# MySensors MQTT Adapter (Home Assistant add-on)

Bridges MySensors gateways to MQTT and publishes Home Assistant MQTT discovery.

## Setup

1. Install the add-on.
2. In the add-on **Configuration** tab, set `config_yaml`.
3. Start the add-on.

## Notes

- The first gateway under `mysensors:` is the default gateway.
- MQTT topics use `adapter.topic_prefix` (default: `ms-mqtt-adapter`).

See `DOCS.md` for examples and option reference.
