package app

import (
	"context"
	"testing"
)

func TestSocialFlow(t *testing.T) {
	queries, conn := setupTestDB(t)
	defer conn.Close()
	svc := NewService(queries)
	ctx := context.Background()

	// 1. Setup: Create two users
	userA, _ := svc.RegisterUser(ctx, "user_a", "password12345")
	userB, _ := svc.RegisterUser(ctx, "user_b", "password12345")

	t.Run("follow user", func(t *testing.T) {
		err := svc.FollowUser(ctx, userA.ID, userB.ID)
		if err != nil {
			t.Fatalf("failed to follow: %v", err)
		}

		following, _ := svc.GetFollowing(ctx, userA.ID, 1, 0)
		if len(following) != 1 || following[0].ID != userB.ID {
			t.Errorf("expected following user_b, got %v", following)
		}

		isFollowing, _ := svc.IsFollowing(ctx, userA.ID, userB.ID)
		if !isFollowing {
			t.Error("expected IsFollowing to be true")
		}
	})

	t.Run("cannot follow self", func(t *testing.T) {
		err := svc.FollowUser(ctx, userA.ID, userA.ID)
		if err == nil {
			t.Error("expected error when following self, got nil")
		}
	})

	t.Run("unfollow user", func(t *testing.T) {
		err := svc.UnfollowUser(ctx, userA.ID, userB.ID)
		if err != nil {
			t.Fatalf("failed to unfollow: %v", err)
		}

		following, _ := svc.GetFollowing(ctx, userA.ID, 1, 0)
		if len(following) != 0 {
			t.Errorf("expected 0 following, got %d", len(following))
		}
	})

	t.Run("paging and offset", func(t *testing.T) {
		user1, _ := svc.RegisterUser(ctx, "user_1", "password12345")
		user2, _ := svc.RegisterUser(ctx, "user_2", "password12345")
		user3, _ := svc.RegisterUser(ctx, "user_3", "password12345")
		user4, _ := svc.RegisterUser(ctx, "user_4", "password12345")
		_ = svc.FollowUser(ctx, userA.ID, user1.ID)
		_ = svc.FollowUser(ctx, userA.ID, user2.ID)
		_ = svc.FollowUser(ctx, userA.ID, user3.ID)
		_ = svc.FollowUser(ctx, userA.ID, user4.ID)

		// Page 1
		following, err := svc.GetFollowing(ctx, userA.ID, 2, 0)
		if err != nil || len(following) != 2 {
			t.Errorf("expected 2 users on page 1, got %d", len(following))
		}

		// Page 2
		following, err = svc.GetFollowing(ctx, userA.ID, 2, 2)
		if err != nil || len(following) != 2 {
			t.Errorf("expected 2 users on page 2, got %d", len(following))
		}
	})
}
