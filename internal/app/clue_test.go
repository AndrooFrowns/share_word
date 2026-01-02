package app

import (
	"context"
	"share_word/internal/db"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClueLogic(t *testing.T) {
	ctx := context.Background()
	svc, queries, _ := SetupTestService(t)

	user, _ := queries.CreateUser(ctx, db.CreateUserParams{ID: "u1", Username: "user1", PasswordHash: "hash"})
	p, err := svc.CreatePuzzle(ctx, "Test Puzzle", user.ID, 5, 5)
	require.NoError(t, err)

	cells, _ := queries.GetCells(ctx, p.ID)
	clues, err := svc.GetFullClues(ctx, p, cells)
	require.NoError(t, err)
	
	found1A := false
	for _, c := range clues {
		if c.Number == 1 && c.Direction == DirectionAcross {
			found1A = true
			assert.Equal(t, "_____", c.Answer)
			assert.Equal(t, "", c.Text)
		}
	}
	assert.True(t, found1A)

	err = queries.UpsertClue(ctx, db.UpsertClueParams{
		PuzzleID:  p.ID,
		Number:    1,
		Direction: string(DirectionAcross),
		Text:      "A first hint",
	})
	require.NoError(t, err)

	clues, err = svc.GetFullClues(ctx, p, cells)
	require.NoError(t, err)

	foundUpdated := false
	for _, c := range clues {
		if c.Number == 1 && c.Direction == DirectionAcross {
			foundUpdated = true
			assert.Equal(t, "A first hint", c.Text)
		}
	}
	assert.True(t, foundUpdated)

	err = queries.UpdateCell(ctx, db.UpdateCellParams{
		PuzzleID: p.ID,
		X:        1,
		Y:        0,
		IsBlock:  true,
	})
	require.NoError(t, err)

	cells, _ = queries.GetCells(ctx, p.ID)
	clues, err = svc.GetFullClues(ctx, p, cells)
	require.NoError(t, err)

	for _, c := range clues {
		if c.Number == 1 && c.Direction == DirectionAcross {
			t.Errorf("1-Across should have been filtered out (too short)")
		}
	}
}