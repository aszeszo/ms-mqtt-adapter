package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"ms-mqtt-adapter/internal/events"
	"ms-mqtt-adapter/internal/mysensors"
	"ms-mqtt-adapter/pkg/api"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/gateway"
	"ms-mqtt-adapter/pkg/mqtt"
	"ms-mqtt-adapter/pkg/tcp"
	"ms-mqtt-adapter/pkg/transport"
	"ms-mqtt-adapter/web"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Configuration file path")
	ingressPort := flag.Int("ingress-port", 8099, "HTTP server port for web UI / ingress")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger, logBroadcast := events.NewBroadcastLogger(cfg.LogLevel, os.Stdout)
	logger.Info("Starting ms-mqtt-adapter", "version", "2.2.2")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app := &Application{
		config:       cfg,
		logger:       logger,
		logBroadcast: logBroadcast,
		configPath:   *configFile,
		ingressPort:  *ingressPort,
		eventBus:     api.NewEventBus(),
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("Application failed", "error", err)
		os.Exit(1)
	}

	logger.Info("ms-mqtt-adapter stopped")
}

// PendingAck tracks a pending ACK request for a MySensors message
type PendingAck struct {
	NodeID    int
	ChildID   int
	VarType   mysensors.VariableType
	Payload   string
	AckChan   chan bool        // Signals when ACK received
	CancelChan chan struct{}   // Signals cancellation (new request arrived)
}

type Application struct {
	config       *config.Config
	logger       *slog.Logger
	logBroadcast *events.BroadcastHandler
	transports   map[string]transport.Transport // gatewayName -> transport
	mqttClient   *mqtt.Client
	tcpServers   map[string]*tcp.Server      // gatewayName -> tcpServer
	gateways     map[string]*gateway.Gateway // gatewayName -> gateway
	syncMgr      *events.EntitySyncManager
	watcher      *fsnotify.Watcher
	ctx          context.Context

	// HTTP / Web UI
	configPath  string
	ingressPort int
	httpServer  *api.Server
	eventBus    *api.EventBus

	// Connection retry management
	transportRetryCount map[string]int
	mqttRetryCount      int
	retryMu             sync.RWMutex

	// ACK tracking for entities with request_ack enabled
	pendingAcks   map[string]*PendingAck  // compositeKey -> pending ACK
	pendingAcksMu sync.Mutex
}

// --- StatusProvider interface implementation ---

func (app *Application) GetConfig() *config.Config     { return app.config }
func (app *Application) GetConfigPath() string          { return app.configPath }

func (app *Application) GetMQTTStatus() api.MQTTStatus {
	connected := false
	if app.mqttClient != nil {
		connected = app.mqttClient.IsConnected()
	}
	return api.MQTTStatus{Connected: connected}
}

func (app *Application) GetMQTTClient() api.MQTTClientProvider {
	if app.mqttClient == nil {
		return nil
	}
	return app.mqttClient
}

func (app *Application) GetTransportStatus() map[string]api.TransportStatus {
	result := make(map[string]api.TransportStatus)
	for name, t := range app.transports {
		transportType := ""
		if gwCfg, ok := app.config.MySensors[name]; ok {
			transportType = gwCfg.Transport
		}
		result[name] = api.TransportStatus{
			Connected: t.IsConnected(),
			Transport: transportType,
		}
	}
	return result
}

func (app *Application) GetGatewayStatus(name string) api.GatewayStatus {
	gw, exists := app.gateways[name]
	if !exists {
		return api.GatewayStatus{}
	}
	transportType := ""
	connected := false
	if gwCfg, ok := app.config.MySensors[name]; ok {
		transportType = gwCfg.Transport
	}
	if t, ok := app.transports[name]; ok {
		connected = t.IsConnected()
	}
	return api.GatewayStatus{
		Connected:        connected,
		Transport:        transportType,
		SeenNodes:        gw.GetSeenNodes(),
		NodeAvailability: gw.GetAllNodeAvailabilityStatus(),
		LastSeenNodeID:   gw.GetLastSeenNodeID(),
		LastIssuedNodeID: gw.GetLastIssuedNodeID(),
	}
}

func (app *Application) GetAllGatewayStatus() map[string]api.GatewayStatus {
	result := make(map[string]api.GatewayStatus)
	for name := range app.gateways {
		result[name] = app.GetGatewayStatus(name)
	}
	return result
}

func (app *Application) GetEntityStates() map[string]string {
	if app.mqttClient == nil {
		return make(map[string]string)
	}
	return app.mqttClient.GetAllStates()
}

// calculateBackoffDelay calculates exponential backoff delay with jitter
func (app *Application) calculateBackoffDelay(retryCount int) time.Duration {
	// Base delay of 2 seconds, max 5 minutes
	baseDelay := 2.0
	maxDelay := 300.0 // 5 minutes

	delay := baseDelay * math.Pow(2, float64(retryCount))
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add some jitter (±25%)
	jitter := delay * 0.25 * (2*float64(time.Now().UnixNano()%1000)/1000.0 - 1)

	return time.Duration((delay + jitter) * float64(time.Second))
}

// retryWithBackoff executes a function with exponential backoff retry logic
func (app *Application) retryWithBackoff(ctx context.Context, operation string, maxRetries int, fn func() error) error {
	var lastErr error
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(); err != nil {
			lastErr = err

			// Check if we've exceeded max retries (if maxRetries >= 0)
			if maxRetries >= 0 && attempt >= maxRetries {
				break
			}

			delay := app.calculateBackoffDelay(attempt)
			if maxRetries >= 0 {
				app.logger.Warn("Operation failed, retrying",
					"operation", operation,
					"attempt", attempt+1,
					"max_attempts", maxRetries+1,
					"retry_in", delay,
					"error", err)
			} else {
				app.logger.Warn("Operation failed, retrying",
					"operation", operation,
					"attempt", attempt+1,
					"retry_in", delay,
					"error", err)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				attempt++
				continue
			}
		} else {
			if attempt > 0 {
				app.logger.Info("Operation succeeded after retries",
					"operation", operation,
					"attempts", attempt+1)
			}
			return nil
		}
	}

	return fmt.Errorf("operation '%s' failed after %d attempts: %w", operation, maxRetries+1, lastErr)
}

// Helper methods for backward compatibility during refactoring
func (app *Application) getDefaultTransport() transport.Transport {
	if app.transports != nil {
		if t, exists := app.transports["default"]; exists {
			return t
		}
		// Return any transport if default doesn't exist
		for _, t := range app.transports {
			return t
		}
	}
	return nil
}

func (app *Application) getDefaultGateway() *gateway.Gateway {
	if app.gateways != nil {
		defaultGatewayName := app.config.GetDefaultGatewayName()
		if g, exists := app.gateways[defaultGatewayName]; exists {
			return g
		}
		// Fallback: return any gateway if default doesn't exist
		for _, g := range app.gateways {
			return g
		}
	}
	return nil
}

func (app *Application) Run(ctx context.Context) error {
	// Store context for use in config reloading
	app.ctx = ctx

	// Initialize retry counters
	app.transportRetryCount = make(map[string]int)
	app.mqttRetryCount = 0

	// Initialize ACK tracking
	app.pendingAcks = make(map[string]*PendingAck)

	// Initialize HTTP server FIRST so the web UI / ingress is always available,
	// even if transport or MQTT connections fail during startup.
	if err := app.initializeHTTPServer(); err != nil {
		return fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	if err := app.initializeTransports(); err != nil {
		return fmt.Errorf("failed to initialize transports: %w", err)
	}

	if err := app.initializeMQTT(); err != nil {
		return fmt.Errorf("failed to initialize MQTT: %w", err)
	}

	if err := app.initializeTCPServers(); err != nil {
		return fmt.Errorf("failed to initialize TCP servers: %w", err)
	}

	if err := app.initializeGateways(); err != nil {
		return fmt.Errorf("failed to initialize gateways: %w", err)
	}

	if err := app.initializeSyncManager(); err != nil {
		return fmt.Errorf("failed to initialize sync manager: %w", err)
	}

	// Start file watcher for config reloading
	if err := app.startConfigWatcher(app.configPath); err != nil {
		app.logger.Warn("Failed to start config watcher", "error", err)
	}

	// Start config file watcher goroutine early so config changes are picked up
	// even if startWithRetry hasn't completed (e.g. empty config on first run)
	go app.handleConfigReloads(ctx)

	// Only start connections if we have something to connect to
	if len(app.config.MySensors) > 0 && app.config.MQTT.Broker != "" {
		if err := app.startWithRetry(ctx); err != nil {
			return fmt.Errorf("failed to start application: %w", err)
		}

		// Always publish Home Assistant discovery topics on startup
		if err := app.publishDiscovery(); err != nil {
			return fmt.Errorf("failed to publish discovery: %w", err)
		}

		// Send initial heartbeat request to all gateways (if heartbeat requests are enabled)
		defaultGatewayName := app.config.GetDefaultGatewayName()
		if defaultGatewayConfig, ok := app.config.MySensors[defaultGatewayName]; ok {
			if defaultGatewayConfig.Gateway.HeartbeatRequestPeriod > 0 {
				app.logger.Info("Sending initial heartbeat requests to gateways")
				for gatewayName, gw := range app.gateways {
					if err := gw.SendHeartbeatRequest(); err != nil {
						app.logger.Error("Failed to send initial heartbeat request", "gateway", gatewayName, "error", err)
					}
				}
			} else {
				app.logger.Info("Heartbeat requests disabled (heartbeat_request_period=0)")
			}
		}

		go app.handleMySensorsMessages()
		go app.handleTCPMessages()
		go app.handleMQTTStateChanges()
		go app.periodicHeartbeatRequest(ctx)
		go app.availabilityMonitor(ctx)
	} else {
		app.logger.Info("Empty or incomplete config - waiting for configuration via web UI")
	}

	app.logger.Info("ms-mqtt-adapter started successfully")

	<-ctx.Done()
	app.logger.Info("Shutting down...")

	return app.shutdown()
}

func (app *Application) initializeTransports() error {
	app.transports = make(map[string]transport.Transport)

	for gatewayName, gatewayConfig := range app.config.MySensors {
		var t transport.Transport
		switch gatewayConfig.Transport {
		case "ethernet":
			t = transport.NewEthernetTransport(
				gatewayConfig.Ethernet.Host,
				gatewayConfig.Ethernet.Port,
				app.logger,
			)
		case "rs485":
			t = transport.NewRS485Transport(
				gatewayConfig.RS485.Device,
				9600,
				app.logger,
			)
		default:
			return fmt.Errorf("unsupported transport type for gateway %s: %s", gatewayName, gatewayConfig.Transport)
		}
		app.transports[gatewayName] = t
	}

	return nil
}

func (app *Application) initializeMQTT() error {
	app.mqttClient = mqtt.NewClient(&app.config.MQTT, &app.config.AdapterTopics, app.config.Devices, app.logger)
	return nil
}

func (app *Application) initializeTCPServers() error {
	app.tcpServers = make(map[string]*tcp.Server)

	for gatewayName, gatewayConfig := range app.config.MySensors {
		if gatewayConfig.TCPService.Enabled {
			app.tcpServers[gatewayName] = tcp.NewServer(gatewayConfig.TCPService.Port, app.logger)
		}
	}
	return nil
}

func (app *Application) initializeGateways() error {
	app.gateways = make(map[string]*gateway.Gateway)

	for gatewayName, gatewayConfig := range app.config.MySensors {
		gatewayTransport := app.transports[gatewayName]
		if gatewayTransport == nil {
			return fmt.Errorf("no transport found for gateway %s", gatewayName)
		}

		// Create a gateway config for this specific gateway
		gatewayConf := &gatewayConfig.Gateway

		gateway := gateway.NewGateway(gatewayName, gatewayConf, app.config, gatewayTransport, app.mqttClient, app.logger)

		// Set up callback to forward outgoing messages to TCP clients and traffic monitor
		if tcpServer, exists := app.tcpServers[gatewayName]; exists {
			gateway.SetMessageSentCallback(func(message *mysensors.Message) {
				tcpServer.BroadcastMessage(message)
				app.publishMySensorsTraffic(gatewayName, "tx", message)
			})
		} else {
			// No TCP server, only publish to traffic monitor
			gateway.SetMessageSentCallback(func(message *mysensors.Message) {
				app.publishMySensorsTraffic(gatewayName, "tx", message)
			})
		}

		app.gateways[gatewayName] = gateway
	}
	return nil
}

func (app *Application) initializeSyncManager() error {
	app.syncMgr = events.NewEntitySyncManager(app.logger)
	return nil
}

func (app *Application) initializeHTTPServer() error {
	staticFS, err := fs.Sub(web.StaticFiles, "dist")
	if err != nil {
		return fmt.Errorf("failed to access embedded static files: %w", err)
	}
	app.httpServer = api.NewServer(app.ingressPort, app, app.eventBus, app.logBroadcast, staticFS, app.logger)
	return app.httpServer.Start(app.ctx)
}

func (app *Application) startWithRetry(ctx context.Context) error {
	// Start connection attempts concurrently
	var wg sync.WaitGroup
	errors := make(chan error, len(app.transports)+1) // +1 for MQTT

	// Connect MQTT with retry
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := app.retryWithBackoff(ctx, "MQTT connection", -1, func() error { // -1 = infinite retries
			return app.mqttClient.Connect(ctx)
		})
		if err != nil {
			errors <- fmt.Errorf("MQTT connection failed permanently: %w", err)
		} else {
			app.logger.Info("MQTT connected successfully")
		}
	}()

	// Connect all transports with retry
	for gatewayName, gatewayTransport := range app.transports {
		wg.Add(1)
		go func(name string, transport transport.Transport) {
			defer wg.Done()
			err := app.retryWithBackoff(ctx, fmt.Sprintf("MySensors gateway '%s'", name), -1, func() error { // -1 = infinite retries
				return transport.Connect(ctx)
			})
			if err != nil {
				errors <- fmt.Errorf("transport connection failed permanently for gateway %s: %w", name, err)
			} else {
				app.logger.Info("MySensors gateway connected successfully", "gateway", name)
			}
		}(gatewayName, gatewayTransport)
	}

	// Start TCP servers (these don't need retry logic as they just bind to ports)
	for gatewayName, tcpServer := range app.tcpServers {
		if err := tcpServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start TCP server for gateway %s: %w", gatewayName, err)
		}
		app.logger.Info("TCP server started", "gateway", gatewayName, "port", tcpServer.Port())
	}

	// Wait for initial connections or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errors:
		// If we get a permanent error, return it
		return err
	case <-done:
		// All connections succeeded
	}

	// Start sync manager (only after connections are established)
	if err := app.syncMgr.Start(ctx, app.config, app.mqttClient, app.transports); err != nil {
		return fmt.Errorf("failed to start sync manager: %w", err)
	}

	// Start connection monitoring and auto-reconnection
	app.startConnectionMonitoring(ctx)

	app.logger.Info("Application started successfully")
	return nil
}

func (app *Application) startConnectionMonitoring(ctx context.Context) {
	// Monitor MySensors transport connections
	for gatewayName, gatewayTransport := range app.transports {
		go func(name string, transport transport.Transport) {
			ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if !transport.IsConnected() {
						app.logger.Info("MySensors gateway disconnected, attempting reconnection", "gateway", name)
						if app.eventBus != nil {
							app.eventBus.Publish(api.Event{
								Type: api.EventConnectionChanged,
								Data: map[string]any{"type": "gateway", "name": name, "connected": false},
							})
						}

						// Attempt reconnection with retry
						err := app.retryWithBackoff(ctx, fmt.Sprintf("MySensors gateway '%s' reconnection", name), -1, func() error {
							return transport.Connect(ctx)
						})

						if err != nil {
							app.logger.Error("Failed to reconnect MySensors gateway", "gateway", name, "error", err)
						} else {
							app.logger.Info("MySensors gateway reconnected successfully", "gateway", name)
							if app.eventBus != nil {
								app.eventBus.Publish(api.Event{
									Type: api.EventConnectionChanged,
									Data: map[string]any{"type": "gateway", "name": name, "connected": true},
								})
							}
						}
					}
				}
			}
		}(gatewayName, gatewayTransport)
	}

	// Monitor MQTT connection
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !app.mqttClient.IsConnected() {
					app.logger.Info("MQTT broker disconnected, attempting reconnection")
					if app.eventBus != nil {
						app.eventBus.Publish(api.Event{
							Type: api.EventConnectionChanged,
							Data: map[string]any{"type": "mqtt", "connected": false},
						})
					}

					// Attempt reconnection with retry
					err := app.retryWithBackoff(ctx, "MQTT reconnection", -1, func() error {
						return app.mqttClient.Connect(ctx)
					})

					if err != nil {
						app.logger.Error("Failed to reconnect MQTT broker", "error", err)
					} else {
						app.logger.Info("MQTT broker reconnected successfully")
						if app.eventBus != nil {
							app.eventBus.Publish(api.Event{
								Type: api.EventConnectionChanged,
								Data: map[string]any{"type": "mqtt", "connected": true},
							})
						}
					}
				}
			}
		}
	}()
}

func (app *Application) publishDiscovery() error {
	for _, device := range app.config.Devices {
		if err := app.mqttClient.PublishHomeAssistantDiscovery(device); err != nil {
			return fmt.Errorf("failed to publish discovery for device %s: %w", device.Name, err)
		}
		app.logger.Info("Published Home Assistant discovery", "device", device.Name)
	}

	// Publish seen nodes for each gateway separately (no global tracking)
	for gatewayName, gateway := range app.gateways {
		gatewaySeenNodes := gateway.GetSeenNodes() // Already returns []int

		// Publish gateway-specific seen nodes
		if err := app.mqttClient.PublishGatewayAdapterStatus(app.config.AdapterTopics.TopicPrefix, gatewayName, gatewaySeenNodes); err != nil {
			return fmt.Errorf("failed to publish gateway adapter status for %s: %w", gatewayName, err)
		}
	}

	return nil
}

func (app *Application) handleMySensorsMessages() {
	// Start a goroutine for each transport
	for gatewayName, gatewayTransport := range app.transports {
		go func(gName string, t transport.Transport) {
			for message := range t.Receive() {
				app.logger.Info("Received MySensors message", "gateway", gName, "message", message.String())
				app.logger.Debug("MySensors message details", "gateway", gName,
					"node_id", message.NodeID, "child_id", message.ChildID, "msg_type", message.MessageType,
					"is_set", message.IsSet(), "is_req", message.IsReq(), "is_internal", message.IsInternal(), "var_type", message.GetVariableType())

				// Broadcast MySensors message for traffic monitoring
				app.publishMySensorsTraffic(gName, "rx", message)

				// Broadcast to corresponding TCP server
				if tcpServer, exists := app.tcpServers[gName]; exists {
					tcpServer.BroadcastMessage(message)
				}

				// Handle message with corresponding gateway
				if gateway, exists := app.gateways[gName]; exists {
					if err := gateway.HandleMessage(message); err != nil {
						app.logger.Error("Gateway message handling failed", "gateway", gName, "error", err, "message", message.String())
					}
					
					// Log availability-related messages
					if message.IsInternal() {
						internalType := message.GetInternalType()
						if internalType == mysensors.I_TIME || internalType == mysensors.I_DISCOVER_RESPONSE {
							app.logger.Debug("Node availability message received", "gateway", gName, "node_id", message.NodeID, "message_type", internalType)
						}
					}
				}

				// Check if this message is an ACK response for a pending request
				app.checkForAckResponse(message)

				app.handleDeviceMessage(message)

				// Handle presentation messages for deduplication tracking
				if message.IsPresentation() {
					sensorType := message.GetSensorType()
					description := message.Payload
					if err := app.mqttClient.PublishPresentationMessage(app.config.AdapterTopics.TopicPrefix, gName, message.NodeID, message.ChildID, fmt.Sprintf("S_%d", int(sensorType)), description); err != nil {
						app.logger.Error("Failed to publish presentation message", "gateway", gName, "error", err, "node_id", message.NodeID, "child_id", message.ChildID)
					} else {
						app.logger.Debug("Published presentation message", "gateway", gName, "node_id", message.NodeID, "child_id", message.ChildID, "sensor_type", sensorType, "description", description)
					}
				}

				// Publish gateway-specific status only (no global tracking)
				if gateway, exists := app.gateways[gName]; exists {
					gatewaySeenNodes := gateway.GetSeenNodes() // Already returns []int

					// Publish gateway-specific seen nodes
					if err := app.mqttClient.PublishGatewayAdapterStatus(app.config.AdapterTopics.TopicPrefix, gName, gatewaySeenNodes); err != nil {
						app.logger.Error("Failed to publish gateway adapter status", "gateway", gName, "error", err)
					}
				}
			}
		}(gatewayName, gatewayTransport)
	}
}

func (app *Application) handleTCPMessages() {
	// Start a goroutine for each TCP server
	for gatewayName, tcpServer := range app.tcpServers {
		go func(gName string, server *tcp.Server) {
			for message := range server.Receive() {
				app.logger.Info("Received message from TCP client", "gateway", gName, "message", message.String())

				// Forward to MySensors gateway
				if gatewayTransport, exists := app.transports[gName]; exists {
					if err := gatewayTransport.Send(message); err != nil {
						app.logger.Error("Failed to forward TCP message to MySensors", "gateway", gName, "error", err, "message", message.String())
					}
				}

				// Also process locally as if it came from a device (for testing/simulation)
				// This allows injecting messages via TCP to update MQTT state
				if gateway, exists := app.gateways[gName]; exists {
					if err := gateway.HandleMessage(message); err != nil {
						app.logger.Error("Gateway message handling failed for TCP message", "gateway", gName, "error", err, "message", message.String())
					}
				}
				app.checkForAckResponse(message)
				app.handleDeviceMessage(message)
			}
		}(gatewayName, tcpServer)
	}
}

func (app *Application) handleMQTTStateChanges() {
	for _, device := range app.config.Devices {
		// Handle entities that can receive commands
		for _, entity := range device.Entities {
			// Only register handlers for entities that can receive commands
			if !entity.CanReceiveCommands() {
				continue
			}

			// Create local copies to avoid closure issues
			currentDevice := device
			currentEntity := entity

			// Create composite key for uniqueness across devices
			compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
			app.mqttClient.RegisterStateChangeHandler(compositeKey, func(deviceName, componentName string, state string) {
				app.logger.Info("MQTT entity command received", "device", deviceName, "entity", componentName, "state", state)

				nodeID, err := app.config.GetEffectiveEntityNodeID(&currentDevice, &currentEntity)
				if err != nil {
					app.logger.Error("Failed to resolve node ID", "device", deviceName, "entity", componentName, "error", err)
					return
				}

				childID, err := app.config.GetEffectiveChildID(&currentEntity)
				if err != nil {
					app.logger.Error("Failed to resolve child ID", "device", deviceName, "entity", componentName, "error", err)
					return
				}

				// Determine which gateway to use
				gatewayName := app.config.GetEffectiveGateway(currentDevice.Gateway, currentEntity.Gateway)

				gatewayTransport, exists := app.transports[gatewayName]
				if !exists {
					app.logger.Error("No transport found for gateway", "gateway", gatewayName, "device", deviceName)
					return
				}

				// Get the MySensors variable type for this entity
				varType, exists := config.GetMySensorsVariableTypeForEntity(currentEntity.EntityType, currentEntity.VariableType)
				if !exists {
					app.logger.Error("Unknown variable type for entity", "entity", componentName, "entityType", currentEntity.EntityType)
					return
				}

				// Check if ACK is requested for this entity
				requestAck := app.config.GetEffectiveEntityRequestAck(&currentEntity)

				if requestAck {
					// Send with ACK handling (timeout/retry)
					app.sendWithAck(compositeKey, gatewayName, gatewayTransport, nodeID, childID, varType, state, &currentEntity)
				} else {
					// Fire and forget (original behavior)
					message := mysensors.NewSetMessageWithAck(nodeID, childID, varType, state, false)
					app.logger.Info("Sending MySensors entity command", "gateway", gatewayName, "message", message.String())

					if err := gatewayTransport.Send(message); err != nil {
						app.logger.Error("Failed to send entity state change to MySensors", "gateway", gatewayName, "error", err,
							"device", deviceName, "entity", componentName, "state", state)
					} else {
						app.logger.Info("MySensors entity command sent successfully", "gateway", gatewayName, "device", deviceName, "entity", componentName,
							"node_id", nodeID, "child_id", childID, "state", state, "message", message.String())

						// Also broadcast the sent message to TCP clients
						if tcpServer, exists := app.tcpServers[gatewayName]; exists {
							tcpServer.BroadcastMessage(message)
						}
					}
				}
			})
		}
	}
}

// sendWithAck sends a MySensors SET message and waits for ACK with timeout/retry logic
func (app *Application) sendWithAck(compositeKey, gatewayName string, gatewayTransport transport.Transport, nodeID, childID int, varType mysensors.VariableType, payload string, entity *config.Entity) {
	ackTimeout := app.config.GetEffectiveAckTimeout(entity)
	ackRetries := app.config.GetEffectiveAckRetries(entity)

	// Cancel any existing pending ACK for this entity (new request restarts the process)
	app.pendingAcksMu.Lock()
	if existingAck, exists := app.pendingAcks[compositeKey]; exists {
		close(existingAck.CancelChan)
		delete(app.pendingAcks, compositeKey)
		app.logger.Debug("Cancelled existing pending ACK for entity", "key", compositeKey)
	}

	// Create new pending ACK
	pending := &PendingAck{
		NodeID:     nodeID,
		ChildID:    childID,
		VarType:    varType,
		Payload:    payload,
		AckChan:    make(chan bool, 1),
		CancelChan: make(chan struct{}),
	}
	app.pendingAcks[compositeKey] = pending
	app.pendingAcksMu.Unlock()

	// Clean up when done
	defer func() {
		app.pendingAcksMu.Lock()
		if current, exists := app.pendingAcks[compositeKey]; exists && current == pending {
			delete(app.pendingAcks, compositeKey)
		}
		app.pendingAcksMu.Unlock()
	}()

	// Retry loop
	for attempt := 0; attempt <= ackRetries; attempt++ {
		message := mysensors.NewSetMessageWithAck(nodeID, childID, varType, payload, true)

		if attempt == 0 {
			app.logger.Info("Sending MySensors entity command with ACK", "gateway", gatewayName, "message", message.String(), "timeout", ackTimeout, "max_retries", ackRetries)
		} else {
			app.logger.Info("Retrying MySensors entity command", "gateway", gatewayName, "message", message.String(), "attempt", attempt, "max_retries", ackRetries)
		}

		if err := gatewayTransport.Send(message); err != nil {
			app.logger.Error("Failed to send entity command to MySensors", "gateway", gatewayName, "error", err, "node_id", nodeID, "child_id", childID)
			return
		}

		// Broadcast to TCP clients
		if tcpServer, exists := app.tcpServers[gatewayName]; exists {
			tcpServer.BroadcastMessage(message)
		}

		// Wait for ACK, timeout, or cancellation
		select {
		case <-pending.AckChan:
			app.logger.Info("ACK received for entity command", "gateway", gatewayName, "node_id", nodeID, "child_id", childID, "attempts", attempt+1)
			return
		case <-pending.CancelChan:
			app.logger.Debug("ACK wait cancelled (new request arrived)", "gateway", gatewayName, "node_id", nodeID, "child_id", childID)
			return
		case <-time.After(ackTimeout):
			if attempt < ackRetries {
				app.logger.Debug("ACK timeout, will retry", "gateway", gatewayName, "node_id", nodeID, "child_id", childID, "attempt", attempt+1)
			}
		case <-app.ctx.Done():
			app.logger.Debug("ACK wait cancelled (context done)", "gateway", gatewayName, "node_id", nodeID, "child_id", childID)
			return
		}
	}

	app.logger.Warn("ACK not received after all retries", "gateway", gatewayName, "node_id", nodeID, "child_id", childID, "retries", ackRetries)
}

// checkForAckResponse checks if an incoming SET message is an ACK response for a pending request
func (app *Application) checkForAckResponse(message *mysensors.Message) {
	if !message.IsSet() {
		return
	}

	app.pendingAcksMu.Lock()
	defer app.pendingAcksMu.Unlock()

	// Check all pending ACKs to see if this message matches
	for key, pending := range app.pendingAcks {
		if pending.NodeID == message.NodeID &&
			pending.ChildID == message.ChildID &&
			pending.VarType == message.GetVariableType() {
			// ACK received - signal the waiting goroutine
			select {
			case pending.AckChan <- true:
				app.logger.Debug("ACK matched for pending request", "key", key, "node_id", message.NodeID, "child_id", message.ChildID)
			default:
				// Channel full or already signaled
			}
			return
		}
	}
}

func (app *Application) publishMySensorsTraffic(gateway, direction string, message *mysensors.Message) {
	if app.eventBus == nil {
		return
	}

	msgType := "UNKNOWN"
	switch message.MessageType {
	case mysensors.PRESENTATION:
		msgType = "PRESENTATION"
	case mysensors.SET:
		msgType = "SET"
	case mysensors.REQ:
		msgType = "REQ"
	case mysensors.INTERNAL:
		msgType = "INTERNAL"
	case mysensors.STREAM:
		msgType = "STREAM"
	}

	trafficMsg := api.MySensorsTrafficMessage{
		Timestamp: time.Now().Format(time.RFC3339),
		Gateway:   gateway,
		Direction: direction,
		Raw:       message.String(),
		NodeID:    message.NodeID,
		ChildID:   message.ChildID,
		Type:      msgType,
		Ack:       message.Ack,
		SubType:   message.SubType,
		Payload:   message.Payload,
	}

	app.eventBus.Publish(api.Event{
		Type: api.EventMySensorsMessage,
		Data: trafficMsg,
	})
}

func (app *Application) handleDeviceMessage(message *mysensors.Message) {
	if !message.IsSet() && !message.IsReq() {
		app.logger.Debug("Ignoring non-SET/REQ message", "message", message.String(), "msg_type", message.MessageType)
		return
	}

	app.logger.Info("Processing device message", "message", message.String(), "node_id", message.NodeID, "child_id", message.ChildID, "var_type", message.GetVariableType())

	var matchedEntities []string

	for _, device := range app.config.Devices {
		// Handle entities with many-to-many mapping (all matches)
		for _, entity := range device.Entities {
			// Only process entities that can report state
			if !entity.CanReportState() {
				continue
			}

			effectiveNodeID, err := app.config.GetEffectiveEntityNodeID(&device, &entity)
			if err != nil {
				app.logger.Error("Failed to resolve node ID for message handling", "device", device.Name, "entity", entity.Name, "error", err)
				continue
			}

			childID, err := app.config.GetEffectiveChildID(&entity)
			if err != nil {
				app.logger.Error("Failed to resolve child ID for message handling", "device", device.Name, "entity", entity.Name, "error", err)
				continue
			}

			if effectiveNodeID == message.NodeID && childID == message.ChildID {
				app.logger.Info("Found matching entity", "device", device.Name, "entity", entity.Name, "node_id", effectiveNodeID, "child_id", childID)
				if message.IsSet() {
					// Get expected variable type for this entity
					expectedVarType, exists := config.GetMySensorsVariableTypeForEntity(entity.EntityType, entity.VariableType)
					app.logger.Info("Variable type check", "entity", entity.Name, "entity_type", entity.EntityType,
						"message_var_type", message.GetVariableType(), "expected_var_type", expectedVarType, "exists", exists, "match", message.GetVariableType() == expectedVarType)

					if exists && message.GetVariableType() == expectedVarType {
						state := message.Payload
						if err := app.mqttClient.PublishEntityState(device, entity, state); err != nil {
							app.logger.Error("Failed to publish entity state", "error", err,
								"device", device.Name, "entity", entity.Name, "state", state)
						} else {
							app.logger.Info("Entity state changed", "device", device.Name, "entity", entity.Name,
								"entity_type", entity.EntityType, "node_id", effectiveNodeID, "child_id", childID, "state", state)
							matchedEntities = append(matchedEntities, fmt.Sprintf("%s:%s", device.Name, entity.Name))

							// Fire WebSocket event
							if app.eventBus != nil {
								uid := entity.GetEffectiveUniqueID(device.ID)
								app.eventBus.Publish(api.Event{
									Type: api.EventEntityStateChanged,
									Data: map[string]string{"unique_id": uid, "state": state},
								})
							}
						}
					}
				}
			}
		}
	}

	// Log only when no matching device found
	if len(matchedEntities) == 0 {
		app.logger.Info("No matching entity found for MySensors message",
			"node_id", message.NodeID, "child_id", message.ChildID, "message_type", message.MessageType, "var_type", message.GetVariableType())
	}
}

func (app *Application) periodicHeartbeatRequest(ctx context.Context) {
	// Use the default gateway's period
	defaultGatewayName := app.config.GetDefaultGatewayName()
	defaultGatewayConfig := app.config.MySensors[defaultGatewayName]
	period := defaultGatewayConfig.Gateway.HeartbeatRequestPeriod

	// If period is 0, disable periodic heartbeat requests
	if period <= 0 {
		app.logger.Info("Periodic heartbeat requests disabled (heartbeat_request_period=0)")
		return
	}

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	app.logger.Info("Starting periodic heartbeat requests", "period", period)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for gatewayName, gateway := range app.gateways {
				if err := gateway.SendHeartbeatRequest(); err != nil {
					app.logger.Error("Failed to send periodic heartbeat request", "gateway", gatewayName, "error", err)
				}
			}
		}
	}
}

// availabilityMonitor periodically checks node availability and publishes status updates
func (app *Application) availabilityMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Check availability every 10 seconds
	defer ticker.Stop()

	app.logger.Info("Starting availability monitor")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check availability for each gateway
			for gatewayName, gw := range app.gateways {
				availabilityStatus := gw.GetAllNodeAvailabilityStatus()
				
				// Log the current availability status for debugging
				app.logger.Debug("Current availability status", "gateway", gatewayName, "status", availabilityStatus)
				
				// For each device, check if its node is available and publish to availability topic
				for _, device := range app.config.Devices {
					// Skip devices that don't use this gateway
					deviceGatewayName := app.config.GetEffectiveGateway(device.Gateway, "")
					app.logger.Debug("Checking device gateway", "device", device.Name, "device_gateway", device.Gateway, "effective_gateway", deviceGatewayName, "current_gateway", gatewayName)
					if deviceGatewayName != gatewayName {
						continue
					}
					
					// Get the effective node ID for this device
					_, err := app.config.GetEffectiveNodeID(&device)
					if err != nil {
						app.logger.Debug("Device has no node ID, skipping availability check", "device", device.Name)
						continue
					}
					
					// For each entity in the device, publish availability status
					for _, entity := range device.Entities {
						// Determine the effective node ID for this entity
						effectiveNodeID, err := app.config.GetEffectiveEntityNodeID(&device, &entity)
						if err != nil {
							app.logger.Debug("Entity has no node ID, skipping availability check", "device", device.Name, "entity", entity.Name)
							continue
						}
						
						// Skip if this entity doesn't use the current gateway
						entityGatewayName := app.config.GetEffectiveGateway(device.Gateway, entity.Gateway)
						app.logger.Debug("Checking entity gateway", "device", device.Name, "entity", entity.Name, "device_gateway", device.Gateway, "entity_gateway", entity.Gateway, "effective_gateway", entityGatewayName, "current_gateway", gatewayName)
						if entityGatewayName != gatewayName {
							continue
						}
						
						// Only publish availability for entities that can report state or receive commands
						if !entity.CanReportState() && !entity.CanReceiveCommands() {
							continue
						}
						
						// Create availability topic for this entity
						uniqueID := entity.GetEffectiveUniqueID(device.ID)
						availabilityTopic := fmt.Sprintf("%s/entity/%s/availability", app.config.AdapterTopics.TopicPrefix, uniqueID)
						
						// Determine availability status
						isAvailable := availabilityStatus[effectiveNodeID]
						payload := "offline"
						if isAvailable {
							payload = "online"
						}
						
						// Debug information
						app.logger.Debug("Availability check details", "gateway", gatewayName, "device", device.Name, "entity", entity.Name, "node_id", effectiveNodeID, "is_available", isAvailable)
						
						// Publish availability status
						if err := app.mqttClient.Publish(availabilityTopic, payload, true); err != nil {
							app.logger.Error("Failed to publish availability status", "gateway", gatewayName, "device", device.Name, "entity", entity.Name, "error", err)
						} else {
							app.logger.Debug("Published availability status", "gateway", gatewayName, "device", device.Name, "entity", entity.Name, "status", payload, "node_id", effectiveNodeID)
						}

						// Fire WebSocket event for availability changes
						if app.eventBus != nil {
							app.eventBus.Publish(api.Event{
								Type: api.EventNodeAvailabilityChanged,
								Data: map[string]any{
									"gateway":   gatewayName,
									"node_id":   effectiveNodeID,
									"available": isAvailable,
								},
							})
						}
					}
				}
			}
		}
	}
}

func (app *Application) startConfigWatcher(configFile string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	app.watcher = watcher

	// Add config file to watcher
	if err := watcher.Add(configFile); err != nil {
		return fmt.Errorf("failed to add config file to watcher: %w", err)
	}

	app.logger.Info("Config watcher started", "file", configFile)
	return nil
}

func (app *Application) handleConfigReloads(ctx context.Context) {
	if app.watcher == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			app.watcher.Close()
			return
		case event := <-app.watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				app.logger.Info("Config file changed, reloading", "file", event.Name)
				if err := app.reloadConfig(event.Name); err != nil {
					app.logger.Error("Failed to reload config", "error", err)
				}
			}
		case err := <-app.watcher.Errors:
			app.logger.Error("Config watcher error", "error", err)
		}
	}
}

func (app *Application) reloadConfig(configFile string) error {
	// Load new config
	newConfig, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// Update application config
	app.config = newConfig

	// Reconfigure MQTT client
	if err := app.mqttClient.Reconfigure(&newConfig.MQTT, &newConfig.AdapterTopics, newConfig.Devices); err != nil {
		return fmt.Errorf("failed to reconfigure MQTT client: %w", err)
	}

	// Reconfigure transports
	for gatewayName, gatewayConfig := range newConfig.MySensors {
		if trans, exists := app.transports[gatewayName]; exists {
			// Only reconnect if transport settings actually changed
			switch gatewayConfig.Transport {
			case "ethernet":
				if ethTransport, ok := trans.(*transport.EthernetTransport); ok {
					changed := ethTransport.Reconfigure(gatewayConfig.Ethernet.Host, gatewayConfig.Ethernet.Port)
					if changed && trans.IsConnected() {
						ethTransport.Disconnect()
						if err := ethTransport.Connect(app.ctx); err != nil {
							app.logger.Error("Failed to reconnect Ethernet transport after reconfiguration", "gateway", gatewayName, "error", err)
						}
					}
				}
			case "rs485":
				if rs485Transport, ok := trans.(*transport.RS485Transport); ok {
					changed := rs485Transport.Reconfigure(gatewayConfig.RS485.Device, 9600)
					if changed && trans.IsConnected() {
						rs485Transport.Disconnect()
						if err := rs485Transport.Connect(app.ctx); err != nil {
							app.logger.Error("Failed to reconnect RS485 transport after reconfiguration", "gateway", gatewayName, "error", err)
						}
					}
				}
			}
		} else {
			// Create new transport for new gateway
			var t transport.Transport
			switch gatewayConfig.Transport {
			case "ethernet":
				t = transport.NewEthernetTransport(
					gatewayConfig.Ethernet.Host,
					gatewayConfig.Ethernet.Port,
					app.logger,
				)
			case "rs485":
				t = transport.NewRS485Transport(
					gatewayConfig.RS485.Device,
					9600,
					app.logger,
				)
			default:
				app.logger.Warn("Unsupported transport type for new gateway", "gateway", gatewayName, "transport", gatewayConfig.Transport)
				continue
			}
			app.transports[gatewayName] = t
			
			// Connect the new transport
			if err := t.Connect(app.ctx); err != nil {
				app.logger.Error("Failed to connect new transport", "gateway", gatewayName, "error", err)
			}
		}
	}

	// Remove transports for deleted gateways
	for gatewayName := range app.config.MySensors {
		if _, exists := newConfig.MySensors[gatewayName]; !exists {
			if transport, exists := app.transports[gatewayName]; exists {
				transport.Disconnect()
				delete(app.transports, gatewayName)
				app.logger.Info("Removed transport for deleted gateway", "gateway", gatewayName)
			}
		}
	}

	// Reconfigure TCP servers
	for gatewayName, gatewayConfig := range newConfig.MySensors {
		if gatewayConfig.TCPService.Enabled {
			if tcpServer, exists := app.tcpServers[gatewayName]; exists {
				// Check if port changed
				if tcpServer.Port() != gatewayConfig.TCPService.Port {
					tcpServer.Stop()
					delete(app.tcpServers, gatewayName)
					app.tcpServers[gatewayName] = tcp.NewServer(gatewayConfig.TCPService.Port, app.logger)
					app.tcpServers[gatewayName].Start(context.Background())
					app.logger.Info("Restarted TCP server with new port", "gateway", gatewayName, "port", gatewayConfig.TCPService.Port)
				}
			} else {
				// Create new TCP server for new gateway
				app.tcpServers[gatewayName] = tcp.NewServer(gatewayConfig.TCPService.Port, app.logger)
				app.tcpServers[gatewayName].Start(context.Background())
				app.logger.Info("Created TCP server for new gateway", "gateway", gatewayName, "port", gatewayConfig.TCPService.Port)
			}
		} else {
			// Stop TCP server if disabled
			if tcpServer, exists := app.tcpServers[gatewayName]; exists {
				tcpServer.Stop()
				delete(app.tcpServers, gatewayName)
				app.logger.Info("Stopped TCP server for disabled gateway", "gateway", gatewayName)
			}
		}
	}

	// Remove TCP servers for deleted gateways
	for gatewayName := range app.config.MySensors {
		if _, exists := newConfig.MySensors[gatewayName]; !exists {
			if tcpServer, exists := app.tcpServers[gatewayName]; exists {
				tcpServer.Stop()
				delete(app.tcpServers, gatewayName)
				app.logger.Info("Removed TCP server for deleted gateway", "gateway", gatewayName)
			}
		}
	}

	// Reconfigure gateways
	for gatewayName, gatewayConfig := range newConfig.MySensors {
		if gw, exists := app.gateways[gatewayName]; exists {
			// Update existing gateway config
			gw.Reconfigure(&gatewayConfig.Gateway)
		} else {
			// Create new gateway for new gateway config
			gatewayTransport := app.transports[gatewayName]
			if gatewayTransport == nil {
				app.logger.Warn("No transport found for new gateway", "gateway", gatewayName)
				continue
			}

			gatewayConf := &gatewayConfig.Gateway

			newGateway := gateway.NewGateway(gatewayName, gatewayConf, newConfig, gatewayTransport, app.mqttClient, app.logger)

			// Set up callback to forward outgoing messages to TCP clients and traffic monitor
			if tcpServer, exists := app.tcpServers[gatewayName]; exists {
				newGateway.SetMessageSentCallback(func(message *mysensors.Message) {
					tcpServer.BroadcastMessage(message)
					app.publishMySensorsTraffic(gatewayName, "tx", message)
				})
			} else {
				// No TCP server, only publish to traffic monitor
				newGateway.SetMessageSentCallback(func(message *mysensors.Message) {
					app.publishMySensorsTraffic(gatewayName, "tx", message)
				})
			}

			app.gateways[gatewayName] = newGateway
			app.logger.Info("Created new gateway", "gateway", gatewayName)
		}
	}

	// Remove gateways for deleted gateways
	for gatewayName := range app.config.MySensors {
		if _, exists := newConfig.MySensors[gatewayName]; !exists {
			if _, exists := app.gateways[gatewayName]; exists {
				// Remove gateway
				delete(app.gateways, gatewayName)
				app.logger.Info("Removed gateway", "gateway", gatewayName)
			}
		}
	}

	// Reconfigure sync manager
	if err := app.syncMgr.Reconfigure(newConfig, app.mqttClient, app.transports); err != nil {
		app.logger.Error("Failed to reconfigure sync manager", "error", err)
		return fmt.Errorf("failed to reconfigure sync manager: %w", err)
	}

	// Re-register MQTT command handlers for all devices (including newly added ones)
	app.handleMQTTStateChanges()

	// Republish Home Assistant discovery topics
	if err := app.publishDiscovery(); err != nil {
		app.logger.Error("Failed to republish discovery topics after config reload", "error", err)
	} else {
		app.logger.Info("Republished Home Assistant discovery topics after config reload")
	}

	// Fire WebSocket event
	if app.eventBus != nil {
		app.eventBus.Publish(api.Event{
			Type: api.EventConfigReloaded,
			Data: map[string]any{},
		})
	}

	app.logger.Info("Config reloaded successfully")
	return nil
}

func (app *Application) shutdown() error {
	app.logger.Info("Shutting down components...")

	// Shutdown HTTP server
	if app.httpServer != nil {
		if err := app.httpServer.Shutdown(context.Background()); err != nil {
			app.logger.Error("HTTP server shutdown error", "error", err)
		}
	}

	if app.syncMgr != nil {
		app.syncMgr.Stop()
	}

	// Stop all TCP servers
	for gatewayName, tcpServer := range app.tcpServers {
		app.logger.Debug("Stopping TCP server", "gateway", gatewayName)
		tcpServer.Stop()
	}

	if app.mqttClient != nil {
		app.mqttClient.Disconnect()
	}

	// Disconnect all transports
	for gatewayName, gatewayTransport := range app.transports {
		app.logger.Debug("Disconnecting transport", "gateway", gatewayName)
		gatewayTransport.Disconnect()
	}

	// Close file watcher
	if app.watcher != nil {
		app.watcher.Close()
	}

	return nil
}
