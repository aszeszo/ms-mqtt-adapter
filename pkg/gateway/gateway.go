package gateway

import (
	"fmt"
	"log/slog"
	"math/rand"
	"ms-mqtt-adapter/internal/mysensors"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/transport"
	"sync"
	"time"
)

type Gateway struct {
	gatewayConfig *config.GatewayConfig
	transport     transport.Transport
	logger        *slog.Logger
	seenNodes     map[int]bool
	seenNodesOrder []int // Track order of node discovery
	nodesMu       sync.RWMutex
	nextNodeID    int
}

func NewGateway(gatewayConfig *config.GatewayConfig, transport transport.Transport, logger *slog.Logger) *Gateway {
	return &Gateway{
		gatewayConfig:  gatewayConfig,
		transport:      transport,
		logger:         logger,
		seenNodes:      make(map[int]bool),
		seenNodesOrder: make([]int, 0),
		nextNodeID:     gatewayConfig.NodeIDRange.Start,
	}
}

func (g *Gateway) HandleMessage(message *mysensors.Message) error {
	g.trackNode(message.NodeID)

	if !message.IsInternal() {
		return nil
	}

	switch message.GetInternalType() {
	case mysensors.I_ID_REQUEST:
		return g.handleIDRequest(message)
	case mysensors.I_TIME:
		return g.handleTimeRequest(message)
	default:
		return nil
	}
}

func (g *Gateway) handleIDRequest(message *mysensors.Message) error {
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

	// Log assignment method used
	assignmentMethod := "sequential"
	if g.gatewayConfig.RandomIDAssignment != nil && *g.gatewayConfig.RandomIDAssignment {
		assignmentMethod = "random"
	}
	
	g.logger.Info("Assigned node ID", "assigned_id", nodeID, "requesting_node", message.NodeID, 
		"method", assignmentMethod)
	g.trackNode(nodeID)
	return nil
}

func (g *Gateway) handleTimeRequest(message *mysensors.Message) error {
	timestamp := time.Now().Unix()
	response := mysensors.NewInternalMessage(message.NodeID, mysensors.I_TIME, fmt.Sprintf("%d", timestamp))

	if err := g.transport.Send(response); err != nil {
		g.logger.Error("Failed to send time response", "error", err, "node", message.NodeID)
		return err
	}

	g.logger.Debug("Sent time response", "node", message.NodeID, "timestamp", timestamp)
	return nil
}

func (g *Gateway) assignNodeID() int {
	g.nodesMu.Lock()
	defer g.nodesMu.Unlock()

	// Check if random ID assignment is enabled
	useRandomAssignment := g.gatewayConfig.RandomIDAssignment != nil && *g.gatewayConfig.RandomIDAssignment

	if useRandomAssignment {
		return g.assignRandomNodeID()
	} else {
		return g.assignSequentialNodeID()
	}
}

func (g *Gateway) assignSequentialNodeID() int {
	// Original sequential assignment logic
	for nodeID := g.gatewayConfig.NodeIDRange.Start; nodeID <= g.gatewayConfig.NodeIDRange.End; nodeID++ {
		if !g.seenNodes[nodeID] {
			return nodeID
		}
	}
	return -1
}

func (g *Gateway) assignRandomNodeID() int {
	// Build list of available node IDs
	var availableIDs []int
	for nodeID := g.gatewayConfig.NodeIDRange.Start; nodeID <= g.gatewayConfig.NodeIDRange.End; nodeID++ {
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

	if wasNew {
		g.logger.Info("New node discovered", "node_id", nodeID)
		g.printSeenNodes()
	}
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

func (g *Gateway) SendVersionRequest() error {
	message := mysensors.NewInternalMessage(0, mysensors.I_VERSION, "")

	if err := g.transport.Send(message); err != nil {
		g.logger.Error("Failed to send version request", "error", err)
		return err
	}

	g.logger.Debug("Sent version request to gateway")
	return nil
}
