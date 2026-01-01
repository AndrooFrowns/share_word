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

	user, err := service.RegisterUser(ctx, "puzzle_owner", "password123456")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("successfully create 5x5 grid", func(t *testing.T) {
		width, height := int64(5), int64(5)
		puzzle, err := service.CreatePuzzle(ctx, "Test Puzzle", user.ID, width, height)
		if err != nil {
			t.Fatalf("failed to create puzzle: %v", err)
		}

		if puzzle.Name != "Test Puzzle" {
			t.Errorf("expected name 'Test Puzzle', got %s", puzzle.Name)
		}

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
		_, err := service.CreatePuzzle(ctx, "Second Puzzle", user.ID, 5, 5)
		if err == nil {
			t.Fatal("expected error due to cooldown, but got nil")
		}
	})

	t.Run("default name for whitespace or empty", func(t *testing.T) {
		service.SkipCooldown = true
		p, err := service.CreatePuzzle(ctx, "   ", user.ID, 5, 5)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name == "" || p.Name == "   " {
			t.Errorf("expected default name, got %q", p.Name)
		}
	})

	t.Run("normalize name whitespace", func(t *testing.T) {
		service.SkipCooldown = true
		p, err := service.CreatePuzzle(ctx, "  My   Cool  Puzzle  ", user.ID, 5, 5)
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
		p, err := service.CreatePuzzle(ctx, longName, user.ID, 5, 5)
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
		service.SkipCooldown = true
		tests := []struct {
			name    string
			width   int64
			height  int64
			wantErr bool
		}{
			{"valid 5x5", 5, 5, false},
			{"valid boundary 255x255", 255, 255, false},
			{"invalid 1x1", 1, 1, true},
			{"invalid 4x4", 4, 4, true},
			{"invalid 0 width", 0, 15, true},
			{"invalid negative width", -5, 5, true},
			{"invalid over limit 256x256", 256, 256, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := service.CreatePuzzle(ctx, tt.name, user.ID, tt.width, tt.height)
				if (err != nil) != tt.wantErr {
					t.Errorf("CreatePuzzle(%d, %d) error = %v, wantErr %v", tt.width, tt.height, err, tt.wantErr)
				}
			})
		}
	})
}

func TestCalculateNumbers(t *testing.T) {
	service := &Service{}

	t.Run("Empty 3x3 Grid", func(t *testing.T) {
		// All Across start at x=0, All Down start at y=0
		// 1 2 3
		// 4 . .
		// 5 . .
		width, height := 3, 3
		cells := make([]db.Cell, 0, 9)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				cells = append(cells, db.Cell{X: int64(x), Y: int64(y), IsBlock: false})
			}
		}

		annotated := service.CalculateNumbers(width, height, cells)
		results := getNumberMap(annotated)

		expected := map[string]int{
			"0,0": 1, "1,0": 2, "2,0": 3,
			"0,1": 4,
			"0,2": 5,
		}

		for pos, num := range expected {
			if results[pos] != num {
				t.Errorf("pos %s: expected %d, got %d", pos, num, results[pos])
			}
		}
	})

	t.Run("Grid with Blocks", func(t *testing.T) {
		// Layout:
		// . . .
		// . # .
		// . . .
		// Numbers should be:
		// 1 2 3
		// 4 # . (Cell 2,1 doesn't start a word)
		// 5 6 . (Cell 2,2 doesn't start a word)
		width, height := 3, 3
		cells := []db.Cell{
			{X: 0, Y: 0, IsBlock: false}, {X: 1, Y: 0, IsBlock: false}, {X: 2, Y: 0, IsBlock: false},
			{X: 0, Y: 1, IsBlock: false}, {X: 1, Y: 1, IsBlock: true},  {X: 2, Y: 1, IsBlock: false},
			{X: 0, Y: 2, IsBlock: false}, {X: 1, Y: 2, IsBlock: false}, {X: 2, Y: 2, IsBlock: false},
		}

		annotated := service.CalculateNumbers(width, height, cells)
		results := getNumberMap(annotated)

		if results["1,1"] != 0 {
			t.Errorf("block at 1,1 should not have a number, got %d", results["1,1"])
		}
		if results["0,0"] != 1 {
			t.Errorf("0,0 should be 1, got %d", results["0,0"])
		}
	})

	t.Run("Lonely letters (no 1-letter words)", func(t *testing.T) {
		// Layout:
		// . # .
		// # # #
		// . . .
		// 0,0 is lonely (across has block, down has block). Should have NO number.
		// 2,0 is lonely. Should have NO number.
		// 0,2 should be #1 (starts 1x3 across)
		width, height := 3, 3
		cells := []db.Cell{
			{X: 0, Y: 0, IsBlock: false}, {X: 1, Y: 0, IsBlock: true},  {X: 2, Y: 0, IsBlock: false},
			{X: 0, Y: 1, IsBlock: true},  {X: 1, Y: 1, IsBlock: true},  {X: 2, Y: 1, IsBlock: true},
			{X: 0, Y: 2, IsBlock: false}, {X: 1, Y: 2, IsBlock: false}, {X: 2, Y: 2, IsBlock: false},
		}

		annotated := service.CalculateNumbers(width, height, cells)
		results := getNumberMap(annotated)

		if results["0,0"] != 0 {
			t.Errorf("0,0 is a lonely cell, should be 0, got %d", results["0,0"])
		}
		if results["0,2"] != 1 {
			t.Errorf("0,2 starts a word, should be 1, got %d", results["0,2"])
		}
	})
}

func getNumberMap(cells []AnnotatedCell) map[string]int {
	m := make(map[string]int)
	for _, c := range cells {
		m[fmt.Sprintf("%d,%d", c.X, c.Y)] = c.Number
	}
	return m
}