package gateway

import (
	"fmt"
	"log/slog"
	"math/rand"
	"ms-mqtt-adapter/internal/mysensors"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/mqtt"
	"ms-mqtt-adapter/pkg/transport"
	"sync"
	"time"
)

type Gateway struct {
	gatewayConfig  *config.GatewayConfig
	gatewayName    string
	appConfig      *config.Config
	transport      transport.Transport
	mqttClient     *mqtt.Client
	logger         *slog.Logger
	seenNodes      map[int]bool
	seenNodesOrder []int // Track order of node discovery
	nodesMu        sync.RWMutex
	nextNodeID     int
	onMessageSent  func(*mysensors.Message) // Callback for sent messages

	// Availability tracking
	availabilityNodes map[int]time.Time // Track last I_TIME/I_DISCOVER_RESPONSE for each node
	availabilityMu    sync.RWMutex

	// Last seen node tracking
	lastSeenNodeID int
	lastSeenMu     sync.RWMutex
}

func NewGateway(gatewayName string, gatewayConfig *config.GatewayConfig, appConfig *config.Config, transport transport.Transport, mqttClient *mqtt.Client, logger *slog.Logger) *Gateway {
	logger.Debug("Creating new gateway", "name", gatewayName, "availability_window", gatewayConfig.AvailabilityWindow)
	return &Gateway{
		gatewayConfig:     gatewayConfig,
		gatewayName:       gatewayName,
		appConfig:         appConfig,
		transport:         transport,
		mqttClient:        mqttClient,
		logger:            logger,
		seenNodes:         make(map[int]bool),
		seenNodesOrder:    make([]int, 0),
		nextNodeID:        gatewayConfig.NodeIDAssignment.NodeIDRange.Start,
		availabilityNodes: make(map[int]time.Time),
	}
}

func (g *Gateway) SetMessageSentCallback(callback func(*mysensors.Message)) {
	g.onMessageSent = callback
}

func (g *Gateway) HandleMessage(message *mysensors.Message) error {
	g.trackNode(message.NodeID)

	// Track availability for ANY message from the node (not just I_TIME/I_DISCOVER_RESPONSE)
	g.trackAvailability(message.NodeID)

	if !message.IsInternal() {
		return nil
	}

	switch message.GetInternalType() {
	case mysensors.I_ID_REQUEST:
		return g.handleIDRequest(message)
	case mysensors.I_TIME:
		return g.handleTimeRequest(message)
	case mysensors.I_DISCOVER_RESPONSE:
		return g.handleDiscoverResponse(message)
	default:
		return nil
	}
}

func (g *Gateway) handleIDRequest(message *mysensors.Message) error {
	// Check if automatic ID assignment is enabled
	if g.gatewayConfig.NodeIDAssignment.Enabled != nil && !*g.gatewayConfig.NodeIDAssignment.Enabled {
		g.logger.Info("Node ID request ignored (automatic assignment disabled)", "requesting_node", message.NodeID)
		return nil
	}

	nodeID := g.assignNodeID()
	if nodeID == -1 {
		g.logger.Warn("No available node IDs", "requesting_node", message.NodeID)
		return fmt.Errorf("no available node IDs")
	}

	response := mysensors.NewInternalMessage(message.NodeID, mysensors.I_ID_RESPONSE, fmt.Sprintf("%d", nodeID))

	if err := g.transport.Send(response); err != nil {
		g.logger.Error("Failed to send ID response", "error", err, "assigned_id", nodeID)
		return err
	}

	// Notify TCP clients about the sent response
	if g.onMessageSent != nil {
		g.onMessageSent(response)
	}

	// Log assignment method used
	assignmentMethod := "sequential"
	if g.gatewayConfig.NodeIDAssignment.RandomIDAssignment != nil && *g.gatewayConfig.NodeIDAssignment.RandomIDAssignment {
		assignmentMethod = "random"
	}

	g.logger.Info("Assigned node ID", "assigned_id", nodeID, "requesting_node", message.NodeID,
		"method", assignmentMethod)
	g.trackNode(nodeID)

	// Publish last issued node ID to MQTT
	if g.mqttClient != nil {
		if err := g.mqttClient.PublishLastIssuedNodeID(g.gatewayName, nodeID); err != nil {
			g.logger.Error("Failed to publish last issued node ID", "error", err)
			// Don't fail the ID assignment if MQTT publish fails
		}
	}

	return nil
}

func (g *Gateway) handleTimeRequest(message *mysensors.Message) error {
	timestamp := time.Now().Unix()
	response := mysensors.NewInternalMessage(message.NodeID, mysensors.I_TIME, fmt.Sprintf("%d", timestamp))

	if err := g.transport.Send(response); err != nil {
		g.logger.Error("Failed to send time response", "error", err, "node", message.NodeID)
		return err
	}

	// Notify TCP clients about the sent response
	if g.onMessageSent != nil {
		g.onMessageSent(response)
	}

	g.logger.Debug("Sent time response", "node", message.NodeID, "timestamp", timestamp)
	return nil
}

func (g *Gateway) handleDiscoverResponse(message *mysensors.Message) error {
	// Log the discovery response
	g.logger.Debug("Received discovery response", "node", message.NodeID, "payload", message.Payload)

	return nil
}

func (g *Gateway) assignNodeID() int {
	g.nodesMu.Lock()
	defer g.nodesMu.Unlock()

	// Check if random ID assignment is enabled
	useRandomAssignment := g.gatewayConfig.NodeIDAssignment.RandomIDAssignment != nil && *g.gatewayConfig.NodeIDAssignment.RandomIDAssignment

	if useRandomAssignment {
		return g.assignRandomNodeID()
	} else {
		return g.assignSequentialNodeID()
	}
}

func (g *Gateway) assignSequentialNodeID() int {
	// Original sequential assignment logic
	for nodeID := g.gatewayConfig.NodeIDAssignment.NodeIDRange.Start; nodeID <= g.gatewayConfig.NodeIDAssignment.NodeIDRange.End; nodeID++ {
		if !g.seenNodes[nodeID] {
			return nodeID
		}
	}
	return -1
}

func (g *Gateway) assignRandomNodeID() int {
	// Build list of available node IDs
	var availableIDs []int
	for nodeID := g.gatewayConfig.NodeIDAssignment.NodeIDRange.Start; nodeID <= g.gatewayConfig.NodeIDAssignment.NodeIDRange.End; nodeID++ {
		if !g.seenNodes[nodeID] {
			availableIDs = append(availableIDs, nodeID)
		}
	}

	// No available IDs
	if len(availableIDs) == 0 {
		return -1
	}

	// Select random ID from available pool
	randomIndex := rand.Intn(len(availableIDs))
	return availableIDs[randomIndex]
}

func (g *Gateway) trackNode(nodeID int) {
	if nodeID == 0 || nodeID == 255 {
		return
	}

	g.nodesMu.Lock()
	wasNew := !g.seenNodes[nodeID]
	if wasNew {
		g.seenNodes[nodeID] = true
		g.seenNodesOrder = append(g.seenNodesOrder, nodeID)
	}
	g.nodesMu.Unlock()

	// Track last seen node
	g.lastSeenMu.Lock()
	g.lastSeenNodeID = nodeID
	g.lastSeenMu.Unlock()

	if wasNew {
		g.logger.Info("New node discovered", "node_id", nodeID)
		g.printSeenNodes()
	}
}

// trackAvailability updates the last seen time for a node when I_TIME or I_DISCOVER_RESPONSE is received
func (g *Gateway) trackAvailability(nodeID int) {
	if nodeID == 0 || nodeID == 255 {
		return
	}

	now := time.Now()
	g.availabilityMu.Lock()
	g.availabilityNodes[nodeID] = now
	g.availabilityMu.Unlock()

	g.logger.Debug("Node availability updated", "node_id", nodeID, "timestamp", now.Unix())
}

// GetNodeAvailabilityStatus returns the availability status for a node
// Returns true if the node has sent a message within the minimum entity availability window for this node
func (g *Gateway) GetNodeAvailabilityStatus(nodeID int) bool {
	if nodeID == 0 || nodeID == 255 {
		return false
	}

	g.availabilityMu.RLock()
	defer g.availabilityMu.RUnlock()

	lastSeen, exists := g.availabilityNodes[nodeID]
	if !exists {
		g.logger.Debug("Node not found in availability tracking", "node_id", nodeID)
		return false
	}

	// Get minimum availability window from entities on this node
	availabilityWindow := g.appConfig.GetMinimumAvailabilityWindowForNode(nodeID, g.gatewayName)

	// Node is available if it has sent a message within the configured availability window
	available := time.Since(lastSeen) < availabilityWindow
	g.logger.Debug("Node availability check", "node_id", nodeID, "last_seen", lastSeen.Unix(), "since_last_seen", time.Since(lastSeen), "window", availabilityWindow, "available", available)
	return available
}

// GetAllNodeAvailabilityStatus returns availability status for all tracked nodes
func (g *Gateway) GetAllNodeAvailabilityStatus() map[int]bool {
	g.availabilityMu.RLock()
	defer g.availabilityMu.RUnlock()

	status := make(map[int]bool)
	now := time.Now()
	for nodeID, lastSeen := range g.availabilityNodes {
		// Get minimum availability window from entities on this node
		availabilityWindow := g.appConfig.GetMinimumAvailabilityWindowForNode(nodeID, g.gatewayName)

		// Node is available if it has sent a message within the configured availability window
		isAvailable := now.Sub(lastSeen) < availabilityWindow
		status[nodeID] = isAvailable
		g.logger.Debug("Node availability status", "node_id", nodeID, "last_seen", lastSeen.Unix(), "now", now.Unix(), "window", availabilityWindow, "available", isAvailable)
	}
	return status
}

func (g *Gateway) printSeenNodes() {
	g.nodesMu.RLock()
	defer g.nodesMu.RUnlock()

	// Use discovery order instead of sorting
	g.logger.Info("Known node IDs", "nodes", g.seenNodesOrder)
}

func (g *Gateway) GetSeenNodes() []int {
	g.nodesMu.RLock()
	defer g.nodesMu.RUnlock()

	// Return a copy to avoid race conditions
	nodeIDs := make([]int, len(g.seenNodesOrder))
	copy(nodeIDs, g.seenNodesOrder)

	return nodeIDs
}

// GetLastSeenNodeID returns the most recently active node ID, or 0 if none.
func (g *Gateway) GetLastSeenNodeID() int {
	g.lastSeenMu.RLock()
	defer g.lastSeenMu.RUnlock()
	return g.lastSeenNodeID
}

func (g *Gateway) SendHeartbeatRequest() error {
	message := mysensors.NewInternalMessage(0, mysensors.I_HEARTBEAT_REQUEST, "")

	if err := g.transport.Send(message); err != nil {
		g.logger.Error("Failed to send heartbeat request", "error", err)
		return err
	}

	// Notify TCP clients about the sent heartbeat request
	if g.onMessageSent != nil {
		g.onMessageSent(message)
	}

	g.logger.Debug("Sent heartbeat request to gateway")
	return nil
}

// Reconfigure updates the gateway configuration
func (g *Gateway) Reconfigure(gatewayConfig *config.GatewayConfig) {
	g.nodesMu.Lock()
	defer g.nodesMu.Unlock()

	g.gatewayConfig = gatewayConfig

	// Reset nextNodeID to start of range
	g.nextNodeID = gatewayConfig.NodeIDAssignment.NodeIDRange.Start

	g.logger.Info("Gateway reconfigured",
		"node_id_range", fmt.Sprintf("%d-%d", gatewayConfig.NodeIDAssignment.NodeIDRange.Start, gatewayConfig.NodeIDAssignment.NodeIDRange.End),
		"node_id_assignment_enabled", gatewayConfig.NodeIDAssignment.Enabled != nil && *gatewayConfig.NodeIDAssignment.Enabled,
		"random_id_assignment", gatewayConfig.NodeIDAssignment.RandomIDAssignment != nil && *gatewayConfig.NodeIDAssignment.RandomIDAssignment,
		"availability_window", gatewayConfig.AvailabilityWindow)
}
