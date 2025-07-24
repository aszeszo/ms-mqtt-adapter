# Optimistic Mode Configuration

Optimistic mode controls how Home Assistant handles switch commands - whether to wait for device confirmation or assume success immediately.

## How It Works

### **Non-Optimistic Mode (Default)**
1. User clicks switch in Home Assistant
2. Command sent to MySensors device via MQTT
3. Home Assistant shows "pending" state
4. Device responds with actual state
5. Home Assistant updates to show confirmed state

**Pros:** Always shows actual device state  
**Cons:** UI feels slower, shows pending state

### **Optimistic Mode**
1. User clicks switch in Home Assistant  
2. Command sent to MySensors device via MQTT
3. Home Assistant immediately assumes success
4. Device eventually responds (updates state if different)

**Pros:** Instant UI response, feels faster  
**Cons:** May briefly show incorrect state if command fails

## Configuration Levels

### Global Setting (Adapter Level)

```yaml
adapter:
  topic_prefix: "ms-mqtt-adapter"
  homeassistant_discovery: true
  optimistic_mode: false  # Default for all relays
```

**Values:**
- `false` (default): Wait for device confirmation
- `true`: Assume commands succeed immediately

### Per-Relay Override

```yaml
devices:
  - name: "My Device"
    relays:
      - name: "Fast Switch"
        optimistic: true   # Override global setting
        
      - name: "Critical Relay"  
        optimistic: false  # Override global setting
        
      - name: "Default Relay"
        # Uses global optimistic_mode setting
```

## Decision Priority

1. **Per-Relay Setting** (`optimistic: true/false`) - Highest priority
2. **Global Setting** (`optimistic_mode: true/false`) - Medium priority  
3. **System Default** (`false`) - Lowest priority

## When to Use Each Mode

### **Use Non-Optimistic Mode (optimistic: false) When:**
- **Critical Controls** - Security systems, pumps, heating
- **Unreliable Network** - Poor MySensors connectivity
- **Important Feedback** - Need to know if command actually worked
- **Debugging** - Troubleshooting device communication

### **Use Optimistic Mode (optimistic: true) When:**
- **UI Responsiveness** - Lights, fans, non-critical switches
- **Reliable Network** - Stable MySensors communication
- **User Experience** - Want instant feedback in interface
- **High-Traffic Controls** - Frequently used switches

## Configuration Examples

### Conservative Setup (Global Non-Optimistic)

```yaml
adapter:
  optimistic_mode: false  # Wait for all confirmations

devices:
  - name: "Living Room"
    relays:
      - name: "Main Light"
        # Uses global: false (waits for confirmation)
        
      - name: "Desk Lamp" 
        optimistic: true  # Override: instant response
```

### Responsive Setup (Global Optimistic)

```yaml
adapter:
  optimistic_mode: true  # Assume success for fast UI

devices:
  - name: "Living Room"
    relays:
      - name: "Desk Lamp"
        # Uses global: true (instant response)
        
      - name: "Security Light"
        optimistic: false  # Override: wait for confirmation
```

### Mixed Environment

```yaml
adapter:
  optimistic_mode: false  # Conservative default

devices:
  - name: "Entertainment Center"
    relays:
      - name: "TV Power"
        optimistic: true   # Fast for frequently used
        
      - name: "Amplifier"
        optimistic: true   # Fast for frequently used
        
  - name: "Security System"
    relays:
      - name: "Alarm Enable"
        optimistic: false  # Critical: must confirm
        
      - name: "Door Lock"
        optimistic: false  # Critical: must confirm
```

## Troubleshooting

### UI Shows Wrong State Briefly
- **Cause:** Optimistic mode assumed success but device failed
- **Solution:** Set `optimistic: false` for that relay
- **Check:** MySensors network reliability

### UI Feels Slow/Unresponsive  
- **Cause:** Non-optimistic mode waiting for confirmations
- **Solution:** Set `optimistic: true` for non-critical relays
- **Check:** MySensors device response time

### Commands Don't Work
- **Cause:** Unrelated to optimistic mode (device/network issue)
- **Debug:** Check MySensors communication, MQTT topics
- **Tip:** Use non-optimistic mode to see actual device responses

## Home Assistant Discovery

The optimistic setting is automatically included in MQTT discovery:

```json
{
  "name": "My Switch",
  "optimistic": false,
  "command_topic": "ms-mqtt-adapter/devices/dev1/relay/switch1/set",
  "state_topic": "ms-mqtt-adapter/devices/dev1/relay/switch1/state"
}
```

## Best Practices

1. **Start Conservative** - Begin with global `optimistic_mode: false`
2. **Test Network** - Verify MySensors reliability before enabling optimistic mode
3. **Categorize Devices** - Use optimistic for UI/comfort, non-optimistic for critical
4. **Monitor Behavior** - Watch for state mismatches in optimistic mode
5. **Document Choices** - Comment why specific relays use optimistic/non-optimistic

## Migration Guide

### From Previous Versions
- **No config changes needed** - Defaults to non-optimistic (existing behavior)
- **To enable globally** - Add `optimistic_mode: true` to adapter section
- **Per-device control** - Add `optimistic: true/false` to individual relays

### Testing Your Setup
1. Set a relay to `optimistic: true`
2. Toggle in Home Assistant - should respond immediately
3. Disconnect MySensors device
4. Toggle again - should show immediate change but eventually revert
5. This confirms optimistic mode is working