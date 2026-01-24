package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIpuz_StringClueNumbers(t *testing.T) {
	// Minimal JSON with string clue number "1"
	jsonData := `{
		"version": "http://ipuz.org/v2",
		"kind": ["http://ipuz.org/crossword#1"],
		"dimensions": {"width": 3, "height": 3},
		"puzzle": [[1,2,3],[4,5,6],[7,8,9]],
		"solution": [["A","B",null],["D","E","F"],["G","H","I"]],
		"clues": {
			"Across": [
				["1", "Clue Text"]
			],
			"Down": []
		}
	}`

	parsed, err := ParseIpuz([]byte(jsonData))
	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	
	// Check that (2,0) is a block (null in solution)
	// Cells are flattened row-by-row. (0,0), (1,0), (2,0)...
	// Index 2 is (2,0)
	assert.True(t, parsed.Cells[2].IsBlock, "Expected cell (2,0) to be a block due to null in solution")

	assert.Equal(t, 1, len(parsed.Clues))
	assert.Equal(t, 1, parsed.Clues[0].Number)
	assert.Equal(t, "Clue Text", parsed.Clues[0].Text)
	assert.Equal(t, DirectionAcross, parsed.Clues[0].Direction)
}
