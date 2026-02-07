#!/usr/bin/with-contenv bashio

# ==============================================================================
# Home Assistant Add-on: MySensors MQTT Adapter
# Runs the MySensors MQTT Adapter
# ==============================================================================

bashio::log.info "Starting MySensors MQTT Adapter..."

CONFIG_PATH="/data/config.yaml"

# Create minimal config if it doesn't exist
if [ ! -f "${CONFIG_PATH}" ]; then
    bashio::log.info "Creating minimal config at ${CONFIG_PATH}"
    bashio::log.info "Configure the adapter via the web UI (Ingress)"
    cat > "${CONFIG_PATH}" <<EOF
# MySensors MQTT Adapter Configuration
# Configure via the web UI - all options are managed through the UI

log_level: info

mysensors: {}

mqtt:
  broker: ""
  port: 1883
  client_id: "ms-mqtt-adapter"

adapter:
  topic_prefix: "ms-mqtt-adapter"
  discovery_prefix: "homeassistant"
  homeassistant_discovery: true

id_aliases:
  relay3_ch0: 0
  relay3_ch1: 1
  relay3_ch2: 2

devices: []
EOF
else
    bashio::log.info "Using existing persistent config at ${CONFIG_PATH}"
fi

# Start the MySensors MQTT Adapter with ingress port
bashio::log.info "Starting ms-mqtt-adapter with config: ${CONFIG_PATH}"
bashio::log.info "Configure via Ingress UI or API endpoints"
exec /ms-mqtt-adapter -config "${CONFIG_PATH}" -ingress-port 8099
