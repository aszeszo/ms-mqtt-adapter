# MySensors MQTT Adapter Documentation

## Installation

1. Add this repository to Home Assistant: **Settings → Add-ons → Add-on Store → ⋮ → Repositories**
2. Install the "MySensors MQTT Adapter" addon
3. Configure the addon options (see below)
4. Start the addon

## Basic Configuration

The addon uses a single `config_yaml` option where you provide the complete YAML configuration. Here's a minimal configuration that relies on defaults:

```yaml
mysensors:
  default:
    ethernet:
      host: "172.30.15.1"

mqtt:
  broker: "core-mosquitto"
  username: "nippy"
  password: "nippy"

devices:
  - name: "Relay 3 #1"
    id: "relay_3_1"
    node_id: 1
    manufacturer: "Nippy"
    model: "Relay 3"
    sw_version: "1.0"
    hw_version: "1.0"
    relays:
      - name: "Relay 1"
        id: "relay_1"
        child_id: 0
      - name: "Relay 2"
        id: "relay_2"
        child_id: 1
      - name: "Relay 3"
        id: "relay_3"
        child_id: 2

  - name: "Input 6 #1"
    id: "input_6_1"
    node_id: 2
    manufacturer: "Nippy"
    model: "Input 6"
    sw_version: "1.0"
    hw_version: "1.0"
    inputs:
      - name: "Input Button 1"
        id: "input_button_1"
        child_id: 0
      - name: "Input Button 2"
        id: "input_button_2"
        child_id: 1
      - name: "Input Button 3"
        id: "input_button_3"
        child_id: 2
      - name: "Input Button 4"
        id: "input_button_4"
        child_id: 3
      - name: "Input Button 5"
        id: "input_button_5"
        child_id: 4
      - name: "Input Button 6"
        id: "input_button_6"
        child_id: 5
```

### What's using defaults in this config:
- **MySensors port**: 5003 (default)
- **Transport**: ethernet (default)
- **MQTT port**: 1883 (default)  
- **TCP service**: disabled (default)
- **Node ID range**: 1-254 (default)
- **Optimistic mode**: false - waits for device confirmation (default)
- **Request ACK**: true - requests acknowledgment from devices (default)
- **Sync**: disabled (default)

## Advanced Configuration Options

### Enable TCP Message Monitoring
Add TCP service to view live MySensors messages for debugging:

```yaml
mysensors:
  default:
    ethernet:
      host: "172.30.15.1"
    tcp_service:
      enabled: true
      port: 5003  # Port required when enabled - connect to this port to view messages
```

### Multiple Gateways
Configure multiple MySensors networks:

```yaml
mysensors:
  default:
    ethernet:
      host: "172.30.15.1"
      port: 5003
    tcp_service:
      enabled: true
      port: 5003  # TCP port required when enabled
  garage:
    ethernet:
      host: "172.30.16.1"
      port: 5003  # Same MySensors port (different host)
    tcp_service:
      enabled: true
      port: 5004  # Different TCP port required

devices:
  - name: "Garage Controller"
    gateway: "garage"  # Uses garage gateway
    node_id: 1
    # ... rest of device config
```

### Per-Device Settings
Override global settings for specific devices:

```yaml
adapter:
  optimistic: false      # Global: wait for confirmation
  request_ack: true      # Global: request ACK

devices:
  - name: "Fast Response Device"
    id: "fast_device"
    node_id: 1
    request_ack: false   # Override: no ACK for this device
    relays:
      - name: "Quick Light"
        id: "quick_light"
        child_id: 0
        optimistic: true  # Override: immediate response
```

### Enable Periodic Sync
Keep device states synchronized:

```yaml
adapter:
  sync:
    enabled: true
    period: "30s"  # Sync every 30 seconds
```

## Troubleshooting

- **Devices not appearing**: Check MySensors gateway connection and node IDs
- **State updates not working**: Enable `request_ack: true` and disable `optimistic: false`
- **Slow responses**: Try `optimistic: true` for immediate UI updates
- **Multiple gateways**: Ensure different TCP ports if TCP service is enabled (port required when enabled)
- **View live messages**: Enable TCP service and connect to the port for debugging