package services

import (
	"context"
	"strings"
	"time"

	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo       UserRepository
	bcryptCost int
}

func NewUserService(repo UserRepository, bcryptCost int) *UserService {
	return &UserService{repo: repo, bcryptCost: bcryptCost}
}

type GetProfileParams struct {
	UserID uuid.UUID
}

type UserProfile struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

func (s *UserService) GetProfile(ctx context.Context, params GetProfileParams) (UserProfile, error) {
	user, err := s.repo.GetByID(ctx, params.UserID)
	if err != nil {
		return UserProfile{}, err
	}

	return UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

type UpdateProfileParams struct {
	UserID uuid.UUID
	Name   string
	Email  string
}

func (s *UserService) UpdateProfile(ctx context.Context, params UpdateProfileParams) (UserProfile, error) {
	if params.Name == "" && params.Email == "" {
		return UserProfile{}, apperror.ErrInvalidInput
	}

	// Fetch current values to fill in fields the caller did not provide.
	current, err := s.repo.GetByID(ctx, params.UserID)
	if err != nil {
		return UserProfile{}, err
	}
	if params.Name == "" {
		params.Name = current.Name
	}
	if params.Email == "" {
		params.Email = current.Email
	} else {
		params.Email = strings.ToLower(params.Email)
	}

	user, err := s.repo.Update(ctx, params.UserID, params.Name, params.Email)
	if err != nil {
		return UserProfile{}, err
	}

	return UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

type ChangePasswordParams struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

func (s *UserService) ChangePassword(ctx context.Context, params ChangePasswordParams) error {
	if len(params.NewPassword) < 8 || len(params.NewPassword) > 72 {
		return apperror.ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, params.UserID)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(params.CurrentPassword))
	if err != nil {
		return apperror.ErrUnauthorized
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(params.NewPassword), s.bcryptCost)
	if err != nil {
		return apperror.Wrap(apperror.ErrInternal, err)
	}

	_, err = s.repo.UpdatePassword(ctx, params.UserID, string(newHash))
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) Delete(ctx context.Context, userID uuid.UUID) error {
	err := s.repo.Delete(ctx, userID)
	if err != nil {
		return err
	}
	return nil
}
