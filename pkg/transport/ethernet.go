package transport

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
	"net"
	"sync"
	"time"
)

type EthernetTransport struct {
	host      string
	port      int
	conn      net.Conn
	connected bool
	mu        sync.RWMutex
	msgChan   chan *mysensors.Message
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *slog.Logger
}

func NewEthernetTransport(host string, port int, logger *slog.Logger) *EthernetTransport {
	return &EthernetTransport{
		host:    host,
		port:    port,
		msgChan: make(chan *mysensors.Message, 100),
		logger:  logger,
	}
}

func (et *EthernetTransport) Connect(ctx context.Context) error {
	et.mu.Lock()
	defer et.mu.Unlock()

	if et.connected {
		return nil
	}

	et.ctx, et.cancel = context.WithCancel(ctx)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", et.host, et.port), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to MySensors gateway: %w", err)
	}

	et.conn = conn
	et.connected = true

	go et.readLoop()

	et.logger.Info("Connected to MySensors Ethernet gateway", "host", et.host, "port", et.port)
	return nil
}

func (et *EthernetTransport) Disconnect() error {
	et.mu.Lock()
	defer et.mu.Unlock()

	if !et.connected {
		return nil
	}

	if et.cancel != nil {
		et.cancel()
	}

	if et.conn != nil {
		et.conn.Close()
	}

	et.connected = false
	et.logger.Info("Disconnected from MySensors Ethernet gateway")
	return nil
}

func (et *EthernetTransport) Send(message *mysensors.Message) error {
	et.mu.RLock()
	defer et.mu.RUnlock()

	if !et.connected || et.conn == nil {
		return fmt.Errorf("not connected to MySensors gateway")
	}

	msgStr := message.String() + "\n"
	_, err := et.conn.Write([]byte(msgStr))
	if err != nil {
		et.logger.Error("Failed to send message to MySensors gateway", "error", err, "message", message.String())
		return fmt.Errorf("failed to send message: %w", err)
	}

	et.logger.Debug("MySensors TX", "message", message.String())
	return nil
}

func (et *EthernetTransport) Receive() <-chan *mysensors.Message {
	return et.msgChan
}

func (et *EthernetTransport) IsConnected() bool {
	et.mu.RLock()
	defer et.mu.RUnlock()
	return et.connected
}

func (et *EthernetTransport) readLoop() {
	defer func() {
		et.mu.Lock()
		et.connected = false
		et.mu.Unlock()
	}()

	scanner := bufio.NewScanner(et.conn)
	for {
		select {
		case <-et.ctx.Done():
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					et.logger.Error("Error reading from MySensors gateway", "error", err)
				}
				return
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			message, err := mysensors.ParseMessage(line)
			if err != nil {
				et.logger.Warn("Failed to parse MySensors message", "error", err, "raw", line)
				continue
			}

			et.logger.Debug("MySensors RX", "message", message.String())

			select {
			case et.msgChan <- message:
			case <-et.ctx.Done():
				return
			default:
				et.logger.Warn("Message channel full, dropping message", "message", message.String())
			}
		}
	}
}
