package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"share_word/internal/app"
	"share_word/internal/db"

	_ "modernc.org/sqlite"
)

func main() {
	dbConn, err := sql.Open("sqlite", "shareword.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	queries := db.New(dbConn)
	service := app.NewService(queries, dbConn)
	ctx := context.Background()

	users := []struct {
		username string
		password string
	}{
		{"alice", "password123456"},
		{"bob", "password123456"},
		{"charlie", "password123456"},
	}

	fmt.Println("Seeding users...")
	createdUsers := make(map[string]string)

	for _, u := range users {
		user, err := service.RegisterUser(ctx, u.username, u.password)
		if err != nil {
			fmt.Printf("User %s already exists or error: %v\n", u.username, err)
			// If user exists, fetch them to get the ID
			existing, _ := queries.GetUserByUsername(ctx, u.username)
			createdUsers[u.username] = existing.ID
			continue
		}
		fmt.Printf("Created user: %s (ID: %s)\n", user.Username, user.ID)
		createdUsers[u.username] = user.ID
	}

	// Create some initial follows
	fmt.Println("\nSeeding relationships...")
	_ = service.FollowUser(ctx, createdUsers["alice"], createdUsers["bob"])
	_ = service.FollowUser(ctx, createdUsers["bob"], createdUsers["charlie"])
	fmt.Println("Alice follows Bob.")
	fmt.Println("Bob follows Charlie.")

	fmt.Println("\nSeeding complete!")
	fmt.Println("Use these links to check profiles once the server is running:")
	for name, id := range createdUsers {
		fmt.Printf("- %s: http://localhost:8080/users/%s\n", name, id)
	}
}
