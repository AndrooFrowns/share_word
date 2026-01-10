-- name: CreateUser :one
INSERT INTO users (id, username, password_hash, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ? LIMIT 1;

-- name: FollowUser :exec
INSERT INTO follows (follower_id, followed_id)
VALUES (?, ?);

-- name: UnfollowUser :exec
DELETE FROM follows
WHERE follower_id = ? AND followed_id = ?;

-- name: GetFollowers :many
SELECT u.* FROM users u
JOIN follows f on u.id = f.follower_id
WHERE f.followed_id = ?
ORDER BY f.created_at DESC
LIMIT ? OFFSET ?;

-- name: GetFollowing :many
SELECT u.* FROM users u
JOIN follows f on u.id = f.followed_id
WHERE f.follower_id = ?
ORDER BY f.created_at DESC
LIMIT ? OFFSET ?;

-- name: IsFollowing :one
SELECT EXISTS (
    SELECT 1 FROM follows
    WHERE follower_id = ? AND followed_id = ?
);

-- name: CreatePuzzle :one
INSERT INTO puzzles (id, owner_id, name, width, height)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdatePuzzleDimensions :exec
UPDATE puzzles SET width = ?, height = ? WHERE id = ?;

-- name: GetClues :many
SELECT * FROM clues WHERE puzzle_id = ?;

-- name: UpsertClue :exec
INSERT INTO clues (puzzle_id, number, direction, text)
VALUES (?, ?, ?, ?)
ON CONFLICT(puzzle_id, number, direction) DO UPDATE SET
    text = excluded.text;

-- name: DeleteAllClues :exec
DELETE FROM clues WHERE puzzle_id = ?;

-- name: GetPuzzle :one
SELECT * FROM puzzles WHERE id = ? LIMIT 1;

-- name: UpdateCell :exec
INSERT INTO cells (puzzle_id, x, y, char, is_block, is_pencil, solution)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(puzzle_id, x, y) DO UPDATE SET
    char = excluded.char,
    is_block = excluded.is_block,
    is_pencil = excluded.is_pencil;

-- name: ImportCell :exec
INSERT INTO cells (puzzle_id, x, y, char, is_block, is_pencil, solution)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(puzzle_id, x, y) DO UPDATE SET
    char = excluded.char,
    solution = excluded.solution,
    is_block = excluded.is_block,
    is_pencil = excluded.is_pencil;

-- name: ToggleBlock :exec
UPDATE cells 
SET is_block = NOT is_block, char = '' 
WHERE puzzle_id = ? AND x = ? AND y = ?;

-- name: GetCells :many
SELECT * FROM cells WHERE puzzle_id = ? ORDER BY y, x;

-- name: DeleteCellsOutside :exec
DELETE FROM cells WHERE puzzle_id = ? AND (x >= ? OR y >= ?);

-- name: DeleteAllCells :exec
DELETE FROM cells WHERE puzzle_id = ?;

-- name: GetPuzzlesByOwner :many
SELECT * FROM puzzles WHERE owner_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: GetPuzzlesFromFollowing :many
SELECT p.* FROM puzzles p
JOIN follows f ON f.followed_id = p.owner_id
WHERE f.follower_id = ?
ORDER BY p.created_at DESC LIMIT ? OFFSET ?;

-- name: GetLastPuzzleByOwner :one
SELECT * FROM puzzles
WHERE owner_id = ?
ORDER BY created_at DESC
LIMIT 1;