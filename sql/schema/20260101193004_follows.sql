-- +goose Up
-- +goose StatementBegin
CREATE TABLE follows (
    follower_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followed_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, followed_id)
);

CREATE INDEX idx_follows_followed ON follows(followed_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE follows;
-- +goose StatementEnd
