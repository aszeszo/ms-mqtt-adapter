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

The addon uses a single `config_yaml` option where you provide the complete YAML configuration. This gives you full flexibility to configure:

- **MySensors gateways** - Ethernet or RS485 connections to your MySensors networks
- **MQTT broker** - Connection to your Home Assistant MQTT broker  
- **Device definitions** - Define your MySensors devices for auto-discovery
- **Adapter behavior** - Customize optimistic mode, sync settings, and more

See the addon documentation for detailed configuration examples and all available options.
