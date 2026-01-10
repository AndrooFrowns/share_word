package transport

import (
	"database/sql"
	"io/fs"
	"net/http"
	"share_word/internal/app"
	"share_word/internal/web/components"
	"share_word/internal/web/static"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/CAFxX/httpcompression/contrib/andybalholm/brotli"
	"github.com/CAFxX/httpcompression/contrib/compress/gzip"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Service        *app.Service
	Router         *chi.Mux
	SessionManager *scs.SessionManager
	IsProd         bool
}

func NewServer(svc *app.Service, db *sql.DB, isProd bool) *Server {
	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db)
	sessionManager.Lifetime = time.Hour * 24 * 7 * 6
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = isProd

	s := &Server{
		Service:        svc,
		Router:         chi.NewRouter(),
		SessionManager: sessionManager,
		IsProd:         isProd,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.Router.Use(middleware.Logger)
	s.Router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
			next.ServeHTTP(w, r)
		})
	})

	// Add Brotli/Gzip compression middleware
	brotliComp, _ := brotli.New(brotli.Options{})
	gzipComp, _ := gzip.New(gzip.Options{})
	compress, _ := httpcompression.Adapter(
		httpcompression.Compressor(brotli.Encoding, 1, brotliComp),
		httpcompression.Compressor(gzip.Encoding, 0, gzipComp),
	)
	s.Router.Use(compress)

	// Use embedded static assets
	staticFS, _ := fs.Sub(static.Assets, ".")
	fs := http.FileServer(http.FS(staticFS))
	s.Router.Handle("/static/*", http.StripPrefix("/static/", fs))

	s.Router.Get("/health", s.handleHealth)

	// SSE Streams
	s.Router.Group(func(r chi.Router) {
		r.Use(s.SessionManager.LoadAndSave)
		r.Get("/puzzles/{id}/stream", s.handlePuzzleStreamSolve)
		r.Get("/puzzles/{id}/edit/stream", s.handlePuzzleStreamEdit)
	})

	// App Routes
	s.Router.Group(func(r chi.Router) {
		r.Use(s.SessionManager.LoadAndSave)

		r.Get("/", s.handleHome)
		r.Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			components.Layout(components.Signup(""), nil, true).Render(r.Context(), w)
		})
		r.Post("/signup", s.handleSignups)
		r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
			components.Layout(components.Login(""), nil, true).Render(r.Context(), w)
		})
		r.Post("/login", s.handleLogin)
		r.Post("/logout", s.handleLogout)

		// Puzzles
		r.Post("/puzzles", s.handleCreatePuzzle)
		r.Get("/puzzles/{id}", s.handleViewPuzzleSolve)
		r.Get("/puzzles/{id}/edit", s.handleViewPuzzleEdit)
		r.Post("/puzzles/{id}/cells/{x}/{y}/set-block", s.handleSetBlock)
		r.Post("/puzzles/{id}/cells/{x}/{y}/set-block/{state}", s.handleSetBlockState)
		r.Post("/puzzles/{id}/cells/{x}/{y}/update", s.handleUpdateCell)
		r.Post("/puzzles/{id}/cells/{x}/{y}/focus", s.handleFocusCell)
		r.Post("/puzzles/{id}/input", s.handlePuzzleInput)
		r.Post("/puzzles/{id}/resize", s.handleResizePuzzle)
		r.Post("/puzzles/{id}/import", s.handleImportPuzzle)
		r.Get("/puzzles/{id}/clues/{number}/{direction}/edit", s.handleEditClue)
				r.Post("/puzzles/{id}/clues/{number}/{direction}/save", s.handleSaveClue)
				r.Post("/puzzles/{id}/clues/{number}/{direction}/focus", s.handleFocusClue)
		
		// Profiles
		r.Get("/users/{id}", s.handleViewProfile)
		r.Post("/users/{id}/follow", s.handleFollow)
		r.Post("/users/{id}/unfollow", s.handleUnfollow)
	})
}
