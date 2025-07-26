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
		for _, entity := range device.Entities {
			// Only sync entities that can receive commands (actuators)
			if !entity.CanReceiveCommands() {
				continue
			}
			
			compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
			if state, exists := sm.mqttClient.GetState(compositeKey); exists {
				nodeID := device.NodeID
				if entity.NodeID != nil {
					nodeID = *entity.NodeID
				}

				// Get MySensors variable type for this entity
				varType, _ := config.GetMySensorsVariableTypeForEntity(entity.EntityType, entity.VariableType)

				requestAck := sm.config.GetEffectiveRequestAck(&device)
				message := mysensors.NewSetMessageWithAck(nodeID, entity.ChildID, varType, state, requestAck)

				if err := sm.transport.Send(message); err != nil {
					sm.logger.Error("Failed to sync entity state", "error", err,
						"device", device.Name, "entity", entity.Name, "state", state)
				} else {
					sm.logger.Debug("Synced entity state",
						"device", device.Name, "entity", entity.Name, "state", state)
				}
			}
		}
	}

	sm.logger.Debug("Periodic sync completed")
}

func (sm *SyncManager) SyncDeviceStates() {
	sm.performSync()
}
