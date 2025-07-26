package events

import (
	"context"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/mqtt"
	"ms-mqtt-adapter/pkg/transport"
	"time"
)

type SyncManager struct {
	config     *config.Config
	mqttClient *mqtt.Client
	transport  transport.Transport
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewSyncManager(cfg *config.Config, mqttClient *mqtt.Client, transport transport.Transport, logger *slog.Logger) *SyncManager {
	return &SyncManager{
		config:     cfg,
		mqttClient: mqttClient,
		transport:  transport,
		logger:     logger,
	}
}

func (sm *SyncManager) Start(ctx context.Context) error {
	if !sm.config.AdapterTopics.Sync.Enabled {
		sm.logger.Info("Periodic sync disabled")
		return nil
	}

	sm.ctx, sm.cancel = context.WithCancel(ctx)

	go sm.syncLoop()
	sm.logger.Info("Sync manager started", "period", sm.config.AdapterTopics.Sync.Period)
	return nil
}

func (sm *SyncManager) Stop() {
	if sm.cancel != nil {
		sm.cancel()
	}
	sm.logger.Info("Sync manager stopped")
}

func (sm *SyncManager) syncLoop() {
	ticker := time.NewTicker(sm.config.AdapterTopics.Sync.Period)
	defer ticker.Stop()

	sm.performSync()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.performSync()
		}
	}
}

func (sm *SyncManager) performSync() {
	sm.logger.Debug("Starting periodic sync")

	for _, device := range sm.config.Devices {
		for _, relay := range device.Relays {
			compositeKey := fmt.Sprintf("%s_%s", device.ID, relay.ID)
			if state, exists := sm.mqttClient.GetState(compositeKey); exists {
				// State is already in 0/1 format
				nodeID := device.NodeID
				if relay.NodeID != nil {
					nodeID = *relay.NodeID
				}

				message := mysensors.NewSetMessageWithAck(nodeID, relay.ChildID, mysensors.V_STATUS, state, true)

				if err := sm.transport.Send(message); err != nil {
					sm.logger.Error("Failed to sync relay state", "error", err,
						"device", device.Name, "relay", relay.Name, "state", state)
				} else {
					sm.logger.Debug("Synced relay state",
						"device", device.Name, "relay", relay.Name, "state", state)
				}
			}
		}
	}

	sm.logger.Debug("Periodic sync completed")
}

func (sm *SyncManager) SyncDeviceStates() {
	sm.performSync()
}
