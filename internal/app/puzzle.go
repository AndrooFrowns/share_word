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
	qtx := s.queries.WithTx(tx)

	lastPuzzle, err := qtx.GetLastPuzzleByOwner(ctx, ownerID)
	if err == nil && !s.SkipCooldown {
		if time.Since(lastPuzzle.CreatedAt) < 30*time.Second {
			return nil, errors.New("please wait a moment before creating another puzzle")
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if width < 1 || height < 1 {
		return nil, errors.New("Grid must be at least 1x1")
	}

	if height > 255 || width > 255 {
		return nil, errors.New("Grid must be at most 255x255")
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

func (s *Service) GetPuzzlesByOwner(ctx context.Context, ownerID string, limit, offset int) ([]db.Puzzle, error) {
	return s.queries.GetPuzzlesByOwner(ctx, db.GetPuzzlesByOwnerParams{
		OwnerID: ownerID,
		Limit:   int64(limit),
		Offset:  int64(offset),
	})
}

func (s *Service) GetPuzzlesFromFollowing(ctx context.Context, followerID string, limit, offset int) ([]db.Puzzle, error) {
	return s.queries.GetPuzzlesFromFollowing(ctx, db.GetPuzzlesFromFollowingParams{
		FollowerID: followerID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
}
