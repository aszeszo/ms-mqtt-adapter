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
	c := &Client{
		config:     cfg,
		adapterCfg: adapterCfg,
		logger:     logger,
		devices:    devices,
		states:     make(map[string]string),
		handlers:   make(map[string]StateChangeHandler),
	}

	// Resubscribe on every connect (including reconnects after connection loss).
	// With CleanSession=true (default), the broker removes all subscriptions on disconnect.
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Info("MQTT connected, resubscribing to topics...")
		if err := c.subscribeToDevices(); err != nil {
			logger.Error("Failed to resubscribe to device topics on reconnect", "error", err)
		}
		if err := c.subscribeToStateTopic(); err != nil {
			logger.Error("Failed to resubscribe to state topics on reconnect", "error", err)
		}
	})

	c.client = mqtt.NewClient(opts)
	return c
}

func (c *Client) Connect(ctx context.Context) error {
	token := c.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT connection timeout")
	}
	if token.Error() != nil {
		return fmt.Errorf("MQTT connection failed: %w", token.Error())
	}

	// Subscriptions are handled by the OnConnectHandler (called on every connect/reconnect)
	c.logger.Info("MQTT client connected")
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

			// Use new entity-based topic structure with unique_id
			uniqueID := entity.GetEffectiveUniqueID(device.ID)
			topic := fmt.Sprintf("%s/entity/%s/set", c.adapterCfg.TopicPrefix, uniqueID)
			// Create composite key for uniqueness across devices (for internal state tracking)
			compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
			// QoS 1 for at-least-once delivery (prevents message loss during network issues)
		token := c.client.Subscribe(topic, 1, c.createEntityHandler(device.Name, entity.Name, compositeKey, device.ID, entity.ID, entity.EntityType))
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

			uniqueID := entity.GetEffectiveUniqueID(device.ID)
			stateTopic := fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, uniqueID)
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

		// Silently ignore empty payloads (HA sometimes sends these during discovery)
		if payload == "" {
			c.logger.Debug("Ignoring empty payload", "topic", msg.Topic())
			return
		}

		// Validate payload based on entity type
		if !c.validateEntityPayload(entityType, payload) {
			c.logger.Warn("Invalid entity payload", "entityType", entityType, "payload", payload)
			return
		}

		// Check if entity is configured for optimistic mode
		optimistic := c.getEffectiveOptimisticModeForEntity(deviceID, entityID)

		if optimistic {
			// Optimistic mode: update MQTT state immediately (assume command will succeed)
			// Find the entity to get its unique_id
			var entityUniqueID string
			for _, device := range c.devices {
				if device.ID == deviceID {
					for _, entity := range device.Entities {
						if entity.ID == entityID {
							entityUniqueID = entity.GetEffectiveUniqueID(device.ID)
							break
						}
					}
					break
				}
			}
			deviceStateTopic := fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, entityUniqueID)
			c.Publish(deviceStateTopic, payload, true)

			c.stateMu.Lock()
			c.states[compositeKey] = payload
			c.stateMu.Unlock()

			c.logger.Debug("Optimistic mode: updated MQTT state immediately", "device", deviceName, "entity", entityName, "state", payload)
		} else {
			// Non-optimistic mode: wait for MySensors device confirmation before updating MQTT state
			c.logger.Debug("Non-optimistic mode: waiting for device confirmation", "device", deviceName, "entity", entityName, "command", payload)
		}

		// Notify the handler to send MySensors command in a goroutine.
		// This is critical: sendWithAck() blocks for up to ackTimeout * retries,
		// which would block paho's single message router goroutine and prevent
		// delivery of all subsequent MQTT messages until the ACK completes.
		if handler, exists := c.handlers[compositeKey]; exists {
			go handler(deviceName, entityName, payload)
		} else {
			c.logger.Warn("MQTT command received but no handler registered",
				"device", deviceName, "entity", entityName, "compositeKey", compositeKey, "topic", msg.Topic())
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
					// Entity-level only setting, defaults to false (non-optimistic)
					if entity.Discovery.Optimistic != nil {
						return *entity.Discovery.Optimistic
					}
					break
				}
			}
			break
		}
	}

	return false // Default to false (non-optimistic mode)
}

func (c *Client) Publish(topic, payload string, retain bool) error {
	token := c.client.Publish(topic, 0, retain, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic %s", topic)
	}
	if token.Error() != nil {
		return fmt.Errorf("publish failed for topic %s: %w", topic, token.Error())
	}

	c.logger.Debug("MQTT TX", "topic", topic, "payload", payload, "retain", retain)
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

// GetAllStates returns a copy of all cached entity states.
func (c *Client) GetAllStates() map[string]string {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	cp := make(map[string]string, len(c.states))
	for k, v := range c.states {
		cp[k] = v
	}
	return cp
}

func (c *Client) PublishHomeAssistantDiscovery(device config.Device) error {
	// Only publish HomeAssistant discovery if enabled
	if c.adapterCfg.HomeAssistantDiscovery == nil || !*c.adapterCfg.HomeAssistantDiscovery {
		return nil
	}

	c.logger.Debug("Publishing/updating Home Assistant discovery configuration", "device", device.Name)

	dd := device.Discovery // local shorthand for device discovery fields
	deviceInfo := map[string]interface{}{
		"identifiers":  []string{device.ID},
		"name":         device.Name,
		"manufacturer": dd.Manufacturer,
		"model":        dd.Model,
		"sw_version":   dd.SWVersion,
		"hw_version":   dd.HWVersion,
	}

	// Add optional device fields
	if dd.ModelID != "" {
		deviceInfo["model_id"] = dd.ModelID
	}
	if dd.SerialNumber != "" {
		deviceInfo["serial_number"] = dd.SerialNumber
	}
	if dd.ConfigurationURL != "" {
		deviceInfo["configuration_url"] = dd.ConfigurationURL
	}
	if dd.SuggestedArea != "" {
		deviceInfo["suggested_area"] = dd.SuggestedArea
	}
	if len(dd.Connections) > 0 {
		deviceInfo["connections"] = dd.Connections
	}
	if dd.ViaDevice != "" {
		deviceInfo["via_device"] = dd.ViaDevice
	}

	// Publish discovery for entities
	for _, entity := range device.Entities {
		entityType, discoveryConfig := c.createEntityDiscoveryConfig(device, entity, deviceInfo)

		configJSON, err := json.Marshal(discoveryConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal entity config: %w", err)
		}

		discoveryTopic := fmt.Sprintf("%s/%s/%s/config", c.adapterCfg.DiscoveryPrefix, entityType, entity.GetEffectiveUniqueID(device.ID))
		if err := c.Publish(discoveryTopic, string(configJSON), true); err != nil {
			return fmt.Errorf("failed to publish entity discovery: %w", err)
		}
		c.logger.Debug("Published/updated Home Assistant discovery for entity", "device", device.Name, "entity", entity.Name, "topic", discoveryTopic)

		// Publish availability status as "online" (unless disabled)
		if entity.Discovery.AvailabilityTopic != "none" {
			if err := c.PublishEntityAvailability(device, entity, true); err != nil {
				return fmt.Errorf("failed to publish entity availability: %w", err)
			}
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
	// Publish to entity-specific state topic using unique_id
	uniqueID := entity.GetEffectiveUniqueID(device.ID)
	deviceStateTopic := fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, uniqueID)

	// Update internal state tracking
	compositeKey := fmt.Sprintf("%s_%s_entity", device.ID, entity.ID)
	c.stateMu.Lock()
	c.states[compositeKey] = value
	c.stateMu.Unlock()

	return c.Publish(deviceStateTopic, value, true)
}

// PublishEntityAvailability publishes the availability status of an entity
func (c *Client) PublishEntityAvailability(device config.Device, entity config.Entity, available bool) error {
	uniqueID := entity.GetEffectiveUniqueID(device.ID)
	availabilityTopic := fmt.Sprintf("%s/entity/%s/availability", c.adapterCfg.TopicPrefix, uniqueID)

	payload := "offline"
	if available {
		payload = "online"
	}

	// Use custom payloads if specified
	if available && entity.Discovery.PayloadAvailable != "" {
		payload = entity.Discovery.PayloadAvailable
	} else if !available && entity.Discovery.PayloadNotAvailable != "" {
		payload = entity.Discovery.PayloadNotAvailable
	}

	c.logger.Debug("Publishing entity availability", "device", device.Name, "entity", entity.Name, "topic", availabilityTopic, "payload", payload)
	return c.Publish(availabilityTopic, payload, true)
}

// createEntityDiscoveryConfig creates Home Assistant discovery configuration for entities
func (c *Client) createEntityDiscoveryConfig(device config.Device, entity config.Entity, deviceInfo map[string]interface{}) (string, map[string]interface{}) {
	d := entity.Discovery // local shorthand for discovery fields

	var haEntityType string
	discoveryConfig := map[string]interface{}{
		"name":        entity.Name,
		"unique_id":   entity.GetEffectiveUniqueID(device.ID),
		"state_topic": fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, entity.GetEffectiveUniqueID(device.ID)),
		"device":      deviceInfo,
	}

	// Add default_entity_id
	// Format: {domain}.{base_id} (e.g., "binary_sensor.piwnica_wlacznik_schody")
	if defaultEntityID, shouldInclude := entity.GetEffectiveDefaultEntityID(device.ID); shouldInclude {
		discoveryConfig["default_entity_id"] = defaultEntityID
	}

	// Add command topic only for entities that can receive commands
	if entity.CanReceiveCommands() {
		discoveryConfig["command_topic"] = fmt.Sprintf("%s/entity/%s/set", c.adapterCfg.TopicPrefix, entity.GetEffectiveUniqueID(device.ID))
	}

	// Origin object - identifies the integration that created this entity
	discoveryConfig["origin"] = map[string]string{
		"name":        "ms-mqtt-adapter",
		"sw_version":  "2.2.6",
		"support_url": "https://github.com/aszeszo/ms-mqtt-adapter",
	}

	// Map entity type to Home Assistant entity type and configure appropriately
	switch entity.EntityType {
	case "switch":
		haEntityType = "switch"
		if d.PayloadOn != "" {
			discoveryConfig["payload_on"] = d.PayloadOn
		} else {
			discoveryConfig["payload_on"] = "1"
		}
		if d.PayloadOff != "" {
			discoveryConfig["payload_off"] = d.PayloadOff
		} else {
			discoveryConfig["payload_off"] = "0"
		}
		if d.StateOn != "" {
			discoveryConfig["state_on"] = d.StateOn
		}
		if d.StateOff != "" {
			discoveryConfig["state_off"] = d.StateOff
		}

	case "light":
		haEntityType = "light"
		if d.PayloadOn != "" {
			discoveryConfig["payload_on"] = d.PayloadOn
		} else {
			discoveryConfig["payload_on"] = "1"
		}
		if d.PayloadOff != "" {
			discoveryConfig["payload_off"] = d.PayloadOff
		} else {
			discoveryConfig["payload_off"] = "0"
		}
		if d.StateOn != "" {
			discoveryConfig["state_on"] = d.StateOn
		}
		if d.StateOff != "" {
			discoveryConfig["state_off"] = d.StateOff
		}

	case "dimmer":
		haEntityType = "light"
		if d.MinValue != nil {
			discoveryConfig["min_mireds"] = *d.MinValue
		}
		if d.MaxValue != nil {
			discoveryConfig["max_mireds"] = *d.MaxValue
		}

	case "text":
		if entity.IsReadOnly() {
			haEntityType = "sensor"
		} else {
			haEntityType = "text"
			if d.Mode != "" {
				discoveryConfig["mode"] = d.Mode
			}
			if d.Pattern != "" {
				discoveryConfig["pattern"] = d.Pattern
			}
		}

	case "number":
		haEntityType = "number"
		if d.MinValue != nil {
			discoveryConfig["min"] = *d.MinValue
		}
		if d.MaxValue != nil {
			discoveryConfig["max"] = *d.MaxValue
		}
		if d.Step != nil {
			discoveryConfig["step"] = *d.Step
		}
		if d.UnitOfMeasurement != "" {
			discoveryConfig["unit_of_measurement"] = d.UnitOfMeasurement
		}
		if d.Mode != "" {
			discoveryConfig["mode"] = d.Mode
		}

	case "select":
		haEntityType = "select"
		if len(d.Options) > 0 {
			discoveryConfig["options"] = d.Options
		}

	case "cover":
		haEntityType = "cover"
		if d.PayloadOpen != "" {
			discoveryConfig["payload_open"] = d.PayloadOpen
		} else {
			discoveryConfig["payload_open"] = "OPEN"
		}
		if d.PayloadClose != "" {
			discoveryConfig["payload_close"] = d.PayloadClose
		} else {
			discoveryConfig["payload_close"] = "CLOSE"
		}
		if d.PayloadStop != "" {
			discoveryConfig["payload_stop"] = d.PayloadStop
		} else {
			discoveryConfig["payload_stop"] = "STOP"
		}
		if d.StateOpen != "" {
			discoveryConfig["state_open"] = d.StateOpen
		}
		if d.StateClosed != "" {
			discoveryConfig["state_closed"] = d.StateClosed
		}
		if d.StateOpening != "" {
			discoveryConfig["state_opening"] = d.StateOpening
		}
		if d.StateClosing != "" {
			discoveryConfig["state_closing"] = d.StateClosing
		}
		if d.StateStopped != "" {
			discoveryConfig["state_stopped"] = d.StateStopped
		}

	case "binary_sensor":
		haEntityType = "binary_sensor"
		if d.PayloadOn != "" {
			discoveryConfig["payload_on"] = d.PayloadOn
		} else {
			discoveryConfig["payload_on"] = "1"
		}
		if d.PayloadOff != "" {
			discoveryConfig["payload_off"] = d.PayloadOff
		} else {
			discoveryConfig["payload_off"] = "0"
		}
		if d.StateOn != "" {
			discoveryConfig["state_on"] = d.StateOn
		}
		if d.StateOff != "" {
			discoveryConfig["state_off"] = d.StateOff
		}
		if d.OffDelay != nil {
			discoveryConfig["off_delay"] = *d.OffDelay
		}
		if d.ExpireAfter != nil {
			discoveryConfig["expire_after"] = *d.ExpireAfter
		}
		if d.ForceUpdate != nil {
			discoveryConfig["force_update"] = *d.ForceUpdate
		}

	case "sensor", "temperature", "humidity", "battery", "voltage", "current", "pressure", "level", "percentage", "weight", "distance", "light_level", "watt", "kwh", "flow", "volume", "ph", "orp", "ec", "var", "va", "power_factor", "custom", "position", "uv", "rain", "rainrate", "wind", "gust", "direction", "impedance":
		haEntityType = "sensor"
		if d.UnitOfMeasurement != "" {
			discoveryConfig["unit_of_measurement"] = d.UnitOfMeasurement
		}
		if d.StateClass != "" {
			discoveryConfig["state_class"] = d.StateClass
		}
		if d.ValueTemplate != "" {
			discoveryConfig["value_template"] = d.ValueTemplate
		}
		if d.ExpireAfter != nil {
			discoveryConfig["expire_after"] = *d.ExpireAfter
		}
		if d.ForceUpdate != nil {
			discoveryConfig["force_update"] = *d.ForceUpdate
		}
		if d.SuggestedDisplayPrecision != nil {
			discoveryConfig["suggested_display_precision"] = *d.SuggestedDisplayPrecision
		}

	default:
		// Default to sensor for unknown types
		haEntityType = "sensor"
	}

	// Apply common configurations
	if d.Icon != "" {
		discoveryConfig["icon"] = d.Icon
	}
	if d.DeviceClass != "" {
		discoveryConfig["device_class"] = d.DeviceClass
	}
	if d.EntityCategory != "" {
		discoveryConfig["entity_category"] = d.EntityCategory
	}
	if d.EntityPicture != "" {
		discoveryConfig["entity_picture"] = d.EntityPicture
	}
	if d.EnabledByDefault != nil {
		discoveryConfig["enabled_by_default"] = *d.EnabledByDefault
	}
	if d.QOS != nil {
		discoveryConfig["qos"] = *d.QOS
	} else {
		discoveryConfig["qos"] = 0
	}
	if d.Retain != nil {
		discoveryConfig["retain"] = *d.Retain
	} else {
		discoveryConfig["retain"] = true
	}
	if d.Optimistic != nil {
		discoveryConfig["optimistic"] = *d.Optimistic
	} else {
		discoveryConfig["optimistic"] = false // Default to false (wait for device confirmation)
	}

	// Availability configuration
	// If AvailabilityTopic is "none", skip availability configuration entirely (assume always available)
	// If empty (not set), use default auto-generated topic
	// If "default", use default auto-generated topic
	// Otherwise use custom topic
	if d.AvailabilityTopic != "none" {
		var availabilityTopic string
		if d.AvailabilityTopic == "" || d.AvailabilityTopic == "default" {
			uniqueID := entity.GetEffectiveUniqueID(device.ID)
			availabilityTopic = fmt.Sprintf("%s/entity/%s/availability", c.adapterCfg.TopicPrefix, uniqueID)
		} else {
			availabilityTopic = d.AvailabilityTopic
		}
		discoveryConfig["availability_topic"] = availabilityTopic
		if d.PayloadAvailable != "" {
			discoveryConfig["payload_available"] = d.PayloadAvailable
		} else {
			discoveryConfig["payload_available"] = "online"
		}
		if d.PayloadNotAvailable != "" {
			discoveryConfig["payload_not_available"] = d.PayloadNotAvailable
		} else {
			discoveryConfig["payload_not_available"] = "offline"
		}

		c.logger.Debug("Entity availability topic configured", "device", device.Name, "entity", entity.Name, "topic", availabilityTopic)
	} else {
		c.logger.Debug("Entity availability disabled (always available)", "device", device.Name, "entity", entity.Name)
	}

	// Template configuration
	if d.JSONAttributesTopic != "" {
		discoveryConfig["json_attributes_topic"] = d.JSONAttributesTopic
	}
	if d.JSONAttributesTemplate != "" {
		discoveryConfig["json_attributes_template"] = d.JSONAttributesTemplate
	}
	if d.StateValueTemplate != "" {
		discoveryConfig["state_value_template"] = d.StateValueTemplate
	}
	if d.CommandTemplate != "" {
		discoveryConfig["command_template"] = d.CommandTemplate
	}
	if d.ValueTemplate != "" {
		discoveryConfig["value_template"] = d.ValueTemplate
	}

	return haEntityType, discoveryConfig
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

func (c *Client) PublishLastIssuedNodeID(gatewayName string, nodeID int) error {
	topic := fmt.Sprintf("%s/gateway/%s/last_id", c.adapterCfg.TopicPrefix, gatewayName)
	payload := fmt.Sprintf("%d", nodeID)
	return c.Publish(topic, payload, true)
}

// PublishPresentationMessage publishes presentation messages under gateway/node structure with deduplication
func (c *Client) PublishPresentationMessage(topicPrefix, gatewayName string, nodeID, childID int, sensorType string, description string) error {
	topic := fmt.Sprintf("%s/gateway/%s/node/%d/presentation", topicPrefix, gatewayName, nodeID)

	// Get existing presentations for deduplication
	existingState, exists := c.GetState(fmt.Sprintf("presentation_%s_%d", gatewayName, nodeID))

	// Create new presentation entry
	newEntry := fmt.Sprintf("child_id:%d;sensor_type:%s;description:%s", childID, sensorType, description)

	var updatedPresentations string
	if exists && existingState != "" {
		// Parse existing presentations and check for duplicates
		existingEntries := strings.Split(existingState, "\n")
		entryExists := false

		for _, entry := range existingEntries {
			if entry == newEntry {
				entryExists = true
				break
			}
		}

		if !entryExists {
			// Append new entry with newline
			updatedPresentations = existingState + "\n" + newEntry
		} else {
			// Entry already exists, no update needed
			updatedPresentations = existingState
		}
	} else {
		// First presentation for this node
		updatedPresentations = newEntry
	}

	// Update internal state
	c.SetState(fmt.Sprintf("presentation_%s_%d", gatewayName, nodeID), updatedPresentations)

	// Publish the accumulated presentations
	return c.Publish(topic, updatedPresentations, true)
}

// Reconfigure updates the MQTT client configuration and resubscribes to topics
func (c *Client) Reconfigure(cfg *config.MQTTConfig, adapterCfg *config.AdapterConfig, devices []config.Device) error {
	c.config = cfg
	c.adapterCfg = adapterCfg
	oldDevices := c.devices
	c.devices = devices

	// Skip resubscription if not connected - will resubscribe on reconnect via OnConnectHandler
	if !c.client.IsConnected() {
		c.logger.Info("MQTT client not connected, will resubscribe on reconnect")
		return nil
	}

	// Unsubscribe from old device topics before resubscribing
	c.unsubscribeDevices(oldDevices)

	// Resubscribe to device topics with new configuration
	if err := c.subscribeToDevices(); err != nil {
		return fmt.Errorf("failed to resubscribe to device topics: %w", err)
	}

	// Resubscribe to state topics
	if err := c.subscribeToStateTopic(); err != nil {
		return fmt.Errorf("failed to resubscribe to state topics: %w", err)
	}

	c.logger.Info("MQTT client reconfigured successfully")
	return nil
}

// unsubscribeDevices unsubscribes from all entity topics for the given device list
func (c *Client) unsubscribeDevices(devices []config.Device) {
	for _, device := range devices {
		for _, entity := range device.Entities {
			uniqueID := entity.GetEffectiveUniqueID(device.ID)

			if entity.CanReceiveCommands() {
				topic := fmt.Sprintf("%s/entity/%s/set", c.adapterCfg.TopicPrefix, uniqueID)
				token := c.client.Unsubscribe(topic)
				token.WaitTimeout(5 * time.Second)
			}

			if entity.CanReportState() {
				stateTopic := fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, uniqueID)
				token := c.client.Unsubscribe(stateTopic)
				token.WaitTimeout(5 * time.Second)
			}
		}
	}
	c.logger.Debug("Unsubscribed from old device topics", "device_count", len(devices))
}

// GetTopicList returns all MQTT topics managed by this adapter (implements MQTTClientProvider)
func (c *Client) GetTopicList(gateways map[string]config.MySensorsConfig) interface{} {
	return c.GetAllTopics(gateways)
}

// DeleteTopics clears MQTT topics based on scope (implements MQTTClientProvider)
func (c *Client) DeleteTopics(scope, deviceID, entityID string, gateways map[string]config.MySensorsConfig) error {
	return c.ClearTopics(scope, deviceID, entityID, gateways)
}

// BrowseAllTopics subscribes to # and collects all messages for the given timeout duration.
func (c *Client) BrowseAllTopics(timeout time.Duration) (any, error) {
	if !c.client.IsConnected() {
		return nil, fmt.Errorf("MQTT client not connected")
	}

	var mu sync.Mutex
	collected := make(map[string]BrokerTopic)

	handler := func(client mqtt.Client, msg mqtt.Message) {
		mu.Lock()
		collected[msg.Topic()] = BrokerTopic{
			Topic:    msg.Topic(),
			Payload:  string(msg.Payload()),
			Retained: msg.Retained(),
		}
		mu.Unlock()
	}

	token := c.client.Subscribe("#", 0, handler)
	if !token.WaitTimeout(5 * time.Second) {
		return nil, fmt.Errorf("subscribe to # timed out")
	}
	if token.Error() != nil {
		return nil, fmt.Errorf("subscribe to # failed: %w", token.Error())
	}

	time.Sleep(timeout)

	unsubToken := c.client.Unsubscribe("#")
	unsubToken.WaitTimeout(5 * time.Second)

	mu.Lock()
	result := make([]BrokerTopic, 0, len(collected))
	for _, t := range collected {
		result = append(result, t)
	}
	mu.Unlock()

	sort.Slice(result, func(i, j int) bool {
		return result[i].Topic < result[j].Topic
	})

	return result, nil
}

// DeleteRetainedTopic clears a single retained message by publishing empty payload with retain flag.
func (c *Client) DeleteRetainedTopic(topic string) error {
	return c.Publish(topic, "", true)
}

// DeleteRetainedTree subscribes to prefix/# to discover all retained topics under a prefix,
// then publishes empty retained messages to clear them. Returns the count of deleted topics.
func (c *Client) DeleteRetainedTree(prefix string, timeout time.Duration) (int, error) {
	if !c.client.IsConnected() {
		return 0, fmt.Errorf("MQTT client not connected")
	}

	var mu sync.Mutex
	var retainedTopics []string

	subTopic := prefix + "/#"
	handler := func(client mqtt.Client, msg mqtt.Message) {
		if msg.Retained() {
			mu.Lock()
			retainedTopics = append(retainedTopics, msg.Topic())
			mu.Unlock()
		}
	}

	token := c.client.Subscribe(subTopic, 0, handler)
	if !token.WaitTimeout(5 * time.Second) {
		return 0, fmt.Errorf("subscribe to %s timed out", subTopic)
	}
	if token.Error() != nil {
		return 0, fmt.Errorf("subscribe to %s failed: %w", subTopic, token.Error())
	}

	time.Sleep(timeout)

	unsubToken := c.client.Unsubscribe(subTopic)
	unsubToken.WaitTimeout(5 * time.Second)

	mu.Lock()
	topics := make([]string, len(retainedTopics))
	copy(topics, retainedTopics)
	mu.Unlock()

	for _, t := range topics {
		if err := c.Publish(t, "", true); err != nil {
			c.logger.Error("Failed to delete retained topic", "topic", t, "error", err)
			return 0, err
		}
	}

	c.logger.Info("Deleted retained topic tree", "prefix", prefix, "count", len(topics))
	return len(topics), nil
}
