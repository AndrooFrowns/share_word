package transport

import (
	"fmt"
	"net/http"
	"share_word/internal/db"
	"share_word/internal/web/components"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

func (s *Server) handleViewPuzzle(w http.ResponseWriter, r *http.Request) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	puzzleID := chi.URLParam(r, "id")
	mode := r.URL.Query().Get("mode")
	if mode != "edit" {
		mode = "solve"
	}

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

	components.Layout(components.PuzzlePage(currentUser, &p, annotated, mode), currentUser).Render(r.Context(), w)
}

func (s *Server) handleToggleBlock(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x, _ := strconv.ParseInt(chi.URLParam(r, "x"), 10, 64)
	y, _ := strconv.ParseInt(chi.URLParam(r, "y"), 10, 64)

	cells, _ := s.Service.Queries.GetCells(r.Context(), puzzleID)
	var targetCell *db.Cell
	for _, c := range cells {
		if c.X == x && c.Y == y {
			targetCell = &c
			break
		}
	}

	if targetCell != nil {
		_ = s.Service.Queries.UpdateCell(r.Context(), db.UpdateCellParams{
			PuzzleID: puzzleID,
			X:        x,
			Y:        y,
			Char:     "",
			IsBlock:  !targetCell.IsBlock,
			IsPencil: false,
		})
	}

	// We pass "edit" here because only Edit mode triggers this
	s.renderGridPatch(w, r, puzzleID, "edit")
}

func (s *Server) handleUpdateCell(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x, _ := strconv.ParseInt(chi.URLParam(r, "x"), 10, 64)
	y, _ := strconv.ParseInt(chi.URLParam(r, "y"), 10, 64)

	var payload struct {
		CellValue string `json:"cellValue"`
		Mode      string `json:"mode"`
	}
	_ = datastar.ReadSignals(r, &payload)

	_ = s.Service.Queries.UpdateCell(r.Context(), db.UpdateCellParams{
		PuzzleID: puzzleID,
		X:        x,
		Y:        y,
		Char:     payload.CellValue,
		IsBlock:  false,
		IsPencil: false,
	})

	s.renderGridPatch(w, r, puzzleID, payload.Mode)
}

func (s *Server) renderGridPatch(w http.ResponseWriter, r *http.Request, puzzleID string, mode string) {
	p, _ := s.Service.Queries.GetPuzzle(r.Context(), puzzleID)
	cells, _ := s.Service.Queries.GetCells(r.Context(), puzzleID)
	annotated := s.Service.CalculateNumbers(int(p.Width), int(p.Height), cells)

	w.WriteHeader(http.StatusOK)
	datastar.NewSSE(w, r).PatchElementTempl(components.Grid(&p, annotated, mode))
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