#!/usr/bin/with-contenv bashio
# ==============================================================================
# Home Assistant Add-on: MySensors MQTT Adapter
# Runs the MySensors MQTT Adapter
# ==============================================================================

bashio::log.info "Starting MySensors MQTT Adapter..."

# Create config from Home Assistant add-on options
CONFIG_PATH="/data/options.json"
APP_CONFIG="/tmp/config.yaml"

# Check if config file exists
if bashio::fs.file_exists "$CONFIG_PATH"; then
    bashio::log.info "Creating configuration from add-on options..."
    
    # Extract configuration values
    LOG_LEVEL=$(bashio::config 'log_level')
    MYSENSORS_HOST=$(bashio::config 'mysensors_host')
    MYSENSORS_PORT=$(bashio::config 'mysensors_port')
    MQTT_BROKER=$(bashio::config 'mqtt_broker')
    MQTT_PORT=$(bashio::config 'mqtt_port')
    MQTT_USERNAME=$(bashio::config 'mqtt_username')
    MQTT_PASSWORD=$(bashio::config 'mqtt_password')
    TOPIC_PREFIX=$(bashio::config 'topic_prefix')
    HA_DISCOVERY=$(bashio::config 'homeassistant_discovery')
    
    # Create configuration file
    cat > "$APP_CONFIG" << EOF
log_level: "${LOG_LEVEL}"

mysensors:
  transport: "ethernet"
  ethernet:
    host: "${MYSENSORS_HOST}"
    port: ${MYSENSORS_PORT}

mqtt:
  broker: "${MQTT_BROKER}"
  port: ${MQTT_PORT}
  username: "${MQTT_USERNAME}"
  password: "${MQTT_PASSWORD}"
  client_id: "ms-mqtt-adapter"

tcp_service:
  enabled: true
  port: 5003

sync:
  enabled: true
  period: "30s"

gateway:
  node_id_range:
    start: 1
    end: 254
  version_request_period: "5s"

adapter:
  topic_prefix: "${TOPIC_PREFIX}"
  homeassistant_discovery: ${HA_DISCOVERY}

devices: []
EOF

    bashio::log.info "Configuration created successfully"
else
    bashio::log.warning "No configuration found, using default config"
    cp /usr/local/bin/config.example.yaml "$APP_CONFIG"
fi

# Run the application
bashio::log.info "Starting MySensors MQTT Adapter with config: $APP_CONFIG"
exec /usr/local/bin/ms-mqtt-adapter -config "$APP_CONFIG"