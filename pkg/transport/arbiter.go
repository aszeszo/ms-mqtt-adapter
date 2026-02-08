package transport

import (
	"context"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
	"sync"
	"sync/atomic"
	"time"
)

// ArbiterConfig configures the half-duplex bus arbiter.
type ArbiterConfig struct {
	BusQuietTime      time.Duration
	InterMessageDelay time.Duration
	MaxTXWait         time.Duration
}

// ArbiterStats exposes runtime statistics for the bus arbiter.
type ArbiterStats struct {
	QueueLength   int   `json:"queue_length"`
	WaitCount     int64 `json:"wait_count"`
	ForceCount    int64 `json:"force_count"`
	TotalTXWaitMs int64 `json:"total_tx_wait_ms"`
	LastRXAgoMs   int64 `json:"last_rx_ago_ms"`
	LastTXAgoMs   int64 `json:"last_tx_ago_ms"`
}

type sendRequest struct {
	message *mysensors.Message
	result  chan error
}

// ArbiterTransport wraps a Transport and adds half-duplex RS485 bus arbitration.
// It tracks incoming message timestamps and delays outgoing messages until the
// bus has been quiet for a configurable duration.
type ArbiterTransport struct {
	inner  Transport
	config ArbiterConfig
	logger *slog.Logger

	lastRXTime atomic.Int64 // UnixNano of last received message
	lastTXTime atomic.Int64 // UnixNano of last sent message

	sendQueue chan sendRequest
	rxChan    chan *mysensors.Message

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Stats
	statsWaitCount   atomic.Int64
	statsForceCount  atomic.Int64
	statsTotalTXWait atomic.Int64 // cumulative nanoseconds
}

func NewArbiterTransport(inner Transport, config ArbiterConfig, logger *slog.Logger) *ArbiterTransport {
	return &ArbiterTransport{
		inner:     inner,
		config:    config,
		logger:    logger,
		sendQueue: make(chan sendRequest, 256),
		rxChan:    make(chan *mysensors.Message, 1000),
	}
}

func (a *ArbiterTransport) Connect(ctx context.Context) error {
	if err := a.inner.Connect(ctx); err != nil {
		return err
	}

	a.ctx, a.cancel = context.WithCancel(ctx)

	a.wg.Add(2)
	go a.rxProxy()
	go a.sendLoop()

	a.logger.Info("Half-duplex bus arbiter started",
		"bus_quiet_time", a.config.BusQuietTime,
		"inter_message_delay", a.config.InterMessageDelay,
		"max_tx_wait", a.config.MaxTXWait)

	return nil
}

func (a *ArbiterTransport) Disconnect() error {
	if a.cancel != nil {
		a.cancel()
	}

	// Drain pending sends with errors
	a.drainQueue()

	// Wait for goroutines to finish
	a.wg.Wait()

	return a.inner.Disconnect()
}

func (a *ArbiterTransport) Send(message *mysensors.Message) error {
	req := sendRequest{
		message: message,
		result:  make(chan error, 1),
	}

	select {
	case a.sendQueue <- req:
	case <-a.ctx.Done():
		return fmt.Errorf("arbiter: context cancelled")
	default:
		return fmt.Errorf("arbiter: send queue full")
	}

	select {
	case err := <-req.result:
		return err
	case <-a.ctx.Done():
		return fmt.Errorf("arbiter: context cancelled while waiting for send")
	}
}

func (a *ArbiterTransport) Receive() <-chan *mysensors.Message {
	return a.rxChan
}

func (a *ArbiterTransport) IsConnected() bool {
	return a.inner.IsConnected()
}

// Inner returns the wrapped transport for type-assertion during reconfiguration.
func (a *ArbiterTransport) Inner() Transport {
	return a.inner
}

// GetStats returns runtime statistics for the bus arbiter.
func (a *ArbiterTransport) GetStats() ArbiterStats {
	now := time.Now().UnixNano()
	lastRX := a.lastRXTime.Load()
	lastTX := a.lastTXTime.Load()

	var lastRXAgo, lastTXAgo int64
	if lastRX > 0 {
		lastRXAgo = (now - lastRX) / int64(time.Millisecond)
	}
	if lastTX > 0 {
		lastTXAgo = (now - lastTX) / int64(time.Millisecond)
	}

	return ArbiterStats{
		QueueLength:   len(a.sendQueue),
		WaitCount:     a.statsWaitCount.Load(),
		ForceCount:    a.statsForceCount.Load(),
		TotalTXWaitMs: a.statsTotalTXWait.Load() / int64(time.Millisecond),
		LastRXAgoMs:   lastRXAgo,
		LastTXAgoMs:   lastTXAgo,
	}
}

// rxProxy reads from the inner transport's Receive channel, updates the last RX
// timestamp, and forwards messages to the arbiter's receive channel.
func (a *ArbiterTransport) rxProxy() {
	defer a.wg.Done()

	innerChan := a.inner.Receive()
	for {
		select {
		case <-a.ctx.Done():
			return
		case msg, ok := <-innerChan:
			if !ok {
				return
			}
			a.lastRXTime.Store(time.Now().UnixNano())

			select {
			case a.rxChan <- msg:
			case <-a.ctx.Done():
				return
			default:
				a.logger.Warn("Arbiter RX channel full, dropping message", "message", msg.String())
			}
		}
	}
}

// sendLoop processes the send queue, waiting for bus quiet before each transmission.
func (a *ArbiterTransport) sendLoop() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		case req := <-a.sendQueue:
			startWait := time.Now()

			forced := a.waitForQuietBus()

			// Enforce inter-message delay
			if a.config.InterMessageDelay > 0 {
				lastTX := time.Unix(0, a.lastTXTime.Load())
				if elapsed := time.Since(lastTX); elapsed < a.config.InterMessageDelay {
					remaining := a.config.InterMessageDelay - elapsed
					select {
					case <-a.ctx.Done():
						req.result <- fmt.Errorf("arbiter: context cancelled during inter-message delay")
						continue
					case <-time.After(remaining):
					}
				}
			}

			waited := time.Since(startWait)
			if waited > time.Millisecond {
				a.statsWaitCount.Add(1)
				a.statsTotalTXWait.Add(int64(waited))
				a.logger.Debug("Bus arbitration: sending after wait",
					"wait", waited, "forced", forced, "message", req.message.String())
			}

			err := a.inner.Send(req.message)
			if err == nil {
				a.lastTXTime.Store(time.Now().UnixNano())
			}

			req.result <- err
		}
	}
}

// waitForQuietBus polls until the bus has been idle for BusQuietTime,
// or until MaxTXWait is exceeded (in which case it returns true for forced).
func (a *ArbiterTransport) waitForQuietBus() (forced bool) {
	deadline := time.Now().Add(a.config.MaxTXWait)

	for {
		lastRX := time.Unix(0, a.lastRXTime.Load())
		quiet := time.Since(lastRX)

		if quiet >= a.config.BusQuietTime {
			return false
		}

		if time.Now().After(deadline) {
			a.statsForceCount.Add(1)
			a.logger.Warn("Bus arbitration timeout, sending anyway",
				"quiet_duration", quiet, "required", a.config.BusQuietTime)
			return true
		}

		// Sleep for remaining quiet time needed, capped at 10ms
		remaining := a.config.BusQuietTime - quiet
		if remaining > 10*time.Millisecond {
			remaining = 10 * time.Millisecond
		}

		select {
		case <-a.ctx.Done():
			return true
		case <-time.After(remaining):
		}
	}
}

// drainQueue drains pending send requests with errors.
func (a *ArbiterTransport) drainQueue() {
	for {
		select {
		case req := <-a.sendQueue:
			req.result <- fmt.Errorf("arbiter: transport disconnecting")
		default:
			return
		}
	}
}
