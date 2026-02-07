package api

import (
	"net/http"
)

func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	state := buildInitialState(s.provider)
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleGetEntityStates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.provider.GetEntityStates())
}

func (s *Server) handleGetGatewayStatus(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	gs := s.provider.GetGatewayStatus(name)
	if gs.SeenNodes == nil {
		writeError(w, http.StatusNotFound, "gateway not found")
		return
	}
	writeJSON(w, http.StatusOK, gatewayStatus{
		Connected:        gs.Connected,
		Transport:        gs.Transport,
		SeenNodes:        gs.SeenNodes,
		NodeAvailability: gs.NodeAvailability,
		LastSeenNodeID:   gs.LastSeenNodeID,
	})
}
