package app

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
	"share_word/internal/db"
	"testing"
)

func setupTestDB(t *testing.T) (*db.Queries, *sql.DB) {
	dbConn, _ := sql.Open("sqlite", ":memory:")
	goose.SetDialect("sqlite3")
	goose.Up(dbConn, "../../sql/schema/")
	return db.New(dbConn), dbConn
}

func TestRegisterUser(t *testing.T) {
	queries, conn := setupTestDB(t)
	defer conn.Close()

	svc := NewService(queries, conn)
	ctx := context.Background()

	user, err := svc.RegisterUser(ctx, "bob", "secret-pass-abcd")
	if err != nil {
		t.Fatal(err)
	}

	if user.Username != "bob" {
		t.Errorf("expected bob, got: %s", user.Username)
	}

	if user.PasswordHash == "secret-pass-abcd" {
		t.Error("password was not hashed")
	}
}

func TestRegisterUser_Validation(t *testing.T) {
	queries, conn := setupTestDB(t)
	defer conn.Close()
	svc := NewService(queries, conn)
	ctx := context.Background()

	tests := []struct {
		name     string
		username string
		password string
		isGood   bool
	}{
		{"valid", "alice_123", "password12345", true},
		{"mininimum username length", "abc", "password12345", true},
		{"Too Short User", "al", "password12345", false},
		{"maximum username length", "abcdefghijklmnopqrstuvwxyz", "password12345", true},
		{"too long username length", "abcdefghijklmnopqrstuvwxyza", "password12345", false},
		{"Empty User", "", "password12345", false},
		{"Empty password", "8234nf", "", false},
		{"Too Long User", "this_username_is_way_too_long_for_us", "password12345", false},
		{"Invalid Chars", "alice!", "password12345", false},
		{"Very Short Password", "alice", "123", false},
		{"Just Short Password", "alice", "password123", false},
		{"Exact Max Password", "alice", "123456789012345678901234567890123456789012345678901234567890123456789012", true},
		{"Too Long Password", "alice", "1234567890123456789012345678901234567890123456789012345678901234567890123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.RegisterUser(ctx, tt.username, tt.password)
			if (err != nil) == tt.isGood {
				t.Errorf("RegisterUser() error = %v, wantErr %v", err, tt.isGood)
			}
		})
	}
}

func TestAuthenticateUser(t *testing.T) {
	queries, conn := setupTestDB(t)
	defer conn.Close()
	svc := NewService(queries, conn)
	ctx := context.Background()

	// 1. Setup: Register a user first
	username := "test_user"
	password := "a-very-long-valid-password"
	_, err := svc.RegisterUser(ctx, username, password)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	t.Run("successful authentication", func(t *testing.T) {
		user, err := svc.AuthenticateUser(ctx, username, password)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if user == nil || user.Username != username {
			t.Errorf("expected user %s, got %v", username, user)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := svc.AuthenticateUser(ctx, username, "wrong-password-123")
		if err == nil {
			t.Error("expected error for wrong password, got nil")
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		_, err := svc.AuthenticateUser(ctx, "ghost", password)
		if err == nil {
			t.Error("expected error for non-existent user, got nil")
		}
	})
}
