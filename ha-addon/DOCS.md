# MySensors MQTT Adapter

Bridge between MySensors networks and MQTT with Home Assistant auto-discovery.

## Configuration

**All configuration is done through the add-on's web UI.**

1. Install the add-on
2. Start the add-on
3. Open the Web UI (click "OPEN WEB UI" or use the sidebar)
4. Configure via the web interface:
   - **MQTT Settings**: Configure your MQTT broker connection
   - **Gateways**: Add MySensors gateways (Ethernet or RS485)
   - **Devices**: Define your MySensors devices and entities
   - **ID Aliases**: (Optional) Create friendly names for node/child IDs

The configuration is persisted to `/data/config.yaml` and survives add-on updates and restarts.

## First-Time Setup

When you first start the add-on, the dashboard will show a **"Configuration Incomplete"** warning listing required settings:

1. **MQTT Broker**: Navigate to **MQTT** tab and configure:
   - Broker address (e.g., `core-mosquitto` for HA's built-in broker)
   - Port (default: 1883)
   - Username/password if required

2. **MySensors Gateway**: Navigate to **Gateways** tab and add at least one gateway:
   - Gateway name (e.g., `main_house`)
   - Transport type: `ethernet` or `rs485`
   - Connection details (IP/port for Ethernet, device path for RS485)

Once MQTT and gateway are configured, the warning will disappear and you can add devices.

## Adding Devices

Navigate to the **Devices** tab and click **Add Device**:

1. **Device Info**:
   - Name (shown in Home Assistant)
   - Device ID (internal identifier)
   - MySensors Node ID
   - Manufacturer, model, version info

2. **Add Entities** for each sensor/actuator:
   - Entity type (switch, sensor, temperature, etc.)
   - Child ID (MySensors child sensor ID)
   - Configure sync, ACK, and other options as needed

## Web UI Features

- **Dashboard**: Status overview showing MQTT connection, gateways, and node availability
- **Devices**: Manage devices and entities
- **Gateways**: Manage MySensors gateway connections
- **MQTT**: Configure MQTT broker and adapter settings
- **MQTT Topics**: Browse and manage all MQTT topics from the broker
- **Aliases**: Create friendly names for node/child IDs
- **Logs**: Real-time log streaming
- **Config Editor**: Advanced YAML editor with validation

## MQTT Topics

- Entity state: `{topic_prefix}/entity/{unique_id}/state`
- Entity command: `{topic_prefix}/entity/{unique_id}/set`
- Discovery: `{discovery_prefix}/{entity_type}/{unique_id}/config`

Default prefixes:
- `topic_prefix`: `ms-mqtt-adapter`
- `discovery_prefix`: `homeassistant`

## Troubleshooting

1. **"Configuration Incomplete" warning**:
   - Check the dashboard for missing required settings
   - Configure MQTT broker and at least one gateway

2. **Entities not appearing in Home Assistant**:
   - Verify MQTT broker connection (check dashboard status)
   - Ensure Home Assistant discovery is enabled (MQTT tab)
   - Check that devices have entities configured

3. **Gateway not connecting**:
   - Verify gateway IP/port or device path
   - Check gateway logs for connection errors
   - Ensure MySensors gateway is running and accessible

4. **Enable debug logging**:
   - Navigate to **MQTT** tab
   - Set Log Level to `debug`
   - Check **Logs** tab for detailed output

## Advanced Configuration

For advanced users, the **Config Editor** tab provides direct YAML editing with:
- Real-time validation
- Syntax highlighting
- Error messages

Configuration file location: `/data/config.yaml`

See the [GitHub repository](https://github.com/aszeszo/ms-mqtt-adapter) for detailed configuration options and examples.
