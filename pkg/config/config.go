package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel      string                     `yaml:"log_level"`
	MySensors     map[string]MySensorsConfig `yaml:"mysensors"`
	MQTT          MQTTConfig                 `yaml:"mqtt"`
	Sync          SyncConfig                 `yaml:"sync"`
	AdapterTopics AdapterConfig             `yaml:"adapter"`
	Devices       []Device                   `yaml:"devices"`
}

type MySensorsConfig struct {
	Transport  string           `yaml:"transport"`
	Ethernet   struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"ethernet"`
	RS485      struct {
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
	TopicPrefix            string `yaml:"topic_prefix"`
	HomeAssistantDiscovery *bool  `yaml:"homeassistant_discovery,omitempty"`
	OptimisticMode         *bool  `yaml:"optimistic_mode,omitempty"`
	RequestAck             *bool  `yaml:"request_ack,omitempty"`
}

type Device struct {
	Name             string  `yaml:"name"`
	ID               string  `yaml:"id"`
	NodeID           int     `yaml:"node_id"`
	Gateway          string  `yaml:"gateway,omitempty"`
	Manufacturer     string  `yaml:"manufacturer"`
	Model            string  `yaml:"model"`
	SWVersion        string  `yaml:"sw_version"`
	HWVersion        string  `yaml:"hw_version"`
	ConfigurationURL string  `yaml:"configuration_url,omitempty"`
	SuggestedArea    string  `yaml:"suggested_area,omitempty"`
	Connections      [][]string `yaml:"connections,omitempty"`
	ViaDevice        string  `yaml:"via_device,omitempty"`
	Relays           []Relay `yaml:"relays"`
	Inputs           []Input `yaml:"inputs"`
}

type Relay struct {
	Name                  string            `yaml:"name"`
	ID                    string            `yaml:"id"`
	ChildID               int               `yaml:"child_id"`
	NodeID                *int              `yaml:"node_id,omitempty"`
	Gateway               string            `yaml:"gateway,omitempty"`
	InitialState          int               `yaml:"initial_state"`
	Icon                  string            `yaml:"icon"`
	DeviceClass           string            `yaml:"device_class"`
	EntityCategory        string            `yaml:"entity_category,omitempty"`
	EnabledByDefault      *bool             `yaml:"enabled_by_default,omitempty"`
	AvailabilityTopic     string            `yaml:"availability_topic,omitempty"`
	PayloadAvailable      string            `yaml:"payload_available,omitempty"`
	PayloadNotAvailable   string            `yaml:"payload_not_available,omitempty"`
	PayloadOn             string            `yaml:"payload_on,omitempty"`
	PayloadOff            string            `yaml:"payload_off,omitempty"`
	StateOn               string            `yaml:"state_on,omitempty"`
	StateOff              string            `yaml:"state_off,omitempty"`
	QOS                   *int              `yaml:"qos,omitempty"`
	Retain                *bool             `yaml:"retain,omitempty"`
	Optimistic            *bool             `yaml:"optimistic,omitempty"`
	JSONAttributesTopic   string            `yaml:"json_attributes_topic,omitempty"`
	JSONAttributesTemplate string           `yaml:"json_attributes_template,omitempty"`
	StateValueTemplate    string            `yaml:"state_value_template,omitempty"`
	CommandTemplate       string            `yaml:"command_template,omitempty"`
}

type Input struct {
	Name                  string `yaml:"name"`
	ID                    string `yaml:"id"`
	ChildID               int    `yaml:"child_id"`
	NodeID                *int   `yaml:"node_id,omitempty"`
	Gateway               string `yaml:"gateway,omitempty"`
	Icon                  string `yaml:"icon"`
	DeviceClass           string `yaml:"device_class"`
	EntityCategory        string `yaml:"entity_category,omitempty"`
	EnabledByDefault      *bool  `yaml:"enabled_by_default,omitempty"`
	AvailabilityTopic     string `yaml:"availability_topic,omitempty"`
	PayloadAvailable      string `yaml:"payload_available,omitempty"`
	PayloadNotAvailable   string `yaml:"payload_not_available,omitempty"`
	PayloadOn             string `yaml:"payload_on,omitempty"`
	PayloadOff            string `yaml:"payload_off,omitempty"`
	StateOn               string `yaml:"state_on,omitempty"`
	StateOff              string `yaml:"state_off,omitempty"`
	QOS                   *int   `yaml:"qos,omitempty"`
	OffDelay              *int   `yaml:"off_delay,omitempty"`
	ExpireAfter           *int   `yaml:"expire_after,omitempty"`
	JSONAttributesTopic   string `yaml:"json_attributes_topic,omitempty"`
	JSONAttributesTemplate string `yaml:"json_attributes_template,omitempty"`
	ValueTemplate         string `yaml:"value_template,omitempty"`
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
				return fmt.Errorf("mysensors gateway '%s' TCP service port is required when enabled", gatewayName)
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

	// Validate that relay node_id:child_id combinations are unique (1:1 mapping only)
	relayTargets := make(map[string][]string) // key: "nodeID:childID", value: list of device:relay names

	for _, device := range config.Devices {
		for _, relay := range device.Relays {
			effectiveNodeID := device.NodeID
			if relay.NodeID != nil {
				effectiveNodeID = *relay.NodeID
			}

			target := fmt.Sprintf("%d:%d", effectiveNodeID, relay.ChildID)
			relayName := fmt.Sprintf("%s:%s", device.Name, relay.Name)
			relayTargets[target] = append(relayTargets[target], relayName)
		}
	}

	// Check for duplicate relay targets
	for target, relays := range relayTargets {
		if len(relays) > 1 {
			return fmt.Errorf("duplicate relay mapping detected for MySensors target %s: %v - relays must have unique node_id:child_id combinations", target, relays)
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

	if config.Sync.Period == 0 {
		config.Sync.Period = 30 * time.Second
	}

	// Set defaults for all MySensors gateways
	nextTCPPort := 5003
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
		
		// Default to sequential ID assignment (false) if not specified
		if gatewayConfig.Gateway.RandomIDAssignment == nil {
			randomAssignment := false
			gatewayConfig.Gateway.RandomIDAssignment = &randomAssignment
		}
		
		// Set TCP service defaults - auto-assign unique ports for multiple gateways
		if gatewayConfig.TCPService.Port == 0 && gatewayConfig.TCPService.Enabled {
			gatewayConfig.TCPService.Port = nextTCPPort
			nextTCPPort++
		}
		// Enable TCP service by default for first gateway ("default") only
		if gatewayName == "default" && len(config.MySensors) == 1 {
			gatewayConfig.TCPService.Enabled = true
			if gatewayConfig.TCPService.Port == 0 {
				gatewayConfig.TCPService.Port = 5003
			}
		}

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
	if config.AdapterTopics.OptimisticMode == nil {
		optimistic := false
		config.AdapterTopics.OptimisticMode = &optimistic
	}
	
	// Default to request ACK (helps with device echoing) if not explicitly set
	if config.AdapterTopics.RequestAck == nil {
		requestAck := true
		config.AdapterTopics.RequestAck = &requestAck
	}

	for i := range config.Devices {
		for j := range config.Devices[i].Relays {
			if config.Devices[i].Relays[j].InitialState == 0 {
				config.Devices[i].Relays[j].InitialState = 0
			}
		}
	}
}
