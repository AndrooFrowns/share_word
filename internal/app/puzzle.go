package app

import (
	"context"
	"errors"
	"fmt"
	"share_word/internal/db"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AnnotatedCell struct {
	db.Cell
	Number int // 0 if no number
}

type Direction string

const (
	DirectionAcross Direction = "across"
	DirectionDown   Direction = "down"
)

type Clue struct {
	Number    int
	Direction Direction
	Text      string
	Answer    string // The derived answer from the grid
}

func (s *Service) GetFullClues(ctx context.Context, p *db.Puzzle, cells []db.Cell) ([]Clue, error) {
	dbClues, err := s.Queries.GetClues(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	clueMap := make(map[string]string)
	for _, c := range dbClues {
		clueMap[fmt.Sprintf("%d-%s", c.Number, c.Direction)] = c.Text
	}

	derived := s.DeriveClues(int(p.Width), int(p.Height), cells)
	for i := range derived {
		txt, ok := clueMap[fmt.Sprintf("%d-%s", derived[i].Number, derived[i].Direction)]
		if ok {
			derived[i].Text = txt
		}
	}
	return derived, nil
}

func (s *Service) DeriveClues(width, height int, cells []db.Cell) []Clue {
	annotated := s.CalculateNumbers(width, height, cells)
	cellMap := make(map[string]AnnotatedCell)
	for _, c := range annotated {
		cellMap[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}

	var clues []Clue

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c, ok := cellMap[fmt.Sprintf("%d,%d", x, y)]
			if !ok || c.Number == 0 || c.IsBlock {
				continue
			}

			// Check Across
			if x == 0 || cellMap[fmt.Sprintf("%d,%d", x-1, y)].IsBlock {
				word := ""
				for currX := x; currX < width; currX++ {
					next := cellMap[fmt.Sprintf("%d,%d", currX, y)]
					if next.IsBlock {
						break
					}
					char := next.Char
					if char == "" {
						char = "_"
					}
					word += char
				}
				if len(word) >= 2 {
					clues = append(clues, Clue{
						Number:    c.Number,
						Direction: DirectionAcross,
						Answer:    word,
					})
				}
			}

			// Check Down
			if y == 0 || cellMap[fmt.Sprintf("%d,%d", x, y-1)].IsBlock {
				word := ""
				for currY := y; currY < height; currY++ {
					next := cellMap[fmt.Sprintf("%d,%d", x, currY)]
					if next.IsBlock {
						break
					}
					char := next.Char
					if char == "" {
						char = "_"
					}
					word += char
				}
				if len(word) >= 2 {
					clues = append(clues, Clue{
						Number:    c.Number,
						Direction: DirectionDown,
						Answer:    word,
					})
				}
			}
		}
	}

	return clues
}

func (s *Service) CalculateNumbers(width, height int, cells []db.Cell) []AnnotatedCell {
	cellMap := make(map[string]db.Cell)
	for _, c := range cells {
		cellMap[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}

	var annotated []AnnotatedCell
	counter := 1

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c, ok := cellMap[fmt.Sprintf("%d,%d", x, y)]
			if !ok {
				continue
			}
			isNumbered := false

			if !c.IsBlock {
				startsAcross := (x == 0 || cellMap[fmt.Sprintf("%d,%d", x-1, y)].IsBlock) &&
					(x+1 < width && !cellMap[fmt.Sprintf("%d,%d", x+1, y)].IsBlock)

				startsDown := (y == 0 || cellMap[fmt.Sprintf("%d,%d", x, y-1)].IsBlock) &&
					(y+1 < height && !cellMap[fmt.Sprintf("%d,%d", x, y+1)].IsBlock)

				if startsAcross || startsDown {
					isNumbered = true
				}
			}

			ac := AnnotatedCell{Cell: c}
			if isNumbered {
				ac.Number = counter
				counter++
			}
			annotated = append(annotated, ac)
		}
	}

	return annotated
}

func (s *Service) CreatePuzzle(ctx context.Context, name string, ownerID string, width, height int64) (*db.Puzzle, error) {
	if width < 5 || height < 5 {
		return nil, errors.New("grid must be at least 5x5")
	}

	if height > 23 || width > 23 {
		return nil, errors.New("grid must be at most 23x23")
	}

	name = strings.TrimSpace(name)
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name = fmt.Sprintf("Puzzle %s", time.Now().Format("2006-01-02 15:04"))
	}
	if len(name) > 100 {
		name = name[:100]
	}

	if !s.SkipCooldown {
		last, _ := s.Queries.GetLastPuzzleByOwner(ctx, ownerID)
		if !last.CreatedAt.IsZero() && time.Since(last.CreatedAt) < 30*time.Second {
			return nil, errors.New("please wait 30 seconds before creating another puzzle")
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.Queries.WithTx(tx)

	puzzle, err := qtx.CreatePuzzle(ctx, db.CreatePuzzleParams{
		ID:      uuid.New().String(),
		OwnerID: ownerID,
		Name:    name,
		Width:   width,
		Height:  height,
	})
	if err != nil {
		return nil, err
	}

	for y := int64(0); y < height; y++ {
		for x := int64(0); x < width; x++ {
			err = qtx.UpdateCell(ctx, db.UpdateCellParams{
				PuzzleID: puzzle.ID,
				X:        x,
				Y:        y,
				Char:     "",
				IsBlock:  false,
				IsPencil: false,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &puzzle, nil
}