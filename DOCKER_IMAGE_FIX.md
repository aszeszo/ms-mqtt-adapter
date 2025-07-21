# Docker Image Format Fix for Home Assistant

## Issue Identified ❌

Home Assistant validation error:
```
Can't read /data/addons/git/3036bad2/ha-addon/config.yaml: does not match regular expression ^([a-z0-9][a-z0-9.\-]*(:[0-9]+)?/)*?([a-z0-9{][a-z0-9.\-_{}]*/)*?([a-z0-9{][a-z0-9.\-_{}]*)$ for dictionary value @ data['image']. Got 'ghcr.io/aszeszo/ms-mqtt-adapter:latest'
```

## Root Cause
Home Assistant has very strict validation rules for Docker image names in add-on configurations. The image name format was not compatible with their regex pattern.

## Solution Applied ✅

### Option 1: Removed Direct Image Reference
- Removed `image:` field from `config.yaml`
- Created local `Dockerfile` that builds using Home Assistant's standard approach

### Option 2: Hybrid Approach - Best of Both Worlds
Created a `Dockerfile` that:

1. **Uses HA Base Images**: Starts with official HA base images
2. **Downloads Pre-built Binary**: Tries to download from GitHub releases first
3. **Fallback to Source Build**: If binary not available, builds from source
4. **Multi-Architecture Support**: Handles different CPU architectures automatically

## Final Structure ✅

```
ha-addon/
├── config.yaml      # ✅ No image field, uses local build
├── build.yaml       # ✅ HA base images configuration
├── Dockerfile       # ✅ Smart build (pre-built binary or source)
├── run.sh           # ✅ Home Assistant integration script
├── DOCS.md          # ✅ User documentation
├── icon.png         # ✅ Add-on icon
└── ...
```

## Benefits

1. **✅ HA Compatible**: Follows Home Assistant's validation rules
2. **✅ Fast Installation**: Uses pre-built binaries when available  
3. **✅ Reliable Fallback**: Builds from source if binaries unavailable
4. **✅ Multi-Architecture**: Supports all HA platforms
5. **✅ Professional Integration**: Uses bashio and HA standards

## Technical Details

The Dockerfile intelligently:
- Detects the target architecture (`x86_64`, `aarch64`, `armv7l`)
- Maps to GitHub release binary names (`amd64`, `arm64`, `arm`)
- Downloads appropriate pre-built binary
- Falls back to Go source compilation if needed
- Installs bashio for Home Assistant integration

This approach gives us the benefits of pre-built images while maintaining full Home Assistant compatibility.