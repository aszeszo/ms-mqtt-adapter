# MySensors MQTT Adapter (Home Assistant add-on repo)

Home Assistant add-on that bridges MySensors gateways to MQTT and publishes MQTT discovery for Home Assistant.

## Install

1. Home Assistant: **Settings → Add-ons → Add-on Store → ⋮ → Repositories**
2. Add repository URL: `https://github.com/aszeszo/ms-mqtt-adapter`
3. Install **MySensors MQTT Adapter** from the store.

## Configure

Configure the add-on using the single option `config_yaml`.

Minimal example:

```yaml
mysensors:
  main_house:
    ethernet:
      host: "192.168.1.100"

mqtt:
  broker: "core-mosquitto"
  username: "your_username"
  password: "your_password"

devices:
  - name: "My Device"
    id: "my_device"
    node_id: 1
    manufacturer: "Example"
    model: "Device"
    sw_version: "1.0"
    hw_version: "1.0"
    entities:
      - name: "Relay 1"
        id: "relay_1"
        child_id: 0
        entity_type: "switch"
        initial_value: "0"
```

## MQTT topics

- Entity state: `ms-mqtt-adapter/entity/{unique_id}/state`
- Entity command: `ms-mqtt-adapter/entity/{unique_id}/set`
- Seen nodes: `ms-mqtt-adapter/gateway/{gateway_name}/seen_nodes`
- Presentations: `ms-mqtt-adapter/gateway/{gateway_name}/node/{node_id}/presentation`

## Docs

- Add-on docs: `ha-addon/DOCS.md`
- Full configuration example: `example.yaml`
