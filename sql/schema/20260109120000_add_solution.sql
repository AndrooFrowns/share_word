-- +goose Up
ALTER TABLE cells ADD COLUMN solution TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE cells DROP COLUMN solution;
