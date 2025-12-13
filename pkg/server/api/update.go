package api

import (
	"encoding/json"
	"net/http"

	"github.com/Yu-Jack/sim-gui/pkg/updater"
)

func (s *Server) handleGetUpdateStatus(w http.ResponseWriter, r *http.Request) {
	// If updater is not initialized, return disabled status
	if s.updater == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updater.UpdateStatus{
			UpdateAvailable: false,
			Message:         "Update checking is disabled",
		})
		return
	}

	status := s.updater.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
