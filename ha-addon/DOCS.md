# Home Assistant Add-on: MySensors MQTT Adapter

Bridge between MySensors network and MQTT with Home Assistant auto-discovery support.

This add-on provides a seamless bridge between your MySensors network and MQTT, with automatic Home Assistant device discovery. It acts as a MySensors gateway and translates MySensors messages to MQTT topics that Home Assistant can understand.

## Installation

1. Add this repository to your Home Assistant add-on store
2. Install the "MySensors MQTT Adapter" add-on
3. Configure the add-on (see configuration section below)
4. Start the add-on

## Configuration

### MySensors Configuration

Configure your MySensors gateway connection:

- **mysensors_host**: IP address of your MySensors Ethernet gateway (e.g., "192.168.1.100")
- **mysensors_port**: Port of your MySensors gateway (default: 5003)

### MQTT Configuration

Configure MQTT broker connection:

- **mqtt_broker**: MQTT broker hostname (use "core-mosquitto" for HA's built-in broker)
- **mqtt_port**: MQTT broker port (default: 1883)
- **mqtt_username**: MQTT username (leave empty if no authentication)
- **mqtt_password**: MQTT password (leave empty if no authentication)

### Add-on Options

- **log_level**: Set logging level (debug, info, warn, error)
- **topic_prefix**: MQTT topic prefix for device topics (default: "ms-mqtt-adapter")
- **homeassistant_discovery**: Enable automatic device discovery (default: true)

## Device Configuration

Since this is a Home Assistant add-on, device configuration is currently handled through the config file. Future versions may include a web UI for device management.

For now, you need to configure your devices in the application's configuration file within the container.

## MQTT Topics

The add-on creates the following MQTT topic structure:

### Device Topics
- `ms-mqtt-adapter/devices/{device_id}/relay/{relay_id}/state` - Relay state
- `ms-mqtt-adapter/devices/{device_id}/relay/{relay_id}/set` - Relay control
- `ms-mqtt-adapter/devices/{device_id}/input/{input_id}/state` - Input state

### Home Assistant Discovery
- `homeassistant/switch/{device_id}_{relay_id}/config` - Switch discovery
- `homeassistant/binary_sensor/{device_id}_{input_id}/config` - Input discovery

### Status Topics
- `ms-mqtt-adapter/seen_nodes` - List of discovered MySensors node IDs

## Features

- **Automatic Discovery**: Devices are automatically discovered in Home Assistant
- **State Persistence**: Device states survive restarts using MQTT retained messages
- **Real-time Control**: Immediate command processing via MQTT
- **Multi-platform**: Supports x86-64, ARM64, and ARM32 architectures
- **Gateway Functions**: Acts as MySensors gateway with node ID assignment
- **TCP Service**: Optional TCP service for debugging and additional integrations

## Troubleshooting

### Check Logs
Monitor the add-on logs for connection issues or configuration errors.

### Common Issues

1. **Cannot connect to MySensors gateway**
   - Verify the IP address and port in configuration
   - Ensure the MySensors gateway is accessible from Home Assistant

2. **MQTT connection failed**
   - Check MQTT broker settings
   - Verify credentials if authentication is enabled

3. **Devices not appearing in Home Assistant**
   - Ensure `homeassistant_discovery` is enabled
   - Check that devices are properly configured
   - Verify MQTT integration is working in Home Assistant

### Debug Mode

Set `log_level: debug` to get detailed logging information including:
- MySensors message traffic
- MQTT message traffic
- Device discovery process
- State management operations

## Support

For issues and feature requests, please visit the [GitHub repository](https://github.com/aszeszo/ms-mqtt-adapter).

## License

MIT License