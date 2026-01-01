-- +goose Up
CREATE TABLE puzzles (
    id          TEXT PRIMARY KEY,
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    width       INTEGER NOT NULL DEFAULT 15,
    height      INTEGER NOT NULL DEFAULT 15,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE cells (
    puzzle_id   TEXT NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    x           INTEGER NOT NULL,
    y           INTEGER NOT NULL,
    char        TEXT NOT NULL DEFAULT '',
    is_block    BOOLEAN NOT NULL DEFAULT FALSE,
    is_pencil   BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (puzzle_id, x, y)
);

CREATE TABLE clues (
    puzzle_id   TEXT NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    number      INTEGER NOT NULL,
    direction   TEXT NOT NULL, 
    text        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (puzzle_id, number, direction)
);

CREATE INDEX idx_cells_puzzle ON cells(puzzle_id);
CREATE INDEX idx_clues_puzzle ON clues(puzzle_id);

-- +goose Down
DROP TABLE clues;
DROP TABLE cells;
DROP TABLE puzzles;
