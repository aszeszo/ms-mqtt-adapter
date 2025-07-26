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
    entities:
      - name: "Relay 1"
        id: "relay_1"
        child_id: 0
        entity_type: "switch"
        initial_value: "0"
        icon: "mdi:electric-switch"
      - name: "Relay 2"
        id: "relay_2"
        child_id: 1
        entity_type: "switch"
        initial_value: "0"
        icon: "mdi:electric-switch"
      - name: "Relay 3"
        id: "relay_3"
        child_id: 2
        entity_type: "switch"
        initial_value: "0"
        icon: "mdi:electric-switch"

  - name: "Input 6 #1"
    id: "input_6_1"
    node_id: 2
    manufacturer: "Nippy"
    model: "Input 6"
    sw_version: "1.0"
    hw_version: "1.0"
    entities:
      - name: "Input Button 1"
        id: "input_button_1"
        child_id: 0
        entity_type: "binary_sensor"
        read_only: true
        device_class: "button"
        icon: "mdi:button-pointer"
      - name: "Input Button 2"
        id: "input_button_2"
        child_id: 1
        entity_type: "binary_sensor"
        read_only: true
        device_class: "button"
        icon: "mdi:button-pointer"
      - name: "Input Button 3"
        id: "input_button_3"
        child_id: 2
        entity_type: "binary_sensor"
        read_only: true
        device_class: "button"
        icon: "mdi:button-pointer"
      - name: "Input Button 4"
        id: "input_button_4"
        child_id: 3
        entity_type: "binary_sensor"
        read_only: true
        device_class: "button"
        icon: "mdi:button-pointer"
      - name: "Input Button 5"
        id: "input_button_5"
        child_id: 4
        entity_type: "binary_sensor"
        read_only: true
        device_class: "button"
        icon: "mdi:button-pointer"
      - name: "Input Button 6"
        id: "input_button_6"
        child_id: 5
        entity_type: "binary_sensor"
        read_only: true
        device_class: "button"
        icon: "mdi:button-pointer"

  # Example device with sensors (numeric measurements)
  - name: "Environment Sensor"
    id: "env_sensor"
    node_id: 5
    manufacturer: "Nippy"
    model: "Environment Pro" 
    sw_version: "1.0"
    hw_version: "1.0"
    entities:
      - name: "Temperature"
        id: "temperature"
        child_id: 0
        entity_type: "temperature"
        read_only: true
        icon: "mdi:thermometer"
      - name: "Humidity"
        id: "humidity"
        child_id: 1
        entity_type: "humidity"
        read_only: true
        icon: "mdi:water-percent"
      - name: "Battery Level"
        id: "battery"
        child_id: 2
        entity_type: "battery"
        read_only: true
        entity_category: "diagnostic"
        icon: "mdi:battery"
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
    entities:
      - name: "Quick Light"
        id: "quick_light"
        child_id: 0
        entity_type: "switch"
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

### Entity Configuration

The adapter uses a unified **entities** configuration. An entity can be a sensor (read-only), actuator (controllable), or both, making configuration simpler and more flexible:

```yaml
devices:
  - name: "Smart Device"
    id: "smart_device"
    node_id: 5
    manufacturer: "Example"
    model: "Multi-Function"
    sw_version: "1.0"
    hw_version: "1.0"
    entities:
      # Switch actuator (equivalent to old "relay")
      - name: "Power Switch"
        id: "power_switch"
        child_id: 0
        entity_type: "switch"           # Maps to MySensors V_STATUS
        initial_value: "0"
        icon: "mdi:power"
      
      # Text actuator (send text messages to device)
      - name: "Display Message"
        id: "display_msg"
        child_id: 1
        entity_type: "text"             # Maps to MySensors V_TEXT
        initial_value: "Hello"
        icon: "mdi:message-text"
      
      # Number actuator (set numeric values)
      - name: "Brightness"
        id: "brightness"
        child_id: 2
        entity_type: "number"           # Maps to MySensors V_PERCENTAGE
        initial_value: "50"
        min_value: 0
        max_value: 100
        step: 5
        unit_of_measurement: "%"
        icon: "mdi:brightness-6"
      
      # Temperature sensor (read-only)
      - name: "Temperature"
        id: "temperature"
        child_id: 3
        entity_type: "temperature"      # Maps to MySensors V_TEMP
        read_only: true
        icon: "mdi:thermometer"
      
      # Binary sensor (read-only)
      - name: "Motion"
        id: "motion"
        child_id: 4
        entity_type: "binary_sensor"    # Maps to MySensors V_STATUS
        read_only: true
        device_class: "motion"
        icon: "mdi:motion-sensor"
```

**Entity Capabilities:**
- `read_only: true` - Entity can only report values (sensor)
- `read_only: false` - Entity can receive commands (actuator) - default for most types
- `write_only: true` - Entity can only receive commands, never reports state

**Available entity types:**

**Actuator Types** (can receive commands):
- **switch**: Binary on/off control (maps to V_STATUS)
- **light**: Light control (maps to V_STATUS)  
- **dimmer**: Dimmer with percentage (maps to V_PERCENTAGE)
- **cover**: Cover/blind control (maps to V_UP/V_DOWN/V_STOP)
- **text**: Text message control (maps to V_TEXT)
- **number**: Numeric value control (maps to V_PERCENTAGE)
- **select**: Selection from predefined options (maps to V_TEXT)
- **climate**: Climate control (maps to V_HVAC_SETPOINT_HEAT)

**Sensor Types** (typically read-only):
- **sensor**: Generic sensor (maps to V_CUSTOM)
- **binary_sensor**: Binary sensor (maps to V_STATUS)
- **temperature**: Temperature sensor (maps to V_TEMP)
- **humidity**: Humidity sensor (maps to V_HUM)
- **battery**: Battery level (maps to V_PERCENTAGE)
- **voltage**: Voltage sensor (maps to V_VOLTAGE)
- **current**: Current sensor (maps to V_CURRENT)
- **pressure**: Pressure sensor (maps to V_PRESSURE)
- **level**: Level sensor (maps to V_LEVEL)
- And many more sensor types...

**Variable type override:**
```yaml
- name: "Custom Entity"
  entity_type: "text"
  variable_type: "V_CUSTOM"  # Override default V_TEXT
```

**MQTT Topics:**
- Command topic: `ms-mqtt-adapter/devices/{device_id}/entity/{entity_id}/set`
- State topic: `ms-mqtt-adapter/devices/{device_id}/entity/{entity_id}/state`


## Troubleshooting

- **Devices not appearing**: Check MySensors gateway connection and node IDs
- **State updates not working**: Enable `request_ack: true` and disable `optimistic: false`
- **Slow responses**: Try `optimistic: true` for immediate UI updates
- **Multiple gateways**: Ensure different TCP ports if TCP service is enabled (port required when enabled)
- **View live messages**: Enable TCP service and connect to the port for debugging