package tcp

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
	"net"
	"sync"
)

type Server struct {
	port      int
	listener  net.Listener
	clients   map[net.Conn]bool
	clientsMu sync.RWMutex
	logger    *slog.Logger
	msgChan   chan *mysensors.Message
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewServer(port int, logger *slog.Logger) *Server {
	return &Server{
		port:    port,
		clients: make(map[net.Conn]bool),
		logger:  logger,
		msgChan: make(chan *mysensors.Message, 100),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	s.listener = listener
	s.logger.Info("TCP server started", "port", s.port)

	go s.acceptConnections()
	return nil
}

func (s *Server) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.clientsMu.Lock()
	for client := range s.clients {
		client.Close()
	}
	s.clientsMu.Unlock()

	s.logger.Info("TCP server stopped")
	return nil
}

func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				s.logger.Error("Failed to accept TCP connection", "error", err)
				continue
			}

			s.clientsMu.Lock()
			s.clients[conn] = true
			s.clientsMu.Unlock()

			s.logger.Info("TCP client connected", "remote", conn.RemoteAddr())
			go s.handleClient(conn)
		}
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
		conn.Close()
		s.logger.Info("TCP client disconnected", "remote", conn.RemoteAddr())
	}()

	scanner := bufio.NewScanner(conn)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					s.logger.Error("Error reading from TCP client", "error", err, "remote", conn.RemoteAddr())
				}
				return
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			message, err := mysensors.ParseMessage(line)
			if err != nil {
				s.logger.Warn("Failed to parse message from TCP client", "error", err, "raw", line, "remote", conn.RemoteAddr())
				continue
			}

			select {
			case s.msgChan <- message:
				s.logger.Debug("Message received from TCP client", "message", message.String(), "remote", conn.RemoteAddr())
			case <-s.ctx.Done():
				return
			default:
				s.logger.Warn("Message channel full, dropping TCP message", "message", message.String())
			}
		}
	}
}

func (s *Server) BroadcastMessage(message *mysensors.Message) {
	msgStr := message.String() + "\n"
	msgBytes := []byte(msgStr)

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for client := range s.clients {
		_, err := client.Write(msgBytes)
		if err != nil {
			s.logger.Error("Failed to send message to TCP client", "error", err, "remote", client.RemoteAddr())
		}
	}
}

func (s *Server) Receive() <-chan *mysensors.Message {
	return s.msgChan
}
