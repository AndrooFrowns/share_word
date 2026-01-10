package app

import (
	"context"
	"os"
	"share_word/internal/db"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestImportPuzzle(t *testing.T) {
	svc, _, _ := SetupTestService(t)
	ctx := context.Background()
	user, _ := svc.Queries.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New().String(),
		Username:     "importer",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
	})

	// Create a placeholder puzzle
	svc.SkipCooldown = true
	p, err := svc.CreatePuzzle(ctx, "Placeholder", user.ID, 15, 15)
	assert.NoError(t, err)

	t.Run("Import .ipuz", func(t *testing.T) {
		data, err := os.ReadFile("testdata/sample.ipuz")
		assert.NoError(t, err)

		err = svc.ImportPuzzle(ctx, p.ID, data, "sample.ipuz")
		assert.NoError(t, err)

		// Verify dimensions
		updated, _ := svc.Queries.GetPuzzle(ctx, p.ID)
		assert.Equal(t, int64(5), updated.Width)
		assert.Equal(t, int64(5), updated.Height)

		// Verify some cells
		cells, _ := svc.Queries.GetCells(ctx, p.ID)
		assert.Equal(t, 25, len(cells))

		// (0,0) should be 'A' but user state empty
		assert.Equal(t, "", cells[0].Char)
		assert.Equal(t, "A", cells[0].Solution)
		// (1,1) should be block
		assert.True(t, cells[6].IsBlock)
	})

	t.Run("Import .puz", func(t *testing.T) {
		data, err := os.ReadFile("testdata/sample.puz")
		assert.NoError(t, err)

		err = svc.ImportPuzzle(ctx, p.ID, data, "sample.puz")
		assert.NoError(t, err)

		updated, _ := svc.Queries.GetPuzzle(ctx, p.ID)
		assert.Equal(t, int64(5), updated.Width)
		assert.Equal(t, int64(5), updated.Height)

		cells, _ := svc.Queries.GetCells(ctx, p.ID)
		assert.Equal(t, 25, len(cells))
		assert.Equal(t, "", cells[0].Char)
		assert.Equal(t, "A", cells[0].Solution)
		assert.True(t, cells[6].IsBlock)
	})
}
