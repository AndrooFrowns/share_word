package app

import (
	"context"
	"share_word/internal/db"
)

func (s *Service) FollowUser(ctx context.Context, followerID, followedID string) error {
	if followerID == followedID {
		return context.Canceled
	}
	return s.Queries.FollowUser(ctx, db.FollowUserParams{
		FollowerID: followerID,
		FollowedID: followedID,
	})
}

func (s *Service) UnfollowUser(ctx context.Context, followerID, followedID string) error {
	return s.Queries.UnfollowUser(ctx, db.UnfollowUserParams{
		FollowerID: followerID,
		FollowedID: followedID,
	})
}

func (s *Service) GetFollowers(ctx context.Context, userID string, limit, offset int64) ([]db.User, error) {
	return s.Queries.GetFollowers(ctx, db.GetFollowersParams{
		FollowedID: userID,
		Limit:      limit,
		Offset:     offset,
	})
}

func (s *Service) GetFollowing(ctx context.Context, userID string, limit, offset int64) ([]db.User, error) {
	return s.Queries.GetFollowing(ctx, db.GetFollowingParams{
		FollowerID: userID,
		Limit:      limit,
		Offset:     offset,
	})
}

func (s *Service) IsFollowing(ctx context.Context, followerID, followedID string) (bool, error) {
	count, err := s.Queries.IsFollowing(ctx, db.IsFollowingParams{
		FollowerID: followerID,
		FollowedID: followedID,
	})
	return count > 0, err
}
