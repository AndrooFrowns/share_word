package app

import (
	"context"
	"share_word/internal/db"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestResizePuzzle(t *testing.T) {
	svc, _, _ := SetupTestService(t)

	ctx := context.Background()
	user, err := svc.Queries.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New().String(),
		Username:     "resizer",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
	})
	assert.NoError(t, err)

	// 1. Create a 5x5 puzzle
	svc.SkipCooldown = true
	p, err := svc.CreatePuzzle(ctx, "Test Resize", user.ID, 5, 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), p.Width)
	assert.Equal(t, int64(5), p.Height)

	// Verify initial cells
	cells, _ := svc.Queries.GetCells(ctx, p.ID)
	assert.Equal(t, 25, len(cells))

	// 2. Expand to 7x7
	err = svc.ResizePuzzle(ctx, p.ID, 7, 7)
	assert.NoError(t, err)

	updatedP, _ := svc.Queries.GetPuzzle(ctx, p.ID)
	assert.Equal(t, int64(7), updatedP.Width)
	assert.Equal(t, int64(7), updatedP.Height)

	cells, _ = svc.Queries.GetCells(ctx, p.ID)
	assert.Equal(t, 49, len(cells))

	// Verify a new cell exists
	found := false
	for _, c := range cells {
		if c.X == 6 && c.Y == 6 {
			found = true
			break
		}
	}
	assert.True(t, found, "New cell at 6,6 should exist")

	// 3. Set a value in a cell that will survive
	err = svc.Queries.UpdateCell(ctx, db.UpdateCellParams{
		PuzzleID: p.ID,
		X:        2,
		Y:        2,
		Char:     "A",
	})
	assert.NoError(t, err)

	// Set a value in a cell that will be deleted
	err = svc.Queries.UpdateCell(ctx, db.UpdateCellParams{
		PuzzleID: p.ID,
		X:        6,
		Y:        6,
		Char:     "B",
	})
	assert.NoError(t, err)

	// 4. Shrink to 6x6
	err = svc.ResizePuzzle(ctx, p.ID, 6, 6)
	assert.NoError(t, err)

	updatedP, _ = svc.Queries.GetPuzzle(ctx, p.ID)
	assert.Equal(t, int64(6), updatedP.Width)
	assert.Equal(t, int64(6), updatedP.Height)

	cells, _ = svc.Queries.GetCells(ctx, p.ID)
	assert.Equal(t, 36, len(cells))

	// Verify "A" survived
	survived := false
	for _, c := range cells {
		if c.X == 2 && c.Y == 2 && c.Char == "A" {
			survived = true
			break
		}
	}
	assert.True(t, survived, "Cell 'A' at 2,2 should still exist")

	// Verify "B" was deleted (implicitly checked by count 36, but explicit check is good)
	deleted := true
	for _, c := range cells {
		if c.X == 6 && c.Y == 6 {
			deleted = false
			break
		}
	}
	assert.True(t, deleted, "Cell 'B' at 6,6 should be deleted")

	// 5. Test Limits
	err = svc.ResizePuzzle(ctx, p.ID, 1, 6)
	assert.Error(t, err, "Should fail resizing to width 1")

	err = svc.ResizePuzzle(ctx, p.ID, 24, 6)
	assert.Error(t, err, "Should fail resizing to width 24")
}
