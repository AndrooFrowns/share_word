package transport

import (
	"net/http"
	"share_word/internal/db"
	"share_word/internal/web/components"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	userID := s.SessionManager.GetString(r.Context(), "userID")
	if userID == "" {
		components.Layout(components.Home(nil), nil).Render(r.Context(), w)
		return
	}

	user, err := s.Service.GetUserByID(r.Context(), userID)
	if err != nil {
		s.SessionManager.Remove(r.Context(), "userID")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Fetch data for the dashboard
	limit, offset := int64(10), int64(0)
	myPuzzles, _ := s.Service.Queries.GetPuzzlesByOwner(r.Context(), db.GetPuzzlesByOwnerParams{OwnerID: userID, Limit: limit, Offset: offset})
	followingPuzzles, _ := s.Service.Queries.GetPuzzlesFromFollowing(r.Context(), db.GetPuzzlesFromFollowingParams{FollowerID: userID, Limit: limit, Offset: offset})

	components.Layout(components.Dashboard(user, myPuzzles, followingPuzzles), user).Render(r.Context(), w)
}