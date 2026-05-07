package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	TokenType string    `json:"token_type"` // "access" | "refresh"
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTManager(secret string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	if len(secret) < 32 {
		panic("JWT secret must be at least 32 characters")
	}
	return &JWTManager{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (m *JWTManager) GenerateTokenPair(userID uuid.UUID) (TokenPair, error) {
	access, err := m.generate(userID, "access", m.accessExpiry)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := m.generate(userID, "refresh", m.refreshExpiry)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (m *JWTManager) ValidateAccessToken(token string) (*Claims, error) {
	return m.validate(token, "access")
}

func (m *JWTManager) ValidateRefreshToken(token string) (*Claims, error) {
	return m.validate(token, "refresh")
}

func (m *JWTManager) RefreshExpiry() time.Duration {
	return m.refreshExpiry
}

func (m *JWTManager) generate(userID uuid.UUID, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(), // jti: ensures each token is unique
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *JWTManager) validate(tokenStr, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type")
	}
	return claims, nil
}
