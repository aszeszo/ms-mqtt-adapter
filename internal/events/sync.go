package events

import (
	"context"
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

// EntitySync represents a single entity's sync configuration
type EntitySync struct {
	device     config.Device
	entity     config.Entity
	transport  transport.Transport
	mqttClient *mqtt.Client
	config     *config.Config
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// EntitySyncManager manages per-entity sync timers
type EntitySyncManager struct {
	syncs  map[string]*EntitySync // key: unique_id
	mu     sync.RWMutex
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

func NewEntitySyncManager(logger *slog.Logger) *EntitySyncManager {
	return &EntitySyncManager{
		syncs:  make(map[string]*EntitySync),
		logger: logger,
	}
}

func (esm *EntitySyncManager) Start(ctx context.Context, cfg *config.Config, mqttClient *mqtt.Client, transports map[string]transport.Transport) error {
	esm.ctx, esm.cancel = context.WithCancel(ctx)

	syncCount := 0
	for _, device := range cfg.Devices {
		for _, entity := range device.Entities {
			// Only create sync for entities that can receive commands and have sync enabled
			if !entity.CanReceiveCommands() || !entity.IsSyncEnabled() {
				continue
			}

			// Determine which transport to use
			gatewayName := cfg.GetEffectiveGateway(device.Gateway, entity.Gateway)
			transport, exists := transports[gatewayName]
			if !exists {
				esm.logger.Error("No transport found for entity sync", "gateway", gatewayName, "device", device.Name, "entity", entity.Name)
				continue
			}

			uniqueID := entity.GetEffectiveUniqueID(device.ID)

			entitySync := &EntitySync{
				device:     device,
				entity:     entity,
				transport:  transport,
				mqttClient: mqttClient,
				config:     cfg,
				logger:     esm.logger,
				ctx:        esm.ctx,
			}

			esm.mu.Lock()
			esm.syncs[uniqueID] = entitySync
			esm.mu.Unlock()

			go entitySync.syncLoop()
			syncCount++

			esm.logger.Debug("Started entity sync", "device", device.Name, "entity", entity.Name, "period", entity.GetEffectiveSyncPeriod())
		}
	}

	if syncCount > 0 {
		esm.logger.Info("Entity sync manager started", "entity_count", syncCount)
	} else {
		esm.logger.Info("No entities configured for sync")
	}

	return nil
}

func (esm *EntitySyncManager) Stop() {
	if esm.cancel != nil {
		esm.cancel()
	}

	esm.mu.Lock()
	defer esm.mu.Unlock()

	for _, entitySync := range esm.syncs {
		if entitySync.cancel != nil {
			entitySync.cancel()
		}
	}

	esm.logger.Info("Entity sync manager stopped")
}

// Reconfigure updates the sync manager with new configuration
func (esm *EntitySyncManager) Reconfigure(cfg *config.Config, mqttClient *mqtt.Client, transports map[string]transport.Transport) error {
	esm.mu.Lock()
	defer esm.mu.Unlock()

	// Stop all existing syncs
	if esm.cancel != nil {
		esm.cancel()
	}

	for _, entitySync := range esm.syncs {
		if entitySync.cancel != nil {
			entitySync.cancel()
		}
	}

	// Create new context for new syncs
	esm.ctx, esm.cancel = context.WithCancel(context.Background())

	// Clear existing syncs
	esm.syncs = make(map[string]*EntitySync)

	// Start new syncs based on updated configuration
	syncCount := 0
	for _, device := range cfg.Devices {
		for _, entity := range device.Entities {
			// Only create sync for entities that can receive commands and have sync enabled
			if !entity.CanReceiveCommands() || !entity.IsSyncEnabled() {
				continue
			}

			// Determine which transport to use
			gatewayName := cfg.GetEffectiveGateway(device.Gateway, entity.Gateway)
			transport, exists := transports[gatewayName]
			if !exists {
				esm.logger.Error("No transport found for entity sync", "gateway", gatewayName, "device", device.Name, "entity", entity.Name)
				continue
			}

			uniqueID := entity.GetEffectiveUniqueID(device.ID)

			entitySync := &EntitySync{
				device:     device,
				entity:     entity,
				transport:  transport,
				mqttClient: mqttClient,
				config:     cfg,
				logger:     esm.logger,
				ctx:        esm.ctx,
			}

			esm.syncs[uniqueID] = entitySync

			go entitySync.syncLoop()
			syncCount++

			esm.logger.Debug("Started entity sync", "device", device.Name, "entity", entity.Name, "period", entity.GetEffectiveSyncPeriod())
		}
	}

	if syncCount > 0 {
		esm.logger.Info("Entity sync manager reconfigured", "entity_count", syncCount)
	} else {
		esm.logger.Info("No entities configured for sync after reconfiguration")
	}

	return nil
}

func (es *EntitySync) syncLoop() {
	period := es.entity.GetEffectiveSyncPeriod()
	splay := es.entity.GetEffectiveSyncSplay() // Auto-defaults to 10% of period (max 30s)
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	// Initial sync (with splay if configured)
	// Splay spreads sync messages to prevent bus contention when many entities sync
	es.performSyncWithSplay(splay)

	for {
		select {
		case <-es.ctx.Done():
			return
		case <-ticker.C:
			es.performSyncWithSplay(splay)
		}
	}
}

// performSyncWithSplay adds a random delay (0 to splay) before performing sync
func (es *EntitySync) performSyncWithSplay(splay time.Duration) {
	if splay > 0 {
		// Random delay between 0 and splay
		delay := time.Duration(rand.Int63n(int64(splay)))
		es.logger.Debug("Sync splay delay", "device", es.device.Name, "entity", es.entity.Name, "delay", delay)
		select {
		case <-es.ctx.Done():
			return
		case <-time.After(delay):
			// Continue to perform sync
		}
	}
	es.performSync()
}

func (es *EntitySync) performSync() {
	compositeKey := fmt.Sprintf("%s_%s_entity", es.device.ID, es.entity.ID)

	if state, exists := es.mqttClient.GetState(compositeKey); exists {
		nodeID, err := es.config.GetEffectiveEntityNodeID(&es.device, &es.entity)
		if err != nil {
			es.logger.Error("Failed to resolve node ID for sync", "device", es.device.Name, "entity", es.entity.Name, "error", err)
			return
		}

		childID, err := es.config.GetEffectiveChildID(&es.entity)
		if err != nil {
			es.logger.Error("Failed to resolve child ID for sync", "device", es.device.Name, "entity", es.entity.Name, "error", err)
			return
		}

		// Get MySensors variable type for this entity
		varType, _ := config.GetMySensorsVariableTypeForEntity(es.entity.EntityType, es.entity.VariableType)

		requestAck := es.config.GetEffectiveEntityRequestAck(&es.entity)
		message := mysensors.NewSetMessageWithAck(nodeID, childID, varType, state, requestAck)

		if err := es.transport.Send(message); err != nil {
			es.logger.Error("Failed to sync entity state", "error", err,
				"device", es.device.Name, "entity", es.entity.Name, "state", state)
		} else {
			es.logger.Debug("Synced entity state",
				"device", es.device.Name, "entity", es.entity.Name, "state", state)
		}
	} else {
		es.logger.Debug("No state available for entity sync", "device", es.device.Name, "entity", es.entity.Name)
	}
}
