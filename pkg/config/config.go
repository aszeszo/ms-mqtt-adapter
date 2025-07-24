package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel      string           `yaml:"log_level"`
	MySensors     MySensorsConfig  `yaml:"mysensors"`
	MQTT          MQTTConfig       `yaml:"mqtt"`
	TCPService    TCPServiceConfig `yaml:"tcp_service"`
	Sync          SyncConfig       `yaml:"sync"`
	Gateway       GatewayConfig    `yaml:"gateway"`
	AdapterTopics AdapterConfig    `yaml:"adapter"`
	Devices       []Device         `yaml:"devices"`
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
	if config.MySensors.Transport == "" {
		return fmt.Errorf("mysensors transport is required")
	}

	if config.MySensors.Transport == "ethernet" {
		if config.MySensors.Ethernet.Host == "" {
			return fmt.Errorf("mysensors ethernet host is required")
		}
		if config.MySensors.Ethernet.Port == 0 {
			return fmt.Errorf("mysensors ethernet port is required")
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

func setDefaults(config *Config) {
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	if config.MQTT.Port == 0 {
		config.MQTT.Port = 1883
	}

	if config.MQTT.ClientID == "" {
		config.MQTT.ClientID = "ms-mqtt-adapter"
	}

	if config.TCPService.Port == 0 {
		config.TCPService.Port = 5003
	}

	if config.Sync.Period == 0 {
		config.Sync.Period = 30 * time.Second
	}

	if config.Gateway.NodeIDRange.Start == 0 {
		config.Gateway.NodeIDRange.Start = 1
	}

	if config.Gateway.NodeIDRange.End == 0 {
		config.Gateway.NodeIDRange.End = 254
	}

	if config.Gateway.VersionRequestPeriod == 0 {
		config.Gateway.VersionRequestPeriod = 5 * time.Second
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
