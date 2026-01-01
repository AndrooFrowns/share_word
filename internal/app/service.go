package app

import (
	"database/sql"
	"share_word/internal/db"
)

type Service struct {
	queries      *db.Queries
	db           *sql.DB
	SkipCooldown bool
}

func NewService(q *db.Queries, database *sql.DB) *Service {
	return &Service{queries: q, db: database, SkipCooldown: false}
}

