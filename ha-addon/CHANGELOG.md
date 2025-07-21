# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-07-21

### Added
- Initial release of MySensors MQTT Adapter Home Assistant add-on
- MySensors to MQTT bridge functionality
- Home Assistant auto-discovery support (optional)
- Support for relays with 1:1 mapping
- Support for inputs with many-to-many mapping
- State persistence using MQTT retained messages
- Immediate command processing via MQTT /set topics
- TCP service for MySensors message replication
- MySensors gateway functions (node ID assignment, time sync)
- Periodic state synchronization
- Configurable version request periods
- Multi-architecture Docker support (x86-64, ARM64, ARM32)
- Home Assistant add-on configuration
- Comprehensive logging with debug mode
- Composite key system for non-unique subdevice IDs
- 0/1 value format for MQTT payloads

### Features
- **Transport Support**: Ethernet gateway with RS485 stub
- **MQTT Topics**: Custom topic structure under configurable prefix
- **Device Mapping**: Flexible relay and input configuration
- **State Management**: Retained MQTT messages override config defaults
- **Real-time Control**: Immediate MySensors command sending
- **Discovery**: Automatic Home Assistant device registration
- **Monitoring**: Sorted seen_nodes topic for network visibility

### Technical Details
- Uses pre-built multi-architecture Docker images
- Supports all Home Assistant architectures
- Comprehensive configuration validation
- Thread-safe state management
- Graceful shutdown handling

### Development
- Entire project "vibe coded" using Claude Code AI assistant
- Production-ready architecture and patterns

[1.0.0]: https://github.com/aszeszo/ms-mqtt-adapter/releases/tag/v1.0.0