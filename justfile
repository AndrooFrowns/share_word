# Justfile for ShareWord

# Default task: list all recipes
default:
    @just --list

# Generate type-safe Go code from SQL and Templ files
generate:
    sqlc generate
    templ generate

# Create a new migration file
migrate-create name:
    goose -dir sql/schema create {{name}} sql

# Apply migrations to local DB
migrate-up:
    goose -dir sql/schema sqlite3 shareword.db up

# Rollback last migration
migrate-down:
    goose -dir sql/schema sqlite3 shareword.db down

# Run all unit tests
test:
    go test -v ./...

# Build the server binary
build: generate
    go build -o shareword cmd/server/main.go

# Run the server in development mode
run: generate
    go run cmd/server/main.go

# Clean up binaries and local database
clean:
    rm -f shareword shareword.db
