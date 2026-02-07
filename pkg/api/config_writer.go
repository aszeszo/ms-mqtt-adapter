package api

import (
	"fmt"
	"ms-mqtt-adapter/pkg/config"
	"os"
	"path/filepath"
	"sync"
)

// ConfigWriter provides thread-safe config validation and persistence.
type ConfigWriter struct {
	mu   sync.Mutex
	path string
}

func NewConfigWriter(path string) *ConfigWriter {
	return &ConfigWriter{path: path}
}

// Save validates, sets defaults, and atomically writes the config to disk.
func (cw *ConfigWriter) Save(cfg *config.Config) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if err := config.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	config.SetDefaults(cfg)

	return config.SaveConfig(cfg, cw.path)
}

// SaveRaw validates raw YAML by loading it, then writes the raw bytes atomically.
func (cw *ConfigWriter) SaveRaw(raw []byte) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	// Validate by loading
	if _, err := config.LoadConfigFromBytes(raw); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	return atomicWrite(cw.path, raw)
}

func (cw *ConfigWriter) Path() string {
	return cw.path
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
