package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"agronode/backend/internal/auth"
	"agronode/backend/internal/repositories"
)

type AuthLoginService struct {
	users   repositories.UserRepository
	session auth.SessionConfig
}

func NewAuthLoginService(users repositories.UserRepository, session auth.SessionConfig) *AuthLoginService {
	return &AuthLoginService{users: users, session: session}
}

func (service *AuthLoginService) Login(email, password string, now time.Time) (auth.SessionClaims, string, error) {
	if service.users == nil {
		return auth.SessionClaims{}, "", errors.New("user repository is not configured")
	}

	user, err := service.users.GetByEmail(context.Background(), strings.TrimSpace(email))
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return auth.SessionClaims{}, "", auth.ErrInvalidCredentials
		}

		return auth.SessionClaims{}, "", err
	}

	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		return auth.SessionClaims{}, "", auth.ErrInvalidCredentials
	}

	claims := auth.SessionClaims{
		Email:          user.Email,
		OrganizationID: user.OrganizationID,
		IssuedAt:       now.UTC().Unix(),
		ExpiresAt:      now.UTC().Add(service.session.TTL).Unix(),
	}

	token, err := auth.SignSessionToken(service.session.Secret, claims)
	if err != nil {
		return auth.SessionClaims{}, "", err
	}

	return claims, token, nil
}
