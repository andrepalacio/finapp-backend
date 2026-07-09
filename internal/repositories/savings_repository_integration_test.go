//go:build integration

package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavingsRepository_CreateAndGetByID(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	savingsRepo := NewSavingsRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)

	goal, err := savingsRepo.Create(context.Background(), CreateSavingsGoalParams{
		WorkspaceID: wsID, Name: "Emergency fund", TargetAmount: 5000000,
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, goal.ID)
	assert.Equal(t, "Emergency fund", goal.Name)

	got, err := savingsRepo.GetByID(context.Background(), goal.ID)
	require.NoError(t, err)
	assert.Equal(t, goal.ID, got.ID)
}

func TestSavingsRepository_GetByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	savingsRepo := NewSavingsRepository(pool)

	_, err := savingsRepo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestSavingsRepository_List(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	savingsRepo := NewSavingsRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)

	for _, name := range []string{"A", "B"} {
		_, err := savingsRepo.Create(context.Background(), CreateSavingsGoalParams{WorkspaceID: wsID, Name: name, TargetAmount: 1000})
		require.NoError(t, err)
	}

	list, err := savingsRepo.List(context.Background(), wsID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestSavingsRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	savingsRepo := NewSavingsRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)

	goal, err := savingsRepo.Create(context.Background(), CreateSavingsGoalParams{WorkspaceID: wsID, Name: "Old", TargetAmount: 1000})
	require.NoError(t, err)

	updated, err := savingsRepo.Update(context.Background(), UpdateSavingsGoalParams{
		ID: goal.ID, WorkspaceID: wsID, Name: "New", TargetAmount: 2000,
	})
	require.NoError(t, err)
	assert.Equal(t, "New", updated.Name)
	assert.Equal(t, float64(2000), updated.TargetAmount)
}

func TestSavingsRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	savingsRepo := NewSavingsRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)

	goal, err := savingsRepo.Create(context.Background(), CreateSavingsGoalParams{WorkspaceID: wsID, Name: "ToDelete", TargetAmount: 1000})
	require.NoError(t, err)

	require.NoError(t, savingsRepo.Delete(context.Background(), goal.ID, wsID))

	_, err = savingsRepo.GetByID(context.Background(), goal.ID)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestSavingsRepository_Contributions_And_TotalContributed(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	savingsRepo := NewSavingsRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)

	goal, err := savingsRepo.Create(context.Background(), CreateSavingsGoalParams{WorkspaceID: wsID, Name: "Goal", TargetAmount: 1000})
	require.NoError(t, err)

	contributedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	c1, err := savingsRepo.CreateContribution(context.Background(), CreateContributionParams{
		GoalID: goal.ID, Amount: 300, ContributedAt: contributedAt, Notes: "first",
	})
	require.NoError(t, err)
	_, err = savingsRepo.CreateContribution(context.Background(), CreateContributionParams{
		GoalID: goal.ID, Amount: 200, ContributedAt: contributedAt,
	})
	require.NoError(t, err)

	total, err := savingsRepo.TotalContributed(context.Background(), goal.ID)
	require.NoError(t, err)
	assert.Equal(t, float64(500), total)

	list, err := savingsRepo.ListContributions(context.Background(), goal.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	got, err := savingsRepo.GetContribution(context.Background(), c1.ID)
	require.NoError(t, err)
	assert.Equal(t, c1.ID, got.ID)

	progress, err := savingsRepo.ListWithProgress(context.Background(), wsID)
	require.NoError(t, err)
	require.Len(t, progress, 1)
	assert.Equal(t, float64(500), progress[0].TotalContributed)
	assert.Equal(t, goal.ID, progress[0].ID)

	require.NoError(t, savingsRepo.DeleteContribution(context.Background(), c1.ID, goal.ID))
	total, err = savingsRepo.TotalContributed(context.Background(), goal.ID)
	require.NoError(t, err)
	assert.Equal(t, float64(200), total)
}

func TestSavingsRepository_TotalContributed_NoContributions(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	savingsRepo := NewSavingsRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)

	goal, err := savingsRepo.Create(context.Background(), CreateSavingsGoalParams{WorkspaceID: wsID, Name: "Empty", TargetAmount: 1000})
	require.NoError(t, err)

	total, err := savingsRepo.TotalContributed(context.Background(), goal.ID)
	require.NoError(t, err)
	assert.Equal(t, float64(0), total)
}
