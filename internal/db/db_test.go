package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestCreateUser(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	goose.SetDialect("sqlite3")
	if err := goose.Up(db, "../../sql/schema"); err != nil {
		t.Fatal(err)
	}

	q := New(db)
	ctx := context.Background()

	now := time.Now().UTC().Round(0)

	user, err := q.CreateUser(ctx, CreateUserParams{
		ID:           "user-1",
		Username:     "alice",
		PasswordHash: "hash",
		CreatedAt:    now,
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("expected id user-1, got: %v", user.ID)
	}
	if user.Username != "alice" {
		t.Errorf("expected username alice, got: %v", user.Username)
	}
	if user.PasswordHash != "hash" {
		t.Errorf("expected has of hash, got: %v", user.PasswordHash)
	}
	if user.CreatedAt != now {
		t.Errorf("expected timestamp of createdAt to match %v, got: %v", now, user.CreatedAt)
	}
}
