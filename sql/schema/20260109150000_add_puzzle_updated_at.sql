-- +goose Up
ALTER TABLE puzzles ADD COLUMN updated_at DATETIME;
UPDATE puzzles SET updated_at = created_at;
-- We can't easily add NOT NULL with a default value that is an expression in one go in some SQLite versions.
-- But since we just want it to be there, we'll just add it.

-- +goose Down
-- No-op