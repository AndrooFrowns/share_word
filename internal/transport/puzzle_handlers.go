package transport

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"share_word/internal/app"
	"share_word/internal/db"
	"share_word/internal/web/components"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/starfederation/datastar-go/datastar"
)

func (s *Server) handleViewPuzzleSolve(w http.ResponseWriter, r *http.Request) {
	s.viewPuzzle(w, r, "solve")
}

func (s *Server) handleViewPuzzleEdit(w http.ResponseWriter, r *http.Request) {
	s.viewPuzzle(w, r, "edit")
}

func (s *Server) viewPuzzle(w http.ResponseWriter, r *http.Request, mode string) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	puzzleID := chi.URLParam(r, "id")

	// Get editing ID from registry
	token := s.SessionManager.Token(r.Context())
	editingClueID := ""
	if val, ok := s.Service.EditingClues.Load(token); ok {
		editingClueID = val.(string)
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

	components.Layout(components.PuzzlePage(currentUser, &p, annotated, clues, mode, s.Service.StartTime, editingClueID), currentUser).Render(r.Context(), w)
}

func (s *Server) handleSetBlock(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x, _ := strconv.ParseInt(chi.URLParam(r, "x"), 10, 64)
	y, _ := strconv.ParseInt(chi.URLParam(r, "y"), 10, 64)

	var payload struct {
		IsBlock bool `json:"isBlock"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	_ = s.Service.Queries.UpdateCell(r.Context(), db.UpdateCellParams{
		PuzzleID: puzzleID,
		X:        x,
		Y:        y,
		Char:     "",
		IsBlock:  payload.IsBlock,
		IsPencil: false,
	})

	s.Service.BroadcastUpdate(puzzleID, true)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSetBlockState(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x, _ := strconv.ParseInt(chi.URLParam(r, "x"), 10, 64)
	y, _ := strconv.ParseInt(chi.URLParam(r, "y"), 10, 64)
	stateStr := chi.URLParam(r, "state")

	var payload struct {
		SymmetryMode string `json:"symmetryMode"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		// Just ignore signal errors here, default to no symmetry if missing
		// or maybe log it? For now, we'll proceed with empty mode if fails.
	}

	isBlock := stateStr == "true"

	p, err := s.Service.Queries.GetPuzzle(r.Context(), puzzleID)
	if err != nil {
		http.Error(w, "puzzle not found", http.StatusNotFound)
		return
	}

	points := app.GetSymmetricCells(x, y, p.Width, p.Height, payload.SymmetryMode)

	for _, pt := range points {
		_ = s.Service.Queries.UpdateCell(r.Context(), db.UpdateCellParams{
			PuzzleID: puzzleID,
			X:        pt.X,
			Y:        pt.Y,
			Char:     "",
			Solution: "",
			IsBlock:  isBlock,
			IsPencil: false,
		})
	}

	s.Service.BroadcastUpdate(puzzleID, true)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUpdateCell(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x, _ := strconv.ParseInt(chi.URLParam(r, "x"), 10, 64)
	y, _ := strconv.ParseInt(chi.URLParam(r, "y"), 10, 64)

	var payload struct {
		CellValue string `json:"cellValue"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	char := payload.CellValue
	if len(char) > 0 {
		char = strings.ToUpper(char[len(char)-1:])
	}

	_ = s.Service.Queries.UpdateCell(r.Context(), db.UpdateCellParams{
		PuzzleID: puzzleID,
		X:        x,
		Y:        y,
		Char:     char,
		IsBlock:  false,
		IsPencil: false,
	})

	s.Service.BroadcastUpdate(puzzleID, false)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleFocusCell(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	x := chi.URLParam(r, "x")
	y := chi.URLParam(r, "y")

	var payload struct {
		ClientID string `json:"clientID"`
	}
	_ = datastar.ReadSignals(r, &payload)

	token := s.SessionManager.Token(r.Context())
	s.Service.FocusedCells.Store(token+":"+payload.ClientID, fmt.Sprintf("%s,%s", x, y))
	s.Service.BroadcastUpdate(puzzleID, false)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSaveClue(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	direction := chi.URLParam(r, "direction")

	var payload struct {
		Text     string `json:"clueText"`
		ClientID string `json:"clientID"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		log.Printf("Error reading signals in handleSaveClue: %v", err)
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	log.Printf("Saving clue: Puzzle=%s, Clue=%d-%s, Text=%q, ClientID=%s", puzzleID, number, direction, payload.Text, payload.ClientID)

	err := s.Service.Queries.UpsertClue(r.Context(), db.UpsertClueParams{
		PuzzleID:  puzzleID,
		Number:    int64(number),
		Direction: direction,
		Text:      payload.Text,
	})
	if err != nil {
		log.Printf("Failed to upsert clue: %v", err)
		http.Error(w, "failed to save clue", http.StatusInternalServerError)
		return
	}

	token := s.SessionManager.Token(r.Context())
	s.Service.EditingClues.Delete(token + ":" + payload.ClientID)
	s.Service.BroadcastUpdate(puzzleID, false)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleEditClue(w http.ResponseWriter, r *http.Request) {
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	direction := chi.URLParam(r, "direction")
	puzzleID := chi.URLParam(r, "id")
	clueID := fmt.Sprintf("%d-%s", number, direction)

	var payload struct {
		ClientID string `json:"clientID"`
	}
	_ = datastar.ReadSignals(r, &payload)

	token := s.SessionManager.Token(r.Context())
	s.Service.EditingClues.Store(token+":"+payload.ClientID, clueID)
	s.Service.BroadcastUpdate(puzzleID, false)
	w.WriteHeader(http.StatusOK)
}
func (s *Server) handlePuzzleStreamSolve(w http.ResponseWriter, r *http.Request) {
	s.handlePuzzleStream(w, r, "solve")
}

func (s *Server) handlePuzzleStreamEdit(w http.ResponseWriter, r *http.Request) {
	s.handlePuzzleStream(w, r, "edit")
}

func (s *Server) handlePuzzleStream(w http.ResponseWriter, r *http.Request, mode string) {
	puzzleID := chi.URLParam(r, "id")
	clientID := r.URL.Query().Get("clientID")
	subject := fmt.Sprintf("puzzles.%s", puzzleID)

	log.Printf("SSE: Client connecting to %s (mode: %s, clientID: %s, remote: %s)", subject, mode, clientID, r.RemoteAddr)

	if s.Service.NC == nil {
		log.Printf("SSE: NATS connection is nil, rejecting client")
		http.Error(w, "realtime service unavailable", http.StatusServiceUnavailable)
		return
	}

	notify := make(chan struct{}, 1)
	sub, err := s.Service.NC.Subscribe(subject, func(msg *nats.Msg) {
		select {
		case notify <- struct{}{}:
		default:
		}
	})
	if err != nil {
		http.Error(w, "failed to subscribe", http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()

	sse := datastar.NewSSE(w, r)

	// Push initial state immediately
	s.pushPuzzleState(r.Context(), sse, puzzleID, mode, clientID)

	// Hot reload check
	if components.EnableHotReload {
		var payload struct {
			ServerVersion int64 `json:"serverVersion"`
		}
		if err := datastar.ReadSignals(r, &payload); err == nil {
			if payload.ServerVersion != s.Service.StartTime {
				fmt.Printf("Hot reload triggered: client %d != server %d\n", payload.ServerVersion, s.Service.StartTime)
				sse.ExecuteScript("window.location.reload()")
				time.Sleep(100 * time.Millisecond) // Give client time to receive the event
				return
			}
		}
	}

	for {
		select {
		case <-r.Context().Done():
			log.Printf("SSE: Client disconnected from %s (clientID: %s)", subject, clientID)
			return
		case <-notify:
			log.Printf("SSE: Pushing update for %s (clientID: %s)", subject, clientID)
			s.pushPuzzleState(r.Context(), sse, puzzleID, mode, clientID)
		}
	}
}

func (s *Server) pushPuzzleState(ctx context.Context, sse *datastar.ServerSentEventGenerator, puzzleID string, mode string, clientID string) {
	p, err := s.Service.Queries.GetPuzzle(ctx, puzzleID)
	if err != nil {
		fmt.Printf("Error getting puzzle in push: %v\n", err)
		return
	}

	cells, err := s.Service.Queries.GetCells(ctx, puzzleID)
	if err != nil {
		return
	}

	token := s.SessionManager.Token(ctx)
	editingClueID := ""
	if val, ok := s.Service.EditingClues.Load(token + ":" + clientID); ok {
		editingClueID = val.(string)
	}

	focusedCell := ""
	if val, ok := s.Service.FocusedCells.Load(token + ":" + clientID); ok {
		focusedCell = val.(string)
	}

	annotated := s.Service.CalculateNumbers(int(p.Width), int(p.Height), cells)
	clues, _ := s.Service.GetFullClues(ctx, &p, cells)

	sse.PatchElementTempl(components.PuzzleUI(&p, annotated, clues, mode, editingClueID, focusedCell))
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

func (s *Server) handleResizePuzzle(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")

	var payload struct {
		Width  int64 `json:"width"`
		Height int64 `json:"height"`
	}
	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	fmt.Printf("Resizing puzzle %s to %dx%d\n", puzzleID, payload.Width, payload.Height)

	err := s.Service.ResizePuzzle(r.Context(), puzzleID, payload.Width, payload.Height)
	if err != nil {
		fmt.Printf("Resize error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.Service.BroadcastUpdate(puzzleID, true)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleImportPuzzle(w http.ResponseWriter, r *http.Request) {
	puzzleID := chi.URLParam(r, "id")

	var payload struct {
		ImportedFiles []struct {
			Name     string `json:"name"`
			Contents string `json:"contents"` // Base64 encoded
			Mime     string `json:"mime"`
		} `json:"importedFiles"`
	}

	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	if len(payload.ImportedFiles) == 0 {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}

	file := payload.ImportedFiles[0]
	fmt.Printf("Importing file: %s (Mime: %s, Len: %d)\n", file.Name, file.Mime, len(file.Contents))

	// We need to strip the prefix if present.
	dataURI := file.Contents
	base64Data := dataURI
	if len(dataURI) > 0 {
		for i, c := range dataURI {
			if c == ',' {
				base64Data = dataURI[i+1:]
				break
			}
		}
	}

	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		http.Error(w, "failed to decode file", http.StatusBadRequest)
		return
	}

	err = s.Service.ImportPuzzle(r.Context(), puzzleID, data, file.Name)
	if err != nil {
		fmt.Printf("Import error: %v\n", err)
		http.Error(w, fmt.Sprintf("import failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Reset the importedFiles signal on the client so they can import again if needed
	datastar.NewSSE(w, r).PatchSignals([]byte(`{"importedFiles": []}`))

	s.Service.BroadcastUpdate(puzzleID, true)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.Service.NC == nil || !s.Service.NC.IsConnected() {
		http.Error(w, "nats not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
