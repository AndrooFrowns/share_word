package app

import (
	"context"
	"errors"
	"fmt"
	"share_word/internal/db"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) AuthenticateUser(ctx context.Context, username, password string) (*db.User, error) {
	err_msg := errors.New("invalid username or password")

	user, err := s.Queries.GetUserByUsername(ctx, username)
	if err != nil {
		// generic on purpose to avoid username harvesting
		return nil, err_msg
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, err_msg
	}

	return &user, nil
}

func (s *Service) RegisterUser(ctx context.Context, username, password string) (*db.User, error) {
	goodUsername := isValidUsername(username)
	if !goodUsername {
		return nil, errors.New("invalid username")
	}
	goodPassword := isValidPassword(password)
	if !goodPassword {
		return nil, errors.New("invalid password")
	}

	_, err := s.Queries.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, errors.New("username taken")
	}

	hashed_password, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	id := uuid.NewString()

	created_user, err := s.Queries.CreateUser(ctx, db.CreateUserParams{
		ID:           id,
		Username:     username,
		PasswordHash: string(hashed_password),
		CreatedAt:    time.Now().UTC().Round(0),
	})

	return &created_user, nil
}

func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 26 {
		return false
	}

	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return true
}

func isValidPassword(password string) bool {
	return len(password) >= 12 && len(password) <= 72
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*db.User, error) {
	user, err := s.Queries.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
