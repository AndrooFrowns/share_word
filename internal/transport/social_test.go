package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProfileFlow(t *testing.T) {
	server, dbConn := setupTestServer(t)
	defer dbConn.Close()
	ctx := context.Background()

	// 1. Create User A (Follower) and User B (Target)
	userA, _ := server.Service.RegisterUser(ctx, "user_a", "password123456")
	userB, _ := server.Service.RegisterUser(ctx, "user_b", "password123456")

	// 2. Login as User A to get a session
	loginBody := fmt.Sprintf(`{"username":"user_a", "password":"password123456"}`)
	reqLogin := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	server.Router.ServeHTTP(wLogin, reqLogin)
	cookie := wLogin.Result().Header.Get("Set-Cookie")

	t.Run("view profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/"+userB.ID, nil)
		req.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "user_b") {
			t.Error("profile page should contain username")
		}
		if !strings.Contains(w.Body.String(), "Follow") {
			t.Error("should show follow button when not following")
		}
	})

	t.Run("follow user via AJAX", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/"+userB.ID+"/follow", nil)
		req.Header.Set("Cookie", cookie)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Unfollow") {
			t.Error("response fragment should contain unfollow button")
		}

		// Verify DB state
		isFollowing, _ := server.Service.IsFollowing(ctx, userA.ID, userB.ID)
		if !isFollowing {
			t.Error("user should be following now")
		}
	})

	t.Run("unfollow user via AJAX", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/"+userB.ID+"/unfollow", nil)
		req.Header.Set("Cookie", cookie)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Follow") {
			t.Error("response fragment should contain follow button")
		}

		// Verify DB state
		isFollowing, _ := server.Service.IsFollowing(ctx, userA.ID, userB.ID)
		if isFollowing {
			t.Error("user should not be following now")
		}
	})

	t.Run("paging followers", func(t *testing.T) {
		// Create 12 followers for User B
		for i := 0; i < 12; i++ {
			u, _ := server.Service.RegisterUser(ctx, fmt.Sprintf("follower_%d", i), "password123456")
			_ = server.Service.FollowUser(ctx, u.ID, userB.ID)
		}

		// Request page 1 (default limit 10)
		req := httptest.NewRequest("GET", "/users/"+userB.ID+"?limit=10&offset=0", nil)
		req.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		if !strings.Contains(w.Body.String(), "Next") {
			t.Error("page 1 should have a 'Next' link")
		}
		if strings.Contains(w.Body.String(), "Prev") {
			t.Error("page 1 should not have a 'Prev' link")
		}

		// Request page 2
		req2 := httptest.NewRequest("GET", "/users/"+userB.ID+"?limit=10&offset=10", nil)
		req2.Header.Set("Cookie", cookie)
		w2 := httptest.NewRecorder()
		server.Router.ServeHTTP(w2, req2)

		if !strings.Contains(w2.Body.String(), "Prev") {
			t.Error("page 2 should have a 'Prev' link")
		}
	})
}
