package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/pkg/config"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client     mqtt.Client
	config     *config.MQTTConfig
	adapterCfg *config.AdapterConfig
	logger     *slog.Logger
	devices    []config.Device
	states     map[string]string
	stateMu    sync.RWMutex
	handlers   map[string]StateChangeHandler
}

type StateChangeHandler func(deviceName, componentName string, state string)

func NewClient(cfg *config.MQTTConfig, adapterCfg *config.AdapterConfig, devices []config.Device, logger *slog.Logger) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Broker, cfg.Port))
	opts.SetClientID(cfg.ClientID)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}
	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.Error("MQTT connection lost", "error", err)
	})
	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		logger.Info("MQTT reconnecting...")
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Info("MQTT connected")
	})

	return &Client{
		client:     mqtt.NewClient(opts),
		config:     cfg,
		adapterCfg: adapterCfg,
		logger:     logger,
		devices:    devices,
		states:     make(map[string]string),
		handlers:   make(map[string]StateChangeHandler),
	}
}

func (c *Client) Connect(ctx context.Context) error {
	token := c.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT connection timeout")
	}
	if token.Error() != nil {
		return fmt.Errorf("MQTT connection failed: %w", token.Error())
	}

	if err := c.subscribeToDevices(); err != nil {
		return fmt.Errorf("failed to subscribe to device topics: %w", err)
	}

	// Subscribe to state topics to capture retained messages
	if err := c.subscribeToStateTopic(); err != nil {
		return fmt.Errorf("failed to subscribe to state topics: %w", err)
	}

	c.logger.Info("MQTT client connected and subscribed to device topics")
	return nil
}

func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	c.logger.Info("MQTT client disconnected")
}

func (c *Client) subscribeToDevices() error {
	for _, device := range c.devices {
		for _, relay := range device.Relays {
			// Subscribe to device-specific topic using device_id/relay/subdevice_id format
			topic := fmt.Sprintf("%s/devices/%s/relay/%s/set", c.adapterCfg.TopicPrefix, device.ID, relay.ID)
			// Create composite key for uniqueness across devices
			compositeKey := fmt.Sprintf("%s_%s", device.ID, relay.ID)
			token := c.client.Subscribe(topic, 0, c.createRelayHandler(device.Name, relay.Name, compositeKey, device.ID, relay.ID))
			if !token.WaitTimeout(5 * time.Second) {
				return fmt.Errorf("subscription timeout for topic %s", topic)
			}
			if token.Error() != nil {
				return fmt.Errorf("subscription failed for topic %s: %w", topic, token.Error())
			}
			c.logger.Debug("Subscribed to relay topic", "topic", topic)
		}
	}
	return nil
}

func (c *Client) subscribeToStateTopic() error {
	for _, device := range c.devices {
		// Subscribe to relay state topics
		for _, relay := range device.Relays {
			stateTopic := fmt.Sprintf("%s/devices/%s/relay/%s/state", c.adapterCfg.TopicPrefix, device.ID, relay.ID)
			compositeKey := fmt.Sprintf("%s_%s", device.ID, relay.ID)
			token := c.client.Subscribe(stateTopic, 0, c.createStateHandler(compositeKey))
			if !token.WaitTimeout(5 * time.Second) {
				return fmt.Errorf("subscription timeout for relay state topic %s", stateTopic)
			}
			if token.Error() != nil {
				return fmt.Errorf("subscription failed for relay state topic %s: %w", stateTopic, token.Error())
			}
			c.logger.Debug("Subscribed to relay state topic", "topic", stateTopic)
		}

		// Subscribe to input state topics
		for _, input := range device.Inputs {
			stateTopic := fmt.Sprintf("%s/devices/%s/input/%s/state", c.adapterCfg.TopicPrefix, device.ID, input.ID)
			compositeKey := fmt.Sprintf("%s_%s", device.ID, input.ID)
			token := c.client.Subscribe(stateTopic, 0, c.createStateHandler(compositeKey))
			if !token.WaitTimeout(5 * time.Second) {
				return fmt.Errorf("subscription timeout for input state topic %s", stateTopic)
			}
			if token.Error() != nil {
				return fmt.Errorf("subscription failed for input state topic %s: %w", stateTopic, token.Error())
			}
			c.logger.Debug("Subscribed to input state topic", "topic", stateTopic)
		}
	}
	return nil
}

func (c *Client) createStateHandler(uniqueID string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		c.logger.Debug("Received retained state message", "topic", msg.Topic(), "payload", payload)

		// Skip empty payloads (might be cleared retained messages)
		if len(payload) == 0 {
			c.logger.Debug("Skipping empty payload (cleared retained message)", "topic", msg.Topic())
			return
		}

		// Validate payload is 0 or 1
		if payload != "0" && payload != "1" {
			c.logger.Warn("Invalid state payload, expected 0 or 1", "payload", payload, "topic", msg.Topic())
			return
		}

		// Store the existing state
		c.stateMu.Lock()
		c.states[uniqueID] = payload
		c.stateMu.Unlock()
	}
}

func (c *Client) createRelayHandler(deviceName, relayName, compositeKey, deviceID, relayID string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		c.logger.Debug("MQTT RX", "topic", msg.Topic(), "payload", payload)

		// Validate payload is 0 or 1
		if payload != "0" && payload != "1" {
			c.logger.Warn("Invalid relay payload, expected 0 or 1", "payload", payload)
			return
		}

		// Check if relay is configured for optimistic mode
		optimistic := c.getEffectiveOptimisticMode(deviceID, relayID)

		if optimistic {
			// Optimistic mode: update MQTT state immediately (assume command will succeed)
			deviceStateTopic := fmt.Sprintf("%s/devices/%s/relay/%s/state", c.adapterCfg.TopicPrefix, deviceID, relayID)
			c.Publish(deviceStateTopic, payload, true)

			c.stateMu.Lock()
			c.states[compositeKey] = payload
			c.stateMu.Unlock()

			c.logger.Debug("Optimistic mode: updated MQTT state immediately", "device", deviceName, "relay", relayName, "state", payload)
		} else {
			// Non-optimistic mode: wait for MySensors device confirmation before updating MQTT state
			c.logger.Debug("Non-optimistic mode: waiting for device confirmation", "device", deviceName, "relay", relayName, "command", payload)
		}

		// Always notify the handler to send MySensors command
		if handler, exists := c.handlers[compositeKey]; exists {
			handler(deviceName, relayName, payload)
		}
	}
}

func (c *Client) RegisterStateChangeHandler(uniqueID string, handler StateChangeHandler) {
	c.handlers[uniqueID] = handler
}

// getEffectiveOptimisticMode determines the effective optimistic mode for a specific relay
// Priority: per-relay setting > global setting > default (false)
func (c *Client) getEffectiveOptimisticMode(deviceID, relayID string) bool {
	// Find the device and relay configuration
	for _, device := range c.devices {
		if device.ID == deviceID {
			for _, relay := range device.Relays {
				if relay.ID == relayID {
					// Check per-relay setting first (highest priority)
					if relay.Optimistic != nil {
						return *relay.Optimistic
					}
					// Fall back to global setting
					if c.adapterCfg.OptimisticMode != nil {
						return *c.adapterCfg.OptimisticMode
					}
					// Default to non-optimistic (false)
					return false
				}
			}
		}
	}
	// If device/relay not found, default to non-optimistic
	return false
}

func (c *Client) Publish(topic, payload string, retain bool) error {
	token := c.client.Publish(topic, 0, retain, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic %s", topic)
	}
	if token.Error() != nil {
		return fmt.Errorf("publish failed for topic %s: %w", topic, token.Error())
	}

	c.logger.Debug("MQTT TX", "topic", topic, "payload", payload)
	return nil
}

func (c *Client) GetState(uniqueID string) (string, bool) {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	state, exists := c.states[uniqueID]
	return state, exists
}

func (c *Client) SetState(uniqueID, state string) {
	c.stateMu.Lock()
	c.states[uniqueID] = state
	c.stateMu.Unlock()
}

func (c *Client) PublishDeviceState(device config.Device, relay config.Relay, state string) error {
	// Publish to device-specific state topic
	deviceStateTopic := fmt.Sprintf("%s/devices/%s/relay/%s/state", c.adapterCfg.TopicPrefix, device.ID, relay.ID)
	
	// Update internal state tracking
	compositeKey := fmt.Sprintf("%s_%s", device.ID, relay.ID)
	c.stateMu.Lock()
	c.states[compositeKey] = state
	c.stateMu.Unlock()
	
	return c.Publish(deviceStateTopic, state, true)
}

func (c *Client) PublishInputState(device config.Device, input config.Input, state string) error {
	// Publish to device-specific state topic using 0/1 values with retain flag
	deviceStateTopic := fmt.Sprintf("%s/devices/%s/input/%s/state", c.adapterCfg.TopicPrefix, device.ID, input.ID)
	
	// Update internal state tracking
	compositeKey := fmt.Sprintf("%s_%s", device.ID, input.ID)
	c.stateMu.Lock()
	c.states[compositeKey] = state
	c.stateMu.Unlock()
	
	return c.Publish(deviceStateTopic, state, true)
}

func (c *Client) PublishHomeAssistantDiscovery(device config.Device) error {
	// Only publish HomeAssistant discovery if enabled
	if c.adapterCfg.HomeAssistantDiscovery == nil || !*c.adapterCfg.HomeAssistantDiscovery {
		return nil
	}

	deviceInfo := map[string]interface{}{
		"identifiers":  []string{device.ID},
		"name":         device.Name,
		"manufacturer": device.Manufacturer,
		"model":        device.Model,
		"sw_version":   device.SWVersion,
		"hw_version":   device.HWVersion,
	}

	// Add optional device fields
	if device.ConfigurationURL != "" {
		deviceInfo["configuration_url"] = device.ConfigurationURL
	}
	if device.SuggestedArea != "" {
		deviceInfo["suggested_area"] = device.SuggestedArea
	}
	if len(device.Connections) > 0 {
		deviceInfo["connections"] = device.Connections
	}
	if device.ViaDevice != "" {
		deviceInfo["via_device"] = device.ViaDevice
	}

	for _, relay := range device.Relays {

		config := map[string]interface{}{
			"name":          relay.Name,
			"unique_id":     fmt.Sprintf("%s_%s", device.ID, relay.ID),
			"command_topic": fmt.Sprintf("%s/devices/%s/relay/%s/set", c.adapterCfg.TopicPrefix, device.ID, relay.ID),
			"state_topic":   fmt.Sprintf("%s/devices/%s/relay/%s/state", c.adapterCfg.TopicPrefix, device.ID, relay.ID),
			"device":        deviceInfo,
		}

		// Set payload and state values with defaults
		if relay.PayloadOn != "" {
			config["payload_on"] = relay.PayloadOn
		} else {
			config["payload_on"] = "1"
		}
		if relay.PayloadOff != "" {
			config["payload_off"] = relay.PayloadOff
		} else {
			config["payload_off"] = "0"
		}
		if relay.StateOn != "" {
			config["state_on"] = relay.StateOn
		} else {
			config["state_on"] = "1"
		}
		if relay.StateOff != "" {
			config["state_off"] = relay.StateOff
		} else {
			config["state_off"] = "0"
		}

		// Apply relay-specific configurations with defaults
		if relay.Optimistic != nil {
			config["optimistic"] = *relay.Optimistic
		} else if c.adapterCfg.OptimisticMode != nil {
			config["optimistic"] = *c.adapterCfg.OptimisticMode
		} else {
			config["optimistic"] = false
		}
		
		if relay.QOS != nil {
			config["qos"] = *relay.QOS
		} else {
			config["qos"] = 0
		}
		
		if relay.Retain != nil {
			config["retain"] = *relay.Retain
		} else {
			config["retain"] = true
		}

		if relay.Icon != "" {
			config["icon"] = relay.Icon
		}
		if relay.DeviceClass != "" {
			config["device_class"] = relay.DeviceClass
		}
		if relay.EntityCategory != "" {
			config["entity_category"] = relay.EntityCategory
		}
		if relay.EnabledByDefault != nil {
			config["enabled_by_default"] = *relay.EnabledByDefault
		}
		if relay.AvailabilityTopic != "" {
			config["availability_topic"] = relay.AvailabilityTopic
			if relay.PayloadAvailable != "" {
				config["payload_available"] = relay.PayloadAvailable
			} else {
				config["payload_available"] = "online"
			}
			if relay.PayloadNotAvailable != "" {
				config["payload_not_available"] = relay.PayloadNotAvailable
			} else {
				config["payload_not_available"] = "offline"
			}
		}
		if relay.JSONAttributesTopic != "" {
			config["json_attributes_topic"] = relay.JSONAttributesTopic
		}
		if relay.JSONAttributesTemplate != "" {
			config["json_attributes_template"] = relay.JSONAttributesTemplate
		}
		if relay.StateValueTemplate != "" {
			config["state_value_template"] = relay.StateValueTemplate
		}
		if relay.CommandTemplate != "" {
			config["command_template"] = relay.CommandTemplate
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal relay config: %w", err)
		}

		discoveryTopic := fmt.Sprintf("homeassistant/switch/%s_%s/config", device.ID, relay.ID)
		if err := c.Publish(discoveryTopic, string(configJSON), true); err != nil {
			return fmt.Errorf("failed to publish relay discovery: %w", err)
		}

		// Honor retained MQTT messages, only use config initial_state if no retained state exists
		compositeKey := fmt.Sprintf("%s_%s", device.ID, relay.ID)
		
		if existingState, exists := c.GetState(compositeKey); !exists {
			// No retained state found, use configured initial state
			initialState := "0"
			if relay.InitialState == 1 {
				initialState = "1"
			}
			c.logger.Debug("No retained state found, applying configured initial state", "device", device.ID, "relay", relay.ID, "initialState", initialState)
			c.SetState(compositeKey, initialState)
			if err := c.PublishDeviceState(device, relay, initialState); err != nil {
				return fmt.Errorf("failed to publish initial relay state: %w", err)
			}
			c.logger.Debug("Published configured initial relay state", "relay", relay.ID, "state", initialState)
		} else {
			// Retained state exists, honor it
			c.logger.Debug("Using existing retained state", "relay", relay.ID, "state", existingState)
		}
	}

	for _, input := range device.Inputs {
		// Always use binary_sensor for inputs
		entityType := "binary_sensor"

		config := map[string]interface{}{
			"name":        input.Name,
			"unique_id":   fmt.Sprintf("%s_%s", device.ID, input.ID),
			"state_topic": fmt.Sprintf("%s/devices/%s/input/%s/state", c.adapterCfg.TopicPrefix, device.ID, input.ID),
			"device":      deviceInfo,
		}

		// Set payload and state values with defaults
		if input.PayloadOn != "" {
			config["payload_on"] = input.PayloadOn
		} else {
			config["payload_on"] = "1"
		}
		if input.PayloadOff != "" {
			config["payload_off"] = input.PayloadOff
		} else {
			config["payload_off"] = "0"
		}
		if input.StateOn != "" {
			config["state_on"] = input.StateOn
		}
		if input.StateOff != "" {
			config["state_off"] = input.StateOff
		}

		// Apply input-specific configurations
		if input.QOS != nil {
			config["qos"] = *input.QOS
		} else {
			config["qos"] = 0
		}

		if input.Icon != "" {
			config["icon"] = input.Icon
		}
		if input.DeviceClass != "" {
			config["device_class"] = input.DeviceClass
		}
		if input.EntityCategory != "" {
			config["entity_category"] = input.EntityCategory
		}
		if input.EnabledByDefault != nil {
			config["enabled_by_default"] = *input.EnabledByDefault
		}
		if input.AvailabilityTopic != "" {
			config["availability_topic"] = input.AvailabilityTopic
			if input.PayloadAvailable != "" {
				config["payload_available"] = input.PayloadAvailable
			} else {
				config["payload_available"] = "online"
			}
			if input.PayloadNotAvailable != "" {
				config["payload_not_available"] = input.PayloadNotAvailable
			} else {
				config["payload_not_available"] = "offline"
			}
		}
		if input.OffDelay != nil {
			config["off_delay"] = *input.OffDelay
		}
		if input.ExpireAfter != nil {
			config["expire_after"] = *input.ExpireAfter
		}
		if input.JSONAttributesTopic != "" {
			config["json_attributes_topic"] = input.JSONAttributesTopic
		}
		if input.JSONAttributesTemplate != "" {
			config["json_attributes_template"] = input.JSONAttributesTemplate
		}
		if input.ValueTemplate != "" {
			config["value_template"] = input.ValueTemplate
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal input config: %w", err)
		}

		discoveryTopic := fmt.Sprintf("homeassistant/%s/%s_%s/config", entityType, device.ID, input.ID)
		if err := c.Publish(discoveryTopic, string(configJSON), true); err != nil {
			return fmt.Errorf("failed to publish input discovery: %w", err)
		}

		// Publish initial state for inputs if no state already exists
		compositeKey := fmt.Sprintf("%s_%s", device.ID, input.ID)
		if existingState, exists := c.GetState(compositeKey); !exists {
			initialState := "0" // Default input state is "off"
			c.SetState(compositeKey, initialState)
			if err := c.PublishInputState(device, input, initialState); err != nil {
				return fmt.Errorf("failed to publish initial input state: %w", err)
			}
			c.logger.Info("Published initial input state", "input", input.ID, "state", initialState)
		} else {
			c.logger.Info("Using existing input state", "input", input.ID, "state", existingState)
		}
	}

	return nil
}

func (c *Client) PublishAdapterStatus(topicPrefix string, nodeIDs []int) error {
	// Sort node IDs before publishing
	sortedNodeIDs := make([]int, len(nodeIDs))
	copy(sortedNodeIDs, nodeIDs)
	sort.Ints(sortedNodeIDs)

	nodeIDStrs := make([]string, len(sortedNodeIDs))
	for i, id := range sortedNodeIDs {
		nodeIDStrs[i] = strconv.Itoa(id)
	}

	nodeIDList := strings.Join(nodeIDStrs, ",")
	topic := fmt.Sprintf("%s/seen_nodes", topicPrefix)

	return c.Publish(topic, nodeIDList, true)
}

func (c *Client) PublishGatewayAdapterStatus(topicPrefix, gatewayName string, nodeIDs []int) error {
	// Sort node IDs before publishing
	sortedNodeIDs := make([]int, len(nodeIDs))
	copy(sortedNodeIDs, nodeIDs)
	sort.Ints(sortedNodeIDs)

	nodeIDStrs := make([]string, len(sortedNodeIDs))
	for i, id := range sortedNodeIDs {
		nodeIDStrs[i] = strconv.Itoa(id)
	}

	nodeIDList := strings.Join(nodeIDStrs, ",")
	topic := fmt.Sprintf("%s/gateway/%s/seen_nodes", topicPrefix, gatewayName)

	return c.Publish(topic, nodeIDList, true)
}
