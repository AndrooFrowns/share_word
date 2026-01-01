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

func setupTestServer(t *testing.T) (*Server, *sql.DB) {
	dbConn, _ := sql.Open("sqlite", ":memory:")
	goose.SetDialect("sqlite3")
	// Run migrations
	if err := goose.Up(dbConn, "../../sql/schema"); err != nil {
		t.Fatal(err)
	}

	queries := db.New(dbConn)
	service := app.NewService(queries, dbConn)
	server := NewServer(service, dbConn)
	return server, dbConn
}

func TestLoginFlow(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	// 1. Create a user first (Register)
	username := "test_user"
	password := "a-very-long-valid-password"
	_, err := server.Service.RegisterUser(context.Background(), username, password)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// 2. Simulate Login POST request
	loginBody := `{"username":"test_user", "password":"a-very-long-valid-password"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	// Verify Redirect instruction sent via SSE
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (SSE), got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "window.location.href = \"/\"") {
		t.Errorf("expected datastar redirect script in response, got: %s", w.Body.String())
	}

	// 3. Capture the Session Cookie
	cookie := w.Result().Header.Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("no session cookie set in response")
	}

	// 4. Try to access "/" with the cookie
	reqHome := httptest.NewRequest("GET", "/", nil)
	reqHome.Header.Set("Cookie", cookie)
	wHome := httptest.NewRecorder()
	server.Router.ServeHTTP(wHome, reqHome)

	if wHome.Code != http.StatusOK {
		t.Errorf("expected status 200 for home page, got %d", wHome.Code)
	}
	if !strings.Contains(wHome.Body.String(), "Welcome, test_user") {
		t.Errorf("home page did not contain welcome message. Got: %s", wHome.Body.String())
	}
}

func TestSignupFlow(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	signupBody := `{"username":"new_user", "password":"a-very-long-valid-password", "confirmPassword":"a-very-long-valid-password"}`
	req := httptest.NewRequest("POST", "/signup", strings.NewReader(signupBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	cookie := w.Result().Header.Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("no session cookie set after signup")
	}

	// Verify home page access
	reqHome := httptest.NewRequest("GET", "/", nil)
	reqHome.Header.Set("Cookie", cookie)
	wHome := httptest.NewRecorder()
	server.Router.ServeHTTP(wHome, reqHome)

	if !strings.Contains(wHome.Body.String(), "Welcome, new_user") {
		t.Error("failed to log in automatically after signup")
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect for unauthorized access, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/login" {
		t.Errorf("expected redirect to /login, got %s", w.Header().Get("Location"))
	}
}

func TestLogout(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	// 1. Setup session
	username := "logout_user"
	password := "a-very-long-valid-password"
	_, _ = server.Service.RegisterUser(context.Background(), username, password)

	// We need to use the router to get the cookie properly
	loginBody := `{"username":"logout_user", "password":"a-very-long-valid-password"}`
	reqLogin := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	reqLogin.Header.Set("Accept", "text/event-stream")
	wLogin := httptest.NewRecorder()
	server.Router.ServeHTTP(wLogin, reqLogin)
	cookie := wLogin.Result().Header.Get("Set-Cookie")

	// 2. Logout
	reqLogout := httptest.NewRequest("POST", "/logout", nil)
	reqLogout.Header.Set("Cookie", cookie)
	wLogout := httptest.NewRecorder()
	server.Router.ServeHTTP(wLogout, reqLogout)

	// 3. Verify cookie is destroyed or at least access is denied
	newCookie := wLogout.Result().Header.Get("Set-Cookie")
	
	reqHome := httptest.NewRequest("GET", "/", nil)
	if newCookie != "" {
		reqHome.Header.Set("Cookie", newCookie)
	} else {
		// some session managers don't send a new cookie on destroy, 
		// they just stop recognizing the old one.
		reqHome.Header.Set("Cookie", cookie) 
	}
	
	wHome := httptest.NewRecorder()
	server.Router.ServeHTTP(wHome, reqHome)

	if wHome.Code != http.StatusSeeOther {
		t.Errorf("expected redirect after logout, got %d", wHome.Code)
	}
}

func TestLoginFailure_Feedback(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	// Register a user
	server.Service.RegisterUser(context.Background(), "alice", "valid-password-12345")

	// Try to login with WRONG password
	loginBody := `{"username":"alice", "password":"wrong-password"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for SSE error response, got %d", w.Code)
	}

	// Verify the response contains the error message we expect to see on the page
	body := w.Body.String()
	if !strings.Contains(body, "Invalid username or password") {
		t.Errorf("expected error message in fragment, got: %s", body)
	}
	// Verify it didn't set a session cookie
	if w.Result().Header.Get("Set-Cookie") != "" {
		t.Error("should not set a session cookie on failed login")
	}
}

func TestSignup_DuplicateUsername(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	// Register alice once
	server.Service.RegisterUser(context.Background(), "alice", "valid-password-12345")

	// Try to register alice AGAIN
	signupBody := `{"username":"alice", "password":"another-password-12345", "confirmPassword":"another-password-12345"}`
	req := httptest.NewRequest("POST", "/signup", strings.NewReader(signupBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "username taken") {
		t.Errorf("expected 'username taken' error, got: %s", body)
	}
}

func TestMalformedJSON(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	// Send garbage JSON
	req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"username": "oops", "password": `)) // Incomplete JSON
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for malformed JSON, got %d", w.Code)
	}
}

func TestOversizedRequest(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()

	// Send a very large body (over 1MB)
	largeBody := strings.Repeat("a", 1024*1024+100)
	req := httptest.NewRequest("POST", "/login", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	// Since the body is never read if the content-length is too large (or if it hits the limit during read),
	// we expect some kind of error. With MaxBytesReader, it typically results in a 413 or a read error.
	if w.Code != http.StatusRequestEntityTooLarge && w.Code != http.StatusBadRequest {
		t.Errorf("expected 413 or 400 for oversized request, got %d", w.Code)
	}
}