package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewJWTManager_PanicsOnShortSecret(t *testing.T) {
	assert.Panics(t, func() {
		NewJWTManager("short", time.Minute, time.Hour)
	})
}

func TestJWTManager_GenerateAndValidate_AccessToken(t *testing.T) {
	m := NewJWTManager("this-is-a-32-character-secret!!!", time.Minute, time.Hour)
	userID := uuid.New()

	pair, err := m.GenerateTokenPair(userID)
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, pair.RefreshToken)

	claims, err := m.ValidateAccessToken(pair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "access", claims.TokenType)
}

func TestJWTManager_GenerateAndValidate_RefreshToken(t *testing.T) {
	m := NewJWTManager("this-is-a-32-character-secret!!!", time.Minute, time.Hour)
	userID := uuid.New()

	pair, err := m.GenerateTokenPair(userID)
	assert.NoError(t, err)

	claims, err := m.ValidateRefreshToken(pair.RefreshToken)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestJWTManager_ValidateAccessToken_WrongType(t *testing.T) {
	m := NewJWTManager("this-is-a-32-character-secret!!!", time.Minute, time.Hour)
	pair, err := m.GenerateTokenPair(uuid.New())
	assert.NoError(t, err)

	_, err = m.ValidateAccessToken(pair.RefreshToken)
	assert.Error(t, err)

	_, err = m.ValidateRefreshToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestJWTManager_ValidateAccessToken_Malformed(t *testing.T) {
	m := NewJWTManager("this-is-a-32-character-secret!!!", time.Minute, time.Hour)

	_, err := m.ValidateAccessToken("not-a-real-token")
	assert.Error(t, err)
}

func TestJWTManager_ValidateAccessToken_Expired(t *testing.T) {
	m := NewJWTManager("this-is-a-32-character-secret!!!", -time.Minute, time.Hour)
	pair, err := m.GenerateTokenPair(uuid.New())
	assert.NoError(t, err)

	_, err = m.ValidateAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestJWTManager_ValidateAccessToken_WrongSecret(t *testing.T) {
	m1 := NewJWTManager("this-is-a-32-character-secret!!!", time.Minute, time.Hour)
	m2 := NewJWTManager("a-totally-different-32-char-key!", time.Minute, time.Hour)

	pair, err := m1.GenerateTokenPair(uuid.New())
	assert.NoError(t, err)

	_, err = m2.ValidateAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestJWTManager_RefreshExpiry(t *testing.T) {
	m := NewJWTManager("this-is-a-32-character-secret!!!", time.Minute, 168*time.Hour)
	assert.Equal(t, 168*time.Hour, m.RefreshExpiry())
}
