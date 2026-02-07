# MySensors MQTT Adapter (add-on)

## Configuration

The add-on uses a single option: `config_yaml`. Paste your full adapter configuration there.

Minimal example:

```yaml
log_level: info

mysensors:
  main_house:
    transport: ethernet
    ethernet:
      host: "172.30.15.1"
      port: 5003

mqtt:
  broker: "core-mosquitto"
  port: 1883
  username: "user"
  password: "pass"
  client_id: "ms-mqtt-adapter"

adapter:
  topic_prefix: "ms-mqtt-adapter"
  homeassistant_discovery: true

id_aliases:
  relay_node: 1
  relay_1: 0

devices:
  - name: "Relay Board"
    id: "relay_board"
    node_id: relay_node
    manufacturer: "Nippy"
    model: "Relay"
    sw_version: "1.0"
    hw_version: "1.0"
    entities:
      - name: "Relay 1"
        id: "relay_1"
        child_id: relay_1
        entity_type: "switch"
        initial_value: "0"
```

## Common options

### Gateways

- `mysensors.<name>.transport`: `ethernet` or `rs485`
- `mysensors.<name>.ethernet.host` / `port`
- `mysensors.<name>.rs485.device`
- `mysensors.<name>.gateway.node_id_assignment.enabled`: enable/disable automatic node id assignment
- `mysensors.<name>.gateway.availability_window`: node availability window
- `mysensors.<name>.tcp_service.enabled`: expose a TCP port to mirror traffic (debugging)

### Entities

Per-entity settings commonly used:

- `node_id` (optional override)
- `gateway` (optional override)
- `sync_period` / `sync_splay`
- `availability_window`
- `optimistic` (wait for device confirmation vs immediate state)
- `request_ack` / `ack_timeout` / `ack_retries`

## MQTT topics

- Entity state: `{topic_prefix}/entity/{unique_id}/state`
- Entity command: `{topic_prefix}/entity/{unique_id}/set`

## Troubleshooting

- Set `log_level: debug` and check the add-on logs.
- Verify the adapter can reach the MySensors gateway and the MQTT broker.
- If entities do not appear in Home Assistant, ensure MQTT discovery is enabled and retained messages are allowed.
