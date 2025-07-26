package config

import (
	"fmt"
	"ms-mqtt-adapter/internal/mysensors"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel      string                     `yaml:"log_level"`
	MySensors     map[string]MySensorsConfig `yaml:"mysensors"`
	MQTT          MQTTConfig                 `yaml:"mqtt"`
	AdapterTopics AdapterConfig              `yaml:"adapter"`
	Devices       []Device                   `yaml:"devices"`
}

type MySensorsConfig struct {
	Transport string `yaml:"transport"`
	Ethernet  struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"ethernet"`
	RS485 struct {
		Device string `yaml:"device"`
	} `yaml:"rs485"`
	Gateway    GatewayConfig    `yaml:"gateway"`
	TCPService TCPServiceConfig `yaml:"tcp_service"`
}

type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	ClientID string `yaml:"client_id"`
}

type TCPServiceConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type SyncConfig struct {
	Enabled bool          `yaml:"enabled"`
	Period  time.Duration `yaml:"period"`
}

type GatewayConfig struct {
	NodeIDRange struct {
		Start int `yaml:"start"`
		End   int `yaml:"end"`
	} `yaml:"node_id_range"`
	VersionRequestPeriod time.Duration `yaml:"version_request_period"`
	RandomIDAssignment   *bool         `yaml:"random_id_assignment,omitempty"`
}

type AdapterConfig struct {
	TopicPrefix            string     `yaml:"topic_prefix"`
	HomeAssistantDiscovery *bool      `yaml:"homeassistant_discovery,omitempty"`
	Optimistic             *bool      `yaml:"optimistic,omitempty"`
	RequestAck             *bool      `yaml:"request_ack,omitempty"`
	Sync                   SyncConfig `yaml:"sync"`
}

type Device struct {
	Name             string     `yaml:"name"`
	ID               string     `yaml:"id"`
	NodeID           int        `yaml:"node_id"`
	Gateway          string     `yaml:"gateway,omitempty"`
	Manufacturer     string     `yaml:"manufacturer"`
	Model            string     `yaml:"model"`
	SWVersion        string     `yaml:"sw_version"`
	HWVersion        string     `yaml:"hw_version"`
	ConfigurationURL string     `yaml:"configuration_url,omitempty"`
	SuggestedArea    string     `yaml:"suggested_area,omitempty"`
	Connections      [][]string `yaml:"connections,omitempty"`
	ViaDevice        string     `yaml:"via_device,omitempty"`
	RequestAck       *bool      `yaml:"request_ack,omitempty"`
	Entities         []Entity   `yaml:"entities"`
}

// Entity represents a unified MySensors entity that can be an input (sensor), output (actuator), or both
type Entity struct {
	Name                   string `yaml:"name"`
	ID                     string `yaml:"id"`
	ChildID                int    `yaml:"child_id"`
	NodeID                 *int   `yaml:"node_id,omitempty"`
	Gateway                string `yaml:"gateway,omitempty"`
	
	// Entity type determines the Home Assistant entity type and capabilities
	EntityType             string `yaml:"entity_type"`                   // "switch", "light", "cover", "text", "number", "select", "sensor", "binary_sensor", etc.
	VariableType           string `yaml:"variable_type,omitempty"`       // MySensors variable type override (e.g., "V_STATUS", "V_TEXT", "V_PERCENTAGE")
	
	// Entity capabilities - determines if entity can read/write
	ReadOnly               *bool   `yaml:"read_only,omitempty"`           // false = can receive commands, true = sensor only (default: based on entity_type)
	WriteOnly              *bool   `yaml:"write_only,omitempty"`          // true = actuator only, false = can report state (default: false)
	
	// Initial and range values (for outputs/actuators)
	InitialValue           string  `yaml:"initial_value,omitempty"`       // Initial value (can be text, number, etc.)
	MinValue               *float64 `yaml:"min_value,omitempty"`          // For number entities
	MaxValue               *float64 `yaml:"max_value,omitempty"`          // For number entities
	Step                   *float64 `yaml:"step,omitempty"`               // For number entities
	Options                []string `yaml:"options,omitempty"`            // For select entities
	
	// Sensor configuration (for inputs/sensors)
	StateClass             string `yaml:"state_class,omitempty"`         // "measurement", "total", "total_increasing"
	
	// Home Assistant configuration
	Icon                   string `yaml:"icon"`
	DeviceClass            string `yaml:"device_class"`
	EntityCategory         string `yaml:"entity_category,omitempty"`
	EnabledByDefault       *bool  `yaml:"enabled_by_default,omitempty"`
	UnitOfMeasurement      string `yaml:"unit_of_measurement,omitempty"`
	
	// MQTT configuration (all optional)
	AvailabilityTopic      string `yaml:"availability_topic,omitempty"`
	PayloadAvailable       string `yaml:"payload_available,omitempty"`
	PayloadNotAvailable    string `yaml:"payload_not_available,omitempty"`
	PayloadOn              string `yaml:"payload_on,omitempty"`           // For switch/light entities
	PayloadOff             string `yaml:"payload_off,omitempty"`          // For switch/light entities
	StateOn                string `yaml:"state_on,omitempty"`             // For switch/light entities
	StateOff               string `yaml:"state_off,omitempty"`            // For switch/light entities
	PayloadOpen            string `yaml:"payload_open,omitempty"`         // For cover entities
	PayloadClose           string `yaml:"payload_close,omitempty"`        // For cover entities
	PayloadStop            string `yaml:"payload_stop,omitempty"`         // For cover entities
	StateOpen              string `yaml:"state_open,omitempty"`           // For cover entities
	StateClosed            string `yaml:"state_closed,omitempty"`         // For cover entities
	QOS                    *int   `yaml:"qos,omitempty"`
	Retain                 *bool  `yaml:"retain,omitempty"`
	Optimistic             *bool  `yaml:"optimistic,omitempty"`
	OffDelay               *int   `yaml:"off_delay,omitempty"`            // For binary sensors
	ExpireAfter            *int   `yaml:"expire_after,omitempty"`         // For sensors
	
	// MQTT template configuration (optional)
	JSONAttributesTopic    string `yaml:"json_attributes_topic,omitempty"`
	JSONAttributesTemplate string `yaml:"json_attributes_template,omitempty"`
	StateValueTemplate     string `yaml:"state_value_template,omitempty"`
	CommandTemplate        string `yaml:"command_template,omitempty"`
	ValueTemplate          string `yaml:"value_template,omitempty"`
}


func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	setDefaults(&config)
	return &config, nil
}

func validateConfig(config *Config) error {
	// Ensure we have at least one MySensors gateway
	if len(config.MySensors) == 0 {
		return fmt.Errorf("at least one mysensors gateway is required")
	}

	// Track TCP service ports to ensure no conflicts
	tcpPorts := make(map[int]string)

	// Validate each MySensors gateway configuration
	for gatewayName, mysensorsConfig := range config.MySensors {
		// Transport will be set to default "ethernet" in setDefaults if not specified
		if mysensorsConfig.Transport != "" && mysensorsConfig.Transport != "ethernet" && mysensorsConfig.Transport != "rs485" {
			return fmt.Errorf("mysensors gateway '%s' transport must be 'ethernet' or 'rs485'", gatewayName)
		}

		if mysensorsConfig.Transport == "ethernet" {
			if mysensorsConfig.Ethernet.Host == "" {
				return fmt.Errorf("mysensors gateway '%s' ethernet host is required", gatewayName)
			}
			if mysensorsConfig.Ethernet.Port == 0 {
				return fmt.Errorf("mysensors gateway '%s' ethernet port is required", gatewayName)
			}
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

	if config.MQTT.Broker == "" {
		return fmt.Errorf("mqtt broker is required")
	}

	// Validate that entity node_id:child_id combinations are unique
	entityTargets := make(map[string][]string) // key: "nodeID:childID", value: list of device:entity names

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
		// Validate entities and add them to the unique target check
		for _, entity := range device.Entities {
			// Validate entity type
			if entity.EntityType == "" {
				return fmt.Errorf("entity_type is required for entity '%s' in device '%s'", entity.Name, device.Name)
			}
			if !validEntityTypes[entity.EntityType] {
				return fmt.Errorf("invalid entity_type '%s' for entity '%s' in device '%s'", entity.EntityType, entity.Name, device.Name)
			}

			// Add to unique target validation
			effectiveNodeID := device.NodeID
			if entity.NodeID != nil {
				effectiveNodeID = *entity.NodeID
			}

			target := fmt.Sprintf("%d:%d", effectiveNodeID, entity.ChildID)
			entityName := fmt.Sprintf("%s:%s", device.Name, entity.Name)
			entityTargets[target] = append(entityTargets[target], entityName)
		}
	}

	// Check for duplicate targets
	for target, names := range entityTargets {
		if len(names) > 1 {
			return fmt.Errorf("duplicate mapping detected for MySensors target %s: %v - all entities must have unique node_id:child_id combinations", target, names)
		}
	}

	return nil
}

// GetEffectiveGateway returns the gateway name to use for a device/relay/input
func (config *Config) GetEffectiveGateway(deviceGateway, componentGateway string) string {
	// Priority: component gateway > device gateway > "default"
	if componentGateway != "" {
		return componentGateway
	}
	if deviceGateway != "" {
		return deviceGateway
	}
	return "default"
}

// GetEffectiveRequestAck returns the effective request_ack setting for a device
func (config *Config) GetEffectiveRequestAck(device *Device) bool {
	// Priority: device setting > global setting > default (true)
	if device.RequestAck != nil {
		return *device.RequestAck
	}
	if config.AdapterTopics.RequestAck != nil {
		return *config.AdapterTopics.RequestAck
	}
	return true // Default to true
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

func setDefaults(config *Config) {
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

	// If there's no "default" gateway but only one gateway, rename it to "default"
	if _, hasDefault := config.MySensors["default"]; !hasDefault && len(config.MySensors) == 1 {
		for name, gatewayConfig := range config.MySensors {
			delete(config.MySensors, name)
			config.MySensors["default"] = gatewayConfig
			break
		}
	}

	if config.MQTT.Port == 0 {
		config.MQTT.Port = 1883
	}

	if config.MQTT.ClientID == "" {
		config.MQTT.ClientID = "ms-mqtt-adapter"
	}

	if config.AdapterTopics.Sync.Period == 0 {
		config.AdapterTopics.Sync.Period = 30 * time.Second
	}

	// Set defaults for all MySensors gateways
	for gatewayName, gatewayConfig := range config.MySensors {
		if gatewayConfig.Gateway.NodeIDRange.Start == 0 {
			gatewayConfig.Gateway.NodeIDRange.Start = 1
		}

		if gatewayConfig.Gateway.NodeIDRange.End == 0 {
			gatewayConfig.Gateway.NodeIDRange.End = 254
		}

		if gatewayConfig.Gateway.VersionRequestPeriod == 0 {
			gatewayConfig.Gateway.VersionRequestPeriod = 5 * time.Second
		}

		// Set default ethernet port if not specified
		if gatewayConfig.Transport == "ethernet" && gatewayConfig.Ethernet.Port == 0 {
			gatewayConfig.Ethernet.Port = 5003
		}

		// Default to sequential ID assignment (false) if not specified
		if gatewayConfig.Gateway.RandomIDAssignment == nil {
			randomAssignment := false
			gatewayConfig.Gateway.RandomIDAssignment = &randomAssignment
		}

		// TCP service is disabled by default and requires explicit port configuration

		config.MySensors[gatewayName] = gatewayConfig
	}

	if config.AdapterTopics.TopicPrefix == "" {
		config.AdapterTopics.TopicPrefix = "ms-mqtt-adapter"
	}

	// Default to enabling HomeAssistant discovery if not explicitly set
	if config.AdapterTopics.HomeAssistantDiscovery == nil {
		enabled := true
		config.AdapterTopics.HomeAssistantDiscovery = &enabled
	}

	// Default to non-optimistic mode (wait for device confirmation) if not explicitly set
	if config.AdapterTopics.Optimistic == nil {
		optimistic := false
		config.AdapterTopics.Optimistic = &optimistic
	}

	// Default to request ACK (helps with device echoing) if not explicitly set
	if config.AdapterTopics.RequestAck == nil {
		requestAck := true
		config.AdapterTopics.RequestAck = &requestAck
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
			switch entity.EntityType {
			case "temperature":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "°C"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "humidity", "battery", "percentage", "level":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "%"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "voltage":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "V"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "current":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "A"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "pressure":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "hPa"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "weight":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "kg"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "distance":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "m"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "light_level":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "lx"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "watt":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "W"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "kwh":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "kWh"
				}
				if entity.StateClass == "" {
					entity.StateClass = "total_increasing"
				}
			case "flow":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "m³/h"
				}
				if entity.StateClass == "" {
					entity.StateClass = "measurement"
				}
			case "volume":
				if entity.UnitOfMeasurement == "" {
					entity.UnitOfMeasurement = "m³"
				}
				if entity.StateClass == "" {
					entity.StateClass = "total_increasing"
				}
			case "dimmer", "number":
				if entity.UnitOfMeasurement == "" && entity.EntityType == "number" {
					entity.UnitOfMeasurement = ""
				}
			case "text", "select", "custom", "sensor", "binary_sensor":
				// Text, select, and sensor entities don't have default units or state class
				if entity.StateClass == "" && (entity.EntityType == "text" || entity.EntityType == "select" || entity.EntityType == "binary_sensor") {
					entity.StateClass = ""
				}
			}
		}
	}
}
