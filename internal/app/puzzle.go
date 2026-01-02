package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"share_word/internal/db"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreatePuzzle(ctx context.Context, puzzle_name, ownerID string, width, height int64) (*db.Puzzle, error) {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := s.Queries.WithTx(tx)

	lastPuzzle, err := qtx.GetLastPuzzleByOwner(ctx, ownerID)
	if err == nil && !s.SkipCooldown {
		if time.Since(lastPuzzle.CreatedAt) < 30*time.Second {
			return nil, errors.New("please wait a moment before creating another puzzle")
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if width < 5 || height < 5 {
		return nil, errors.New("Grid must be at least 5x5")
	}

	if height > 23 || width > 23 {
		return nil, errors.New("Grid must be at most 23x23")
	}

	puzzle_name = strings.Join(strings.Fields(puzzle_name), " ")
	if len(puzzle_name) > 100 {
		puzzle_name = puzzle_name[:100]
	}

	if puzzle_name == "" {
		puzzle_name = fmt.Sprintf("puzzle_%v", time.Now().UTC().Round(0))
	}

	id := uuid.NewString()

	created_puzzle, err := qtx.CreatePuzzle(
		ctx,
		db.CreatePuzzleParams{
			ID:      id,
			OwnerID: ownerID,
			Name:    puzzle_name,
			Width:   width,
			Height:  height,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating puzzle: %w", err)
	}

	for y := int64(0); y < height; y++ {
		for x := int64(0); x < width; x++ {
			err := qtx.UpdateCell(ctx, db.UpdateCellParams{
				PuzzleID: id,
				X:        int64(x),
				Y:        int64(y),
				Char:     "",
				IsBlock:  false,
				IsPencil: false,
			})
			if err != nil {
				return nil, fmt.Errorf("initializing cell (%d,%d): %w", x, y, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &created_puzzle, nil
}

type AnnotatedCell struct {
	db.Cell
	Number int // The clue number, 0 if no number
}

func (s *Service) CalculateNumbers(width, height int, cells []db.Cell) []AnnotatedCell {
	// Create a 2D lookup for convenience
	grid := make([][]db.Cell, height)
	for i := range grid {
		grid[i] = make([]db.Cell, width)
	}
	for _, c := range cells {
		grid[c.Y][c.X] = c
	}

	annotated := make([]AnnotatedCell, 0, len(cells))
	currentNumber := 1

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := grid[y][x]
			isNumber := false

			if !cell.IsBlock {
				startsAcross := (x == 0 || grid[y][x-1].IsBlock) &&
					(x < width-1 && !grid[y][x+1].IsBlock)

				startsDown := (y == 0 || grid[y-1][x].IsBlock) &&
					(y < height-1 && !grid[y+1][x].IsBlock)

				if startsAcross || startsDown {
					isNumber = true
				}
			}

			num := 0
			if isNumber {
				num = currentNumber
				currentNumber++
			}

			annotated = append(annotated, AnnotatedCell{
				Cell:   cell,
				Number: num,
			})
		}
	}

	return annotated
}

func (s *Service) GetPuzzlesByOwner(ctx context.Context, ownerID string, limit, offset int) ([]db.Puzzle, error) {
	return s.Queries.GetPuzzlesByOwner(ctx, db.GetPuzzlesByOwnerParams{
		OwnerID: ownerID,
		Limit:   int64(limit),
		Offset:  int64(offset),
	})
}

func (s *Service) GetPuzzlesFromFollowing(ctx context.Context, followerID string, limit, offset int) ([]db.Puzzle, error) {
	return s.Queries.GetPuzzlesFromFollowing(ctx, db.GetPuzzlesFromFollowingParams{
		FollowerID: followerID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
}