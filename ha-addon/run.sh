#!/usr/bin/with-contenv bashio

# ==============================================================================
# Home Assistant Add-on: MySensors MQTT Adapter
# Runs the MySensors MQTT Adapter
# ==============================================================================

bashio::log.info "Starting MySensors MQTT Adapter..."

# Create configuration directory
mkdir -p /config/ms-mqtt-adapter

# Generate configuration file from addon options
CONFIG_FILE="/config/ms-mqtt-adapter/config.yaml"

bashio::log.info "Generating configuration file: ${CONFIG_FILE}"

# Get log level from addon options
LOG_LEVEL=$(bashio::config 'log_level')

# Create basic configuration
cat > "${CONFIG_FILE}" << EOF
log_level: "${LOG_LEVEL}"

mysensors:
  default:
    transport: "ethernet"
    ethernet:
      host: "127.0.0.1"
      port: 5003
    gateway:
      node_id_range:
        start: 1
        end: 254
      version_request_period: "5s"
    tcp_service:
      enabled: true
      port: 5003

mqtt:
  broker: "core-mosquitto"
  port: 1883
  username: ""
  password: ""
  client_id: "ms-mqtt-adapter"

sync:
  enabled: true
  period: "30s"

adapter:
  topic_prefix: "ms-mqtt-adapter"
  homeassistant_discovery: true
  optimistic_mode: false
  request_ack: true

devices: []
EOF

# Create devices configuration if it doesn't exist
DEVICES_FILE="/config/ms-mqtt-adapter/devices.yaml"
if ! bashio::fs.file_exists "${DEVICES_FILE}"; then
    bashio::log.info "Creating example devices configuration: ${DEVICES_FILE}"
    cat > "${DEVICES_FILE}" << EOF
# MySensors Device Configuration
# Add your MySensors devices here

devices:
  - name: "Example Relay Controller"
    id: "example_relay"
    node_id: 1
    gateway: "default"
    manufacturer: "MySensors"
    model: "Relay Controller"
    sw_version: "1.0"
    hw_version: "1.0"
    relays:
      - name: "Light Switch"
        id: "light_switch"
        child_id: 0
        initial_state: 0
        optimistic: false
        icon: "mdi:lightbulb"
        device_class: "switch"

# Remove or comment out the example above and add your actual devices
EOF
fi

# Display configuration information
bashio::log.info "Configuration file: ${CONFIG_FILE}"
bashio::log.info "Devices file: ${DEVICES_FILE}"
bashio::log.info "Edit these files to configure your MySensors devices"

# Start the MySensors MQTT Adapter
bashio::log.info "Starting ms-mqtt-adapter..."
exec /ms-mqtt-adapter --config "${CONFIG_FILE}"