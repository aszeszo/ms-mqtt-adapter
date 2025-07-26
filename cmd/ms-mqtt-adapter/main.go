package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"ms-mqtt-adapter/internal/events"
	"ms-mqtt-adapter/internal/mysensors"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/gateway"
	"ms-mqtt-adapter/pkg/mqtt"
	"ms-mqtt-adapter/pkg/tcp"
	"ms-mqtt-adapter/pkg/transport"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Configuration file path")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger := events.NewLogger(cfg.LogLevel)
	logger.Info("Starting ms-mqtt-adapter", "version", "1.0.0")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app := &Application{
		config: cfg,
		logger: logger,
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("Application failed", "error", err)
		os.Exit(1)
	}

	logger.Info("ms-mqtt-adapter stopped")
}

type Application struct {
	config     *config.Config
	logger     *slog.Logger
	transports map[string]transport.Transport  // gatewayName -> transport
	mqttClient *mqtt.Client
	tcpServers map[string]*tcp.Server          // gatewayName -> tcpServer
	gateways   map[string]*gateway.Gateway     // gatewayName -> gateway
	syncMgr    *events.SyncManager
	
	// Connection retry management
	transportRetryCount map[string]int
	mqttRetryCount      int
	retryMu             sync.RWMutex
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
		if g, exists := app.gateways["default"]; exists {
			return g
		}
		// Return any gateway if default doesn't exist
		for _, g := range app.gateways {
			return g
		}
	}
	return nil
}

func (app *Application) Run(ctx context.Context) error {
	// Initialize retry counters
	app.transportRetryCount = make(map[string]int)
	app.mqttRetryCount = 0
	
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

	if err := app.startWithRetry(ctx); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	// Wait briefly for retained messages to be processed before publishing discovery
	time.Sleep(1 * time.Second)

	if err := app.publishDiscovery(); err != nil {
		return fmt.Errorf("failed to publish discovery: %w", err)
	}

	// Perform initial sync if sync is enabled
	if app.config.AdapterTopics.Sync.Enabled {
		app.logger.Info("Performing initial device state sync")
		app.syncMgr.SyncDeviceStates()
	}

	// Send initial version request to all gateways
	app.logger.Info("Sending initial version requests to gateways")
	for gatewayName, gw := range app.gateways {
		if err := gw.SendVersionRequest(); err != nil {
			app.logger.Error("Failed to send initial version request", "gateway", gatewayName, "error", err)
		}
	}

	go app.handleMySensorsMessages()
	go app.handleTCPMessages()
	go app.handleMQTTStateChanges()
	go app.periodicVersionRequest(ctx)

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
		gatewayConf := &config.GatewayConfig{
			NodeIDRange:            gatewayConfig.Gateway.NodeIDRange,
			VersionRequestPeriod:   gatewayConfig.Gateway.VersionRequestPeriod,
			RandomIDAssignment:     gatewayConfig.Gateway.RandomIDAssignment,
		}
		
		app.gateways[gatewayName] = gateway.NewGateway(gatewayConf, gatewayTransport, app.logger)
	}
	return nil
}

func (app *Application) initializeSyncManager() error {
	// Use default transport for sync manager
	defaultTransport := app.getDefaultTransport()
	if defaultTransport == nil {
		return fmt.Errorf("no transport available for sync manager")
	}
	app.syncMgr = events.NewSyncManager(app.config, app.mqttClient, defaultTransport, app.logger)
	return nil
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
	if err := app.syncMgr.Start(ctx); err != nil {
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
						
						// Attempt reconnection with retry
						err := app.retryWithBackoff(ctx, fmt.Sprintf("MySensors gateway '%s' reconnection", name), -1, func() error {
							return transport.Connect(ctx)
						})
						
						if err != nil {
							app.logger.Error("Failed to reconnect MySensors gateway", "gateway", name, "error", err)
						} else {
							app.logger.Info("MySensors gateway reconnected successfully", "gateway", name)
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
					
					// Attempt reconnection with retry
					err := app.retryWithBackoff(ctx, "MQTT reconnection", -1, func() error {
						return app.mqttClient.Connect(ctx)
					})
					
					if err != nil {
						app.logger.Error("Failed to reconnect MQTT broker", "error", err)
					} else {
						app.logger.Info("MQTT broker reconnected successfully")
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

	// Publish seen nodes for each gateway separately and combined
	allSeenNodesMap := make(map[int]bool)
	for gatewayName, gateway := range app.gateways {
		gatewaySeenNodes := gateway.GetSeenNodes() // Already returns []int
		
		// Add to combined map
		for _, nodeID := range gatewaySeenNodes {
			allSeenNodesMap[nodeID] = true
		}
		
		// Publish gateway-specific seen nodes
		if err := app.mqttClient.PublishGatewayAdapterStatus(app.config.AdapterTopics.TopicPrefix, gatewayName, gatewaySeenNodes); err != nil {
			return fmt.Errorf("failed to publish gateway adapter status for %s: %w", gatewayName, err)
		}
	}
	
	// Convert combined map to slice
	var allSeenNodes []int
	for nodeID := range allSeenNodesMap {
		allSeenNodes = append(allSeenNodes, nodeID)
	}
	
	// Publish combined seen nodes
	if err := app.mqttClient.PublishAdapterStatus(app.config.AdapterTopics.TopicPrefix, allSeenNodes); err != nil {
		return fmt.Errorf("failed to publish adapter status: %w", err)
	}

	return nil
}

func (app *Application) handleMySensorsMessages() {
	// Start a goroutine for each transport
	for gatewayName, gatewayTransport := range app.transports {
		go func(gName string, t transport.Transport) {
			for message := range t.Receive() {
				app.logger.Debug("Received MySensors message", "gateway", gName, "message", message.String())

				// Broadcast to corresponding TCP server
				if tcpServer, exists := app.tcpServers[gName]; exists {
					tcpServer.BroadcastMessage(message)
				}

				// Handle message with corresponding gateway
				if gateway, exists := app.gateways[gName]; exists {
					if err := gateway.HandleMessage(message); err != nil {
						app.logger.Error("Gateway message handling failed", "gateway", gName, "error", err, "message", message.String())
					}
				}

				app.handleDeviceMessage(message)

				// Publish gateway-specific status and combined status
				if gateway, exists := app.gateways[gName]; exists {
					gatewaySeenNodes := gateway.GetSeenNodes() // Already returns []int
					
					// Publish gateway-specific seen nodes
					if err := app.mqttClient.PublishGatewayAdapterStatus(app.config.AdapterTopics.TopicPrefix, gName, gatewaySeenNodes); err != nil {
						app.logger.Error("Failed to publish gateway adapter status", "gateway", gName, "error", err)
					}
				}

				// Update combined adapter status with all seen nodes
				allSeenNodesMap := make(map[int]bool)
				for _, gw := range app.gateways {
					seenNodes := gw.GetSeenNodes() // Already returns []int
					for _, nodeID := range seenNodes {
						allSeenNodesMap[nodeID] = true
					}
				}
				
				// Convert map to slice
				var allSeenNodes []int
				for nodeID := range allSeenNodesMap {
					allSeenNodes = append(allSeenNodes, nodeID)
				}
				
				if err := app.mqttClient.PublishAdapterStatus(app.config.AdapterTopics.TopicPrefix, allSeenNodes); err != nil {
					app.logger.Error("Failed to publish adapter status", "error", err)
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
				if gatewayTransport, exists := app.transports[gName]; exists {
					if err := gatewayTransport.Send(message); err != nil {
						app.logger.Error("Failed to forward TCP message to MySensors", "gateway", gName, "error", err, "message", message.String())
					}
				}
			}
		}(gatewayName, tcpServer)
	}
}

func (app *Application) handleMQTTStateChanges() {
	for _, device := range app.config.Devices {
		for _, relay := range device.Relays {
			// Create local copies to avoid closure issues
			currentDevice := device
			currentRelay := relay
			
			// Create composite key for uniqueness across devices
			compositeKey := fmt.Sprintf("%s_%s", device.ID, relay.ID)
			app.mqttClient.RegisterStateChangeHandler(compositeKey, func(deviceName, componentName string, state string) {
				app.logger.Info("MQTT command received", "device", deviceName, "component", componentName, "state", state)
				
				mysensorsState := state // State is already 0/1 format

				nodeID := currentDevice.NodeID
				if currentRelay.NodeID != nil {
					nodeID = *currentRelay.NodeID
				}

				// Determine which gateway to use
				gatewayName := "default"
				if currentDevice.Gateway != "" {
					gatewayName = currentDevice.Gateway
				}
				
				gatewayTransport, exists := app.transports[gatewayName]
				if !exists {
					app.logger.Error("No transport found for gateway", "gateway", gatewayName, "device", deviceName)
					return
				}

				// Use configured ACK bit setting (priority: device > global > default true)
				requestAck := app.config.GetEffectiveRequestAck(&currentDevice)
				message := mysensors.NewSetMessageWithAck(nodeID, currentRelay.ChildID, mysensors.V_STATUS, mysensorsState, requestAck)
				
				app.logger.Info("Sending MySensors command", "gateway", gatewayName, "message", message.String())
				
				if err := gatewayTransport.Send(message); err != nil {
					app.logger.Error("Failed to send state change to MySensors", "gateway", gatewayName, "error", err,
						"device", deviceName, "component", componentName, "state", state)
				} else {
					app.logger.Info("MySensors command sent successfully", "gateway", gatewayName, "device", deviceName, "relay", componentName, 
						"node_id", nodeID, "child_id", currentRelay.ChildID, "state", mysensorsState, "message", message.String())
				}
			})
		}
	}
}

func (app *Application) handleDeviceMessage(message *mysensors.Message) {
	if !message.IsSet() && !message.IsReq() {
		return
	}

	relayHandled := false
	var matchedInputs []string

	for _, device := range app.config.Devices {
		// Handle relays with 1:1 mapping (first match only)
		if !relayHandled {
			for _, relay := range device.Relays {
				effectiveNodeID := device.NodeID
				if relay.NodeID != nil {
					effectiveNodeID = *relay.NodeID
				}

				if effectiveNodeID == message.NodeID && relay.ChildID == message.ChildID {
					if message.IsSet() && message.GetVariableType() == mysensors.V_STATUS {
						state := message.Payload // Use payload directly (should be 0 or 1)
						if err := app.mqttClient.PublishDeviceState(device, relay, state); err != nil {
							app.logger.Error("Failed to publish device state", "error", err,
								"device", device.Name, "relay", relay.Name, "state", state)
						} else {
							app.logger.Debug("Published relay state", "device", device.Name, "relay", relay.Name,
								"node_id", effectiveNodeID, "child_id", relay.ChildID, "state", state)
						}
						relayHandled = true
						break // Only handle first matching relay
					}
				}
			}
		}

		// Handle inputs with many-to-many mapping (all matches)
		for _, input := range device.Inputs {
			effectiveNodeID := device.NodeID
			if input.NodeID != nil {
				effectiveNodeID = *input.NodeID
			}

			if effectiveNodeID == message.NodeID && input.ChildID == message.ChildID {
				if message.IsSet() {
					// Check if this is a sensor value message
					if config.IsBinarySensor(input.SensorType) {
						// Handle binary sensor (existing logic)
						state := message.Payload // Use payload directly (should be 0 or 1)
						if err := app.mqttClient.PublishInputState(device, input, state); err != nil {
							app.logger.Error("Failed to publish input state", "error", err,
								"device", device.Name, "input", input.Name, "state", state)
						} else {
							app.logger.Info("Input state changed", "device", device.Name, "input", input.Name,
								"node_id", effectiveNodeID, "child_id", input.ChildID, "state", state)
							matchedInputs = append(matchedInputs, fmt.Sprintf("%s:%s", device.Name, input.Name))
						}
					} else {
						// Handle numeric sensor
						expectedVarType, exists := config.GetMySensorsVariableType(input.SensorType)
						if exists && message.GetVariableType() == expectedVarType {
							sensorValue := message.Payload
							if err := app.mqttClient.PublishSensorState(device, input, sensorValue); err != nil {
								app.logger.Error("Failed to publish sensor state", "error", err,
									"device", device.Name, "sensor", input.Name, "value", sensorValue)
							} else {
								app.logger.Info("Sensor value changed", "device", device.Name, "sensor", input.Name,
									"sensor_type", input.SensorType, "node_id", effectiveNodeID, "child_id", input.ChildID, "value", sensorValue)
								matchedInputs = append(matchedInputs, fmt.Sprintf("%s:%s", device.Name, input.Name))
							}
						}
					}
				}
			}
		}
	}

	// Log only when no matching device found
	if len(matchedInputs) == 0 && !relayHandled {
		app.logger.Debug("No matching device found for MySensors message",
			"node_id", message.NodeID, "child_id", message.ChildID)
	}
}

func (app *Application) periodicVersionRequest(ctx context.Context) {
	// Use the default gateway's period, or first available gateway
	var period time.Duration
	if defaultGatewayConfig, exists := app.config.MySensors["default"]; exists {
		period = defaultGatewayConfig.Gateway.VersionRequestPeriod
	} else {
		// Use first gateway's period
		for _, gatewayConfig := range app.config.MySensors {
			period = gatewayConfig.Gateway.VersionRequestPeriod
			break
		}
	}
	
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	app.logger.Info("Starting periodic version requests", "period", period)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for gatewayName, gateway := range app.gateways {
				if err := gateway.SendVersionRequest(); err != nil {
					app.logger.Error("Failed to send periodic version request", "gateway", gatewayName, "error", err)
				}
			}
		}
	}
}

func (app *Application) shutdown() error {
	app.logger.Info("Shutting down components...")

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

	return nil
}
