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

  # Example device with sensors (numeric measurements)
  - name: "Environment Sensor"
    id: "env_sensor"
    node_id: 5
    manufacturer: "Nippy"
    model: "Environment Pro" 
    sw_version: "1.0"
    hw_version: "1.0"
    inputs:
      - name: "Temperature"
        id: "temperature"
        child_id: 0
        sensor_type: "temperature"  # Numeric temperature sensor
      - name: "Humidity"
        id: "humidity"
        child_id: 1
        sensor_type: "humidity"     # Numeric humidity sensor
      - name: "Battery Level"
        id: "battery"
        child_id: 2
        sensor_type: "battery"      # Numeric battery sensor
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

### Outputs

The adapter supports outputs for controlling MySensors devices with different value types. Outputs can be text, numeric, switches, or selectable options:

```yaml
devices:
  - name: "Smart Display #1"
    id: "smart_display_1"
    node_id: 6
    manufacturer: "Nippy"
    model: "Smart Display"
    sw_version: "1.0"
    hw_version: "1.0"
    outputs:
      # Text output - allows setting arbitrary text messages
      - name: "Display Message"
        id: "display_message"
        child_id: 0
        output_type: "text"           # Maps to MySensors V_TEXT
        initial_value: "Hello World"
        icon: "mdi:message-text"
        entity_category: "config"
      
      # Number output - allows setting numeric values with range
      - name: "Brightness Level"
        id: "brightness_level"
        child_id: 1
        output_type: "number"         # Maps to MySensors V_PERCENTAGE
        initial_value: "50"
        min_value: 0
        max_value: 100
        step: 5
        unit_of_measurement: "%"
        icon: "mdi:brightness-6"
        
      # Select output - allows choosing from predefined options
      - name: "Display Mode"
        id: "display_mode"
        child_id: 2
        output_type: "select"         # Maps to MySensors V_TEXT
        initial_value: "normal"
        options: ["normal", "bright", "dim", "off"]
        icon: "mdi:monitor"
        
      # Switch output - equivalent to relay but using outputs
      - name: "Backlight Power"
        id: "backlight_power"
        child_id: 3
        output_type: "switch"         # Maps to MySensors V_STATUS
        initial_value: "1"
        icon: "mdi:lightbulb"
```

**Available output types**:
- **switch**: Binary on/off control (maps to V_STATUS)
- **light**: Light control (maps to V_STATUS)
- **dimmer**: Dimmer control with percentage (maps to V_PERCENTAGE)
- **cover**: Cover/blind control (maps to V_UP/V_DOWN/V_STOP)
- **text**: Text message control (maps to V_TEXT)
- **number**: Numeric value control (maps to V_PERCENTAGE)
- **select**: Selection from predefined options (maps to V_TEXT)
- **climate**: Climate control (maps to V_HVAC_SETPOINT_HEAT)

**Variable type override**: You can override the default MySensors variable type:
```yaml
- name: "Custom Output"
  output_type: "text"
  variable_type: "V_CUSTOM"  # Override default V_TEXT
```

### Sensor Types

The adapter supports both binary and numeric sensors:

#### Binary Sensors (Default)
For buttons, switches, motion detectors, etc. that report 0/1 values:

```yaml
inputs:
  - name: "Motion Sensor"
    id: "motion"
    child_id: 0
    sensor_type: "binary"  # Optional - this is the default
    device_class: "motion"
```

#### Numeric Sensors
For temperature, humidity, battery levels, etc. that report numeric values:

```yaml
inputs:
  - name: "Temperature"
    id: "temperature"
    child_id: 0
    sensor_type: "temperature"    # Maps to MySensors V_TEMP
    # Defaults: unit="°C", state_class="measurement"
    
  - name: "Humidity"
    id: "humidity" 
    child_id: 1
    sensor_type: "humidity"       # Maps to MySensors V_HUM
    # Defaults: unit="%", state_class="measurement"
    
  - name: "Battery"
    id: "battery"
    child_id: 2
    sensor_type: "battery"        # Maps to MySensors V_PERCENTAGE
    # Defaults: unit="%", state_class="measurement", entity_category="diagnostic"
    
  - name: "Voltage"
    id: "voltage"
    child_id: 3
    sensor_type: "voltage"        # Maps to MySensors V_VOLTAGE
    # Defaults: unit="V", state_class="measurement"
    
  - name: "Current"
    id: "current"
    child_id: 4
    sensor_type: "current"        # Maps to MySensors V_CURRENT
    # Defaults: unit="A", state_class="measurement"
    
  - name: "Pressure"
    id: "pressure"
    child_id: 5
    sensor_type: "pressure"       # Maps to MySensors V_PRESSURE
    # Defaults: unit="hPa", state_class="measurement"
    
  - name: "Level"
    id: "level"
    child_id: 6
    sensor_type: "level"          # Maps to MySensors V_LEVEL
    # Defaults: unit="%", state_class="measurement"
```

**Available sensor types**: 
- **Basic**: `binary`, `temperature`, `humidity`, `battery`, `voltage`, `current`, `pressure`, `level`, `percentage`
- **Power/Energy**: `watt`, `kwh`, `var`, `va`, `power_factor`  
- **Environmental**: `light_level`, `uv`, `ph`, `orp`, `ec`
- **Weather**: `wind`, `gust`, `direction`, `rain`, `rainrate`
- **Physical**: `weight`, `distance`, `volume`, `flow`
- **Special**: `text`, `custom`, `position`, `impedance`

**Custom units**: You can override default units:
```yaml
- name: "Custom Pressure"
  sensor_type: "pressure"
  unit_of_measurement: "PSI"  # Override default "hPa"
```

## Troubleshooting

- **Devices not appearing**: Check MySensors gateway connection and node IDs
- **State updates not working**: Enable `request_ack: true` and disable `optimistic: false`
- **Slow responses**: Try `optimistic: true` for immediate UI updates
- **Multiple gateways**: Ensure different TCP ports if TCP service is enabled (port required when enabled)
- **View live messages**: Enable TCP service and connect to the port for debugging