#!/usr/bin/with-contenv bashio

# ==============================================================================
# Home Assistant Add-on: MySensors MQTT Adapter
# Runs the MySensors MQTT Adapter
# ==============================================================================

bashio::log.info "Starting MySensors MQTT Adapter..."

# Extract config_yaml from addon options and save as /config.yaml
if bashio::config.has_value 'config_yaml'; then
    bashio::log.info "Extracting config_yaml from addon options"
    CONFIG_YAML=$(bashio::config 'config_yaml')
    echo "${CONFIG_YAML}" > /config.yaml
    bashio::log.info "Configuration saved to /config.yaml"
else
    bashio::log.error "No config_yaml provided in addon options"
    exit 1
fi

# Start the MySensors MQTT Adapter
bashio::log.info "Starting ms-mqtt-adapter with config: /config.yaml"
exec /ms-mqtt-adapter -config /config.yaml