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
    MYSENSORS_TRANSPORT=$(bashio::config 'mysensors.transport')
    MYSENSORS_HOST=$(bashio::config 'mysensors.ethernet.host')
    MYSENSORS_PORT=$(bashio::config 'mysensors.ethernet.port')
    MQTT_BROKER=$(bashio::config 'mqtt.broker')
    MQTT_PORT=$(bashio::config 'mqtt.port')
    MQTT_USERNAME=$(bashio::config 'mqtt.username')
    MQTT_PASSWORD=$(bashio::config 'mqtt.password')
    MQTT_CLIENT_ID=$(bashio::config 'mqtt.client_id')
    TCP_ENABLED=$(bashio::config 'tcp_service.enabled')
    TCP_PORT=$(bashio::config 'tcp_service.port')
    SYNC_ENABLED=$(bashio::config 'sync.enabled')
    SYNC_PERIOD=$(bashio::config 'sync.period')
    TOPIC_PREFIX=$(bashio::config 'adapter.topic_prefix')
    HA_DISCOVERY=$(bashio::config 'adapter.homeassistant_discovery')
    VERSION_REQUEST_PERIOD=$(bashio::config 'gateway.version_request_period')
    
    # Create configuration file
    cat > "$APP_CONFIG" << EOF
log_level: "${LOG_LEVEL}"

mysensors:
  transport: "${MYSENSORS_TRANSPORT}"
  ethernet:
    host: "${MYSENSORS_HOST}"
    port: ${MYSENSORS_PORT}

mqtt:
  broker: "${MQTT_BROKER}"
  port: ${MQTT_PORT}
  username: "${MQTT_USERNAME}"
  password: "${MQTT_PASSWORD}"
  client_id: "${MQTT_CLIENT_ID}"

tcp_service:
  enabled: ${TCP_ENABLED}
  port: ${TCP_PORT}

sync:
  enabled: ${SYNC_ENABLED}
  period: "${SYNC_PERIOD}"

gateway:
  node_id_range:
    start: 1
    end: 254
  version_request_period: "${VERSION_REQUEST_PERIOD}"

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