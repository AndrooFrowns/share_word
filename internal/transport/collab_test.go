package transport

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTabIsolation(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := context.Background()

	// 1. Setup user and puzzle
	_, _ = s.Service.RegisterUser(ctx, "tabuser", "pass123")
	loginBody := `{"username":"tabuser", "password":"pass123"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)
	cookie := rr.Header().Get("Set-Cookie")

	p, err := s.Service.CreatePuzzle(ctx, "Tab Test", "tabuser", 5, 5)
	require.NoError(t, err)

	// 2. Focus cell 0,0 in Tab A
	tabA := "client-a"
	focusReqA := httptest.NewRequest("POST", fmt.Sprintf("/puzzles/%s/cells/0/0/focus", p.ID), strings.NewReader(fmt.Sprintf(`{"clientID":"%s"}`, tabA)))
	focusReqA.Header.Set("Cookie", cookie)
	focusReqA.Header.Set("Content-Type", "application/json")
	focusReqA.Header.Set("Datastar-Request", "true")
	s.Router.ServeHTTP(httptest.NewRecorder(), focusReqA)

	// 3. Focus cell 1,1 in Tab B (same session)
	tabB := "client-b"
	focusReqB := httptest.NewRequest("POST", fmt.Sprintf("/puzzles/%s/cells/1/1/focus", p.ID), strings.NewReader(fmt.Sprintf(`{"clientID":"%s"}`, tabB)))
	focusReqB.Header.Set("Cookie", cookie)
	focusReqB.Header.Set("Content-Type", "application/json")
	focusReqB.Header.Set("Datastar-Request", "true")
	s.Router.ServeHTTP(httptest.NewRecorder(), focusReqB)

	// 4. Verify internal state isolation by scanning the map
	var valA, valB string
	s.Service.FocusedCells.Range(func(k, v any) bool {
		key := k.(string)
		if strings.HasSuffix(key, ":"+tabA) {
			valA = v.(string)
		}
		if strings.HasSuffix(key, ":"+tabB) {
			valB = v.(string)
		}
		return true
	})

	assert.Equal(t, "0,0", valA)
	assert.Equal(t, "1,1", valB)
}

func TestInputSanitization(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := context.Background()

	p, err := s.Service.CreatePuzzle(ctx, "Sanity Test", "owner", 5, 5)
	require.NoError(t, err)

	// Update cell with lowercase long string
	updateBody := `{"cellValue":"abc"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/puzzles/%s/cells/0/0/update", p.ID), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")
	s.Router.ServeHTTP(httptest.NewRecorder(), req)

	// Verify it was sanitized to 'C' (last char, uppercase)
	cells, _ := s.Service.Queries.GetCells(ctx, p.ID)
	found := false
	for _, c := range cells {
		if c.X == 0 && c.Y == 0 {
			assert.Equal(t, "C", c.Char)
			found = true
		}
	}
	assert.True(t, found)
}
