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

func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

func (c *Client) subscribeToDevices() error {
	for _, device := range c.devices {
		// Subscribe to entity command topics
		for _, entity := range device.Entities {
			// Only subscribe to command topics for entities that can receive commands
			if !entity.CanReceiveCommands() {
				continue
			}
			
			// Subscribe to device-specific topic using device_id/entity/subdevice_id format
			topic := fmt.Sprintf("%s/devices/%s/entity/%s/set", c.adapterCfg.TopicPrefix, device.ID, entity.ID)
			// Create composite key for uniqueness across devices
			compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
			token := c.client.Subscribe(topic, 0, c.createEntityHandler(device.Name, entity.Name, compositeKey, device.ID, entity.ID, entity.EntityType))
			if !token.WaitTimeout(5 * time.Second) {
				return fmt.Errorf("subscription timeout for topic %s", topic)
			}
			if token.Error() != nil {
				return fmt.Errorf("subscription failed for topic %s: %w", topic, token.Error())
			}
			c.logger.Debug("Subscribed to entity topic", "topic", topic)
		}
	}
	return nil
}

func (c *Client) subscribeToStateTopic() error {
	for _, device := range c.devices {
		// Subscribe to entity state topics
		for _, entity := range device.Entities {
			// Only subscribe to state topics for entities that can report state
			if !entity.CanReportState() {
				continue
			}
			
			stateTopic := fmt.Sprintf("%s/devices/%s/entity/%s/state", c.adapterCfg.TopicPrefix, device.ID, entity.ID)
			compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
			token := c.client.Subscribe(stateTopic, 0, c.createEntityStateHandler(compositeKey, entity.EntityType))
			if !token.WaitTimeout(5 * time.Second) {
				return fmt.Errorf("subscription timeout for entity state topic %s", stateTopic)
			}
			if token.Error() != nil {
				return fmt.Errorf("subscription failed for entity state topic %s: %w", stateTopic, token.Error())
			}
			c.logger.Debug("Subscribed to entity state topic", "topic", stateTopic)
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

		// For sensor topics (contains "_sensor"), accept any numeric value
		if strings.Contains(uniqueID, "_sensor") {
			// Store the sensor value
			c.stateMu.Lock()
			c.states[uniqueID] = payload
			c.stateMu.Unlock()
		} else {
			// For binary sensor topics, validate payload is 0 or 1
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
}


func (c *Client) createEntityHandler(deviceName, entityName, compositeKey, deviceID, entityID, entityType string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		c.logger.Debug("MQTT RX", "topic", msg.Topic(), "payload", payload)

		// Validate payload based on entity type
		if !c.validateEntityPayload(entityType, payload) {
			c.logger.Warn("Invalid entity payload", "entityType", entityType, "payload", payload)
			return
		}

		// Check if entity is configured for optimistic mode
		optimistic := c.getEffectiveOptimisticModeForEntity(deviceID, entityID)

		if optimistic {
			// Optimistic mode: update MQTT state immediately (assume command will succeed)
			deviceStateTopic := fmt.Sprintf("%s/devices/%s/entity/%s/state", c.adapterCfg.TopicPrefix, deviceID, entityID)
			c.Publish(deviceStateTopic, payload, true)

			c.stateMu.Lock()
			c.states[compositeKey] = payload
			c.stateMu.Unlock()

			c.logger.Debug("Optimistic mode: updated MQTT state immediately", "device", deviceName, "entity", entityName, "state", payload)
		} else {
			// Non-optimistic mode: wait for MySensors device confirmation before updating MQTT state
			c.logger.Debug("Non-optimistic mode: waiting for device confirmation", "device", deviceName, "entity", entityName, "command", payload)
		}

		// Always notify the handler to send MySensors command
		if handler, exists := c.handlers[compositeKey]; exists {
			handler(deviceName, entityName, payload)
		}
	}
}

func (c *Client) RegisterStateChangeHandler(uniqueID string, handler StateChangeHandler) {
	c.handlers[uniqueID] = handler
}


func (c *Client) createEntityStateHandler(uniqueID string, entityType string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		c.logger.Debug("Received retained entity state message", "topic", msg.Topic(), "payload", payload, "entityType", entityType)

		// Skip empty payloads (might be cleared retained messages)
		if len(payload) == 0 {
			c.logger.Debug("Skipping empty payload (cleared retained message)", "topic", msg.Topic())
			return
		}

		// Validate payload based on entity type
		if !c.validateEntityPayload(entityType, payload) {
			c.logger.Warn("Invalid retained entity state payload", "entityType", entityType, "payload", payload, "topic", msg.Topic())
			return
		}

		// Store the entity state
		c.stateMu.Lock()
		c.states[uniqueID] = payload
		c.stateMu.Unlock()
		
		c.logger.Debug("Stored retained entity state", "uniqueID", uniqueID, "payload", payload)
	}
}

// validateEntityPayload validates the payload based on entity type
func (c *Client) validateEntityPayload(entityType, payload string) bool {
	// Reuse output validation logic for actuator entity types
	switch entityType {
	case "switch", "light":
		return payload == "0" || payload == "1" || payload == "ON" || payload == "OFF"
	case "dimmer", "number":
		if payload == "0" || payload == "1" {
			return true
		}
		// Could add numeric validation here
		return true
	case "text", "select":
		return true
	case "cover":
		return payload == "UP" || payload == "DOWN" || payload == "STOP" || 
			   payload == "OPEN" || payload == "CLOSE"
	case "sensor", "binary_sensor", "temperature", "humidity", "battery", 
		 "voltage", "current", "pressure", "level", "percentage", "weight", 
		 "distance", "light_level", "watt", "kwh", "flow", "volume", "ph", 
		 "orp", "ec", "var", "va", "power_factor", "custom", "position", 
		 "uv", "rain", "rainrate", "wind", "gust", "direction", "impedance":
		// Sensor entity types accept any payload (they're reporting values)
		return true
	default:
		// For unknown types, accept any payload
		return true
	}
}

// getEffectiveOptimisticModeForEntity determines the effective optimistic mode for a specific entity
func (c *Client) getEffectiveOptimisticModeForEntity(deviceID, entityID string) bool {
	// Find the device and entity configuration
	for _, device := range c.devices {
		if device.ID == deviceID {
			for _, entity := range device.Entities {
				if entity.ID == entityID {
					// Priority: entity setting > device setting > global setting > default (false)
					if entity.Optimistic != nil {
						return *entity.Optimistic
					}
					break
				}
			}
			// No entity-specific setting found, check device setting
			break
		}
	}

	// Fall back to global setting
	if c.adapterCfg.Optimistic != nil {
		return *c.adapterCfg.Optimistic
	}

	return false // Default to false (non-optimistic)
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

	// Publish discovery for entities
	for _, entity := range device.Entities {
		entityType, discoveryConfig := c.createEntityDiscoveryConfig(device, entity, deviceInfo)
		
		configJSON, err := json.Marshal(discoveryConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal entity config: %w", err)
		}

		discoveryTopic := fmt.Sprintf("homeassistant/%s/%s_%s/config", entityType, device.ID, entity.ID)
		if err := c.Publish(discoveryTopic, string(configJSON), true); err != nil {
			return fmt.Errorf("failed to publish entity discovery: %w", err)
		}

		// Publish initial state for entities that can report state if no state already exists
		if entity.CanReportState() {
			compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
			if existingState, exists := c.GetState(compositeKey); !exists {
				initialValue := entity.InitialValue
				if initialValue == "" {
					// Set default initial values based on entity type
					switch entity.EntityType {
					case "switch", "light", "binary_sensor":
						initialValue = "0"
					case "dimmer", "number", "percentage", "level":
						initialValue = "0"
					case "text", "select", "sensor":
						initialValue = ""
					default:
						initialValue = "0"
					}
				}
				
				// Only publish initial state for read-only sensors that are binary sensors
				// For other sensors, we wait for data from MySensors device
				if entity.IsReadOnly() && entity.EntityType != "binary_sensor" {
					c.logger.Debug("Skipping initial state for read-only sensor (waiting for MySensors data)", "entity", entity.ID, "type", entity.EntityType)
				} else {
					c.SetState(compositeKey, initialValue)
					if err := c.PublishEntityState(device, entity, initialValue); err != nil {
						return fmt.Errorf("failed to publish initial entity state: %w", err)
					}
					c.logger.Debug("Published initial entity state", "entity", entity.ID, "state", initialValue)
				}
			} else {
				c.logger.Debug("Using existing retained entity state", "entity", entity.ID, "state", existingState)
			}
		}
	}

	return nil
}

// PublishEntityState publishes the state of an entity
func (c *Client) PublishEntityState(device config.Device, entity config.Entity, value string) error {
	// Publish to device-specific state topic
	deviceStateTopic := fmt.Sprintf("%s/devices/%s/entity/%s/state", c.adapterCfg.TopicPrefix, device.ID, entity.ID)
	
	// Update internal state tracking
	compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
	c.stateMu.Lock()
	c.states[compositeKey] = value
	c.stateMu.Unlock()
	
	return c.Publish(deviceStateTopic, value, true)
}

// createEntityDiscoveryConfig creates Home Assistant discovery configuration for entities
func (c *Client) createEntityDiscoveryConfig(device config.Device, entity config.Entity, deviceInfo map[string]interface{}) (string, map[string]interface{}) {
	var haEntityType string
	discoveryConfig := map[string]interface{}{
		"name":        entity.Name,
		"unique_id":   fmt.Sprintf("%s_%s_entity", device.ID, entity.ID),
		"state_topic": fmt.Sprintf("%s/devices/%s/entity/%s/state", c.adapterCfg.TopicPrefix, device.ID, entity.ID),
		"device":      deviceInfo,
	}

	// Add command topic only for entities that can receive commands
	if entity.CanReceiveCommands() {
		discoveryConfig["command_topic"] = fmt.Sprintf("%s/devices/%s/entity/%s/set", c.adapterCfg.TopicPrefix, device.ID, entity.ID)
	}

	// Map entity type to Home Assistant entity type and configure appropriately
	switch entity.EntityType {
	case "switch":
		haEntityType = "switch"
		// Set payload values with defaults
		if entity.PayloadOn != "" {
			discoveryConfig["payload_on"] = entity.PayloadOn
		} else {
			discoveryConfig["payload_on"] = "1"
		}
		if entity.PayloadOff != "" {
			discoveryConfig["payload_off"] = entity.PayloadOff
		} else {
			discoveryConfig["payload_off"] = "0"
		}
		if entity.StateOn != "" {
			discoveryConfig["state_on"] = entity.StateOn
		}
		if entity.StateOff != "" {
			discoveryConfig["state_off"] = entity.StateOff
		}

	case "light":
		haEntityType = "light"
		// Set payload values with defaults
		if entity.PayloadOn != "" {
			discoveryConfig["payload_on"] = entity.PayloadOn
		} else {
			discoveryConfig["payload_on"] = "1"
		}
		if entity.PayloadOff != "" {
			discoveryConfig["payload_off"] = entity.PayloadOff
		} else {
			discoveryConfig["payload_off"] = "0"
		}
		if entity.StateOn != "" {
			discoveryConfig["state_on"] = entity.StateOn
		}
		if entity.StateOff != "" {
			discoveryConfig["state_off"] = entity.StateOff
		}

	case "dimmer":
		haEntityType = "light"
		if entity.MinValue != nil {
			discoveryConfig["min_mireds"] = *entity.MinValue
		}
		if entity.MaxValue != nil {
			discoveryConfig["max_mireds"] = *entity.MaxValue
		}

	case "text":
		if entity.IsReadOnly() {
			// For read-only text entities, use sensor instead of text
			haEntityType = "sensor"
		} else {
			haEntityType = "text"
		}

	case "number":
		haEntityType = "number"
		if entity.MinValue != nil {
			discoveryConfig["min"] = *entity.MinValue
		}
		if entity.MaxValue != nil {
			discoveryConfig["max"] = *entity.MaxValue
		}
		if entity.Step != nil {
			discoveryConfig["step"] = *entity.Step
		}
		if entity.UnitOfMeasurement != "" {
			discoveryConfig["unit_of_measurement"] = entity.UnitOfMeasurement
		}

	case "select":
		haEntityType = "select"
		if len(entity.Options) > 0 {
			discoveryConfig["options"] = entity.Options
		}

	case "cover":
		haEntityType = "cover"
		// Cover-specific payloads
		if entity.PayloadOpen != "" {
			discoveryConfig["payload_open"] = entity.PayloadOpen
		} else {
			discoveryConfig["payload_open"] = "OPEN"
		}
		if entity.PayloadClose != "" {
			discoveryConfig["payload_close"] = entity.PayloadClose
		} else {
			discoveryConfig["payload_close"] = "CLOSE"
		}
		if entity.PayloadStop != "" {
			discoveryConfig["payload_stop"] = entity.PayloadStop
		} else {
			discoveryConfig["payload_stop"] = "STOP"
		}
		if entity.StateOpen != "" {
			discoveryConfig["state_open"] = entity.StateOpen
		}
		if entity.StateClosed != "" {
			discoveryConfig["state_closed"] = entity.StateClosed
		}

	case "binary_sensor":
		haEntityType = "binary_sensor"
		// Set payload values with defaults
		if entity.PayloadOn != "" {
			discoveryConfig["payload_on"] = entity.PayloadOn
		} else {
			discoveryConfig["payload_on"] = "1"
		}
		if entity.PayloadOff != "" {
			discoveryConfig["payload_off"] = entity.PayloadOff
		} else {
			discoveryConfig["payload_off"] = "0"
		}
		if entity.StateOn != "" {
			discoveryConfig["state_on"] = entity.StateOn
		}
		if entity.StateOff != "" {
			discoveryConfig["state_off"] = entity.StateOff
		}
		if entity.OffDelay != nil {
			discoveryConfig["off_delay"] = *entity.OffDelay
		}
		if entity.ExpireAfter != nil {
			discoveryConfig["expire_after"] = *entity.ExpireAfter
		}

	case "sensor", "temperature", "humidity", "battery", "voltage", "current", "pressure", "level", "percentage", "weight", "distance", "light_level", "watt", "kwh", "flow", "volume", "ph", "orp", "ec", "var", "va", "power_factor", "custom", "position", "uv", "rain", "rainrate", "wind", "gust", "direction", "impedance":
		haEntityType = "sensor"
		if entity.UnitOfMeasurement != "" {
			discoveryConfig["unit_of_measurement"] = entity.UnitOfMeasurement
		}
		if entity.StateClass != "" {
			discoveryConfig["state_class"] = entity.StateClass
		}
		if entity.ValueTemplate != "" {
			discoveryConfig["value_template"] = entity.ValueTemplate
		}

	default:
		// Default to sensor for unknown types
		haEntityType = "sensor"
	}

	// Apply common configurations
	if entity.Icon != "" {
		discoveryConfig["icon"] = entity.Icon
	}
	if entity.DeviceClass != "" {
		discoveryConfig["device_class"] = entity.DeviceClass
	}
	if entity.EntityCategory != "" {
		discoveryConfig["entity_category"] = entity.EntityCategory
	}
	if entity.EnabledByDefault != nil {
		discoveryConfig["enabled_by_default"] = *entity.EnabledByDefault
	}
	if entity.QOS != nil {
		discoveryConfig["qos"] = *entity.QOS
	} else {
		discoveryConfig["qos"] = 0
	}
	if entity.Retain != nil {
		discoveryConfig["retain"] = *entity.Retain
	} else {
		discoveryConfig["retain"] = true
	}
	if entity.Optimistic != nil {
		discoveryConfig["optimistic"] = *entity.Optimistic
	} else if c.adapterCfg.Optimistic != nil {
		discoveryConfig["optimistic"] = *c.adapterCfg.Optimistic
	} else {
		discoveryConfig["optimistic"] = false
	}

	// Availability configuration
	if entity.AvailabilityTopic != "" {
		discoveryConfig["availability_topic"] = entity.AvailabilityTopic
		if entity.PayloadAvailable != "" {
			discoveryConfig["payload_available"] = entity.PayloadAvailable
		} else {
			discoveryConfig["payload_available"] = "online"
		}
		if entity.PayloadNotAvailable != "" {
			discoveryConfig["payload_not_available"] = entity.PayloadNotAvailable
		} else {
			discoveryConfig["payload_not_available"] = "offline"
		}
	}

	// Template configuration
	if entity.JSONAttributesTopic != "" {
		discoveryConfig["json_attributes_topic"] = entity.JSONAttributesTopic
	}
	if entity.JSONAttributesTemplate != "" {
		discoveryConfig["json_attributes_template"] = entity.JSONAttributesTemplate
	}
	if entity.StateValueTemplate != "" {
		discoveryConfig["state_value_template"] = entity.StateValueTemplate
	}
	if entity.CommandTemplate != "" {
		discoveryConfig["command_template"] = entity.CommandTemplate
	}
	if entity.ValueTemplate != "" {
		discoveryConfig["value_template"] = entity.ValueTemplate
	}

	return haEntityType, discoveryConfig
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
