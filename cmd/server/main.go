package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"share_word/internal/app"
	"share_word/internal/db"
	"share_word/internal/transport"
	"share_word/internal/web/components"
	"share_word/sql/schema"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "shareword.db"
	}

	env := os.Getenv("ENV")
	isProd := env == "production"

	components.Version = fmt.Sprintf("%d", time.Now().Unix())

	dbConn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		log.Fatal(err)
		return
	}
	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal(err)
		return
	}

	// Run migrations from embedded FS
	goose.SetBaseFS(schema.Migrations)
	if err := goose.Up(dbConn, "."); err != nil {
		log.Fatal(err)
		return
	}

	queries := db.New(dbConn)
	service := app.NewService(queries, dbConn)
	defer service.Shutdown()
	server := transport.NewServer(service, dbConn, isProd)

	log.Printf("Server starting in %s mode on http://localhost:%s\n", env, port)
	if err := http.ListenAndServe(":"+port, server.Router); err != nil {
		log.Fatal(err)
	}
}
