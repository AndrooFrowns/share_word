package transport

import (
	"net/http"
	"share_word/internal/web/components"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	userID := s.SessionManager.GetString(r.Context(), "userID")
	if userID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := s.Service.GetUserByID(r.Context(), userID)
	if err != nil {
		s.SessionManager.Remove(r.Context(), "userID")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	myPuzzles, _ := s.Service.GetPuzzlesByOwner(r.Context(), user.ID, 10, 0)
	followingPuzzles, _ := s.Service.GetPuzzlesFromFollowing(r.Context(), user.ID, 10, 0)

	components.Layout(components.Dashboard(user, myPuzzles, followingPuzzles), user).Render(r.Context(), w)
}
