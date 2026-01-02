package transport

import (
	"net/http"
	"share_word/internal/db"
	"share_word/internal/web/components"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleViewProfile(w http.ResponseWriter, r *http.Request) {
	currentUserID := s.SessionManager.GetString(r.Context(), "userID")
	targetUserID := chi.URLParam(r, "id")

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, _ := strconv.ParseInt(limitStr, 10, 64)
	offset, _ := strconv.ParseInt(offsetStr, 10, 64)

	if limit <= 0 {
		limit = 10
	}

	targetUser, err := s.Service.GetUserByID(r.Context(), targetUserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	followers, _ := s.Service.GetFollowers(r.Context(), targetUserID, limit, offset)
	following, _ := s.Service.GetFollowing(r.Context(), targetUserID, limit, offset)
	isFollowing := false
	var currentUser *db.User

	if currentUserID != "" {
		u, _ := s.Service.GetUserByID(r.Context(), currentUserID)
		currentUser = u
		val, _ := s.Service.Queries.IsFollowing(r.Context(), db.IsFollowingParams{
			FollowerID: currentUserID,
			FollowedID: targetUserID,
		})
		isFollowing = val > 0
	}

	puzzles, _ := s.Service.Queries.GetPuzzlesByOwner(r.Context(), db.GetPuzzlesByOwnerParams{
		OwnerID: targetUser.ID,
		Limit:   10,
		Offset:  0,
	})

	components.Layout(components.Profile(targetUser, isFollowing, followers, following, limit, offset, puzzles), currentUser).Render(r.Context(), w)
}

func (s *Server) handleFollow(w http.ResponseWriter, r *http.Request) {
	followerID := s.SessionManager.GetString(r.Context(), "userID")
	followedID := chi.URLParam(r, "id")

	if followerID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	_ = s.Service.FollowUser(r.Context(), followerID, followedID)
	http.Redirect(w, r, "/users/"+followedID, http.StatusSeeOther)
}

func (s *Server) handleUnfollow(w http.ResponseWriter, r *http.Request) {
	followerID := s.SessionManager.GetString(r.Context(), "userID")
	followedID := chi.URLParam(r, "id")

	if followerID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	_ = s.Service.UnfollowUser(r.Context(), followerID, followedID)
	http.Redirect(w, r, "/users/"+followedID, http.StatusSeeOther)
}