package transport

import (
	"net/http"
	"share_word/internal/web/components"

	"github.com/starfederation/datastar-go/datastar"
)

func (s *Server) handleSignups(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username        string `json:"username"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if payload.Password != payload.ConfirmPassword {
		sse := datastar.NewSSE(w, r, datastar.WithCompression())
		sse.PatchElementTempl(components.Signup("Passwords do not match"))
		return
	}

	user, err := s.Service.RegisterUser(r.Context(), payload.Username, payload.Password)
	if err != nil {
		sse := datastar.NewSSE(w, r, datastar.WithCompression())
		sse.PatchElementTempl(components.Signup(err.Error()))
		return
	}

	// Security best practice: renew token on login/signup
	if err := s.SessionManager.RenewToken(r.Context()); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	s.SessionManager.Put(r.Context(), "userID", user.ID)

	// Ensure headers are written before SSE starts
	w.WriteHeader(http.StatusOK)
	sse := datastar.NewSSE(w, r, datastar.WithCompression())
	sse.Redirect("/")
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := datastar.ReadSignals(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.Service.AuthenticateUser(r.Context(), payload.Username, payload.Password)
	if err != nil {
		sse := datastar.NewSSE(w, r, datastar.WithCompression())
		sse.PatchElementTempl(components.Login("Invalid username or password"))
		return
	}

	// Security best practice: renew token on login/signup
	if err := s.SessionManager.RenewToken(r.Context()); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	s.SessionManager.Put(r.Context(), "userID", user.ID)

	// Ensure headers are written before SSE starts
	w.WriteHeader(http.StatusOK)
	sse := datastar.NewSSE(w, r, datastar.WithCompression())
	sse.Redirect("/")
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.SessionManager.Destroy(r.Context())
	datastar.NewSSE(w, r, datastar.WithCompression()).Redirect("/login")
}
