package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

type ParsedPuzzle struct {
	Title  string
	Author string
	Width  int
	Height int
	Cells  []ParsedCell
	Clues  []ParsedClue
}

type ParsedCell struct {
	X       int
	Y       int
	Char    string
	IsBlock bool
}

type ParsedClue struct {
	Number    int
	Direction Direction
	Text      string
}

func ParsePuzzleFile(filename string, data []byte) (*ParsedPuzzle, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".puz" {
		return ParsePuz(data)
	}
	if ext == ".ipuz" {
		return ParseIpuz(data)
	}

	// Fallback check magic bytes for .puz
	if len(data) > 0x10 && string(data[2:13]) == "ACROSS&DOWN" {
		return ParsePuz(data)
	}

	return nil, errors.New("unsupported file format")
}

func ParsePuz(data []byte) (*ParsedPuzzle, error) {
	if len(data) < 0x34 {
		return nil, errors.New("invalid puz file: too short")
	}

	r := bytes.NewReader(data)

	// Skip to width/height at 0x2C
	header := make([]byte, 0x34)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	if string(header[2:13]) != "ACROSS&DOWN" {
		return nil, errors.New("invalid puz file: missing magic bytes")
	}

	width := int(header[0x2C])
	height := int(header[0x2D])
	// numClues := binary.LittleEndian.Uint16(header[0x2E:0x30])

	numCells := width * height

	// Solution Grid
	solution := make([]byte, numCells)
	if _, err := io.ReadFull(r, solution); err != nil {
		return nil, err
	}

	// Player State Grid
	state := make([]byte, numCells)
	if _, err := io.ReadFull(r, state); err != nil {
		return nil, err
	}

	var cells []ParsedCell
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			char := solution[y*width+x]
			isBlock := char == '.'
			val := ""
			if !isBlock {
				val = string(char)
			}
			cells = append(cells, ParsedCell{
				X:       x,
				Y:       y,
				Char:    val,
				IsBlock: isBlock,
			})
		}
	}

	// Helper to read null-terminated strings
	readString := func() string {
		var buf []byte
		for {
			b, err := r.ReadByte()
			if err != nil || b == 0 {
				break
			}
			buf = append(buf, b)
		}
		decoded, _ := charmap.ISO8859_1.NewDecoder().Bytes(buf)
		return string(decoded)
	}

	title := readString()
	author := readString()
	_ = readString() // copyright

	var clues []ParsedClue
	grid := make([][]bool, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			grid[y][x] = solution[y*width+x] == '.'
		}
	}

	counter := 1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if grid[y][x] {
				continue
			}

			startsAcross := (x == 0 || grid[y][x-1]) && (x+1 < width && !grid[y][x+1])
			startsDown := (y == 0 || grid[y-1][x]) && (y+1 < height && !grid[y+1][x])

			if startsAcross || startsDown {
				if startsAcross {
					clues = append(clues, ParsedClue{Number: counter, Direction: DirectionAcross, Text: readString()})
				}
				if startsDown {
					clues = append(clues, ParsedClue{Number: counter, Direction: DirectionDown, Text: readString()})
				}
				counter++
			}
		}
	}

	return &ParsedPuzzle{
		Title:  title,
		Author: author,
		Width:  width,
		Height: height,
		Cells:  cells,
		Clues:  clues,
	}, nil
}

// ipuz parser remains same, but I'll make it more tolerant
type ipuzFile struct {
	Version    string   `json:"version"`
	Kind       []string `json:"kind"`
	Dimensions struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions"`
	Puzzle   [][]interface{}        `json:"puzzle"`
	Solution [][]interface{}        `json:"solution"`
	Title    string                 `json:"title"`
	Author   string                 `json:"author"`
	Clues    map[string]interface{} `json:"clues"`
}

func ParseIpuz(data []byte) (*ParsedPuzzle, error) {
	var f ipuzFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}

	width := f.Dimensions.Width
	height := f.Dimensions.Height
	if width == 0 || height == 0 {
		if len(f.Puzzle) > 0 {
			height = len(f.Puzzle)
			width = len(f.Puzzle[0])
		} else {
			return nil, errors.New("invalid ipuz dimensions")
		}
	}

	var cells []ParsedCell
	gridSource := f.Solution
	if len(gridSource) == 0 {
		gridSource = f.Puzzle
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var val interface{}
			if y < len(gridSource) && x < len(gridSource[y]) {
				val = gridSource[y][x]
			}

			isBlock := false
			char := ""

			switch v := val.(type) {
			case string:
				if v == "#" {
					isBlock = true
				} else {
					char = v
				}
			case map[string]interface{}:
				if c, ok := v["cell"]; ok {
					if s, ok := c.(string); ok {
						char = s
					}
				}
			case nil:
				isBlock = false
			}

			if char == "#" {
				isBlock = true
				char = ""
			}

			cells = append(cells, ParsedCell{
				X:       x,
				Y:       y,
				Char:    strings.ToUpper(char),
				IsBlock: isBlock,
			})
		}
	}

	var parsedClues []ParsedClue

	processClues := func(dirStr string, dir Direction) {
		if list, ok := f.Clues[dirStr]; ok {
			if arr, ok := list.([]interface{}); ok {
				for _, item := range arr {
					// Handle [number, text] or { "number": 1, "clue": "text" }
					if clueArr, ok := item.([]interface{}); ok && len(clueArr) >= 2 {
						numFloat, _ := clueArr[0].(float64)
						text, _ := clueArr[1].(string)
						parsedClues = append(parsedClues, ParsedClue{
							Number:    int(numFloat),
							Direction: dir,
							Text:      text,
						})
					} else if clueMap, ok := item.(map[string]interface{}); ok {
						numFloat, _ := clueMap["number"].(float64)
						text, _ := clueMap["clue"].(string)
						if text == "" {
							text, _ = clueMap["text"].(string)
						}
						parsedClues = append(parsedClues, ParsedClue{
							Number:    int(numFloat),
							Direction: dir,
							Text:      text,
						})
					}
				}
			}
		}
	}

	processClues("Across", DirectionAcross)
	processClues("Down", DirectionDown)
	processClues("across", DirectionAcross)
	processClues("down", DirectionDown)

	return &ParsedPuzzle{
		Title:  f.Title,
		Author: f.Author,
		Width:  width,
		Height: height,
		Cells:  cells,
		Clues:  parsedClues,
	}, nil
}
