package gateway

import (
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/transport"
	"sort"
	"sync"
	"time"
)

type Gateway struct {
	config     *config.Config
	transport  transport.Transport
	logger     *slog.Logger
	seenNodes  map[int]bool
	nodesMu    sync.RWMutex
	nextNodeID int
}

func NewGateway(cfg *config.Config, transport transport.Transport, logger *slog.Logger) *Gateway {
	return &Gateway{
		config:     cfg,
		transport:  transport,
		logger:     logger,
		seenNodes:  make(map[int]bool),
		nextNodeID: cfg.Gateway.NodeIDRange.Start,
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

	g.logger.Info("Assigned node ID", "assigned_id", nodeID, "requesting_node", message.NodeID)
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

	for nodeID := g.config.Gateway.NodeIDRange.Start; nodeID <= g.config.Gateway.NodeIDRange.End; nodeID++ {
		if !g.seenNodes[nodeID] {
			return nodeID
		}
	}

	return -1
}

func (g *Gateway) trackNode(nodeID int) {
	if nodeID == 0 || nodeID == 255 {
		return
	}

	g.nodesMu.Lock()
	wasNew := !g.seenNodes[nodeID]
	g.seenNodes[nodeID] = true
	g.nodesMu.Unlock()

	if wasNew {
		g.logger.Info("New node discovered", "node_id", nodeID)
		g.printSeenNodes()
	}
}

func (g *Gateway) printSeenNodes() {
	g.nodesMu.RLock()
	defer g.nodesMu.RUnlock()

	var nodeIDs []int
	for nodeID := range g.seenNodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Ints(nodeIDs)

	g.logger.Info("Known node IDs", "nodes", nodeIDs)
}

func (g *Gateway) GetSeenNodes() []int {
	g.nodesMu.RLock()
	defer g.nodesMu.RUnlock()

	var nodeIDs []int
	for nodeID := range g.seenNodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Ints(nodeIDs)

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
