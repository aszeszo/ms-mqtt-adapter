package transport

import (
	"context"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
)

type RS485Transport struct {
	device   string
	baudRate int
	logger   *slog.Logger
	msgChan  chan *mysensors.Message
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
	return fmt.Errorf("RS485 transport not implemented yet")
}

func (rt *RS485Transport) Disconnect() error {
	return fmt.Errorf("RS485 transport not implemented yet")
}

func (rt *RS485Transport) Send(message *mysensors.Message) error {
	return fmt.Errorf("RS485 transport not implemented yet")
}

func (rt *RS485Transport) Receive() <-chan *mysensors.Message {
	return rt.msgChan
}

func (rt *RS485Transport) IsConnected() bool {
	return false
}
