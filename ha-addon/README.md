# MySensors MQTT Adapter (Home Assistant add-on)

Bridges MySensors gateways to MQTT and publishes Home Assistant MQTT discovery.

## Setup

1. Install the add-on.
2. Start the add-on.
3. Open the Web UI (click "OPEN WEB UI" or use the sidebar).
4. Configure via the web interface (MQTT broker, gateways, devices).

## Notes

- All configuration is managed through the web UI.
- Configuration is persisted to `/data/config.yaml` and survives updates.
- MQTT topics use `adapter.topic_prefix` (default: `ms-mqtt-adapter`).

See `DOCS.md` for detailed setup instructions and configuration guide.
