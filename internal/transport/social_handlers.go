package transport

import (
	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"net/http"
	"share_word/internal/db"
	"share_word/internal/web/components"
	"strconv"
)

func (s *Server) handleFollowInvite(w http.ResponseWriter, r *http.Request) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	followedID := chi.URLParam(r, "id")

	_ = s.Service.FollowUser(r.Context(), currentUserID, followedID)
	http.Redirect(w, r, "/users/"+followedID, http.StatusSeeOther)
}

func (s *Server) handleViewProfile(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	s.renderProfile(w, r, targetID, false)
}

func (s *Server) handleFollow(w http.ResponseWriter, r *http.Request) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	targetID := chi.URLParam(r, "id")
	_ = s.Service.FollowUser(r.Context(), currentUserID, targetID)
	s.renderProfile(w, r, targetID, true)
}

func (s *Server) handleUnfollow(w http.ResponseWriter, r *http.Request) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	targetID := chi.URLParam(r, "id")
	_ = s.Service.UnfollowUser(r.Context(), currentUserID, targetID)
	s.renderProfile(w, r, targetID, true)
}

func (s *Server) renderProfile(w http.ResponseWriter, r *http.Request, targetID string, isFragment bool) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	targetUser, err := s.Service.GetUserByID(r.Context(), targetID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if limit <= 0 {
		limit = 10
	}

	followers, _ := s.Service.GetFollowers(r.Context(), targetID, limit, offset)
	following, _ := s.Service.GetFollowing(r.Context(), targetID, limit, offset)
	isFollowing, _ := s.Service.IsFollowing(r.Context(), currentUserID, targetID)
	puzzles, _ := s.Service.GetPuzzlesByOwner(r.Context(), targetID, 10, 0)

	component := components.Profile(targetUser, isFollowing, followers, following, limit, offset, puzzles)

	if isFragment {
		w.WriteHeader(http.StatusOK)
		datastar.NewSSE(w, r).PatchElementTempl(component)
	} else {
		var currentUser *db.User
		if currentUserID != "" {
			currentUser, _ = s.Service.GetUserByID(r.Context(), currentUserID)
		}
		components.Layout(component, currentUser).Render(r.Context(), w)
	}
}
