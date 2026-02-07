package api

import (
	"encoding/json"
	"io"
	"ms-mqtt-adapter/pkg/config"
	"net/http"
	"os"
)

// --- Full config ---

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := s.saveConfig(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Log level ---

func (s *Server) handleGetLogLevel(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"log_level": cfg.LogLevel})
}

func (s *Server) handlePutLogLevel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LogLevel string `json:"log_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.LogLevel = body.LogLevel
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- MQTT ---

func (s *Server) handleGetMQTT(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg.MQTT)
}

func (s *Server) handlePutMQTT(w http.ResponseWriter, r *http.Request) {
	var mqtt config.MQTTConfig
	if err := json.NewDecoder(r.Body).Decode(&mqtt); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.MQTT = mqtt
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Adapter ---

func (s *Server) handleGetAdapter(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg.AdapterTopics)
}

func (s *Server) handlePutAdapter(w http.ResponseWriter, r *http.Request) {
	var adapter config.AdapterConfig
	if err := json.NewDecoder(r.Body).Decode(&adapter); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.AdapterTopics = adapter
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- ID Aliases ---

func (s *Server) handleGetAliases(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	aliases := cfg.IDaliases
	if aliases == nil {
		aliases = make(map[string]int)
	}
	writeJSON(w, http.StatusOK, aliases)
}

func (s *Server) handlePutAliases(w http.ResponseWriter, r *http.Request) {
	var aliases map[string]int
	if err := json.NewDecoder(r.Body).Decode(&aliases); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.IDaliases = aliases
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- MySensors gateways ---

func (s *Server) handleGetMySensors(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg.MySensors)
}

func (s *Server) handlePutMySensors(w http.ResponseWriter, r *http.Request) {
	var gateways map[string]config.MySensorsConfig
	if err := json.NewDecoder(r.Body).Decode(&gateways); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.MySensors = gateways
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetGatewayConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	gw, exists := cfg.MySensors[name]
	if !exists {
		writeError(w, http.StatusNotFound, "gateway not found")
		return
	}
	writeJSON(w, http.StatusOK, gw)
}

func (s *Server) handlePutGatewayConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var gw config.MySensorsConfig
	if err := json.NewDecoder(r.Body).Decode(&gw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg.MySensors == nil {
		cfg.MySensors = make(map[string]config.MySensorsConfig)
	}
	cfg.MySensors[name] = gw
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteGatewayConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, exists := cfg.MySensors[name]; !exists {
		writeError(w, http.StatusNotFound, "gateway not found")
		return
	}
	delete(cfg.MySensors, name)
	// Remove devices associated with this gateway
	filtered := cfg.Devices[:0]
	for _, d := range cfg.Devices {
		if d.Gateway != name {
			filtered = append(filtered, d)
		}
	}
	cfg.Devices = filtered
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Devices ---

func (s *Server) handleGetDevices(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg.Devices)
}

func (s *Server) handlePutDevices(w http.ResponseWriter, r *http.Request) {
	var devices []config.Device
	if err := json.NewDecoder(r.Body).Decode(&devices); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.Devices = devices
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePostDevice(w http.ResponseWriter, r *http.Request) {
	var dev config.Device
	if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dev.ID == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}
	// Check for duplicate ID
	for _, d := range cfg.Devices {
		if d.ID == dev.ID {
			writeError(w, http.StatusConflict, "device with this ID already exists")
			return
		}
	}
	cfg.Devices = append(cfg.Devices, dev)
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) findDeviceIndex(cfg *config.Config, id string) int {
	for i, d := range cfg.Devices {
		if d.ID == id {
			return i
		}
	}
	return -1
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := s.findDeviceIndex(cfg, id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	writeJSON(w, http.StatusOK, cfg.Devices[idx])
}

func (s *Server) handlePutDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var dev config.Device
	if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := s.findDeviceIndex(cfg, id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	dev.ID = id // Ensure ID stays consistent
	cfg.Devices[idx] = dev
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := s.findDeviceIndex(cfg, id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	cfg.Devices = append(cfg.Devices[:idx], cfg.Devices[idx+1:]...)
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Entities (nested under devices) ---

func (s *Server) handleGetEntities(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := s.findDeviceIndex(cfg, id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	writeJSON(w, http.StatusOK, cfg.Devices[idx].Entities)
}

func (s *Server) handlePostEntity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var entity config.Entity
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := s.findDeviceIndex(cfg, id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	// Check for duplicate entity ID
	for _, e := range cfg.Devices[idx].Entities {
		if e.ID == entity.ID {
			writeError(w, http.StatusConflict, "entity with this ID already exists")
			return
		}
	}
	cfg.Devices[idx].Entities = append(cfg.Devices[idx].Entities, entity)
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) findEntityIndex(dev *config.Device, eid string) int {
	for i, e := range dev.Entities {
		if e.ID == eid {
			return i
		}
	}
	return -1
}

func (s *Server) handleGetEntity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	eid := r.PathValue("eid")
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dIdx := s.findDeviceIndex(cfg, id)
	if dIdx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	eIdx := s.findEntityIndex(&cfg.Devices[dIdx], eid)
	if eIdx == -1 {
		writeError(w, http.StatusNotFound, "entity not found")
		return
	}
	writeJSON(w, http.StatusOK, cfg.Devices[dIdx].Entities[eIdx])
}

func (s *Server) handlePutEntity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	eid := r.PathValue("eid")
	var entity config.Entity
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dIdx := s.findDeviceIndex(cfg, id)
	if dIdx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	eIdx := s.findEntityIndex(&cfg.Devices[dIdx], eid)
	if eIdx == -1 {
		writeError(w, http.StatusNotFound, "entity not found")
		return
	}
	entity.ID = eid
	cfg.Devices[dIdx].Entities[eIdx] = entity
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteEntity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	eid := r.PathValue("eid")
	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dIdx := s.findDeviceIndex(cfg, id)
	if dIdx == -1 {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	eIdx := s.findEntityIndex(&cfg.Devices[dIdx], eid)
	if eIdx == -1 {
		writeError(w, http.StatusNotFound, "entity not found")
		return
	}
	entities := cfg.Devices[dIdx].Entities
	cfg.Devices[dIdx].Entities = append(entities[:eIdx], entities[eIdx+1:]...)
	if err := s.saveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Validate / Raw YAML ---

func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" || contentType == "" {
		// Validate as JSON config struct
		var cfg config.Config
		if err := json.Unmarshal(body, &cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if err := config.ValidateConfig(&cfg); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"valid": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"valid": true})
		return
	}

	// Validate as raw YAML
	if _, err := config.LoadConfigFromBytes(body); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true})
}

func (s *Server) handleGetRawConfig(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(s.configWriter.Path())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Write(data)
}

func (s *Server) handlePutRawConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := s.configWriter.SaveRaw(body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
