package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
	"os"
	"sync"
	"time"
	
	"github.com/tarm/serial"
)

type RS485Transport struct {
	device    string
	baudRate  int
	port      io.ReadWriteCloser
	connected bool
	mu        sync.RWMutex
	msgChan   chan *mysensors.Message
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *slog.Logger
}

func NewRS485Transport(device string, baudRate int, logger *slog.Logger) *RS485Transport {
	return &RS485Transport{
		device:   device,
		baudRate: baudRate,
		logger:   logger,
		msgChan:  make(chan *mysensors.Message, 100),
	}
}

func (rt *RS485Transport) Connect(ctx context.Context) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.connected {
		return nil
	}

	rt.ctx, rt.cancel = context.WithCancel(ctx)

	// Check if device exists
	if _, err := os.Stat(rt.device); os.IsNotExist(err) {
		return fmt.Errorf("RS485 device does not exist: %s", rt.device)
	}

	config := &serial.Config{
		Name:        rt.device,
		Baud:        rt.baudRate,
		ReadTimeout: time.Second * 1,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		return fmt.Errorf("failed to open RS485 port: %w", err)
	}

	rt.port = port
	rt.connected = true

	go rt.readLoop()

	rt.logger.Info("Connected to MySensors RS485 gateway", "device", rt.device, "baud", rt.baudRate)
	return nil
}

func (rt *RS485Transport) Disconnect() error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if !rt.connected {
		return nil
	}

	if rt.cancel != nil {
		rt.cancel()
	}

	if rt.port != nil {
		rt.port.Close()
	}

	rt.connected = false
	rt.logger.Info("Disconnected from MySensors RS485 gateway")
	return nil
}

func (rt *RS485Transport) Send(message *mysensors.Message) error {
	rt.mu.RLock()
	port := rt.port
	connected := rt.connected
	rt.mu.RUnlock()

	if !connected || port == nil {
		return fmt.Errorf("not connected to MySensors RS485 gateway")
	}

	msgStr := message.String() + "\n"
	_, err := port.Write([]byte(msgStr))
	if err != nil {
		rt.logger.Error("Failed to send message to MySensors RS485 gateway", "error", err, "message", message.String())
		// Mark as disconnected on write error
		rt.mu.Lock()
		rt.connected = false
		rt.mu.Unlock()
		return fmt.Errorf("failed to send message: %w", err)
	}

	rt.logger.Debug("MySensors RS485 TX", "message", message.String())
	return nil
}

func (rt *RS485Transport) Receive() <-chan *mysensors.Message {
	return rt.msgChan
}

func (rt *RS485Transport) IsConnected() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.connected
}

func (rt *RS485Transport) readLoop() {
	defer func() {
		rt.mu.Lock()
		rt.connected = false
		if rt.port != nil {
			rt.port.Close()
		}
		rt.mu.Unlock()
		rt.logger.Warn("MySensors RS485 gateway connection lost", "device", rt.device)
	}()

	scanner := bufio.NewScanner(rt.port)
	for {
		select {
		case <-rt.ctx.Done():
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					rt.logger.Error("Error reading from MySensors RS485 gateway", "error", err)
				} else {
					rt.logger.Info("MySensors RS485 gateway closed connection")
				}
				return
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			message, err := mysensors.ParseMessage(line)
			if err != nil {
				rt.logger.Warn("Failed to parse MySensors message", "error", err, "raw", line)
				continue
			}

			rt.logger.Debug("MySensors RS485 RX", "message", message.String())

			select {
			case rt.msgChan <- message:
			case <-rt.ctx.Done():
				return
			default:
				rt.logger.Warn("Message channel full, dropping message", "message", message.String())
			}
		}
	}
}
