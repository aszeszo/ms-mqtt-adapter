package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/events"
	"ms-mqtt-adapter/internal/mysensors"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/gateway"
	"ms-mqtt-adapter/pkg/mqtt"
	"ms-mqtt-adapter/pkg/tcp"
	"ms-mqtt-adapter/pkg/transport"
	"os"
	"os/signal"
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
	transport  transport.Transport
	mqttClient *mqtt.Client
	tcpServer  *tcp.Server
	gateway    *gateway.Gateway
	syncMgr    *events.SyncManager
}

func (app *Application) Run(ctx context.Context) error {
	if err := app.initializeTransport(); err != nil {
		return fmt.Errorf("failed to initialize transport: %w", err)
	}

	if err := app.initializeMQTT(); err != nil {
		return fmt.Errorf("failed to initialize MQTT: %w", err)
	}

	if err := app.initializeTCPServer(); err != nil {
		return fmt.Errorf("failed to initialize TCP server: %w", err)
	}

	if err := app.initializeGateway(); err != nil {
		return fmt.Errorf("failed to initialize gateway: %w", err)
	}

	if err := app.initializeSyncManager(); err != nil {
		return fmt.Errorf("failed to initialize sync manager: %w", err)
	}

	if err := app.start(ctx); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	// Wait briefly for retained messages to be processed before publishing discovery
	time.Sleep(1 * time.Second)

	if err := app.publishDiscovery(); err != nil {
		return fmt.Errorf("failed to publish discovery: %w", err)
	}

	// Perform initial sync if sync is enabled
	if app.config.Sync.Enabled {
		app.logger.Info("Performing initial device state sync")
		app.syncMgr.SyncDeviceStates()
	}

	// Send initial version request to gateway
	app.logger.Info("Sending initial version request to gateway")
	if err := app.gateway.SendVersionRequest(); err != nil {
		app.logger.Error("Failed to send initial version request", "error", err)
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

func (app *Application) initializeTransport() error {
	switch app.config.MySensors.Transport {
	case "ethernet":
		app.transport = transport.NewEthernetTransport(
			app.config.MySensors.Ethernet.Host,
			app.config.MySensors.Ethernet.Port,
			app.logger,
		)
	case "rs485":
		app.transport = transport.NewRS485Transport(
			app.config.MySensors.RS485.Device,
			9600,
			app.logger,
		)
	default:
		return fmt.Errorf("unsupported transport type: %s", app.config.MySensors.Transport)
	}

	return nil
}

func (app *Application) initializeMQTT() error {
	app.mqttClient = mqtt.NewClient(&app.config.MQTT, &app.config.AdapterTopics, app.config.Devices, app.logger)
	return nil
}

func (app *Application) initializeTCPServer() error {
	if app.config.TCPService.Enabled {
		app.tcpServer = tcp.NewServer(app.config.TCPService.Port, app.logger)
	}
	return nil
}

func (app *Application) initializeGateway() error {
	app.gateway = gateway.NewGateway(app.config, app.transport, app.logger)
	return nil
}

func (app *Application) initializeSyncManager() error {
	app.syncMgr = events.NewSyncManager(app.config, app.mqttClient, app.transport, app.logger)
	return nil
}

func (app *Application) start(ctx context.Context) error {
	if err := app.transport.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect transport: %w", err)
	}

	if err := app.mqttClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	if app.tcpServer != nil {
		if err := app.tcpServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start TCP server: %w", err)
		}
	}

	if err := app.syncMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sync manager: %w", err)
	}

	return nil
}

func (app *Application) publishDiscovery() error {
	for _, device := range app.config.Devices {
		if err := app.mqttClient.PublishHomeAssistantDiscovery(device); err != nil {
			return fmt.Errorf("failed to publish discovery for device %s: %w", device.Name, err)
		}
		app.logger.Info("Published Home Assistant discovery", "device", device.Name)
	}

	seenNodes := app.gateway.GetSeenNodes()
	if err := app.mqttClient.PublishAdapterStatus(app.config.AdapterTopics.TopicPrefix, seenNodes); err != nil {
		return fmt.Errorf("failed to publish adapter status: %w", err)
	}

	return nil
}

func (app *Application) handleMySensorsMessages() {
	for message := range app.transport.Receive() {
		if app.tcpServer != nil {
			app.tcpServer.BroadcastMessage(message)
		}

		if err := app.gateway.HandleMessage(message); err != nil {
			app.logger.Error("Gateway message handling failed", "error", err, "message", message.String())
		}

		app.handleDeviceMessage(message)

		seenNodes := app.gateway.GetSeenNodes()
		if err := app.mqttClient.PublishAdapterStatus(app.config.AdapterTopics.TopicPrefix, seenNodes); err != nil {
			app.logger.Error("Failed to publish adapter status", "error", err)
		}
	}
}

func (app *Application) handleTCPMessages() {
	if app.tcpServer == nil {
		return
	}

	for message := range app.tcpServer.Receive() {
		if err := app.transport.Send(message); err != nil {
			app.logger.Error("Failed to forward TCP message to MySensors", "error", err, "message", message.String())
		}
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
				mysensorsState := state // State is already 0/1 format

				nodeID := currentDevice.NodeID
				if currentRelay.NodeID != nil {
					nodeID = *currentRelay.NodeID
				}

				message := mysensors.NewSetMessage(nodeID, currentRelay.ChildID, mysensors.V_STATUS, mysensorsState)
				if err := app.transport.Send(message); err != nil {
					app.logger.Error("Failed to send state change to MySensors", "error", err,
						"device", deviceName, "component", componentName, "state", state)
				} else {
					app.logger.Info("State change sent to MySensors",
						"device", deviceName, "component", componentName, "state", mysensorsState)
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
					state := message.Payload // Use payload directly (should be 0 or 1)
					if err := app.mqttClient.PublishInputState(device, input, state); err != nil {
						app.logger.Error("Failed to publish input state", "error", err,
							"device", device.Name, "input", input.Name, "state", state)
					} else {
						app.logger.Info("Input state changed", "device", device.Name, "input", input.Name,
							"node_id", effectiveNodeID, "child_id", input.ChildID, "state", state)
						matchedInputs = append(matchedInputs, fmt.Sprintf("%s:%s", device.Name, input.Name))
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
	ticker := time.NewTicker(app.config.Gateway.VersionRequestPeriod)
	defer ticker.Stop()

	app.logger.Info("Starting periodic version requests", "period", app.config.Gateway.VersionRequestPeriod)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := app.gateway.SendVersionRequest(); err != nil {
				app.logger.Error("Failed to send periodic version request", "error", err)
			}
		}
	}
}

func (app *Application) shutdown() error {
	app.logger.Info("Shutting down components...")

	if app.syncMgr != nil {
		app.syncMgr.Stop()
	}

	if app.tcpServer != nil {
		app.tcpServer.Stop()
	}

	if app.mqttClient != nil {
		app.mqttClient.Disconnect()
	}

	if app.transport != nil {
		app.transport.Disconnect()
	}

	return nil
}
