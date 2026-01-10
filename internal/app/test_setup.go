package app

import (
	"database/sql"
	"share_word/internal/db"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// SetupTestService is a helper for integration tests that need a real DB and Service.
func SetupTestService(t *testing.T) (*Service, *db.Queries, *sql.DB) {
	dbConn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	goose.SetDialect("sqlite3")
	// Try standard relative paths for migrations
	err = goose.Up(dbConn, "../../sql/schema")
	if err != nil {
		err = goose.Up(dbConn, "../sql/schema")
		if err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}
	}

	queries := db.New(dbConn)
	service := NewService(queries, dbConn)

	t.Cleanup(func() {
		service.Shutdown()
		dbConn.Close()
	})

	return service, queries, dbConn
}
