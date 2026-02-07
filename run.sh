#!/usr/bin/with-contenv bashio

# ==============================================================================
# Home Assistant Add-on: MySensors MQTT Adapter
# Runs the MySensors MQTT Adapter
# ==============================================================================

bashio::log.info "Starting MySensors MQTT Adapter..."

CONFIG_PATH="/data/config.yaml"

# Seed config from addon options only if persistent config does not exist
if [ ! -f "${CONFIG_PATH}" ]; then
    if bashio::config.has_value 'config_yaml'; then
        bashio::log.info "Seeding initial config from addon options to ${CONFIG_PATH}"
        CONFIG_YAML=$(bashio::config 'config_yaml')
        echo "${CONFIG_YAML}" > "${CONFIG_PATH}"
    else
        bashio::log.error "No config_yaml provided in addon options and no existing config found"
        exit 1
    fi
else
    bashio::log.info "Using existing persistent config at ${CONFIG_PATH}"
fi

# Start the MySensors MQTT Adapter with ingress port
bashio::log.info "Starting ms-mqtt-adapter with config: ${CONFIG_PATH}"
exec /ms-mqtt-adapter -config "${CONFIG_PATH}" -ingress-port 8099
