# ms-mqtt-adapter Specification (v2.2.0 codebase)

This document specifies the behavior of the Go-based **MySensors MQTT Adapter** found in this repository. It is written to be sufficiently precise that the adapter can be re-implemented from scratch while preserving observable behavior and interfaces.

## 1. Purpose and Scope

### 1.1 Purpose

The adapter bridges one or more **MySensors gateways** to an **MQTT broker** and optionally:

- publishes **Home Assistant MQTT Discovery** configuration for configured devices/entities
- mirrors MySensors traffic to/from external tools via a **TCP replication service** (per gateway)

### 1.2 Non-goals

The current implementation does **not** aim to:

- implement the full MySensors protocol (it handles a small subset of internal message types)
- provide TLS, advanced auth, or certificate management for MQTT
- offer a UI; configuration is via YAML only

## 2. Glossary

- **Gateway**: A MySensors gateway instance (Ethernet/TCP or RS485/serial) through which node traffic flows.
- **Node**: A MySensors node identified by `node_id` (1..254 typically; 0/255 are reserved).
- **Child**: A MySensors child sensor/actuator on a node, identified by `child_id` (0..255).
- **Entity**: A configured mapping between `(gateway, node_id, child_id, variable_type)` and MQTT/HA concepts.
- **Device**: A Home Assistant device grouping one or more entities.
- **Unique ID**: Per-entity identifier used in MQTT topics and HA discovery topics.

## 3. External Interfaces

### 3.1 MySensors Wire Format

The adapter parses and emits the canonical MySensors serial/TCP text format:

```
node_id;child_id;message_type;ack;sub_type;payload\n
```

- `message_type` is an integer enum:
  - `0` PRESENTATION
  - `1` SET
  - `2` REQ
  - `3` INTERNAL
  - `4` STREAM (parsed but otherwise ignored)
- `ack` is `"0"` or `"1"`; parsing treats `"1"` as true, else false.
- `sub_type` meaning depends on `message_type`:
  - PRESENTATION: sensor type
  - SET/REQ: variable type
  - INTERNAL: internal type

Only newline-delimited frames are supported. The adapter ignores blank lines.

### 3.2 MySensors Transports

Each configured gateway uses exactly one transport:

1. **Ethernet (TCP client)**: adapter connects to `host:port` and reads/writes newline-delimited MySensors frames.
2. **RS485 (serial)**: adapter opens a serial device and reads/writes newline-delimited frames.

Transport properties:

- message receive is buffered (channel size 1000); overflow drops messages with a log warning
- write errors mark the transport as disconnected and trigger reconnection logic

### 3.3 MQTT Broker

The adapter uses MQTT (via Eclipse Paho client) over plain TCP:

- broker URL: `tcp://{broker}:{port}`
- username/password supported
- client id configurable
- auto reconnect enabled

The adapter:

- subscribes to entity command topics: `{topic_prefix}/entity/{unique_id}/set`
- subscribes to entity state topics to capture retained state on startup:
  `{topic_prefix}/entity/{unique_id}/state`
- publishes retained messages to various topics (see Section 6)

QoS is always `0` in the current implementation.

### 3.4 Optional TCP Replication Service

Per gateway, an optional TCP server can be enabled to:

- broadcast all MySensors frames seen by the adapter (inbound from gateway and outbound commands/responses)
- accept injected MySensors frames from clients and forward them to the gateway and local processing pipeline

This is intended for debugging and for integration with external MySensors tools.

## 4. Configuration

### 4.1 Source and Reloading

- The adapter reads a YAML config file, default path `config.yaml`.
- CLI flag: `-config <path>`.
- A file watcher (fsnotify) watches the config path for write/create events and attempts a live reload.

Important: config reload is implemented but has limitations (see Section 10).

### 4.2 YAML Schema (top level)

```yaml
log_level: "info"                     # debug|info|warn|error; default: info

mysensors:                            # required, at least one entry
  <gateway_name>:
    transport: "ethernet"             # ethernet|rs485; default set later
    ethernet:
      host: "192.0.2.10"              # required for ethernet usage
      port: 5003
    rs485:
      device: "/dev/ttyUSB0"

    gateway:
      node_id_assignment:
        enabled: true                 # default: true
        node_id_range:
          start: 1                    # default: 1
          end: 254                    # default: 254
        random_id_assignment: false   # default: false

      heartbeat_request_period: "10s" # default in code: 0 (disabled)
      availability_window: "30s"      # default: 30s

    tcp_service:
      enabled: false                  # default: false
      port: 5004                      # required when enabled

mqtt:
  broker: "core-mosquitto"            # required
  port: 1883                          # default: 1883
  username: "user"
  password: "pass"
  client_id: "ms-mqtt-adapter"        # default: ms-mqtt-adapter

adapter:
  topic_prefix: "ms-mqtt-adapter"     # default: ms-mqtt-adapter
  homeassistant_discovery: true       # default: true

id_aliases:                           # optional; alias -> int (0..255)
  relay_node: 1
  relay_1: 0

devices:                              # optional; if absent/empty, no entities are exposed (gateway topics still publish)
  - name: "Relay Board"
    id: "relay_board"
    node_id: relay_node               # int or alias; can be omitted if each entity sets node_id
    gateway: "main_house"             # optional; otherwise "default gateway" heuristic
    discovery:                        # Home Assistant MQTT discovery device info
      manufacturer: "Example"
      model: "Relay"
      sw_version: "1.0"
      hw_version: "1.0"
      configuration_url: "http://..."
      suggested_area: "Basement"      # Suggested HA area (device-level only)
      connections:
        - ["mac", "AA:BB:CC:DD:EE:FF"]
      via_device: "some_device_id"
    entities:
      - name: "Relay 1"
        id: "relay_1"
        child_id: relay_1             # int or alias (required)
        entity_type: "switch"         # required; see 4.4
        initial_value: "0"
        # ... entity options (below)
```

### 4.3 Gateway Configuration Details

`mysensors.<name>.transport`:

- `"ethernet"`: connect via TCP to `ethernet.host:ethernet.port`
- `"rs485"`: open `rs485.device` as serial port

Notes / observed behavior to preserve:

- If `transport` is omitted, the code later defaults it to `"ethernet"`.
- Validation of `ethernet.host`/`ethernet.port` is only enforced when `transport` is explicitly `"ethernet"`. If `transport` is omitted, missing host/port may pass validation but will fail at runtime.
- RS485 baud rate is fixed at **9600** in code (no YAML knob).

`mysensors.<name>.gateway.node_id_assignment`:

- If enabled, the adapter handles `I_ID_REQUEST` and responds with `I_ID_RESPONSE` using either:
  - sequential selection from `start..end` (first free id)
  - random selection from the remaining pool if `random_id_assignment: true`

`mysensors.<name>.gateway.heartbeat_request_period`:

- A period > 0 causes the adapter to send `I_HEARTBEAT_REQUEST` to each gateway periodically.
- A value of 0 disables periodic heartbeat requests.
- **Observed default in code is 0** unless explicitly set in YAML.

`mysensors.<name>.gateway.availability_window`:

- default: 30s
- used as fallback availability window for nodes on this gateway unless any entity on the node overrides it (see 7.8).

`mysensors.<name>.tcp_service`:

- when enabled, starts a TCP server on the configured port.
- the port must be unique across gateways (enforced by validation).

### 4.4 Entity Types

Entity types are validated against a fixed allowlist:

Actuator-ish:

- `switch`, `light`, `dimmer`, `cover`, `text`, `number`, `select`, `climate`, `rgb_light`, `rgbw_light`

Sensor-ish:

- `sensor`, `binary_sensor`, `temperature`, `humidity`, `battery`, `voltage`, `current`, `pressure`, `level`,
  `percentage`, `weight`, `distance`, `light_level`, `watt`, `kwh`, `flow`, `volume`, `ph`, `orp`, `ec`,
  `var`, `va`, `power_factor`, `custom`, `position`, `uv`, `rain`, `rainrate`, `wind`, `gust`, `direction`,
  `impedance`

### 4.5 Entity Fields

Core mapping fields:

- `child_id` (required): int or alias, resolved via `id_aliases`
- `node_id` (optional): int or alias; overrides device `node_id`
- `gateway` (optional): overrides device gateway selection
- `entity_type` (required)
- `variable_type` (optional): a MySensors variable type name like `V_STATUS`, `V_TEXT`, etc.

Identity fields:

- `unique_id` (optional): if absent, defaults to `{device.id}_{entity.id}`
- `default_entity_id` (optional pointer string):
  - if unset: defaults to `{domain}.{device.id}_{entity.id}` and is included in discovery
  - if set to `""` (empty string): `default_entity_id` is omitted from discovery payload
  - if set to custom value: uses that full entity ID (must include domain prefix, e.g., `switch.basement_main_power`)
  - Format: `{domain}.{name}` where domain is determined by entity type (e.g., `binary_sensor`, `switch`, `sensor`)

Read/write capability flags:

- `read_only` (optional bool)
- `write_only` (optional bool)

Observed default behavior:

- If `read_only` is not set:
  - entity types `sensor` and `binary_sensor` default to read-only
  - all other entity types default to not read-only (even for `temperature`, `humidity`, etc.)
- If `write_only` is not set: defaults to false

Sync fields:

- `sync_period` (duration, default 0 = disabled)
- `sync_splay` (duration, default 0; ignored unless `0 < sync_splay < sync_period`)

Availability fields:

- `availability_window` (duration, optional): candidate window for node availability computation; minimum across entities on same node wins.

Actuator configuration fields (used mostly for HA discovery):

- `initial_value` (string; defaults depend on `entity_type`)
- number: `min_value`, `max_value`, `step`
- select: `options`

Home Assistant / MQTT discovery metadata fields (entity-level):

- `icon`, `device_class`, `entity_category`, `enabled_by_default`, `unit_of_measurement`, `state_class`
- payload fields: `payload_on`, `payload_off`, etc.
- template fields: `value_template`, `json_attributes_template`, etc.

Note: `suggested_area` is a device-level field only (set in the device's `discovery:` section).

ACK fields:

- `request_ack` (bool): default true
- `ack_timeout` (duration): default 250ms
- `ack_retries` (int): default 3

Important: several MQTT-related fields are only used to populate HA discovery JSON; the adapter’s own publishing uses fixed QoS=0 and often hard-coded retain behavior (see Section 6 and Section 10).

### 4.6 ID Alias Resolution

`node_id` and `child_id` accept:

- integer YAML values
- string aliases that map via `id_aliases`

YAML quirk handled:

- YAML may unmarshal integers as `float64`; the code accepts float64 and casts to int.

### 4.7 Validation Rules (as implemented)

On config load, validation runs before defaults are applied. The current rules are:

- `mysensors` must contain at least one gateway.
- For each `mysensors.<gateway>`:
  - `transport` is allowed to be empty, `"ethernet"`, or `"rs485"`. Any other value errors.
  - If `transport` is explicitly `"ethernet"`:
    - `ethernet.host` must be non-empty
    - `ethernet.port` must be non-zero
  - If `tcp_service.enabled: true`:
    - `tcp_service.port` must be non-zero
    - ports must be unique across gateways
- `mqtt.broker` must be non-empty.
- `id_aliases` (if present):
  - alias names must be non-empty
  - id must be in range `0..255` (inclusive)
- For each `device`:
  - if `device.node_id` is set, it must resolve (int, float64, or alias)
  - if `device.node_id` is not set, then **all** entities in the device must set `entity.node_id`
- For each `entity`:
  - `entity.entity_type` must be set and in the allowlist (Section 4.4)
  - `entity.child_id` must resolve
  - effective `node_id` (entity.node_id or device.node_id) must resolve
- Duplicate `(node_id, child_id)` mappings are allowed, but at most one entity per target may have sync enabled (`sync_period > 0`).

Notably not validated:

- whether `device.id`, `entity.id`, `unique_id` are unique
- whether `device.gateway` / `entity.gateway` refer to a declared `mysensors` gateway
- RS485 device path presence (it is checked on connect instead)

### 4.8 Defaulting Rules (as implemented)

After validation, defaults are applied:

Top-level:

- `log_level`: defaults to `"info"`
- `mqtt.port`: defaults to `1883`
- `mqtt.client_id`: defaults to `"ms-mqtt-adapter"`
- `adapter.topic_prefix`: defaults to `"ms-mqtt-adapter"`
- `adapter.homeassistant_discovery`: defaults to `true`

Per gateway:

- if `transport` is empty, it is set to `"ethernet"`
- `gateway.node_id_assignment.node_id_range.start`: defaults to `1` if `0`
- `gateway.node_id_assignment.node_id_range.end`: defaults to `254` if `0`
- `gateway.node_id_assignment.enabled`: defaults to `true` if unset
- `gateway.node_id_assignment.random_id_assignment`: defaults to `false` if unset
- `gateway.availability_window`: defaults to `30s` if `0` (and prints a line to stdout via `fmt.Printf`)
- if transport is `"ethernet"` and `ethernet.port` is `0`, it defaults to `5003`
- `gateway.heartbeat_request_period`: no default is applied; if omitted it remains `0` (disabled)

Per entity:

- `initial_value` is defaulted when empty:
  - `switch`, `light`, `binary_sensor`: `"0"`
  - `dimmer`, `number`, `percentage`, `level`: `"0"`
  - `text`, `select`, `sensor`: `""`
  - everything else: `"0"`
- default units/state classes (if fields are empty):
  - `temperature`: unit `"\u00b0C"`, state_class `"measurement"`
  - `humidity`, `battery`, `percentage`, `level`: unit `"%"`, state_class `"measurement"`
  - `voltage`: unit `"V"`, state_class `"measurement"`
  - `current`: unit `"A"`, state_class `"measurement"`
  - `pressure`: unit `"hPa"`, state_class `"measurement"`
  - `weight`: unit `"kg"`, state_class `"measurement"`
  - `distance`: unit `"m"`, state_class `"measurement"`
  - `light_level`: unit `"lx"`, state_class `"measurement"`
  - `watt`: unit `"W"`, state_class `"measurement"`
  - `kwh`: unit `"kWh"`, state_class `"total_increasing"`
  - `flow`: unit `"m\u00b3/h"`, state_class `"measurement"`
  - `volume`: unit `"m\u00b3"`, state_class `"total_increasing"`
- `optimistic`: defaults to `false` if unset
- `request_ack`: defaults to `true` if unset

Duration values in YAML are expected to use Go duration strings (e.g. `"250ms"`, `"30s"`, `"5m"`).

## 5. Internal Architecture (Modules)

Recommended module split for a reimplementation (matches the current layout):

- `internal/mysensors`
  - MySensors message parsing/formatting and enums
- `pkg/transport`
  - `Transport` interface and concrete implementations:
    - Ethernet transport
    - RS485 transport
- `pkg/mqtt`
  - MQTT client wrapper:
    - subscribe to entity topics
    - publish discovery and states
    - maintain in-memory state cache
- `pkg/gateway`
  - per-gateway logic:
    - node id assignment
    - time responses
    - node tracking (seen nodes)
    - availability tracking (last-seen timestamps)
- `pkg/tcp`
  - optional TCP replication server
- `internal/events`
  - logger and entity sync manager
- `cmd/ms-mqtt-adapter`
  - main application wiring, concurrency, retry/backoff, config reload

## 6. MQTT Topic Contract

All adapter topics are namespaced under:

- `topic_prefix` (default: `ms-mqtt-adapter`)

### 6.1 Entity Topics

For each entity:

- State topic: `{topic_prefix}/entity/{unique_id}/state`
  - retained
  - payload: opaque string (usually `"0"`, `"1"`, numbers, or text depending on entity)
- Command topic: `{topic_prefix}/entity/{unique_id}/set`
  - adapter subscribes to this for entities that can receive commands
  - payload: opaque string validated loosely based on entity type
- Availability topic (published by adapter): `{topic_prefix}/entity/{unique_id}/availability`
  - retained
  - payload: `"online"` or `"offline"`

### 6.2 Gateway / Adapter Topics

Per gateway:

- Seen nodes: `{topic_prefix}/gateway/{gateway_name}/seen_nodes`
  - retained
  - payload: comma-separated sorted list of node IDs (e.g. `"1,2,31"`) or `""` if none
- Last issued node id: `{topic_prefix}/gateway/{gateway_name}/last_id`
  - retained
  - payload: decimal node id assigned in the last `I_ID_REQUEST` handling
- Presentations (per node): `{topic_prefix}/gateway/{gateway_name}/node/{node_id}/presentation`
  - retained
  - payload: newline-separated entries, accumulated and deduplicated

Presentation entry format (single line):

```
child_id:<child_id>;sensor_type:S_<n>;description:<payload>
```

### 6.3 Home Assistant Discovery Topics

If `adapter.homeassistant_discovery` is enabled, for each entity:

- Discovery topic: `homeassistant/{ha_component}/{unique_id}/config`
  - retained JSON payload

Where `ha_component` is derived from `entity_type` (see 8.2).

## 7. Runtime Behavior

### 7.1 Startup Sequence

1. Parse CLI flags (`-config`, default `config.yaml`).
2. Load YAML config, validate, then apply defaults.
3. Initialize:
   - logger
   - one transport per gateway
   - MQTT client wrapper
   - optional TCP servers per gateway
   - one `Gateway` handler per gateway
   - entity sync manager
4. Start a config file watcher (best-effort).
5. Connect with retry/backoff (infinite retries):
   - MQTT broker
   - each MySensors transport
6. Start TCP servers (no retry loop besides process restart / reload).
7. Start entity sync manager (after successful connections).
8. Publish Home Assistant discovery (and initial states as needed).
9. Optionally send initial heartbeat requests (if heartbeat enabled).
10. Start background goroutines:
    - per-transport receive loop handling
    - per-TCP-server receive loop handling
    - MQTT command handler registration
    - periodic heartbeat sender
    - availability monitor
    - config reload loop

Shutdown is triggered by SIGINT/SIGTERM; the adapter disconnects MQTT, stops TCP servers, disconnects transports, stops sync manager, and closes the config watcher.

### 7.2 Connection Retry and Monitoring

Initial connect uses exponential backoff:

- base delay: 2s
- exponential factor: `2^attempt`
- max delay: 300s
- jitter: +/- 25%

After initial connect, a monitor checks every 30s:

- if a transport is disconnected -> reconnect with infinite retry/backoff
- if MQTT is disconnected -> reconnect with infinite retry/backoff

### 7.3 Inbound MySensors Message Pipeline

For each gateway transport, the adapter consumes `Transport.Receive()`:

For each message:

1. Log receipt.
2. If TCP replication is enabled for the gateway: broadcast the message to all TCP clients.
3. Pass message to the per-gateway handler (`Gateway.HandleMessage`) which:
   - tracks node as "seen"
   - updates last-seen timestamp (availability) for the node
   - if message is `INTERNAL`, handles selected internal subtypes:
     - `I_ID_REQUEST` -> assign ID and send `I_ID_RESPONSE`
     - `I_TIME` -> send current Unix time in seconds
     - `I_DISCOVER_RESPONSE` -> log only
4. Check whether the message satisfies any pending ACK wait (see 7.6).
5. Attempt to map the message to configured entities (see 7.4) and publish state.
6. If message is `PRESENTATION`, publish/accumulate presentation info (6.2).
7. Publish gateway seen_nodes status (6.2).

### 7.4 Entity Matching and State Publishing

Device message handling is only invoked for messages where:

- `message_type` is SET or REQ

Current observed behavior:

- REQ messages are logged but do not cause a response or state publish.
- SET messages may publish entity state.

Matching rules for SET messages:

1. For each configured device and entity:
   - skip entities that cannot report state (`write_only: true`)
2. Resolve effective node id:
   - entity.node_id if set, else device.node_id
3. Resolve child id: entity.child_id
4. If `(node_id, child_id)` matches `(message.NodeID, message.ChildID)` then:
   - compute expected MySensors variable type:
     - `variable_type` override if set and recognized
     - otherwise mapping based on `entity_type` (see 8.1)
   - publish only if `message.SubType == expectedVarType`
   - publish payload verbatim to `{topic_prefix}/entity/{unique_id}/state` (retained)

Multiple entities may map to the same `(node_id, child_id)`; all matching entities will be updated. The configuration validator only restricts this when multiple entities have `sync_period` enabled for the same target.

### 7.5 MQTT Command Handling (Outbound to MySensors)

For each command-capable entity, the MQTT client subscribes to:

`{topic_prefix}/entity/{unique_id}/set`

On a received command:

1. Validate payload based on entity type (loosely):
   - `switch`/`light`: accepts `0|1|ON|OFF`
   - `cover`: accepts `UP|DOWN|STOP|OPEN|CLOSE`
   - most sensor-like types accept any payload
2. If entity is in optimistic mode:
   - immediately publish the same payload to the state topic (retained)
   - store payload in the internal state cache
3. Invoke the registered application handler to send a MySensors SET message:
   - determine effective gateway (entity.gateway > device.gateway > default gateway heuristic)
   - determine effective node_id/child_id
   - determine MySensors variable type from entity type/override
   - send a SET message with payload verbatim
   - if `request_ack` is true, use the ACK wait/retry behavior (7.6)

### 7.6 ACK Request/Wait Semantics

When `request_ack` is true for an entity command:

- the adapter sends the MySensors SET frame with `ack` bit set to `1`.
- it then waits for any inbound SET message matching:
  - same `node_id`
  - same `child_id`
  - same `variable type` (subtype)

The ACK match does **not** currently verify:

- the MySensors `ack` flag on the response
- the payload value

Timeout/retry:

- wait `ack_timeout` (default 250ms)
- retry up to `ack_retries` times (default 3)
- total attempts = `ack_retries + 1`

New command cancellation:

- if a new command arrives for the same entity while an ACK wait is in progress, the previous wait is cancelled and replaced.

### 7.7 Entity Sync (Periodic State Push to Nodes)

For each entity where:

- entity can receive commands (`read_only` false)
- and `sync_period > 0`

The sync manager runs a periodic loop:

1. Optionally wait a random delay in `[0, sync_splay)` if `0 < sync_splay < sync_period`.
2. Look up last known entity state in the MQTT client’s internal cache (keyed by `{device.id}_{entity.id}_entity`).
3. If state exists:
   - send a MySensors SET message to the entity’s `(node_id, child_id)` with the cached payload
   - include ACK bit depending on `request_ack`
4. If no state exists, skip.

### 7.8 Node Availability Tracking and Publishing

Per gateway, the adapter tracks last-seen timestamps for nodes:

- last-seen is updated on **any** message received from a node (excluding node 0 and 255).

Every 10 seconds, the availability monitor:

1. Computes per-node availability for that gateway:
   - each node is `online` if `now - last_seen < availability_window`
2. `availability_window` is chosen as:
   - the minimum `entity.availability_window` across all entities that map to the node, if any are set and > 0
   - otherwise the gateway’s `gateway.availability_window` (default 30s)
3. For each device/entity that uses the current gateway and can either report state or receive commands:
   - publish `{topic_prefix}/entity/{unique_id}/availability` with payload `"online"` or `"offline"` (retained)

Important observed quirks to preserve:

- The runtime availability publisher always uses the default availability topic format and hard-coded payloads `"online"/"offline"`, regardless of per-entity `availability_topic` or custom payload settings used in discovery.
- Minimum-availability calculation does not filter by gateway when scanning entities for a node id; overlapping node IDs across gateways may interact.

### 7.9 Presentation Message Publishing

When a MySensors PRESENTATION message is received:

- publish/accumulate a textual presentation list under:
  `{topic_prefix}/gateway/{gateway}/node/{node_id}/presentation`
- each line is deduplicated; order is append-only by first-seen

## 8. Home Assistant MQTT Discovery

### 8.1 Entity Type -> MySensors Variable Type Mapping

Unless `entity.variable_type` overrides it, the adapter maps entity types to MySensors variable types as follows:

Actuator-ish:

- `switch`, `light` -> `V_STATUS`
- `dimmer`, `number` -> `V_PERCENTAGE`
- `cover` -> `V_UP` (commands may use `UP/DOWN/STOP` payloads)
- `text`, `select` -> `V_TEXT`
- `climate` -> `V_HVAC_SETPOINT_HEAT`
- `rgb_light` -> `V_RGB`
- `rgbw_light` -> `V_RGBW`

Sensor-ish:

- `binary_sensor` -> `V_STATUS`
- `sensor`, `custom` -> `V_CUSTOM`
- `temperature` -> `V_TEMP`
- `humidity` -> `V_HUM`
- `battery`, `percentage` -> `V_PERCENTAGE`
- `voltage` -> `V_VOLTAGE`
- `current` -> `V_CURRENT`
- `pressure` -> `V_PRESSURE`
- `level` -> `V_LEVEL`
- `weight` -> `V_WEIGHT`
- `distance` -> `V_DISTANCE`
- `light_level` -> `V_LIGHT_LEVEL`
- `watt` -> `V_WATT`
- `kwh` -> `V_KWH`
- `flow` -> `V_FLOW`
- `volume` -> `V_VOLUME`
- `ph` -> `V_PH`
- `orp` -> `V_ORP`
- `ec` -> `V_EC`
- `var` -> `V_VAR`
- `va` -> `V_VA`
- `power_factor` -> `V_POWER_FACTOR`
- `position` -> `V_POSITION`
- `uv` -> `V_UV`
- `rain` -> `V_RAIN`
- `rainrate` -> `V_RAINRATE`
- `wind` -> `V_WIND`
- `gust` -> `V_GUST`
- `direction` -> `V_DIRECTION`
- `impedance` -> `V_IMPEDANCE`

### 8.2 Entity Type -> Home Assistant Component Mapping

Discovery publishes to `homeassistant/{component}/{unique_id}/config` with `component` derived from `entity_type`:

- `switch` -> `switch`
- `light` -> `light`
- `dimmer` -> `light` (note: code uses `min_mireds`/`max_mireds` fields if min/max provided)
- `text` -> `text` unless read-only, then `sensor`
- `number` -> `number`
- `select` -> `select`
- `cover` -> `cover`
- `binary_sensor` -> `binary_sensor`
- any of the sensor-ish types -> `sensor`
- unknown -> `sensor`

### 8.3 Discovery Payload Shape

For every entity, the discovery JSON includes at minimum:

- `name`
- `unique_id`
- `state_topic` = `{topic_prefix}/entity/{unique_id}/state`
- `device` object:
  - `identifiers: [device.id]`
  - `name`, `manufacturer`, `model`, `sw_version`, `hw_version`
  - optional: `configuration_url`, `suggested_area`, `connections`, `via_device`
- `default_entity_id` (format: `{domain}.{name}`) if configured per 4.5
- `command_topic` for command-capable entities:
  `{topic_prefix}/entity/{unique_id}/set`

Common optional fields set if configured:

- `icon`, `device_class`, `entity_category`, `enabled_by_default`
- `qos` (default 0), `retain` (default true), `optimistic` (default false)
- template fields: `json_attributes_topic`, `json_attributes_template`, `value_template`, etc.

Availability fields:

- `availability_topic` is always set in discovery by current code:
  `{topic_prefix}/entity/{unique_id}/availability` unless overridden in config
- `payload_available` defaults to `"online"`
- `payload_not_available` defaults to `"offline"`

Per-type fields:

- switch/light/binary_sensor:
  - `payload_on` default `"1"`, `payload_off` default `"0"`
  - optional `state_on`, `state_off`
- cover:
  - `payload_open` default `"OPEN"`, `payload_close` default `"CLOSE"`, `payload_stop` default `"STOP"`
  - optional `state_open`, `state_closed`
- number:
  - `min`, `max`, `step`, `unit_of_measurement` if provided
- select:
  - `options` if provided
- sensor:
  - `unit_of_measurement`, `state_class`, `value_template` if provided

Initial state publishing on discovery:

- If an entity can report state and no retained state is known yet:
  - publishes `initial_value` to the state topic (retained)
  - except: if entity is read-only and not a `binary_sensor`, initial publish is skipped (wait for device data)

## 9. Build and Packaging

### 9.1 Go Build

- Module name: `ms-mqtt-adapter`
- Go version in `go.mod`: `go 1.24`
- Build target: `./cmd/ms-mqtt-adapter`

Typical commands:

- build: `go build -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter`
- tests: `go test ./...`

### 9.2 Home Assistant Add-on Container

- Docker multi-stage build:
  - build in `golang:1.24-alpine`
  - runtime base image provided via `BUILD_FROM` (Home Assistant base images)
- The container entrypoint runs `/run.sh`, which:
  - reads the add-on option `config_yaml`
  - writes it to `/config.yaml`
  - executes `/ms-mqtt-adapter -config /config.yaml`

### 9.3 CI / Publishing

GitHub Actions builds and publishes multi-arch images to GHCR and then creates a manifest list for `latest` and version tags.

## 10. Known Quirks / Limitations (Observed)

These are behaviors present in the current code and should be considered if the goal is a faithful reimplementation:

1. Default gateway selection:
   - "default gateway" is the first key returned by iterating over the `mysensors` map. Go map iteration order is not guaranteed; with multiple gateways, the default may be effectively arbitrary between runs.
2. Config validation vs defaults:
   - validation enforces `ethernet.port` only when `transport: ethernet` is explicitly set; omitting `transport` may allow port omission (later defaulted to 5003).
3. RS485 baud rate:
   - fixed at 9600 (no config knob), despite some docs/examples hinting at configurability.
4. Sensor read-only defaults:
   - only `sensor` and `binary_sensor` default to read-only; types like `temperature` default to writable unless `read_only: true` is set.
5. Availability config mismatch:
   - runtime publisher always writes to `{topic_prefix}/entity/{unique_id}/availability` with payloads `online/offline`, regardless of custom `availability_topic` or payload fields configured for discovery.
6. Retain/QoS behavior:
   - state and availability publishes are retained; QoS is always 0. Entity `retain`/`qos` config only affects the discovery JSON metadata, not actual publish behavior.
7. REQ message handling:
   - inbound MySensors REQ messages are not responded to.
8. ACK semantics:
   - "ACK received" is inferred by observing any SET message matching node/child/varType; ack bit and payload are not verified.
9. Live reload limitations:
   - MQTT subscriptions are refreshed on reload, but application-level command handlers are not re-registered for newly added entities.
   - gateway removal paths in reload logic are unreliable due to updating `app.config` early.
