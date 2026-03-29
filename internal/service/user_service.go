package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"eshop/internal/repository"
	"eshop/internal/utils"
)

type UserService struct {
	repo      *repository.UserRepository
	jwtSecret string
}

func NewUserService(repo *repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{repo: repo, jwtSecret: jwtSecret}
}

func (s *UserService) Register(ctx context.Context, username, password, nickname string) (*repository.User, error) {
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("username is required")
	}
	if len(password) < 6 {
		return nil, errors.New("password length must be >= 6")
	}
	if strings.TrimSpace(nickname) == "" {
		nickname = username
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := &repository.User{
		Username:     strings.TrimSpace(username),
		PasswordHash: hash,
		Nickname:     strings.TrimSpace(nickname),
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.repo.GetByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return "", err
	}
	if !utils.CheckPassword(user.PasswordHash, password) {
		return "", errors.New("invalid username or password")
	}
	token, err := utils.GenerateJWT(s.jwtSecret, user.ID, user.Username, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return token, nil
}

func (s *UserService) GetProfile(ctx context.Context, userID uint64) (*repository.User, error) {
	if userID == 0 {
		return nil, errors.New("invalid user id")
	}
	return s.repo.GetByID(ctx, userID)
}
