package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"share_word/internal/app"
	"share_word/internal/db"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	dbConn, _ := sql.Open("sqlite", ":memory:")
	goose.SetDialect("sqlite3")
	goose.Up(dbConn, "../../sql/schema/")
	return dbConn
}

func setupTestServer(t *testing.T) (*Server, *db.Queries, func()) {
	dbConn := setupTestDB(t)
	queries := db.New(dbConn)
	service := app.NewService(queries, dbConn)
	server := NewServer(service, dbConn, false)

	cleanup := func() {
		service.Shutdown()
		dbConn.Close()
	}

	return server, queries, cleanup
}

func TestLoginFlow(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	password := "password123456"
	_, err := server.Service.RegisterUser(ctx, "testuser", password)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"username":"testuser", "password":"password123456"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")

	rr := httptest.NewRecorder()
	server.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), `window.location.href = "/"`) {
		t.Errorf("expected redirect to root in response, got: %s", rr.Body.String())
	}

	cookieHeader := rr.Header().Get("Set-Cookie")
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", cookieHeader)
	rr = httptest.NewRecorder()
	server.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Create a New Puzzle") {
		t.Errorf("expected dashboard (Create a New Puzzle) in response, got body length %d", len(rr.Body.String()))
	}
}

func TestSignupFlow(t *testing.T) {
	server, queries, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"username":"newuser", "password":"password123456", "confirmPassword":"password123456"}`
	req := httptest.NewRequest("POST", "/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")

	rr := httptest.NewRecorder()
	server.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `window.location.href = "/"`) {
		t.Errorf("expected redirect to root in response, got: %s", rr.Body.String())
	}

	_, err := queries.GetUserByUsername(context.Background(), "newuser")
	if err != nil {
		t.Error("user was not created")
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	server.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Collaborative Crosswords") {
		t.Errorf("expected welcome message for guest, got body length %d", len(rr.Body.String()))
	}
}

func TestLogout(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = server.Service.RegisterUser(ctx, "logoutuser", "password123456")

	loginBody := `{"username":"logoutuser", "password":"password123456"}`
	loginReq := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.Header.Set("Datastar-Request", "true")
	loginRR := httptest.NewRecorder()
	server.Router.ServeHTTP(loginRR, loginReq)

	cookieHeader := loginRR.Header().Get("Set-Cookie")

	logoutReq := httptest.NewRequest("POST", "/logout", nil)
	logoutReq.Header.Set("Cookie", cookieHeader)
	logoutRR := httptest.NewRecorder()
	server.Router.ServeHTTP(logoutRR, logoutReq)

	if logoutRR.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", logoutRR.Code)
	}
	if !strings.Contains(logoutRR.Body.String(), `window.location.href = "/login"`) {
		t.Errorf("expected redirect to login in response, got: %s", logoutRR.Body.String())
	}
}

func TestLoginFailure_Feedback(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"username":"wrong", "password":"wrong_password_long_enough"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")

	rr := httptest.NewRecorder()
	server.Router.ServeHTTP(rr, req)

	if !strings.Contains(rr.Body.String(), "Invalid username or password") {
		t.Error("expected error message in response")
	}
}

func TestSignup_DuplicateUsername(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = server.Service.RegisterUser(ctx, "existing", "password123456")

	body := `{"username":"existing", "password":"newpassword123456", "confirmPassword":"newpassword123456"}`
	req := httptest.NewRequest("POST", "/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")

	rr := httptest.NewRecorder()
	server.Router.ServeHTTP(rr, req)

	if !strings.Contains(rr.Body.String(), "username taken") {
		t.Errorf("expected username taken error, got: %s", rr.Body.String())
	}
}
