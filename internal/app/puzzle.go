package app

import (
	"context"
	"errors"
	"fmt"
	"share_word/internal/db"
	"sort"
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

func (s *Service) GetFullClues(ctx context.Context, pID string, cells []db.Cell) ([]Clue, error) {
	dbClues, err := s.Queries.GetClues(ctx, pID)
	if err != nil {
		return nil, err
	}
	clueMap := make(map[string]string)
	for _, c := range dbClues {
		clueMap[fmt.Sprintf("%d-%s", c.Number, c.Direction)] = c.Text
	}

	p, err := s.Queries.GetPuzzle(ctx, pID)
	if err != nil {
		return nil, err
	}

	derived := s.DeriveClues(int(p.Width), int(p.Height), cells)
	for i := range derived {
		key := fmt.Sprintf("%d-%s", derived[i].Number, derived[i].Direction)
		if txt, ok := clueMap[key]; ok {
			derived[i].Text = txt
			delete(clueMap, key) // Remove from map so we know it's used
		}
	}

	// Append orphans
	for key, text := range clueMap {
		var num int
		var dir string
		// Parse key "1-across"
		if _, err := fmt.Sscanf(key, "%d-%s", &num, &dir); err == nil {
			derived = append(derived, Clue{
				Number:    num,
				Direction: Direction(dir),
				Text:      text,
				Answer:    "", // No grid answer
			})
		}
	}

	// Sort
	sort.Slice(derived, func(i, j int) bool {
		if derived[i].Number != derived[j].Number {
			return derived[i].Number < derived[j].Number
		}
		return derived[i].Direction == DirectionAcross // Across first
	})

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
					// Use solution for derived answers if available, otherwise char?
					// For generating clue *structure*, we rely on block/non-block.
					// For the *answer* field, we should probably use the solution if present.
					char := next.Solution
					if char == "" {
						char = next.Char
					}
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
					char := next.Solution
					if char == "" {
						char = next.Char
					}
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
				Solution: "",
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

func (s *Service) ResizePuzzle(ctx context.Context, puzzleID string, newWidth, newHeight int64) error {
	if newWidth < 2 || newHeight < 2 {
		return errors.New("grid must be at least 2x2")
	}
	if newWidth > 23 || newHeight > 23 {
		return errors.New("grid must be at most 23x23")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := s.Queries.WithTx(tx)

	p, err := qtx.GetPuzzle(ctx, puzzleID)
	if err != nil {
		return err
	}

	oldWidth := p.Width
	oldHeight := p.Height

	err = qtx.UpdatePuzzleDimensions(ctx, db.UpdatePuzzleDimensionsParams{
		Width:  newWidth,
		Height: newHeight,
		ID:     puzzleID,
	})
	if err != nil {
		return err
	}

	if newWidth < oldWidth || newHeight < oldHeight {
		err = qtx.DeleteCellsOutside(ctx, db.DeleteCellsOutsideParams{
			PuzzleID: puzzleID,
			X:        newWidth,
			Y:        newHeight,
		})
		if err != nil {
			return err
		}
	}

	// Only insert newly created cells
	for y := int64(0); y < newHeight; y++ {
		for x := int64(0); x < newWidth; x++ {
			if x >= oldWidth || y >= oldHeight {
				err = qtx.UpdateCell(ctx, db.UpdateCellParams{
					PuzzleID: puzzleID,
					X:        x,
					Y:        y,
					Char:     "",
					Solution: "",
					IsBlock:  false,
					IsPencil: false,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return tx.Commit()
}

func (s *Service) ImportPuzzle(ctx context.Context, puzzleID string, data []byte, filename string) error {
	parsed, err := ParsePuzzleFile(filename, data)
	if err != nil {
		return err
	}

	// For import, we allow any size up to a reasonable limit
	if parsed.Width > 30 || parsed.Height > 30 {
		return errors.New("puzzle too large (max 30x30)")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := s.Queries.WithTx(tx)

	// Update Dimensions
	err = qtx.UpdatePuzzleDimensions(ctx, db.UpdatePuzzleDimensionsParams{
		Width:  int64(parsed.Width),
		Height: int64(parsed.Height),
		ID:     puzzleID,
	})
	if err != nil {
		return err
	}

	// Clear existing
	if err := qtx.DeleteAllCells(ctx, puzzleID); err != nil {
		return err
	}
	if err := qtx.DeleteAllClues(ctx, puzzleID); err != nil {
		return err
	}

	// Insert Cells
	for _, cell := range parsed.Cells {
		err = qtx.ImportCell(ctx, db.ImportCellParams{
			PuzzleID: puzzleID,
			X:        int64(cell.X),
			Y:        int64(cell.Y),
			Char:     "",        // User state starts empty
			Solution: cell.Char, // Correct answer
			IsBlock:  cell.IsBlock,
			IsPencil: false,
		})
		if err != nil {
			return err
		}
	}

	// Insert Clues
	for _, clue := range parsed.Clues {
		err = qtx.UpsertClue(ctx, db.UpsertClueParams{
			PuzzleID:  puzzleID,
			Number:    int64(clue.Number),
			Direction: string(clue.Direction),
			Text:      clue.Text,
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Service) GetActiveWordCells(width, height int, cells []db.Cell, focusedX, focusedY int64, direction Direction) map[string]bool {
	activeCells := make(map[string]bool)
	if focusedX < 0 || focusedY < 0 {
		return activeCells
	}

	cellMap := make(map[string]db.Cell)
	for _, c := range cells {
		cellMap[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}

	// Check if starting cell is valid
	start, ok := cellMap[fmt.Sprintf("%d,%d", focusedX, focusedY)]
	if !ok || start.IsBlock {
		return activeCells
	}

	activeCells[fmt.Sprintf("%d,%d", focusedX, focusedY)] = true

	if direction == DirectionAcross {
		// Scan left
		for x := focusedX - 1; x >= 0; x-- {
			c, ok := cellMap[fmt.Sprintf("%d,%d", x, focusedY)]
			if !ok || c.IsBlock {
				break
			}
			activeCells[fmt.Sprintf("%d,%d", x, focusedY)] = true
		}
		// Scan right
		for x := focusedX + 1; x < int64(width); x++ {
			c, ok := cellMap[fmt.Sprintf("%d,%d", x, focusedY)]
			if !ok || c.IsBlock {
				break
			}
			activeCells[fmt.Sprintf("%d,%d", x, focusedY)] = true
		}
	} else {
		// Scan up
		for y := focusedY - 1; y >= 0; y-- {
			c, ok := cellMap[fmt.Sprintf("%d,%d", focusedX, y)]
			if !ok || c.IsBlock {
				break
			}
			activeCells[fmt.Sprintf("%d,%d", focusedX, y)] = true
		}
		// Scan down
		for y := focusedY + 1; y < int64(height); y++ {
			c, ok := cellMap[fmt.Sprintf("%d,%d", focusedX, y)]
			if !ok || c.IsBlock {
				break
			}
			activeCells[fmt.Sprintf("%d,%d", focusedX, y)] = true
		}
	}

	return activeCells
}


func (s *Service) GetClueJumpTarget(ctx context.Context, pID string, cells []db.Cell, x, y int64, dir Direction, forward bool) (int64, int64, Direction) {
	p, err := s.Queries.GetPuzzle(ctx, pID)
	if err != nil {
		return x, y, dir
	}

	width, height := int(p.Width), int(p.Height)
	clues, _ := s.GetFullClues(ctx, pID, cells)
	curClue := s.GetActiveClue(width, height, cells, clues, x, y, dir)
	if curClue == nil {
		if len(clues) > 0 {
			return s.getClueStart(width, height, cells, &clues[0])
		}
		return x, y, dir
	}

	// Create a logical order: all Across clues, then all Down clues
	var logicalClues []Clue
	for _, c := range clues {
		if c.Direction == DirectionAcross {
			logicalClues = append(logicalClues, c)
		}
	}
	for _, c := range clues {
		if c.Direction == DirectionDown {
			logicalClues = append(logicalClues, c)
		}
	}

	if len(logicalClues) == 0 {
		return x, y, dir
	}

	// Find current clue index in logical list
	curIdx := -1
	for i := range logicalClues {
		if logicalClues[i].Number == curClue.Number && logicalClues[i].Direction == curClue.Direction {
			curIdx = i
			break
		}
	}

	if curIdx == -1 {
		return x, y, dir
	}

	var targetClue Clue
	if forward {
		targetIdx := (curIdx + 1) % len(logicalClues)
		targetClue = logicalClues[targetIdx]
	} else {
		targetIdx := (curIdx - 1 + len(logicalClues)) % len(logicalClues)
		targetClue = logicalClues[targetIdx]
	}

	return s.getClueStart(width, height, cells, &targetClue)
}

func (s *Service) getClueStart(width, height int, cells []db.Cell, targetClue *Clue) (int64, int64, Direction) {
	annotated := s.CalculateNumbers(width, height, cells)
	for _, ac := range annotated {
		if ac.Number == targetClue.Number {
			return ac.X, ac.Y, targetClue.Direction
		}
	}
	return 0, 0, targetClue.Direction // Fallback
}

func (s *Service) GetAutoAdvanceTarget(ctx context.Context, pID string, cells []db.Cell, x, y int64, dir Direction, forward bool) (int64, int64, Direction) {
	p, err := s.Queries.GetPuzzle(ctx, pID)
	if err != nil {
		return x, y, dir
	}

	width, height := int(p.Width), int(p.Height)
	cellMap := make(map[string]db.Cell)
	for _, c := range cells {
		cellMap[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}

	// 1. Try immediate next cell in current word
	nx, ny := x, y
	delta := int64(1)
	if !forward {
		delta = -1
	}

	if dir == DirectionAcross {
		nx += delta
	} else {
		ny += delta
	}

	if nx >= 0 && nx < int64(width) && ny >= 0 && ny < int64(height) {
		if !cellMap[fmt.Sprintf("%d,%d", nx, ny)].IsBlock {
			return nx, ny, dir
		}
	}

	// 2. We are at the end/start of a word, find the next/prev clue
	clues, _ := s.GetFullClues(ctx, pID, cells)
	curClue := s.GetActiveClue(width, height, cells, clues, x, y, dir)
	if curClue == nil {
		return x, y, dir
	}

	var targetClue *Clue
	if forward {
		// Find next clue in same direction
		for i := range clues {
			if clues[i].Direction == dir && clues[i].Number > curClue.Number {
				targetClue = &clues[i]
				break
			}
		}
		// If not found, switch direction
		if targetClue == nil {
			otherDir := DirectionAcross
			if dir == DirectionAcross {
				otherDir = DirectionDown
			}
			for i := range clues {
				if clues[i].Direction == otherDir {
					targetClue = &clues[i]
					break
				}
			}
		}
	} else {
		// Find prev clue in same direction
		for i := len(clues) - 1; i >= 0; i-- {
			if clues[i].Direction == dir && clues[i].Number < curClue.Number {
				targetClue = &clues[i]
				break
			}
		}
		// If not found, switch direction (to the last clue of other direction)
		if targetClue == nil {
			otherDir := DirectionAcross
			if dir == DirectionAcross {
				otherDir = DirectionDown
			}
			for i := len(clues) - 1; i >= 0; i-- {
				if clues[i].Direction == otherDir {
					targetClue = &clues[i]
					break
				}
			}
		}
	}

	// Wrap around if still nil
	if targetClue == nil && len(clues) > 0 {
		if forward {
			targetClue = &clues[0]
		} else {
			targetClue = &clues[len(clues)-1]
		}
	}

	if targetClue != nil {
		annotated := s.CalculateNumbers(width, height, cells)
		if forward {
			// Start of clue
			for _, ac := range annotated {
				if ac.Number == targetClue.Number {
					return ac.X, ac.Y, targetClue.Direction
				}
			}
		} else {
			// End of clue
			var lastX, lastY int64
			foundStart := false
			for _, ac := range annotated {
				if ac.Number == targetClue.Number {
					lastX, lastY = ac.X, ac.Y
					foundStart = true
					break
				}
			}
			if foundStart {
				// Scan to end of word
				for {
					tx, ty := lastX, lastY
					if targetClue.Direction == DirectionAcross {
						tx++
					} else {
						ty++
					}
					if tx < int64(width) && ty < int64(height) && !cellMap[fmt.Sprintf("%d,%d", tx, ty)].IsBlock {
						lastX, lastY = tx, ty
					} else {
						break
					}
				}
				return lastX, lastY, targetClue.Direction
			}
		}
	}

	return x, y, dir
}

func (s *Service) GetActiveClue(width, height int, cells []db.Cell, clues []Clue, fx, fy int64, dir Direction) *Clue {
	cellMap := make(map[string]db.Cell)
	for _, c := range cells {
		cellMap[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}

	curX, curY := fx, fy
	if dir == DirectionAcross {
		for curX > 0 {
			prev, ok := cellMap[fmt.Sprintf("%d,%d", curX-1, curY)]
			if !ok || prev.IsBlock {
				break
			}
			curX--
		}
	} else {
		for curY > 0 {
			prev, ok := cellMap[fmt.Sprintf("%d,%d", curX, curY-1)]
			if !ok || prev.IsBlock {
				break
			}
			curY--
		}
	}

	annotated := s.CalculateNumbers(width, height, cells)
	var clueNum int
	for _, ac := range annotated {
		if ac.X == curX && ac.Y == curY {
			clueNum = ac.Number
			break
		}
	}

	if clueNum == 0 {
		return nil
	}

	for i := range clues {
		if clues[i].Number == clueNum && clues[i].Direction == dir {
			return &clues[i]
		}
	}

	return nil
}

func (s *Service) GetNextCell(width, height int, cells []db.Cell, x, y int64, dir Direction, forward bool) (int64, int64) {
	cellMap := make(map[string]db.Cell)
	for _, c := range cells {
		cellMap[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}

	nx, ny := x, y
	delta := int64(1)
	if !forward {
		delta = -1
	}

	for {
		if dir == DirectionAcross {
			nx += delta
		} else {
			ny += delta
		}

		if nx < 0 || nx >= int64(width) || ny < 0 || ny >= int64(height) {
			return x, y // Stop at boundaries
		}

		if !cellMap[fmt.Sprintf("%d,%d", nx, ny)].IsBlock {
			return nx, ny
		}
	}
}

type Point struct {
	X, Y int64
}

func GetSymmetricCells(x, y, width, height int64, mode string) []Point {
	points := []Point{{X: x, Y: y}}

	switch mode {
	case "horizontal":
		// Mirror across vertical axis (left-right)
		points = append(points, Point{X: width - 1 - x, Y: y})
	case "vertical":
		// Mirror across horizontal axis (top-bottom)
		points = append(points, Point{X: x, Y: height - 1 - y})
	case "both":
		// Mirror horizontal, vertical, and diagonal?
		// "Both" usually means full rectangular symmetry?
		// Or H + V (which implies 4 points).
		p2 := Point{X: width - 1 - x, Y: y}              // H
		p3 := Point{X: x, Y: height - 1 - y}             // V
		p4 := Point{X: width - 1 - x, Y: height - 1 - y} // Rotational/Both
		points = append(points, p2, p3, p4)
	case "rotational":
		// 180 degree rotation
		points = append(points, Point{X: width - 1 - x, Y: height - 1 - y})
	}

	// Deduplicate (e.g. center point)
	seen := make(map[string]bool)
	var unique []Point
	for _, p := range points {
		k := fmt.Sprintf("%d,%d", p.X, p.Y)
		if !seen[k] {
			seen[k] = true
			unique = append(unique, p)
		}
	}
	return unique
}
