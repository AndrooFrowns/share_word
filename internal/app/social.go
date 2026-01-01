package app

import (
	"context"
	"errors"
	"share_word/internal/db"
)

func (s *Service) FollowUser(ctx context.Context, followerID, followedID string) error {
	if followedID == followerID {
		return errors.New("cannot follow yourself")
	}

	return s.queries.FollowUser(ctx, db.FollowUserParams{
		FollowerID: followerID,
		FollowedID: followedID,
	})
}

func (s *Service) UnfollowUser(ctx context.Context, followerID, followedID string) error {
	return s.queries.UnfollowUser(
		ctx,
		db.UnfollowUserParams{
			FollowerID: followerID,
			FollowedID: followedID,
		},
	)
}

func (s *Service) GetFollowing(ctx context.Context, userID string, limit, offset int64) ([]db.User, error) {
	return s.queries.GetFollowing(ctx, db.GetFollowingParams{
		FollowerID: userID,
		Limit:      limit,
		Offset:     offset,
	})
}

func (s *Service) GetFollowers(ctx context.Context, userID string, limit, offset int64) ([]db.User, error) {
	return s.queries.GetFollowers(ctx, db.GetFollowersParams{
		FollowedID: userID,
		Limit:      limit,
		Offset:     offset,
	})
}

func (s *Service) IsFollowing(ctx context.Context, followerID, followedID string) (bool, error) {
	isFollowing, err := s.queries.IsFollowing(ctx, db.IsFollowingParams{
		FollowerID: followerID,
		FollowedID: followedID,
	})

	if err != nil {
		return false, err
	}

	return isFollowing > 0, nil
}
