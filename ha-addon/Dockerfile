ARG BUILD_FROM
FROM ${BUILD_FROM}

# Install runtime dependencies
RUN apk add --no-cache \
    bash \
    curl \
    jq

# Install bashio
RUN curl -J -L -o /tmp/bashio.tar.gz \
    "https://github.com/hassio-addons/bashio/archive/v0.16.2.tar.gz" \
    && mkdir /tmp/bashio \
    && tar zxvf /tmp/bashio.tar.gz --strip 1 -C /tmp/bashio \
    && mv /tmp/bashio/lib /usr/lib/bashio \
    && ln -s /usr/lib/bashio/bashio /usr/bin/bashio \
    && rm -fr /tmp/bashio.tar.gz /tmp/bashio

# Download the pre-built application
RUN ARCH=$(uname -m) \
    && if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi \
    && if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi \
    && if [ "$ARCH" = "armv7l" ]; then ARCH="arm"; fi \
    && echo "Downloading for architecture: $ARCH" \
    && curl -L -o /tmp/ms-mqtt-adapter \
    "https://github.com/aszeszo/ms-mqtt-adapter/releases/latest/download/ms-mqtt-adapter-linux-$ARCH" \
    || echo "Binary not found, will build from source"

# If binary download failed, build from source
RUN if [ ! -f /tmp/ms-mqtt-adapter ]; then \
    echo "Building from source..." \
    && apk add --no-cache go git \
    && git clone https://github.com/aszeszo/ms-mqtt-adapter.git /tmp/src \
    && cd /tmp/src \
    && go build -o /tmp/ms-mqtt-adapter ./cmd/ms-mqtt-adapter \
    && apk del go git; \
    fi

# Install the application
RUN mv /tmp/ms-mqtt-adapter /usr/local/bin/ms-mqtt-adapter \
    && chmod +x /usr/local/bin/ms-mqtt-adapter

# Copy configuration and run script
COPY run.sh /
RUN chmod a+x /run.sh

CMD ["/run.sh"]