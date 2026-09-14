package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type SessionClaims struct {
	Email          string `json:"email"`
	OrganizationID uint   `json:"organizationId"`
	IssuedAt       int64  `json:"iat"`
	ExpiresAt      int64  `json:"exp"`
}

type SessionConfig struct {
	Secret string
	TTL    time.Duration
}

func (config SessionConfig) Validate() error {
	if strings.TrimSpace(config.Secret) == "" {
		return errors.New("session secret is required")
	}
	if config.TTL <= 0 {
		return errors.New("session ttl must be positive")
	}
	return nil
}

func SignSessionToken(secret string, claims SessionClaims) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("session secret is required")
	}

	if claims.Email == "" || claims.OrganizationID == 0 || claims.ExpiresAt <= 0 {
		return "", ErrInvalidToken
	}

	return signClaims(secret, claims)
}

func ValidateSessionToken(secret, token string, now time.Time) (SessionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return SessionClaims{}, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return SessionClaims{}, ErrInvalidToken
	}

	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return SessionClaims{}, ErrInvalidToken
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	expectedSignature := mac.Sum(nil)
	if !hmac.Equal(signatureBytes, expectedSignature) {
		return SessionClaims{}, ErrInvalidToken
	}

	var claims SessionClaims
	if err := json.NewDecoder(bytes.NewReader(payloadBytes)).Decode(&claims); err != nil {
		return SessionClaims{}, ErrInvalidToken
	}

	if claims.Email == "" || claims.OrganizationID == 0 || claims.ExpiresAt <= 0 {
		return SessionClaims{}, ErrInvalidToken
	}

	if now.UTC().Unix() > claims.ExpiresAt {
		return SessionClaims{}, ErrInvalidToken
	}

	return claims, nil
}

func signClaims(secret string, claims SessionClaims) (string, error) {
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	signature := mac.Sum(nil)

	return fmt.Sprintf("%s.%s",
		base64.RawURLEncoding.EncodeToString(payloadBytes),
		base64.RawURLEncoding.EncodeToString(signature),
	), nil
}

func RandomSecret() (string, error) {
	bytesBuffer := make([]byte, 32)
	if _, err := rand.Read(bytesBuffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytesBuffer), nil
}
