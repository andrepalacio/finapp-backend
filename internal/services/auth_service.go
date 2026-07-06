package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository is the interface shared by auth and user services.
// The concrete implementation lives in internal/repositories.
type UserRepository interface {
	Create(ctx context.Context, params repositories.CreateUserParams) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
	Update(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) (models.User, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

type AuthService struct {
	repo       UserRepository
	redis      *redis.Client
	jwt        *pkgauth.JWTManager
	bcryptCost int
}

type RegisterParams struct {
	Email    string
	Password string
	Name     string
}

type LoginParams struct {
	Email    string
	Password string
}

func NewAuthService(repo UserRepository, rdb *redis.Client, jwt *pkgauth.JWTManager, bcryptCost int) *AuthService {
	return &AuthService{repo: repo, redis: rdb, jwt: jwt, bcryptCost: bcryptCost}
}

func (s *AuthService) Register(ctx context.Context, p RegisterParams) (pkgauth.TokenPair, error) {
	// bcrypt silently truncates at 72 bytes — enforce the limit explicitly.
	if len(p.Password) < 8 || len(p.Password) > 72 {
		return pkgauth.TokenPair{}, apperror.ErrInvalidInput
	}

	p.Email = strings.ToLower(p.Email)

	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), s.bcryptCost)
	if err != nil {
		return pkgauth.TokenPair{}, apperror.Wrap(apperror.ErrInternal, err)
	}

	user, err := s.repo.Create(ctx, repositories.CreateUserParams{
		Email:        p.Email,
		PasswordHash: string(hash),
		Name:         p.Name,
	})
	if err != nil {
		return pkgauth.TokenPair{}, err
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *AuthService) Login(ctx context.Context, p LoginParams) (pkgauth.TokenPair, error) {
	p.Email = strings.ToLower(p.Email)

	user, err := s.repo.GetByEmail(ctx, p.Email)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return pkgauth.TokenPair{}, apperror.ErrUnauthorized
		}
		return pkgauth.TokenPair{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(p.Password)); err != nil {
		return pkgauth.TokenPair{}, apperror.ErrUnauthorized
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (pkgauth.TokenPair, error) {
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return pkgauth.TokenPair{}, apperror.ErrUnauthorized
	}

	// GetDel atomically reads and deletes — prevents TOCTOU races where two concurrent
	// requests with the same token both pass a separate GET check.
	key := refreshKey(refreshToken)
	stored, err := s.redis.GetDel(ctx, key).Result()
	if err != nil || stored != claims.UserID.String() {
		return pkgauth.TokenPair{}, apperror.ErrUnauthorized
	}

	return s.issueTokens(ctx, claims.UserID)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	s.redis.Del(ctx, refreshKey(refreshToken)) //nolint:errcheck
	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, userID uuid.UUID) (pkgauth.TokenPair, error) {
	pair, err := s.jwt.GenerateTokenPair(userID)
	if err != nil {
		return pkgauth.TokenPair{}, apperror.Wrap(apperror.ErrInternal, err)
	}

	key := refreshKey(pair.RefreshToken)
	if err := s.redis.Set(ctx, key, userID.String(), s.jwt.RefreshExpiry()).Err(); err != nil {
		return pkgauth.TokenPair{}, apperror.Wrap(apperror.ErrInternal, err)
	}

	return pair, nil
}

func refreshKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("refresh:%x", sum)
}
