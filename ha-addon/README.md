# MySensors MQTT Adapter

Bridge between MySensors sensor networks and Home Assistant via MQTT.

## What it does

- **Connects MySensors devices** to Home Assistant automatically
- **Multiple gateway support** - connect ethernet and RS485 gateways simultaneously  
- **Auto-discovery** - devices appear in Home Assistant without manual configuration
- **Real-time control** - switch relays, read sensors, monitor device status
- **TCP monitoring** - view live MySensors traffic for debugging

## Device Support

- **Relays/Switches** - Control lights, pumps, fans, outlets
- **Binary Sensors** - Motion detectors, door/window contacts, buttons

## Configuration

Configure MySensors gateways, MQTT broker, and device definitions through the addon options. The adapter automatically creates Home Assistant entities for all configured devices.
