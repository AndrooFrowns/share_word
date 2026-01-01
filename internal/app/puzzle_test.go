package app

import (
	"context"
	"database/sql"
	"fmt"
	"share_word/internal/db"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func setupPuzzleTest(t *testing.T) (*Service, *sql.DB) {
	dbConn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	goose.SetDialect("sqlite3")
	if err := goose.Up(dbConn, "../../sql/schema"); err != nil {
		t.Fatal(err)
	}

	queries := db.New(dbConn)
	service := NewService(queries, dbConn)
	return service, dbConn
}

func TestCreatePuzzle(t *testing.T) {
	service, dbConn := setupPuzzleTest(t)
	defer dbConn.Close()
	ctx := context.Background()

	// 1. Setup a user
	user, err := service.RegisterUser(ctx, "puzzle_owner", "password123456")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("successfully create 3x3 grid", func(t *testing.T) {
		width, height := int64(3), int64(3)
		puzzle, err := service.CreatePuzzle(ctx, "Test Puzzle", user.ID, width, height)
		if err != nil {
			t.Fatalf("failed to create puzzle: %v", err)
		}

		if puzzle.Name != "Test Puzzle" {
			t.Errorf("expected name 'Test Puzzle', got %s", puzzle.Name)
		}

		// Verify cells were initialized
		cells, err := service.queries.GetCells(ctx, puzzle.ID)
		if err != nil {
			t.Fatal(err)
		}

		expectedCells := int(width * height)
		if len(cells) != expectedCells {
			t.Errorf("expected %d cells, got %d", expectedCells, len(cells))
		}
	})

	t.Run("enforce creation cooldown", func(t *testing.T) {
		// Attempt to create another puzzle immediately for the same user
		_, err := service.CreatePuzzle(ctx, "Second Puzzle", user.ID, 5, 5)
		if err == nil {
			t.Fatal("expected error due to cooldown, but got nil")
		}

		expectedErr := "please wait a moment before creating another puzzle"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	})

	t.Run("default name for whitespace or empty", func(t *testing.T) {
		service.SkipCooldown = true
		p, err := service.CreatePuzzle(ctx, "   ", user.ID, 3, 3)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name == "" || p.Name == "   " {
			t.Errorf("expected default name, got %q", p.Name)
		}
	})

	t.Run("normalize name whitespace", func(t *testing.T) {
		service.SkipCooldown = true
		p, err := service.CreatePuzzle(ctx, "  My   Cool  Puzzle  ", user.ID, 3, 3)
		if err != nil {
			t.Fatal(err)
		}
		expected := "My Cool Puzzle"
		if p.Name != expected {
			t.Errorf("expected %q, got %q", expected, p.Name)
		}
	})

	t.Run("truncate long name", func(t *testing.T) {
		service.SkipCooldown = true
		longName := "This is a very long puzzle name that should be truncated because it exceeds the maximum allowed length of one hundred characters"
		p, err := service.CreatePuzzle(ctx, longName, user.ID, 3, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Name) > 100 {
			t.Errorf("expected length <= 100, got %d", len(p.Name))
		}
		if p.Name != longName[:100] {
			t.Errorf("expected truncation, got %q", p.Name)
		}
	})

	t.Run("dimension edge cases", func(t *testing.T) {
		service.SkipCooldown = true // ENABLE BYPASS FOR THIS TEST

		tests := []struct {
			name    string
			width   int64
			height  int64
			wantErr bool
		}{
			{"valid 15x15", 15, 15, false},
			{"valid 1x1", 1, 1, false},
			{"valid boundary 255x15", 255, 15, false},
			{"valid boundary 15x255", 15, 255, false},
			{"valid boundary 255x255", 255, 255, false},
			{"invalid 0 width", 0, 15, true},
			{"invalid 0 height", 15, 0, true},
			{"invalid negative width", -5, 5, true},
			{"invalid negative height", 5, -5, true},
			{"invalid over limit 256x256", 256, 256, true},
			{"invalid over limit 256x15", 256, 15, true},
			{"invalid over limit 15x256", 15, 256, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pName := fmt.Sprintf("Puzzle %s", tt.name)
				_, err := service.CreatePuzzle(ctx, pName, user.ID, tt.width, tt.height)
				if (err != nil) != tt.wantErr {
					t.Errorf("CreatePuzzle(%d, %d) error = %v, wantErr %v", tt.width, tt.height, err, tt.wantErr)
				}
			})
		}
	})
}