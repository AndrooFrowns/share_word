package transport

import (
	"fmt"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

func (s *Server) handleCreatePuzzle(w http.ResponseWriter, r *http.Request) {
	userID := s.SessionManager.GetString(r.Context(), "userID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload struct {
		Name   string `json:"name"`
		Width  int64  `json:"width"`
		Height int64  `json:"height"`
	}

	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	p, err := s.Service.CreatePuzzle(r.Context(), payload.Name, userID, payload.Width, payload.Height)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	datastar.NewSSE(w, r).Redirect(fmt.Sprintf("/puzzles/%s", p.ID))
}
