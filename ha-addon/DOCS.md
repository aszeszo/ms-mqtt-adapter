# MySensors MQTT Adapter Documentation

## Installation

1. Add this repository to Home Assistant: **Settings → Add-ons → Add-on Store → ⋮ → Repositories**
2. Install the "MySensors MQTT Adapter" addon
3. Configure the addon options (see below)
4. Start the addon

## Basic Configuration

### MySensors Gateway
```yaml
mysensors:
  default:
    transport: ethernet
    ethernet:
      host: "192.168.1.100"
      port: 5003
    gateway:
      node_id_range:
        start: 1
        end: 254
    tcp_service:
      enabled: true
      port: 5003
```

### MQTT Broker
```yaml
mqtt:
  broker: core-mosquitto
  port: 1883
  username: ""
  password: ""
```

## Device Configuration

Add devices in the main configuration file at `/config/ms-mqtt-adapter/config.yaml`:

```yaml
devices:
  - name: "Living Room Relays"
    id: "living_room"
    node_id: 1
    relays:
      - name: "Main Light"
        id: "main_light"
        child_id: 0
        optimistic: false  # Wait for device confirmation
      - name: "Fan"
        id: "fan"
        child_id: 1
        optimistic: true   # Immediate response
```

## Multiple Gateways

```yaml
mysensors:
  default:
    transport: ethernet
    ethernet:
      host: "192.168.1.100"
      port: 5003
    tcp_service:
      port: 5003
  garage:
    transport: rs485
    rs485:
      device: "/dev/ttyUSB0"
    tcp_service:
      port: 5004  # Different port required
```

Then specify gateway in device config:
```yaml
devices:
  - name: "Garage Controller"
    gateway: "garage"  # Uses garage gateway
    node_id: 1
```

## TCP Monitoring

Connect to `<addon-ip>:5003` to view live MySensors messages for debugging.

## Troubleshooting

- **Devices not appearing**: Check MySensors gateway connection and node IDs
- **State updates not working**: Try `request_ack: true` and `optimistic: false`
- **Multiple gateways**: Ensure different TCP ports for each gateway