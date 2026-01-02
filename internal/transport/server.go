package transport

import (
	"database/sql"
	"net/http"
	"share_word/internal/app"
	"share_word/internal/web/components"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Service        *app.Service
	Router         *chi.Mux
	SessionManager *scs.SessionManager
}

func NewServer(svc *app.Service, db *sql.DB) *Server {
	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db)
	sessionManager.Lifetime = time.Hour * 24 * 7 * 6

	s := &Server{
		Service:        svc,
		Router:         chi.NewRouter(),
		SessionManager: sessionManager,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.Router.Use(middleware.Logger)
	s.Router.Use(s.SessionManager.LoadAndSave)
	s.Router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB limit
			next.ServeHTTP(w, r)
		})
	})

	fs := http.FileServer(http.Dir("internal/web/static"))
	s.Router.Handle("/static/*", http.StripPrefix("/static/", fs))

	s.Router.Get("/", s.handleHome)

	s.Router.Get("/signup", func(w http.ResponseWriter, r *http.Request) {
		component := components.Layout(components.Signup(""), nil)
		component.Render(r.Context(), w)
	})
	s.Router.Post("/signup", s.handleSignups)

	s.Router.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		components.Layout(components.Login(""), nil).Render(r.Context(), w)
	})
	s.Router.Post("/login", s.handleLogin)

	s.Router.Post("/logout", s.handleLogout)

	// Puzzles
	s.Router.Post("/puzzles", s.handleCreatePuzzle)
	s.Router.Get("/puzzles/{id}", s.handleViewPuzzle)
	s.Router.Post("/puzzles/{id}/cells/{x}/{y}/toggle-block", s.handleToggleBlock)
	s.Router.Post("/puzzles/{id}/cells/{x}/{y}/update", s.handleUpdateCell)
	s.Router.Get("/puzzles/{id}/clues/{number}/{direction}/edit", s.handleEditClue)
	s.Router.Get("/puzzles/{id}/clues/{number}/{direction}/view", s.handleViewClueItem)
	s.Router.Post("/puzzles/{id}/clues/{number}/{direction}/live", s.handleLiveUpdateClue)
	s.Router.Post("/puzzles/{id}/clues/{number}/{direction}/save", s.handleSaveClue)

	// Profiles
	s.Router.Get("/users/{id}", s.handleViewProfile)
	s.Router.Post("/users/{id}/follow", s.handleFollow)
	s.Router.Post("/users/{id}/unfollow", s.handleUnfollow)
}

