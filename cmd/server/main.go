package main

import (
	"database/sql"
	"log"
	"net/http"
	"share_word/internal/app"
	"share_word/internal/db"
	"share_word/internal/transport"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func main() {
	dbConn, err := sql.Open("sqlite", "shareword.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		log.Fatal(err)
		return
	}
	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal(err)
		return
	}
	if err := goose.Up(dbConn, "sql/schema"); err != nil {
		log.Fatal(err)
		return
	}

	queries := db.New(dbConn)
	service := app.NewService(queries, dbConn)
	defer service.Shutdown()
	server := transport.NewServer(service, dbConn)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", server.Router); err != nil {
		log.Fatal(err)
	}
}
