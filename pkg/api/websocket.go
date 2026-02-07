package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"ms-mqtt-adapter/pkg/config"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event types sent over WebSocket.
const (
	EventInitialState            = "initial_state"
	EventEntityStateChanged      = "entity_state_changed"
	EventNodeAvailabilityChanged = "node_availability_changed"
	EventConnectionChanged       = "connection_changed"
	EventConfigReloaded          = "config_reloaded"
	EventPong                    = "pong"
	EventMySensorsMessage        = "mysensors_message"
)

// Event is a typed message sent through the EventBus.
type Event struct {
	Type string `json:"event"`
	Data any    `json:"data,omitempty"`
}

// EventBus broadcasts events to all registered listeners.
type EventBus struct {
	mu        sync.RWMutex
	listeners []chan Event
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

func (eb *EventBus) Subscribe() chan Event {
	ch := make(chan Event, 64)
	eb.mu.Lock()
	eb.listeners = append(eb.listeners, ch)
	eb.mu.Unlock()
	return ch
}

func (eb *EventBus) Unsubscribe(ch chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for i, l := range eb.listeners {
		if l == ch {
			eb.listeners = append(eb.listeners[:i], eb.listeners[i+1:]...)
			close(ch)
			return
		}
	}
}

// Publish sends an event to all listeners (non-blocking; drops if buffer full).
func (eb *EventBus) Publish(evt Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.listeners {
		select {
		case ch <- evt:
		default:
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleWebSocket upgrades the connection and streams events.
func handleWebSocket(provider StatusProvider, bus *EventBus, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", "error", err)
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Send initial state
		initial := buildInitialState(provider)
		if err := conn.WriteJSON(Event{Type: EventInitialState, Data: initial}); err != nil {
			return
		}

		// Subscribe to events
		ch := bus.Subscribe()
		defer bus.Unsubscribe(ch)

		// Read pump (handle ping and refresh from client)
		go func() {
			defer cancel()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				var m struct {
					Event string `json:"event"`
				}
				if json.Unmarshal(msg, &m) == nil {
					switch m.Event {
					case "ping":
						conn.WriteJSON(Event{Type: EventPong})
					case "refresh":
						initial := buildInitialState(provider)
						conn.WriteJSON(Event{Type: EventInitialState, Data: initial})
					}
				}
			}
		}()

		// Write pump
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteJSON(evt); err != nil {
					return
				}
			}
		}
	}
}

type initialState struct {
	MQTT         mqttStatus                `json:"mqtt"`
	Gateways     map[string]gatewayStatus  `json:"gateways"`
	Entities     map[string]string         `json:"entities"`
	ConfigStatus configStatusInfo         `json:"config_status"`
}

type configStatusInfo struct {
	Complete      bool     `json:"complete"`
	MissingFields []string `json:"missing_fields"`
}

// MySensorsTrafficMessage represents a MySensors message for traffic monitoring
type MySensorsTrafficMessage struct {
	Timestamp string `json:"timestamp"`
	Gateway   string `json:"gateway"`
	Direction string `json:"direction"` // "rx" or "tx"
	Raw       string `json:"raw"`
	NodeID    int    `json:"node_id"`
	ChildID   int    `json:"child_id"`
	Type      string `json:"type"`
	Ack       bool   `json:"ack"`
	SubType   int    `json:"sub_type"`
	Payload   string `json:"payload"`
}

type mqttStatus struct {
	Connected bool   `json:"connected"`
	Broker    string `json:"broker"`
	Port      int    `json:"port"`
}

type gatewayStatus struct {
	Connected        bool           `json:"connected"`
	Transport        string         `json:"transport"`
	SeenNodes        []int          `json:"seen_nodes"`
	NodeAvailability map[int]bool   `json:"node_availability"`
	LastSeenNodeID   int            `json:"last_seen_node_id"`
	LastIssuedNodeID int            `json:"last_issued_node_id"`
}

func buildInitialState(p StatusProvider) initialState {
	cfg := p.GetConfig()
	mqtt := p.GetMQTTStatus()

	gateways := make(map[string]gatewayStatus)
	for name, gs := range p.GetAllGatewayStatus() {
		gateways[name] = gatewayStatus{
			Connected:        gs.Connected,
			Transport:        gs.Transport,
			SeenNodes:        gs.SeenNodes,
			NodeAvailability: gs.NodeAvailability,
			LastSeenNodeID:   gs.LastSeenNodeID,
			LastIssuedNodeID: gs.LastIssuedNodeID,
		}
	}

	// Check config completeness
	completeness := config.CheckConfigCompleteness(cfg)

	return initialState{
		MQTT: mqttStatus{
			Connected: mqtt.Connected,
			Broker:    cfg.MQTT.Broker,
			Port:      cfg.MQTT.Port,
		},
		Gateways: gateways,
		Entities: p.GetEntityStates(),
		ConfigStatus: configStatusInfo{
			Complete:      completeness.Complete,
			MissingFields: completeness.MissingFields,
		},
	}
}

// handleTrafficWebSocket streams MySensors traffic messages
func handleTrafficWebSocket(bus *EventBus, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", "error", err)
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Subscribe to MySensors message events only
		ch := bus.Subscribe()
		defer bus.Unsubscribe(ch)

		// Goroutine to read from event bus and send to WebSocket
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				select {
				case <-ctx.Done():
					return
				case evt, ok := <-ch:
					if !ok {
						return
					}
					// Only forward MySensors message events
					if evt.Type == EventMySensorsMessage {
						if err := conn.WriteJSON(evt); err != nil {
							return
						}
					}
				}
			}
		}()

		// Read pump (handle client disconnect)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				break
			}
		}

		<-done
	}
}
