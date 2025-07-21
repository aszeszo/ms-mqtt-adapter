# MySensors MQTT Adapter - Home Assistant Add-on Repository

This repository contains a Home Assistant add-on that bridges MySensors networks with MQTT, providing automatic device discovery for Home Assistant.

## Installation

### Adding the Repository

1. In Home Assistant, go to **Settings** → **Add-ons** → **Add-on Store**
2. Click the **⋮** menu (three dots) in the top right
3. Select **Repositories**
4. Add this URL: `https://github.com/aszeszo/ms-mqtt-adapter`
5. Click **Add** and wait for the repository to be processed

### Installing the Add-on

1. After adding the repository, refresh the Add-on Store
2. Find "MySensors MQTT Adapter" in the store
3. Click on it and select **Install**
4. Wait for the installation to complete

## Configuration

Before starting the add-on, configure it in the **Configuration** tab:

### Basic Configuration

```yaml
mysensors_host: "192.168.1.100"  # Your MySensors gateway IP
mysensors_port: 5003
mqtt_broker: core-mosquitto      # Use HA's built-in broker
mqtt_port: 1883
mqtt_username: ""                # Optional
mqtt_password: ""                # Optional
topic_prefix: ms-mqtt-adapter
homeassistant_discovery: true
```

### Advanced Options

- **log_level**: Set to `debug` for troubleshooting

## Features

- 🔄 **Automatic Discovery**: Devices appear automatically in Home Assistant
- 🏠 **Full HA Integration**: Native Home Assistant device support
- ⚡ **Real-time Control**: Immediate command processing
- 🔄 **State Persistence**: Device states survive restarts
- 🌐 **Multi-platform**: Supports all Home Assistant architectures
- 🛠️ **Gateway Functions**: Node ID assignment and time sync

## Device Types Supported

- **Relays/Switches**: 1:1 mapping with MySensors nodes
- **Inputs/Sensors**: Many-to-many mapping support
- **Binary Sensors**: Motion, door/window sensors, etc.

## MQTT Topics

The add-on creates organized topic structure:
- `ms-mqtt-adapter/devices/{device}/relay/{relay}/state`
- `ms-mqtt-adapter/devices/{device}/relay/{relay}/set`
- `ms-mqtt-adapter/devices/{device}/input/{input}/state`

## Troubleshooting

### Common Issues

1. **Gateway Connection Failed**
   - Verify MySensors gateway IP and port
   - Check network connectivity

2. **MQTT Issues**
   - Ensure Mosquitto broker is running
   - Check MQTT credentials if authentication is enabled

3. **Devices Not Appearing**
   - Verify `homeassistant_discovery` is enabled
   - Check add-on logs for errors

### Getting Help

- **Logs**: Check add-on logs in the **Log** tab
- **Debug Mode**: Set `log_level: debug` for detailed information
- **Issues**: Report problems on [GitHub](https://github.com/aszeszo/ms-mqtt-adapter/issues)

## Architecture Support

This add-on supports all Home Assistant platforms:
- amd64 (Intel/AMD 64-bit)
- aarch64 (ARM 64-bit)
- armv7 (ARM 32-bit)
- armhf (ARM hard-float)
- i386 (Intel 32-bit)

## Development

This project was developed using Claude Code AI assistant, demonstrating modern AI-assisted software development practices.

## License

MIT License - see repository for details.