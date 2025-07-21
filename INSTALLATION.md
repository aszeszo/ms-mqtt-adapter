# Installation Instructions

## Home Assistant Add-on Repository Setup Complete ✅

This repository is now properly configured as a Home Assistant add-on repository using the pre-built Docker image `ghcr.io/aszeszo/ms-mqtt-adapter:latest`.

## Repository Structure

```
/ (repository root)
├── repository.yaml       # Repository metadata
├── logo.png              # Repository logo (256x128)
├── README.md             # Repository documentation
└── ha-addon/             # Add-on directory
    ├── config.yaml       # Add-on configuration (uses pre-built image)
    ├── DOCS.md           # Add-on documentation
    ├── CHANGELOG.md      # Version history
    ├── icon.png          # Add-on icon (128x128)
    └── README.md         # Add-on summary
```

## Installation Steps

1. **Ensure Docker Image is Built**: 
   - The GitHub Actions workflow should have built `ghcr.io/aszeszo/ms-mqtt-adapter:latest`
   - Verify the image exists and supports all architectures

2. **Add Repository to Home Assistant**:
   - Go to **Settings** → **Add-ons** → **Add-on Store**
   - Click **⋮** menu → **Repositories**
   - Add: `https://github.com/aszeszo/ms-mqtt-adapter`

3. **Install Add-on**:
   - Find "MySensors MQTT Adapter" in the store
   - Click **Install**
   - Configure and start

## Key Features

- ✅ Uses pre-built multi-architecture Docker image
- ✅ No local building required (fast installation)
- ✅ Supports all Home Assistant architectures
- ✅ Clean, minimal configuration
- ✅ Professional documentation

## Configuration

The add-on uses a simplified configuration structure:
- `mysensors_host` - MySensors gateway IP
- `mqtt_broker` - MQTT broker (core-mosquitto for HA built-in)
- `homeassistant_discovery` - Enable/disable auto-discovery

## Next Steps

1. Commit and push all changes to GitHub
2. Verify the Docker image is available at `ghcr.io/aszeszo/ms-mqtt-adapter:latest`
3. Test the repository in Home Assistant
4. The add-on should now appear and install successfully