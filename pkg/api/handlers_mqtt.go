package api

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleGetMQTTTopics(w http.ResponseWriter, r *http.Request) {
	mqttClient := s.provider.GetMQTTClient()
	if mqttClient == nil {
		writeError(w, http.StatusServiceUnavailable, "MQTT client not available")
		return
	}

	topics, err := mqttClient.BrowseAllTopics(3 * time.Second)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, topics)
}

func (s *Server) handleDeleteMQTTTopics(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topics []string `json:"topics,omitempty"`
		Prefix string   `json:"prefix,omitempty"`
		// Legacy fields for backward compatibility
		Scope    string `json:"scope,omitempty"`
		DeviceID string `json:"device_id,omitempty"`
		EntityID string `json:"entity_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	mqttClient := s.provider.GetMQTTClient()
	if mqttClient == nil {
		writeError(w, http.StatusServiceUnavailable, "MQTT client not available")
		return
	}

	// New API: delete specific topics
	if len(req.Topics) > 0 {
		for _, topic := range req.Topics {
			if err := mqttClient.DeleteRetainedTopic(topic); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": len(req.Topics)})
		return
	}

	// New API: delete topic tree by prefix
	if req.Prefix != "" {
		count, err := mqttClient.DeleteRetainedTree(req.Prefix, 3*time.Second)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": count})
		return
	}

	// Legacy API: delete by scope
	if req.Scope != "" {
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
		return
	}

	writeError(w, http.StatusBadRequest, "one of 'topics', 'prefix', or 'scope' is required")
}
