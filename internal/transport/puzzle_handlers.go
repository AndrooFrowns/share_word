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
	clues, err := s.Service.GetFullClues(r.Context(), &p, cells)
	if err != nil {
		http.Error(w, "failed to load clues", http.StatusInternalServerError)
		return
	}

	var currentUser *db.User
	if currentUserID != "" {
		u, _ := s.Service.GetUserByID(r.Context(), currentUserID)
		currentUser = u
	}

	components.Layout(components.PuzzlePage(currentUser, &p, annotated, clues, mode), currentUser).Render(r.Context(), w)
}

func (s *Server) handleToggleBlock(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x, _ := strconv.ParseInt(chi.URLParam(r, "x"), 10, 64)
	y, _ := strconv.ParseInt(chi.URLParam(r, "y"), 10, 64)

	cells, err := s.Service.Queries.GetCells(r.Context(), puzzleID)
	if err != nil {
		http.Error(w, "failed to load cells", http.StatusInternalServerError)
		return
	}

	var targetCell *db.Cell
	for _, c := range cells {
		if c.X == x && c.Y == y {
			targetCell = &c
			break
		}
	}

	if targetCell != nil {
		err = s.Service.Queries.UpdateCell(r.Context(), db.UpdateCellParams{
			PuzzleID: puzzleID,
			X:        x,
			Y:        y,
			Char:     "",
			IsBlock:  !targetCell.IsBlock,
			IsPencil: false,
		})
		if err != nil {
			http.Error(w, "failed to update cell", http.StatusInternalServerError)
			return
		}
	}

	s.renderPuzzleUI(w, r, puzzleID, "edit", "", "")
}

func (s *Server) handleUpdateCell(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x, _ := strconv.ParseInt(chi.URLParam(r, "x"), 10, 64)
	y, _ := strconv.ParseInt(chi.URLParam(r, "y"), 10, 64)

	var payload struct {
		CellValue string `json:"cellValue"`
		Mode      string `json:"mode"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	err := s.Service.Queries.UpdateCell(r.Context(), db.UpdateCellParams{
		PuzzleID: puzzleID,
		X:        x,
		Y:        y,
		Char:     payload.CellValue,
		IsBlock:  false,
		IsPencil: false,
	})
	if err != nil {
		http.Error(w, "failed to update cell", http.StatusInternalServerError)
		return
	}

	s.renderPuzzleUI(w, r, puzzleID, payload.Mode, "", "")
}

func (s *Server) handleEditClue(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	direction := chi.URLParam(r, "direction")
	editingClueID := fmt.Sprintf("%d-%s", number, direction)

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
	clues, err := s.Service.GetFullClues(r.Context(), &p, cells)
	if err != nil {
		http.Error(w, "failed to load clues", http.StatusInternalServerError)
		return
	}

	clueText := ""
	for _, c := range clues {
		if c.Number == number && string(c.Direction) == direction {
			clueText = c.Text
			break
		}
	}

	s.renderPuzzleUI(w, r, puzzleID, "edit", editingClueID, clueText)
}

func (s *Server) handleViewClueItem(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	s.renderPuzzleUI(w, r, puzzleID, "edit", "", "")
}

func (s *Server) handleLiveUpdateClue(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	direction := chi.URLParam(r, "direction")

	var payload struct {
		ClueText string `json:"clueText"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_ = s.Service.Queries.UpsertClue(r.Context(), db.UpsertClueParams{
		PuzzleID:  puzzleID,
		Number:    int64(number),
		Direction: direction,
		Text:      payload.ClueText,
	})

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSaveClue(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	direction := chi.URLParam(r, "direction")

	var payload struct {
		ClueText string `json:"clueText"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	err := s.Service.Queries.UpsertClue(r.Context(), db.UpsertClueParams{
		PuzzleID:  puzzleID,
		Number:    int64(number),
		Direction: direction,
		Text:      payload.ClueText,
	})
	if err != nil {
		http.Error(w, "failed to save clue", http.StatusInternalServerError)
		return
	}

	s.renderPuzzleUI(w, r, puzzleID, "edit", "", "")
}

func (s *Server) renderPuzzleUI(w http.ResponseWriter, r *http.Request, puzzleID string, mode string, editingClueID string, clueText string) {
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
	clues, err := s.Service.GetFullClues(r.Context(), &p, cells)
	if err != nil {
		http.Error(w, "failed to load clues", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	datastar.NewSSE(w, r).PatchElementTempl(components.PuzzleUI(&p, annotated, clues, mode, editingClueID, clueText))
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