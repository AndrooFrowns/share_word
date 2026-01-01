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
