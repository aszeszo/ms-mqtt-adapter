package mqtt

import (
	"fmt"
	"ms-mqtt-adapter/pkg/config"
)

type TopicInfo struct {
	Topic    string `json:"topic"`
	Type     string `json:"type"` // state, command, availability, discovery, gateway
	DeviceID string `json:"device_id,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
	Gateway  string `json:"gateway,omitempty"`
	Retained bool   `json:"retained"`
}

type TopicList struct {
	AdapterTopics   []TopicInfo `json:"adapter_topics"`
	DiscoveryTopics []TopicInfo `json:"discovery_topics"`
	GatewayTopics   []TopicInfo `json:"gateway_topics"`
}

// GetAllTopics returns all MQTT topics managed by this adapter
func (c *Client) GetAllTopics(gateways map[string]config.MySensorsConfig) TopicList {
	list := TopicList{
		AdapterTopics:   []TopicInfo{},
		DiscoveryTopics: []TopicInfo{},
		GatewayTopics:   []TopicInfo{},
	}

	for _, device := range c.devices {
		for _, entity := range device.Entities {
			uniqueID := entity.GetEffectiveUniqueID(device.ID)

			// State topic
			list.AdapterTopics = append(list.AdapterTopics, TopicInfo{
				Topic:    fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, uniqueID),
				Type:     "state",
				DeviceID: device.ID,
				EntityID: entity.ID,
				Retained: true,
			})

			// Command topic (if entity can receive commands)
			if entity.CanReceiveCommands() {
				list.AdapterTopics = append(list.AdapterTopics, TopicInfo{
					Topic:    fmt.Sprintf("%s/entity/%s/set", c.adapterCfg.TopicPrefix, uniqueID),
					Type:     "command",
					DeviceID: device.ID,
					EntityID: entity.ID,
					Retained: false,
				})
			}

			// Availability topic
			list.AdapterTopics = append(list.AdapterTopics, TopicInfo{
				Topic:    fmt.Sprintf("%s/entity/%s/availability", c.adapterCfg.TopicPrefix, uniqueID),
				Type:     "availability",
				DeviceID: device.ID,
				EntityID: entity.ID,
				Retained: true,
			})

			// Discovery topic
			entityType := entity.GetHAEntityType()
			list.DiscoveryTopics = append(list.DiscoveryTopics, TopicInfo{
				Topic:    fmt.Sprintf("homeassistant/%s/%s/config", entityType, uniqueID),
				Type:     "discovery",
				DeviceID: device.ID,
				EntityID: entity.ID,
				Retained: true,
			})
		}
	}

	// Gateway topics
	for name := range gateways {
		list.GatewayTopics = append(list.GatewayTopics, TopicInfo{
			Topic:    fmt.Sprintf("%s/gateway/%s/last_id", c.adapterCfg.TopicPrefix, name),
			Type:     "gateway",
			Gateway:  name,
			Retained: true,
		})
	}

	return list
}

// ClearTopics clears retained messages for topics matching the filter
func (c *Client) ClearTopics(scope, deviceID, entityID string, gateways map[string]config.MySensorsConfig) error {
	topics := c.getTopicsToDelete(scope, deviceID, entityID, gateways)

	c.logger.Info("Clearing MQTT topics", "scope", scope, "device_id", deviceID, "entity_id", entityID, "count", len(topics))

	for _, topic := range topics {
		// Publish empty retained message to clear
		if err := c.Publish(topic, "", true); err != nil {
			c.logger.Error("Failed to clear topic", "topic", topic, "error", err)
			return err
		}
		c.logger.Debug("Cleared topic", "topic", topic)
	}

	return nil
}

func (c *Client) getTopicsToDelete(scope, deviceID, entityID string, gateways map[string]config.MySensorsConfig) []string {
	var topics []string

	switch scope {
	case "all":
		list := c.GetAllTopics(gateways)
		for _, t := range list.AdapterTopics {
			if t.Retained {
				topics = append(topics, t.Topic)
			}
		}
		for _, t := range list.DiscoveryTopics {
			topics = append(topics, t.Topic)
		}
		for _, t := range list.GatewayTopics {
			topics = append(topics, t.Topic)
		}

	case "device":
		for _, device := range c.devices {
			if device.ID == deviceID {
				for _, entity := range device.Entities {
					uniqueID := entity.GetEffectiveUniqueID(device.ID)
					topics = append(topics,
						fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, uniqueID),
						fmt.Sprintf("%s/entity/%s/availability", c.adapterCfg.TopicPrefix, uniqueID),
						fmt.Sprintf("homeassistant/%s/%s/config", entity.GetHAEntityType(), uniqueID),
					)
				}
			}
		}

	case "entity":
		for _, device := range c.devices {
			if device.ID == deviceID {
				for _, entity := range device.Entities {
					if entity.ID == entityID {
						uniqueID := entity.GetEffectiveUniqueID(device.ID)
						topics = append(topics,
							fmt.Sprintf("%s/entity/%s/state", c.adapterCfg.TopicPrefix, uniqueID),
							fmt.Sprintf("%s/entity/%s/availability", c.adapterCfg.TopicPrefix, uniqueID),
							fmt.Sprintf("homeassistant/%s/%s/config", entity.GetHAEntityType(), uniqueID),
						)
					}
				}
			}
		}

	case "discovery":
		list := c.GetAllTopics(gateways)
		for _, t := range list.DiscoveryTopics {
			topics = append(topics, t.Topic)
		}

	case "adapter":
		list := c.GetAllTopics(gateways)
		for _, t := range list.AdapterTopics {
			if t.Retained {
				topics = append(topics, t.Topic)
			}
		}
	}

	return topics
}
