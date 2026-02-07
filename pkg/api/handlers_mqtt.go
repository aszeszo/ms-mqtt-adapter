package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleGetMQTTTopics(w http.ResponseWriter, r *http.Request) {
	mqttClient := s.provider.GetMQTTClient()
	if mqttClient == nil {
		writeError(w, http.StatusServiceUnavailable, "MQTT client not available")
		return
	}

	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	topics := mqttClient.GetTopicList(cfg.MySensors)
	writeJSON(w, http.StatusOK, topics)
}

func (s *Server) handleDeleteMQTTTopics(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope    string `json:"scope"`
		DeviceID string `json:"device_id,omitempty"`
		EntityID string `json:"entity_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Scope == "" {
		writeError(w, http.StatusBadRequest, "scope is required")
		return
	}

	mqttClient := s.provider.GetMQTTClient()
	if mqttClient == nil {
		writeError(w, http.StatusServiceUnavailable, "MQTT client not available")
		return
	}

	cfg, err := s.currentConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := mqttClient.DeleteTopics(req.Scope, req.DeviceID, req.EntityID, cfg.MySensors); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
