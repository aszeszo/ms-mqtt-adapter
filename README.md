# ms-mqtt-adapter

A bridge between MySensors network and MQTT with Home Assistant auto-discovery support.

*This project was "vibe coded" using [Claude Code](https://claude.ai/code) - an AI-powered development tool that helped design, implement, and debug the entire codebase through natural language conversations.*

## Features

- **MySensors to MQTT Bridge**: Seamlessly bridges MySensors network with MQTT broker
- **Home Assistant Auto-Discovery**: Automatically publishes device configurations for Home Assistant (optional)
- **Flexible Device Mapping**: Support for relays (1:1 mapping) and inputs (many-to-many mapping)
- **State Persistence**: Honors retained MQTT messages over configuration defaults
- **Immediate Command Processing**: Real-time MySensors command sending via MQTT /set topics
- **TCP Service**: Optional TCP service for MySensors message replication and forwarding
- **Gateway Functions**: Acts as MySensors gateway with node ID assignment and time synchronization
- **Periodic State Sync**: Ensures MQTT and MySensors states remain synchronized
- **Pluggable Transports**: Ethernet gateway support with RS485 stub for future expansion
- **Composite Keys**: Supports non-unique subdevice IDs across different devices

## Configuration

Create a `config.yaml` file based on `config.example.yaml`:

```yaml
log_level: "info"  # debug, info, warn, error

mysensors:
  transport: "ethernet"  # ethernet, rs485
  ethernet:
    host: "192.168.1.100"
    port: 5003

mqtt:
  broker: "localhost"
  port: 1883
  username: ""
  password: ""
  client_id: "ms-mqtt-adapter"

adapter:
  topic_prefix: "ms-mqtt-adapter"
  homeassistant_discovery: true  # Enable/disable HomeAssistant auto-discovery

gateway:
  version_request_period: "5s"  # How often to send version requests to gateway

devices:
  - name: "Relay Device"
    id: "relay_device_1"
    node_id: 1
    relays:
      - name: "Relay 1"
        id: "relay_1"
        child_id: 0
        initial_state: 0  # 0=OFF, 1=ON (only used if no retained MQTT state)
        icon: "hue:socket-eu"
        device_class: "switch"
    inputs:
      - name: "Button 1"
        id: "button_1"
        child_id: 1
        icon: "hue:friends-of-hue-senic"

# ... see config.example.yaml for full configuration
```

## Building and Running

```bash
# Build the application
go build -o ms-mqtt-adapter cmd/ms-mqtt-adapter/main.go

# Run with default config
./ms-mqtt-adapter

# Run with custom config
./ms-mqtt-adapter -config /path/to/config.yaml
```

## Project Structure

```
ms-mqtt-adapter/
├── cmd/ms-mqtt-adapter/     # Main application
├── pkg/
│   ├── config/              # Configuration management
│   ├── transport/           # MySensors transport layer
│   ├── mqtt/                # MQTT client and Home Assistant integration
│   ├── tcp/                 # TCP service for message replication
│   └── gateway/             # MySensors gateway functionality
├── internal/
│   ├── mysensors/           # MySensors message parsing
│   └── events/              # Event handling and synchronization
└── config.example.yaml      # Example configuration
```

## MySensors Protocol

The adapter handles MySensors message format:
```
node-id;child-sensor-id;message-type;ack;sub-type;payload
```

### Supported Message Types
- **SET**: Sensor data updates
- **REQ**: Data requests
- **INTERNAL**: Gateway functions (ID assignment, time sync)
- **PRESENTATION**: Device/sensor registration

### Gateway Functions
- **Node ID Assignment**: Responds to `I_ID_REQUEST` messages
- **Time Synchronization**: Responds to `I_TIME` requests
- **Version Requests**: Periodically sends version requests to gateway (configurable period)

## MQTT Topics

### Device Topics (New Structure)
- `{topic_prefix}/devices/{device_id}/relay/{relay_id}/state` - Relay state
- `{topic_prefix}/devices/{device_id}/relay/{relay_id}/set` - Relay control
- `{topic_prefix}/devices/{device_id}/input/{input_id}/state` - Input state

### Home Assistant Discovery (Optional)
- `homeassistant/switch/{device_id}_{relay_id}/config` - Switch discovery
- `homeassistant/binary_sensor/{device_id}_{input_id}/config` - Input discovery

### Adapter Status
- `{topic_prefix}/seen_nodes` - Comma-separated list of discovered node IDs (sorted)

## Key Features

### State Persistence
- **Retained MQTT messages** take precedence over config `initial_state`
- Only applies config `initial_state` when no retained state exists
- Ensures state survives restarts and reconnections

### Device Mapping
- **Relays**: 1:1 mapping (one MySensors node:child maps to one relay)
- **Inputs**: Many-to-many mapping (one MySensors node:child can map to multiple inputs)
- **Composite Keys**: `{device_id}_{subdevice_id}` allows non-unique subdevice IDs

### Immediate Command Processing
- Commands sent to `/set` topics are processed immediately
- MySensors messages sent in real-time (not just during sync cycles)
- State changes reflected immediately in MQTT

## Value Format
- All MQTT payloads use **0/1 format** (not ON/OFF)
- Compatible with HomeAssistant expectations
- Consistent across all device types

## Logging

Log levels: `debug`, `info`, `warn`, `error`

Debug mode logs:
- MySensors RX/TX messages
- MQTT RX/TX messages  
- State management operations
- Discovery process details

## TCP Service

When enabled, provides a TCP service on configurable port (default 5003) that:
- Replicates all MySensors messages to connected clients
- Accepts messages from clients and forwards to MySensors network
- Useful for debugging and additional integrations

## Development Note

This entire project was developed using Claude Code, demonstrating the power of AI-assisted software development. From initial architecture discussions to bug fixes and feature implementations, the codebase evolved through natural language conversations with the AI, showcasing how modern development tools can accelerate the creation of complex, production-ready software.

## License

MIT License