package transport

import (
	"fmt"
	"net/http"
	"share_word/internal/db"
	"share_word/internal/web/components"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

func (s *Server) handleViewPuzzle(w http.ResponseWriter, r *http.Request) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	puzzleID := chi.URLParam(r, "id")

	p, err := s.Service.Queries.GetPuzzle(r.Context(), puzzleID)
	if err != nil {
		http.Error(w, "puzzle not found", http.StatusNotFound)
		return
	}

	cells, err := s.Service.Queries.GetCells(r.Context(), puzzleID)
	if err != nil {
		http.Error(w, "failed to load cells", http.StatusInternalServerError)
		return
	}

	annotated := s.Service.CalculateNumbers(int(p.Width), int(p.Height), cells)

	var currentUser *db.User
	if currentUserID != "" {
		u, _ := s.Service.GetUserByID(r.Context(), currentUserID)
		currentUser = u
	}

	components.Layout(components.PuzzlePage(currentUser, &p, annotated), currentUser).Render(r.Context(), w)
}

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
