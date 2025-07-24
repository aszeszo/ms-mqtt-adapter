# MQTT Payload and State Customization

The MySensors MQTT Adapter supports full customization of MQTT payload and state values for Home Assistant integration. This allows you to integrate with systems that use different ON/OFF representations than the default "1"/"0" values.

## Configuration Fields

### For Relays (Switches)

```yaml
relays:
  - name: "My Switch"
    id: "my_switch"
    child_id: 0
    payload_on: "1"      # What to send to turn ON (default: "1")
    payload_off: "0"     # What to send to turn OFF (default: "0") 
    state_on: "1"        # What represents ON state (default: "1")
    state_off: "0"       # What represents OFF state (default: "0")
```

### For Inputs (Binary Sensors)

```yaml
inputs:
  - name: "My Sensor"
    id: "my_sensor"
    child_id: 0
    payload_on: "1"      # What represents ON/triggered (default: "1")
    payload_off: "0"     # What represents OFF/normal (default: "0")
    state_on: "1"        # Optional: ON state representation
    state_off: "0"       # Optional: OFF state representation
```

## Usage Examples

### Default Behavior (No Configuration Needed)

```yaml
relays:
  - name: "Standard Switch"
    id: "standard_switch"
    child_id: 0
    # Uses: payload_on="1", payload_off="0", state_on="1", state_off="0"
```

### Legacy System Integration

```yaml
relays:
  - name: "Legacy Relay"
    id: "legacy_relay"
    child_id: 0
    payload_on: "ON"
    payload_off: "OFF"
    state_on: "ON"
    state_off: "OFF"
```

### Boolean String Values

```yaml
relays:
  - name: "Boolean Switch"
    id: "boolean_switch"
    child_id: 0
    payload_on: "true"
    payload_off: "false"
    state_on: "true"
    state_off: "false"
```

### Custom Sensor States

```yaml
inputs:
  - name: "Door Sensor"
    id: "door_sensor"
    child_id: 0
    payload_on: "OPEN"
    payload_off: "CLOSED"
    state_on: "OPEN"
    state_off: "CLOSED"
```

### Numeric Values

```yaml
relays:
  - name: "Power Level"
    id: "power_level"
    child_id: 0
    payload_on: "100"
    payload_off: "0"
    state_on: "100"
    state_off: "0"
```

## How It Works

### Command Flow (Relays)
1. Home Assistant sends command using `payload_on` or `payload_off` values
2. Adapter receives command and sends to MySensors network
3. MySensors device responds with state
4. Adapter publishes state using `state_on` or `state_off` values

### State Flow (Inputs)
1. MySensors device sends state change
2. Adapter publishes to MQTT using `payload_on` or `payload_off` values
3. Home Assistant interprets based on configured values

### MQTT Topics

The payload customization affects these MQTT topics:

**Relays:**
- Command: `{topic_prefix}/devices/{device_id}/relay/{relay_id}/set`
- State: `{topic_prefix}/devices/{device_id}/relay/{relay_id}/state`

**Inputs:**
- State: `{topic_prefix}/devices/{device_id}/input/{input_id}/state`

### Home Assistant Discovery

The custom payload values are automatically included in Home Assistant MQTT discovery messages:

```json
{
  "name": "Legacy Relay",
  "unique_id": "device1_relay1",
  "command_topic": "ms-mqtt-adapter/devices/device1/relay/relay1/set",
  "state_topic": "ms-mqtt-adapter/devices/device1/relay/relay1/state",
  "payload_on": "ON",
  "payload_off": "OFF",
  "state_on": "ON",
  "state_off": "OFF"
}
```

## Migration from Fixed Values

If you're upgrading from a version that used fixed "1"/"0" values:

1. **No changes needed** - Default behavior remains the same
2. **To customize** - Add the payload fields to your configuration
3. **Existing MQTT retained messages** - Will continue to work with new payload values

## Best Practices

1. **Consistency** - Use the same payload format across similar devices
2. **Clarity** - Choose descriptive values (e.g., "OPEN"/"CLOSED" vs "1"/"0" for doors)
3. **Compatibility** - Consider what values your MySensors devices expect
4. **Testing** - Verify both directions (command → device and device → state) work correctly

## Troubleshooting

### Commands Not Working
- Check that `payload_on`/`payload_off` match what your MySensors device expects
- Verify MQTT command topics are being published correctly

### States Not Updating
- Ensure `state_on`/`state_off` values match what you want displayed in Home Assistant
- Check that MySensors device is sending expected values

### Home Assistant Not Recognizing States
- Verify discovery message contains correct payload values
- Clear and republish discovery messages if needed