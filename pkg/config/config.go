package config

import (
	"fmt"
	"ms-mqtt-adapter/internal/mysensors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel      string                     `yaml:"log_level" json:"log_level"`
	MySensors     map[string]MySensorsConfig `yaml:"mysensors" json:"mysensors"`
	MQTT          MQTTConfig                 `yaml:"mqtt" json:"mqtt"`
	AdapterTopics AdapterConfig              `yaml:"adapter" json:"adapter"`
	IDaliases     map[string]int             `yaml:"id_aliases,omitempty" json:"id_aliases,omitempty"`
	Devices       []Device                   `yaml:"devices" json:"devices"`
}

type EthernetConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type RS485Config struct {
	Device string `yaml:"device" json:"device"`
}

type MySensorsConfig struct {
	Transport  string           `yaml:"transport" json:"transport"`
	Ethernet   EthernetConfig   `yaml:"ethernet" json:"ethernet"`
	RS485      RS485Config      `yaml:"rs485" json:"rs485"`
	Gateway    GatewayConfig    `yaml:"gateway" json:"gateway"`
	TCPService TCPServiceConfig `yaml:"tcp_service" json:"tcp_service"`
	HalfDuplex HalfDuplexConfig `yaml:"half_duplex" json:"half_duplex"`
}

type HalfDuplexConfig struct {
	Enabled           bool          `yaml:"enabled" json:"enabled"`
	BusQuietTime      time.Duration `yaml:"bus_quiet_time" json:"bus_quiet_time"`
	InterMessageDelay time.Duration `yaml:"inter_message_delay" json:"inter_message_delay"`
	MaxTXWait         time.Duration `yaml:"max_tx_wait" json:"max_tx_wait"`
}

type MQTTConfig struct {
	Broker   string `yaml:"broker" json:"broker"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	ClientID string `yaml:"client_id" json:"client_id"`
}

type TCPServiceConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Port    int  `yaml:"port" json:"port"`
}

type NodeIDRangeConfig struct {
	Start int `yaml:"start" json:"start"`
	End   int `yaml:"end" json:"end"`
}

type NodeIDAssignmentConfig struct {
	Enabled            *bool             `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	NodeIDRange        NodeIDRangeConfig `yaml:"node_id_range" json:"node_id_range"`
	RandomIDAssignment *bool             `yaml:"random_id_assignment,omitempty" json:"random_id_assignment,omitempty"`
}

type GatewayConfig struct {
	NodeIDAssignment       NodeIDAssignmentConfig `yaml:"node_id_assignment" json:"node_id_assignment"`
	HeartbeatRequestPeriod time.Duration          `yaml:"heartbeat_request_period" json:"heartbeat_request_period"`
	AvailabilityWindow     time.Duration          `yaml:"availability_window" json:"availability_window"`
}

type AdapterConfig struct {
	TopicPrefix            string `yaml:"topic_prefix" json:"topic_prefix"`
	DiscoveryPrefix        string `yaml:"discovery_prefix" json:"discovery_prefix"`
	HomeAssistantDiscovery *bool  `yaml:"homeassistant_discovery,omitempty" json:"homeassistant_discovery,omitempty"`
}

// DeviceDiscoveryConfig contains all fields that map to HA MQTT discovery device info keys.
type DeviceDiscoveryConfig struct {
	Manufacturer     string     `yaml:"manufacturer" json:"manufacturer"`
	Model            string     `yaml:"model" json:"model"`
	ModelID          string     `yaml:"model_id,omitempty" json:"model_id,omitempty"`
	SWVersion        string     `yaml:"sw_version" json:"sw_version"`
	HWVersion        string     `yaml:"hw_version" json:"hw_version"`
	SerialNumber     string     `yaml:"serial_number,omitempty" json:"serial_number,omitempty"`
	ConfigurationURL string     `yaml:"configuration_url,omitempty" json:"configuration_url,omitempty"`
	SuggestedArea    string     `yaml:"suggested_area,omitempty" json:"suggested_area,omitempty"`
	Connections      [][]string `yaml:"connections,omitempty" json:"connections,omitempty"`
	ViaDevice        string     `yaml:"via_device,omitempty" json:"via_device,omitempty"`
}

type Device struct {
	Name      string                `yaml:"name" json:"name"`
	ID        string                `yaml:"id" json:"id"`
	NodeID    any                   `yaml:"node_id,omitempty" json:"node_id,omitempty"`
	Gateway   string                `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	Discovery DeviceDiscoveryConfig `yaml:"discovery,omitempty" json:"discovery,omitempty"`
	Entities  []Entity              `yaml:"entities" json:"entities"`
}

// Entity represents a unified MySensors entity that can be an input (sensor), output (actuator), or both
type Entity struct {
	Name               string  `yaml:"name" json:"name"`
	ID                 string  `yaml:"id" json:"id"`
	UniqueID           string  `yaml:"unique_id,omitempty" json:"unique_id,omitempty"`
	DefaultEntityID    *string `yaml:"default_entity_id,omitempty" json:"default_entity_id,omitempty"`
	ChildID            any     `yaml:"child_id" json:"child_id"`
	NodeID             any     `yaml:"node_id,omitempty" json:"node_id,omitempty"`
	Gateway            string  `yaml:"gateway,omitempty" json:"gateway,omitempty"`

	EntityType         string `yaml:"entity_type" json:"entity_type"`
	VariableType       string `yaml:"variable_type,omitempty" json:"variable_type,omitempty"`

	ReadOnly           *bool  `yaml:"read_only,omitempty" json:"read_only,omitempty"`
	WriteOnly          *bool  `yaml:"write_only,omitempty" json:"write_only,omitempty"`

	SyncPeriod         *time.Duration `yaml:"sync_period,omitempty" json:"sync_period,omitempty"`
	SyncSplay          *time.Duration `yaml:"sync_splay,omitempty" json:"sync_splay,omitempty"`

	AvailabilityWindow *time.Duration `yaml:"availability_window,omitempty" json:"availability_window,omitempty"`

	InitialValue       string `yaml:"initial_value,omitempty" json:"initial_value,omitempty"`

	RequestAck         *bool          `yaml:"request_ack,omitempty" json:"request_ack,omitempty"`
	AckTimeout         *time.Duration `yaml:"ack_timeout,omitempty" json:"ack_timeout,omitempty"`
	AckRetries         *int           `yaml:"ack_retries,omitempty" json:"ack_retries,omitempty"`

	// Discovery contains all HA MQTT discovery payload fields
	Discovery          DiscoveryConfig `yaml:"discovery,omitempty" json:"discovery,omitempty"`
}

// DiscoveryConfig contains all fields that map to Home Assistant MQTT discovery payload keys.
type DiscoveryConfig struct {
	// Display & classification
	Icon                       string `yaml:"icon,omitempty" json:"icon,omitempty"`
	DeviceClass                string `yaml:"device_class,omitempty" json:"device_class,omitempty"`
	EntityCategory             string `yaml:"entity_category,omitempty" json:"entity_category,omitempty"`
	EntityPicture              string `yaml:"entity_picture,omitempty" json:"entity_picture,omitempty"`
	EnabledByDefault           *bool  `yaml:"enabled_by_default,omitempty" json:"enabled_by_default,omitempty"`

	// Sensor/number configuration
	UnitOfMeasurement          string   `yaml:"unit_of_measurement,omitempty" json:"unit_of_measurement,omitempty"`
	StateClass                 string   `yaml:"state_class,omitempty" json:"state_class,omitempty"`
	SuggestedDisplayPrecision  *int     `yaml:"suggested_display_precision,omitempty" json:"suggested_display_precision,omitempty"`
	ForceUpdate                *bool    `yaml:"force_update,omitempty" json:"force_update,omitempty"`
	MinValue                   *float64 `yaml:"min_value,omitempty" json:"min_value,omitempty"`
	MaxValue                   *float64 `yaml:"max_value,omitempty" json:"max_value,omitempty"`
	Step                       *float64 `yaml:"step,omitempty" json:"step,omitempty"`
	Options                    []string `yaml:"options,omitempty" json:"options,omitempty"`
	Mode                       string   `yaml:"mode,omitempty" json:"mode,omitempty"`
	Pattern                    string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`

	// Availability
	AvailabilityTopic          string `yaml:"availability_topic,omitempty" json:"availability_topic,omitempty"`
	PayloadAvailable           string `yaml:"payload_available,omitempty" json:"payload_available,omitempty"`
	PayloadNotAvailable        string `yaml:"payload_not_available,omitempty" json:"payload_not_available,omitempty"`

	// Switch/light/binary_sensor payloads
	PayloadOn                  string `yaml:"payload_on,omitempty" json:"payload_on,omitempty"`
	PayloadOff                 string `yaml:"payload_off,omitempty" json:"payload_off,omitempty"`
	StateOn                    string `yaml:"state_on,omitempty" json:"state_on,omitempty"`
	StateOff                   string `yaml:"state_off,omitempty" json:"state_off,omitempty"`

	// Cover payloads
	PayloadOpen                string `yaml:"payload_open,omitempty" json:"payload_open,omitempty"`
	PayloadClose               string `yaml:"payload_close,omitempty" json:"payload_close,omitempty"`
	PayloadStop                string `yaml:"payload_stop,omitempty" json:"payload_stop,omitempty"`
	StateOpen                  string `yaml:"state_open,omitempty" json:"state_open,omitempty"`
	StateClosed                string `yaml:"state_closed,omitempty" json:"state_closed,omitempty"`
	StateOpening               string `yaml:"state_opening,omitempty" json:"state_opening,omitempty"`
	StateClosing               string `yaml:"state_closing,omitempty" json:"state_closing,omitempty"`
	StateStopped               string `yaml:"state_stopped,omitempty" json:"state_stopped,omitempty"`

	// MQTT transport
	QOS                        *int  `yaml:"qos,omitempty" json:"qos,omitempty"`
	Retain                     *bool `yaml:"retain,omitempty" json:"retain,omitempty"`
	Optimistic                 *bool `yaml:"optimistic,omitempty" json:"optimistic,omitempty"`

	// Timing
	OffDelay                   *int `yaml:"off_delay,omitempty" json:"off_delay,omitempty"`
	ExpireAfter                *int `yaml:"expire_after,omitempty" json:"expire_after,omitempty"`

	// Templates
	JSONAttributesTopic        string `yaml:"json_attributes_topic,omitempty" json:"json_attributes_topic,omitempty"`
	JSONAttributesTemplate     string `yaml:"json_attributes_template,omitempty" json:"json_attributes_template,omitempty"`
	StateValueTemplate         string `yaml:"state_value_template,omitempty" json:"state_value_template,omitempty"`
	CommandTemplate            string `yaml:"command_template,omitempty" json:"command_template,omitempty"`
	ValueTemplate              string `yaml:"value_template,omitempty" json:"value_template,omitempty"`
}


func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	return LoadConfigFromBytes(data)
}

// LoadConfigFromBytes parses, validates and applies defaults to a config from raw YAML bytes.
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	SetDefaults(&config)
	return &config, nil
}

// SaveConfig marshals the config to YAML and writes it atomically to the given path.
func SaveConfig(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}

// ConfigCompleteness represents the completeness status of the configuration
type ConfigCompleteness struct {
	Complete      bool     `json:"complete"`
	MissingFields []string `json:"missing_fields"`
}

// CheckConfigCompleteness checks if all required config fields are present
func CheckConfigCompleteness(config *Config) ConfigCompleteness {
	var missing []string

	// Check MySensors gateways
	if len(config.MySensors) == 0 {
		missing = append(missing, "mysensors (at least one gateway required)")
	}

	// Check MQTT broker
	if config.MQTT.Broker == "" {
		missing = append(missing, "mqtt.broker")
	}

	return ConfigCompleteness{
		Complete:      len(missing) == 0,
		MissingFields: missing,
	}
}

func ValidateConfig(config *Config) error {
	// Track TCP service ports to ensure no conflicts
	tcpPorts := make(map[int]string)

	// Validate each MySensors gateway configuration (if any)
	for gatewayName, mysensorsConfig := range config.MySensors {
		// Transport will be set to default "ethernet" in setDefaults if not specified
		if mysensorsConfig.Transport != "" && mysensorsConfig.Transport != "ethernet" && mysensorsConfig.Transport != "rs485" {
			return fmt.Errorf("mysensors gateway '%s' transport must be 'ethernet' or 'rs485'", gatewayName)
		}

		// Validate TCP service ports for conflicts
		if mysensorsConfig.TCPService.Enabled {
			if mysensorsConfig.TCPService.Port == 0 {
				return fmt.Errorf("mysensors gateway '%s': tcp_service port must be explicitly specified when enabled", gatewayName)
			}
			if existingGateway, exists := tcpPorts[mysensorsConfig.TCPService.Port]; exists {
				return fmt.Errorf("mysensors gateway '%s' TCP service port %d conflicts with gateway '%s'",
					gatewayName, mysensorsConfig.TCPService.Port, existingGateway)
			}
			tcpPorts[mysensorsConfig.TCPService.Port] = gatewayName
		}
	}

	// Validate id_aliases
	if config.IDaliases != nil {
		for alias, id := range config.IDaliases {
			if alias == "" {
				return fmt.Errorf("id_aliases cannot have empty alias names")
			}
			if id < 0 {
				return fmt.Errorf("id_aliases '%s': ID must be non-negative, got %d", alias, id)
			}
			if id > 255 {
				return fmt.Errorf("id_aliases '%s': ID must be <= 255 for MySensors compatibility, got %d", alias, id)
			}
		}
	}

	// Validate entity sync restrictions for duplicate mappings
	type entityInfo struct {
		name           string
		hasSyncEnabled bool
	}
	entityTargets := make(map[string][]entityInfo) // key: "nodeID:childID", value: list of entity info

	// Validate entities
	validEntityTypes := map[string]bool{
		// Actuator types
		"switch":       true,
		"light":        true,
		"dimmer":       true,
		"cover":        true,
		"text":         true,
		"number":       true,
		"select":       true,
		"climate":      true,
		"rgb_light":    true,
		"rgbw_light":   true,
		
		// Sensor types
		"sensor":        true,
		"binary_sensor": true,
		"temperature":   true,
		"humidity":      true,
		"battery":       true,
		"voltage":       true,
		"current":       true,
		"pressure":      true,
		"level":         true,
		"percentage":    true,
		"weight":        true,
		"distance":      true,
		"light_level":   true,
		"watt":          true,
		"kwh":           true,
		"flow":          true,
		"volume":        true,
		"ph":            true,
		"orp":           true,
		"ec":            true,
		"var":           true,
		"va":            true,
		"power_factor":  true,
		"custom":        true,
		"position":      true,
		"uv":            true,
		"rain":          true,
		"rainrate":      true,
		"wind":          true,
		"gust":          true,
		"direction":     true,
		"impedance":     true,
	}

	for _, device := range config.Devices {
		// Check if device has node_id or if all entities have node_id
		deviceHasNodeID := device.NodeID != nil
		if deviceHasNodeID {
			// Validate device node_id can be resolved
			_, err := config.GetEffectiveNodeID(&device)
			if err != nil {
				return fmt.Errorf("invalid node_id for device '%s': %w", device.Name, err)
			}
		}

		// If device has no node_id, validate that all entities have node_id
		if !deviceHasNodeID {
			for _, entity := range device.Entities {
				if entity.NodeID == nil {
					return fmt.Errorf("device '%s' has no node_id, so entity '%s' must have node_id configured", device.Name, entity.Name)
				}
			}
		}

		// Validate entities and add them to the unique target check
		for _, entity := range device.Entities {
			// Validate entity type
			if entity.EntityType == "" {
				return fmt.Errorf("entity_type is required for entity '%s' in device '%s'", entity.Name, device.Name)
			}
			if !validEntityTypes[entity.EntityType] {
				return fmt.Errorf("invalid entity_type '%s' for entity '%s' in device '%s'", entity.EntityType, entity.Name, device.Name)
			}

			// Validate entity child_id can be resolved
			childID, err := config.GetEffectiveChildID(&entity)
			if err != nil {
				return fmt.Errorf("invalid child_id for entity '%s' in device '%s': %w", entity.Name, device.Name, err)
			}

			// Add to unique target validation
			effectiveNodeID, err := config.GetEffectiveEntityNodeID(&device, &entity)
			if err != nil {
				return fmt.Errorf("invalid effective node_id for entity '%s' in device '%s': %w", entity.Name, device.Name, err)
			}

			target := fmt.Sprintf("%d:%d", effectiveNodeID, childID)
			entityName := fmt.Sprintf("%s:%s", device.Name, entity.Name)
			hasSyncEnabled := entity.IsSyncEnabled()
			
			entityTargets[target] = append(entityTargets[target], entityInfo{
				name:           entityName,
				hasSyncEnabled: hasSyncEnabled,
			})
		}
	}

	// Check sync restrictions for duplicate targets
	for target, entities := range entityTargets {
		if len(entities) > 1 {
			// Count entities with sync enabled
			syncEnabledCount := 0
			var syncEnabledEntities []string
			var allEntityNames []string
			
			for _, entity := range entities {
				allEntityNames = append(allEntityNames, entity.name)
				if entity.hasSyncEnabled {
					syncEnabledCount++
					syncEnabledEntities = append(syncEnabledEntities, entity.name)
				}
			}
			
			// Allow duplicate mappings but restrict sync
			if syncEnabledCount > 1 {
				return fmt.Errorf("multiple entities with sync enabled detected for MySensors target %s: %v - at most one entity per target can have sync enabled", target, syncEnabledEntities)
			}
		}
	}

	return nil
}

// GetDefaultGatewayName returns the first gateway name in the configuration as the default
func (config *Config) GetDefaultGatewayName() string {
	// Use the first gateway in the map (Go maps have deterministic iteration order in Go 1.0+)
	for gatewayName := range config.MySensors {
		return gatewayName
	}
	
	// This should never happen if config validation passed
	return ""
}

// GetEffectiveGateway returns the gateway name to use for a device/entity
func (config *Config) GetEffectiveGateway(deviceGateway, componentGateway string) string {
	// Priority: component gateway > device gateway > default gateway
	if componentGateway != "" {
		return componentGateway
	}
	if deviceGateway != "" {
		return deviceGateway
	}
	return config.GetDefaultGatewayName()
}

// GetEffectiveEntityRequestAck returns the effective request_ack setting for an entity
func (config *Config) GetEffectiveEntityRequestAck(entity *Entity) bool {
	// Entity-level only setting, defaults to true
	if entity.RequestAck != nil {
		return *entity.RequestAck
	}
	return true // Default to true (require ACK)
}

// GetEffectiveAckTimeout returns the effective ACK timeout for an entity
func (config *Config) GetEffectiveAckTimeout(entity *Entity) time.Duration {
	if entity.AckTimeout != nil {
		return *entity.AckTimeout
	}
	return 250 * time.Millisecond // Default: 250ms
}

// GetEffectiveAckRetries returns the effective ACK retry count for an entity
func (config *Config) GetEffectiveAckRetries(entity *Entity) int {
	if entity.AckRetries != nil {
		return *entity.AckRetries
	}
	return 3 // Default: 3 retries (total ~1s with 250ms timeout)
}

// ResolveID resolves a node_id or child_id which can be either an int or string alias
func (config *Config) ResolveID(idValue any) (int, error) {
	switch v := idValue.(type) {
	case int:
		return v, nil
	case string:
		if alias, exists := config.IDaliases[v]; exists {
			return alias, nil
		}
		return 0, fmt.Errorf("unknown ID alias: %s", v)
	case float64:
		// YAML sometimes unmarshals integers as float64
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid ID type: %T, expected int or string", idValue)
	}
}

// GetEffectiveNodeID returns the resolved node ID for a device
func (config *Config) GetEffectiveNodeID(device *Device) (int, error) {
	if device.NodeID == nil {
		return 0, fmt.Errorf("device '%s' has no node_id configured", device.Name)
	}
	return config.ResolveID(device.NodeID)
}

// GetEffectiveEntityNodeID returns the resolved node ID for an entity (with device fallback)
func (config *Config) GetEffectiveEntityNodeID(device *Device, entity *Entity) (int, error) {
	if entity.NodeID != nil {
		return config.ResolveID(entity.NodeID)
	}
	if device.NodeID == nil {
		return 0, fmt.Errorf("entity '%s' in device '%s' has no node_id and device has no fallback node_id", entity.Name, device.Name)
	}
	return config.GetEffectiveNodeID(device)
}

// GetEffectiveChildID returns the resolved child ID for an entity
func (config *Config) GetEffectiveChildID(entity *Entity) (int, error) {
	return config.ResolveID(entity.ChildID)
}

// GetMinimumAvailabilityWindowForNode returns the minimum availability_window from all entities on a given node
// Returns the gateway default if no entities have availability_window set, or 0 if all entities have 0s
func (config *Config) GetMinimumAvailabilityWindowForNode(nodeID int, gatewayName string) time.Duration {
	var minWindow *time.Duration

	// Iterate through all devices and entities to find those matching this node
	for _, device := range config.Devices {
		for _, entity := range device.Entities {
			// Get the effective node ID for this entity
			effectiveNodeID, err := config.GetEffectiveEntityNodeID(&device, &entity)
			if err != nil || effectiveNodeID != nodeID {
				continue
			}

			// Check if this entity has a non-zero availability_window set
			if entity.AvailabilityWindow != nil && *entity.AvailabilityWindow > 0 {
				if minWindow == nil || *entity.AvailabilityWindow < *minWindow {
					minWindow = entity.AvailabilityWindow
				}
			}
		}
	}

	// If we found at least one entity with availability_window, return the minimum
	if minWindow != nil {
		return *minWindow
	}

	// Otherwise, return the gateway's default availability_window
	if gw, exists := config.MySensors[gatewayName]; exists {
		return gw.Gateway.AvailabilityWindow
	}

	// Fallback to 30 seconds if gateway not found
	return 30 * time.Second
}

// GetMySensorsVariableType returns the MySensors variable type for a sensor type
func GetMySensorsVariableType(sensorType string) (mysensors.VariableType, bool) {
	mapping := map[string]mysensors.VariableType{
		"temperature":  mysensors.V_TEMP,
		"humidity":     mysensors.V_HUM,
		"battery":      mysensors.V_PERCENTAGE,
		"voltage":      mysensors.V_VOLTAGE,
		"current":      mysensors.V_CURRENT,
		"pressure":     mysensors.V_PRESSURE,
		"level":        mysensors.V_LEVEL,
		"percentage":   mysensors.V_PERCENTAGE,
		"weight":       mysensors.V_WEIGHT,
		"distance":     mysensors.V_DISTANCE,
		"light_level":  mysensors.V_LIGHT_LEVEL,
		"watt":         mysensors.V_WATT,
		"kwh":          mysensors.V_KWH,
		"flow":         mysensors.V_FLOW,
		"volume":       mysensors.V_VOLUME,
		"ph":           mysensors.V_PH,
		"orp":          mysensors.V_ORP,
		"ec":           mysensors.V_EC,
		"var":          mysensors.V_VAR,
		"va":           mysensors.V_VA,
		"power_factor": mysensors.V_POWER_FACTOR,
		"text":         mysensors.V_TEXT,
		"custom":       mysensors.V_CUSTOM,
		"position":     mysensors.V_POSITION,
		"uv":           mysensors.V_UV,
		"rain":         mysensors.V_RAIN,
		"rainrate":     mysensors.V_RAINRATE,
		"wind":         mysensors.V_WIND,
		"gust":         mysensors.V_GUST,
		"direction":    mysensors.V_DIRECTION,
		"impedance":    mysensors.V_IMPEDANCE,
	}

	varType, exists := mapping[sensorType]
	return varType, exists
}

// IsBinarySensor returns true if the sensor type represents a binary sensor
func IsBinarySensor(sensorType string) bool {
	return sensorType == "binary" || sensorType == ""
}

// GetMySensorsVariableTypeForOutput returns the MySensors variable type for an output type
func GetMySensorsVariableTypeForOutput(outputType, variableTypeOverride string) (mysensors.VariableType, bool) {
	// If variable type is explicitly specified, use it
	if variableTypeOverride != "" {
		mapping := map[string]mysensors.VariableType{
			"V_STATUS":             mysensors.V_STATUS,
			"V_PERCENTAGE":         mysensors.V_PERCENTAGE,
			"V_TEXT":               mysensors.V_TEXT,
			"V_TEMP":               mysensors.V_TEMP,
			"V_HUM":                mysensors.V_HUM,
			"V_PRESSURE":           mysensors.V_PRESSURE,
			"V_VOLTAGE":            mysensors.V_VOLTAGE,
			"V_CURRENT":            mysensors.V_CURRENT,
			"V_LEVEL":              mysensors.V_LEVEL,
			"V_WATT":               mysensors.V_WATT,
			"V_KWH":                mysensors.V_KWH,
			"V_DISTANCE":           mysensors.V_DISTANCE,
			"V_WEIGHT":             mysensors.V_WEIGHT,
			"V_LIGHT_LEVEL":        mysensors.V_LIGHT_LEVEL,
			"V_FLOW":               mysensors.V_FLOW,
			"V_VOLUME":             mysensors.V_VOLUME,
			"V_UP":                 mysensors.V_UP,
			"V_DOWN":               mysensors.V_DOWN,
			"V_STOP":               mysensors.V_STOP,
			"V_RGB":                mysensors.V_RGB,
			"V_RGBW":               mysensors.V_RGBW,
			"V_HVAC_SETPOINT_HEAT": mysensors.V_HVAC_SETPOINT_HEAT,
			"V_HVAC_SETPOINT_COOL": mysensors.V_HVAC_SETPOINT_COOL,
			"V_HVAC_FLOW_MODE":     mysensors.V_HVAC_FLOW_MODE,
			"V_CUSTOM":             mysensors.V_CUSTOM,
			"V_POSITION":           mysensors.V_POSITION,
			"V_IR_SEND":            mysensors.V_IR_SEND,
			"V_PH":                 mysensors.V_PH,
			"V_ORP":                mysensors.V_ORP,
			"V_EC":                 mysensors.V_EC,
			"V_VAR":                mysensors.V_VAR,
			"V_VA":                 mysensors.V_VA,
			"V_POWER_FACTOR":       mysensors.V_POWER_FACTOR,
		}
		
		if varType, exists := mapping[variableTypeOverride]; exists {
			return varType, true
		}
	}
	
	// Default mappings based on output type
	defaultMapping := map[string]mysensors.VariableType{
		"switch":      mysensors.V_STATUS,
		"light":       mysensors.V_STATUS,
		"dimmer":      mysensors.V_PERCENTAGE,
		"cover":       mysensors.V_UP, // Cover uses V_UP/V_DOWN/V_STOP
		"text":        mysensors.V_TEXT,
		"number":      mysensors.V_PERCENTAGE,
		"select":      mysensors.V_TEXT,
		"climate":     mysensors.V_HVAC_SETPOINT_HEAT,
		"rgb_light":   mysensors.V_RGB,
		"rgbw_light":  mysensors.V_RGBW,
	}
	
	if varType, exists := defaultMapping[outputType]; exists {
		return varType, true
	}
	
	// Default to V_STATUS for unknown types
	return mysensors.V_STATUS, false
}

// Entity helper functions

// IsReadOnly returns true if the entity is read-only (sensor)
func (e *Entity) IsReadOnly() bool {
	if e.ReadOnly != nil {
		return *e.ReadOnly
	}
	// Default based on entity type
	switch e.EntityType {
	case "sensor", "binary_sensor":
		return true
	default:
		return false
	}
}

// IsWriteOnly returns true if the entity is write-only (actuator)
func (e *Entity) IsWriteOnly() bool {
	if e.WriteOnly != nil {
		return *e.WriteOnly
	}
	return false // Default to false (can report state)
}

// CanReceiveCommands returns true if the entity can receive MQTT commands
func (e *Entity) CanReceiveCommands() bool {
	return !e.IsReadOnly()
}

// CanReportState returns true if the entity can report state via MQTT
func (e *Entity) CanReportState() bool {
	return !e.IsWriteOnly()
}

// GetEffectiveUniqueID returns the unique ID for the entity, using UniqueID if set, otherwise device_id + entity_id
func (e *Entity) GetEffectiveUniqueID(deviceID string) string {
	if e.UniqueID != "" {
		return e.UniqueID
	}
	return fmt.Sprintf("%s_%s", deviceID, e.ID)
}

// GetEffectiveDefaultEntityID returns the default_entity_id for the entity
// Returns empty string if DefaultEntityID is explicitly set to empty string (to exclude from discovery)
// Format: {domain}.{base_id} where base_id is DefaultEntityID if set, otherwise device_id + entity_id
func (e *Entity) GetEffectiveDefaultEntityID(deviceID string) (string, bool) {
	// Determine the base ID (the part after the domain)
	var baseID string
	if e.DefaultEntityID == nil {
		// Default: use device_id + entity_id format
		baseID = fmt.Sprintf("%s_%s", deviceID, e.ID)
	} else if *e.DefaultEntityID == "" {
		// Explicitly set to empty string - exclude from discovery
		return "", false
	} else {
		// Custom value provided - use it as-is
		baseID = *e.DefaultEntityID
	}

	// Construct full default_entity_id in format: {domain}.{base_id}
	domain := e.GetHAEntityType()
	return fmt.Sprintf("%s.%s", domain, baseID), true
}

// GetEffectiveSyncPeriod returns the sync period for the entity, 0 means sync is disabled
func (e *Entity) GetEffectiveSyncPeriod() time.Duration {
	if e.SyncPeriod != nil {
		return *e.SyncPeriod
	}
	return 0 // Default: sync disabled
}

// IsSyncEnabled returns true if sync is enabled for this entity
func (e *Entity) IsSyncEnabled() bool {
	return e.GetEffectiveSyncPeriod() > 0
}

// GetHAEntityType returns the Home Assistant entity type for this entity
func (e *Entity) GetHAEntityType() string {
	switch e.EntityType {
	case "switch", "light", "dimmer", "cover":
		return e.EntityType
	case "rgb_light":
		return "light"
	case "rgbw_light":
		return "light"
	case "binary_sensor", "sensor":
		return e.EntityType
	case "temperature", "humidity", "battery", "voltage", "current", "pressure",
		"weight", "distance", "light_level", "watt", "kwh", "flow", "volume",
		"ph", "orp", "ec", "var", "va", "power_factor", "uv", "rain", "rainrate",
		"wind", "gust", "direction", "impedance":
		return "sensor"
	case "text", "number", "select", "climate":
		return e.EntityType
	case "custom":
		// Custom types default to sensor
		return "sensor"
	case "position":
		return "sensor"
	default:
		return "sensor"
	}
}

// GetEffectiveSyncSplay returns the sync splay for the entity
// Returns 0 if splay is not set or if splay >= sync_period
func (e *Entity) GetEffectiveSyncSplay() time.Duration {
	if e.SyncSplay == nil || *e.SyncSplay <= 0 {
		return 0
	}
	splay := *e.SyncSplay
	period := e.GetEffectiveSyncPeriod()
	// Ignore splay if it's >= sync_period (would cause issues)
	if splay >= period {
		return 0
	}
	return splay
}

// GetMySensorsVariableTypeForEntity returns the MySensors variable type for an entity
func GetMySensorsVariableTypeForEntity(entityType, variableTypeOverride string) (mysensors.VariableType, bool) {
	// If variable type is explicitly specified, use it
	if variableTypeOverride != "" {
		mapping := map[string]mysensors.VariableType{
			"V_STATUS":             mysensors.V_STATUS,
			"V_PERCENTAGE":         mysensors.V_PERCENTAGE,
			"V_TEXT":               mysensors.V_TEXT,
			"V_TEMP":               mysensors.V_TEMP,
			"V_HUM":                mysensors.V_HUM,
			"V_PRESSURE":           mysensors.V_PRESSURE,
			"V_VOLTAGE":            mysensors.V_VOLTAGE,
			"V_CURRENT":            mysensors.V_CURRENT,
			"V_LEVEL":              mysensors.V_LEVEL,
			"V_WATT":               mysensors.V_WATT,
			"V_KWH":                mysensors.V_KWH,
			"V_DISTANCE":           mysensors.V_DISTANCE,
			"V_WEIGHT":             mysensors.V_WEIGHT,
			"V_LIGHT_LEVEL":        mysensors.V_LIGHT_LEVEL,
			"V_FLOW":               mysensors.V_FLOW,
			"V_VOLUME":             mysensors.V_VOLUME,
			"V_UP":                 mysensors.V_UP,
			"V_DOWN":               mysensors.V_DOWN,
			"V_STOP":               mysensors.V_STOP,
			"V_RGB":                mysensors.V_RGB,
			"V_RGBW":               mysensors.V_RGBW,
			"V_HVAC_SETPOINT_HEAT": mysensors.V_HVAC_SETPOINT_HEAT,
			"V_HVAC_SETPOINT_COOL": mysensors.V_HVAC_SETPOINT_COOL,
			"V_HVAC_FLOW_MODE":     mysensors.V_HVAC_FLOW_MODE,
			"V_CUSTOM":             mysensors.V_CUSTOM,
			"V_POSITION":           mysensors.V_POSITION,
			"V_IR_SEND":            mysensors.V_IR_SEND,
			"V_PH":                 mysensors.V_PH,
			"V_ORP":                mysensors.V_ORP,
			"V_EC":                 mysensors.V_EC,
			"V_VAR":                mysensors.V_VAR,
			"V_VA":                 mysensors.V_VA,
			"V_POWER_FACTOR":       mysensors.V_POWER_FACTOR,
		}
		
		if varType, exists := mapping[variableTypeOverride]; exists {
			return varType, true
		}
	}
	
	// Default mappings based on entity type
	defaultMapping := map[string]mysensors.VariableType{
		// Actuator types
		"switch":       mysensors.V_STATUS,
		"light":        mysensors.V_STATUS,
		"dimmer":       mysensors.V_PERCENTAGE,
		"cover":        mysensors.V_UP, // Cover uses V_UP/V_DOWN/V_STOP
		"text":         mysensors.V_TEXT,
		"number":       mysensors.V_PERCENTAGE,
		"select":       mysensors.V_TEXT,
		"climate":      mysensors.V_HVAC_SETPOINT_HEAT,
		"rgb_light":    mysensors.V_RGB,
		"rgbw_light":   mysensors.V_RGBW,
		
		// Sensor types (from existing GetMySensorsVariableType function)
		"binary_sensor": mysensors.V_STATUS,
		"sensor":        mysensors.V_CUSTOM, // Default sensor type
		"temperature":   mysensors.V_TEMP,
		"humidity":      mysensors.V_HUM,
		"battery":       mysensors.V_PERCENTAGE,
		"voltage":       mysensors.V_VOLTAGE,
		"current":       mysensors.V_CURRENT,
		"pressure":      mysensors.V_PRESSURE,
		"level":         mysensors.V_LEVEL,
		"percentage":    mysensors.V_PERCENTAGE,
		"weight":        mysensors.V_WEIGHT,
		"distance":      mysensors.V_DISTANCE,
		"light_level":   mysensors.V_LIGHT_LEVEL,
		"watt":          mysensors.V_WATT,
		"kwh":           mysensors.V_KWH,
		"flow":          mysensors.V_FLOW,
		"volume":        mysensors.V_VOLUME,
		"ph":            mysensors.V_PH,
		"orp":           mysensors.V_ORP,
		"ec":            mysensors.V_EC,
		"var":           mysensors.V_VAR,
		"va":            mysensors.V_VA,
		"power_factor":  mysensors.V_POWER_FACTOR,
		"custom":        mysensors.V_CUSTOM,
		"position":      mysensors.V_POSITION,
		"uv":            mysensors.V_UV,
		"rain":          mysensors.V_RAIN,
		"rainrate":      mysensors.V_RAINRATE,
		"wind":          mysensors.V_WIND,
		"gust":          mysensors.V_GUST,
		"direction":     mysensors.V_DIRECTION,
		"impedance":     mysensors.V_IMPEDANCE,
	}
	
	if varType, exists := defaultMapping[entityType]; exists {
		return varType, true
	}
	
	// Default to V_STATUS for unknown types
	return mysensors.V_STATUS, false
}

func SetDefaults(config *Config) {
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	// Ensure MySensors map is initialized
	if config.MySensors == nil {
		config.MySensors = make(map[string]MySensorsConfig)
	}

	// Set default transport type to "ethernet" if not specified
	for gatewayName, gatewayConfig := range config.MySensors {
		if gatewayConfig.Transport == "" {
			gatewayConfig.Transport = "ethernet"
			config.MySensors[gatewayName] = gatewayConfig
		}
	}

	// Note: We no longer automatically rename gateways to "default"
	// The GetDefaultGatewayName() function handles finding the appropriate default

	if config.MQTT.Port == 0 {
		config.MQTT.Port = 1883
	}

	if config.MQTT.ClientID == "" {
		config.MQTT.ClientID = "ms-mqtt-adapter"
	}

	// Sync configuration is now per-entity, no global default needed

	// Set defaults for all MySensors gateways
	for gatewayName, gatewayConfig := range config.MySensors {
		if gatewayConfig.Gateway.NodeIDAssignment.NodeIDRange.Start == 0 {
			gatewayConfig.Gateway.NodeIDAssignment.NodeIDRange.Start = 1
		}

		if gatewayConfig.Gateway.NodeIDAssignment.NodeIDRange.End == 0 {
			gatewayConfig.Gateway.NodeIDAssignment.NodeIDRange.End = 254
		}

		if gatewayConfig.Gateway.AvailabilityWindow == 0 {
			gatewayConfig.Gateway.AvailabilityWindow = 120 * time.Second // 2 minutes - prevents flapping for slow/periodic devices
			// Log the default value being set
			fmt.Printf("Setting default availability window for gateway %s to %v\n", gatewayName, gatewayConfig.Gateway.AvailabilityWindow)
		}

		// Set default ethernet port if not specified
		if gatewayConfig.Transport == "ethernet" && gatewayConfig.Ethernet.Port == 0 {
			gatewayConfig.Ethernet.Port = 5003
		}

		// Default to enabled ID assignment if not specified
		if gatewayConfig.Gateway.NodeIDAssignment.Enabled == nil {
			enabled := true
			gatewayConfig.Gateway.NodeIDAssignment.Enabled = &enabled
		}

		// Default to sequential ID assignment (false) if not specified
		if gatewayConfig.Gateway.NodeIDAssignment.RandomIDAssignment == nil {
			randomAssignment := false
			gatewayConfig.Gateway.NodeIDAssignment.RandomIDAssignment = &randomAssignment
		}

		// TCP service is disabled by default and requires explicit port configuration

		// Set half-duplex defaults when enabled
		if gatewayConfig.HalfDuplex.Enabled {
			if gatewayConfig.HalfDuplex.BusQuietTime == 0 {
				gatewayConfig.HalfDuplex.BusQuietTime = 100 * time.Millisecond
			}
			if gatewayConfig.HalfDuplex.InterMessageDelay == 0 {
				gatewayConfig.HalfDuplex.InterMessageDelay = 50 * time.Millisecond
			}
			if gatewayConfig.HalfDuplex.MaxTXWait == 0 {
				gatewayConfig.HalfDuplex.MaxTXWait = 2 * time.Second
			}
		}

		config.MySensors[gatewayName] = gatewayConfig
	}

	if config.AdapterTopics.TopicPrefix == "" {
		config.AdapterTopics.TopicPrefix = "ms-mqtt-adapter"
	}

	if config.AdapterTopics.DiscoveryPrefix == "" {
		config.AdapterTopics.DiscoveryPrefix = "homeassistant"
	}

	// Default to enabling HomeAssistant discovery if not explicitly set
	if config.AdapterTopics.HomeAssistantDiscovery == nil {
		enabled := true
		config.AdapterTopics.HomeAssistantDiscovery = &enabled
	}

	for i := range config.Devices {

		// Set defaults for entities
		for j := range config.Devices[i].Entities {
			entity := &config.Devices[i].Entities[j]
			
			// Set default initial values based on entity type
			if entity.InitialValue == "" {
				switch entity.EntityType {
				case "switch", "light", "binary_sensor":
					entity.InitialValue = "0"
				case "dimmer", "number", "percentage", "level":
					entity.InitialValue = "0"
				case "text", "select", "sensor":
					entity.InitialValue = ""
				default:
					entity.InitialValue = "0"
				}
			}
			
			// Set default units and state class based on entity type
			d := &entity.Discovery
			switch entity.EntityType {
			case "temperature":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "°C"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "humidity", "battery", "percentage", "level":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "%"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "voltage":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "V"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "current":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "A"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "pressure":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "hPa"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "weight":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "kg"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "distance":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "m"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "light_level":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "lx"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "watt":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "W"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "kwh":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "kWh"
				}
				if d.StateClass == "" {
					d.StateClass = "total_increasing"
				}
			case "flow":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "m³/h"
				}
				if d.StateClass == "" {
					d.StateClass = "measurement"
				}
			case "volume":
				if d.UnitOfMeasurement == "" {
					d.UnitOfMeasurement = "m³"
				}
				if d.StateClass == "" {
					d.StateClass = "total_increasing"
				}
			}

			// Default optimistic mode to false (wait for device confirmation)
			if d.Optimistic == nil {
				optimistic := false
				d.Optimistic = &optimistic
			}

			// Default request_ack to true (require device confirmation)
			if entity.RequestAck == nil {
				requestAck := true
				entity.RequestAck = &requestAck
			}
		}
	}
}
