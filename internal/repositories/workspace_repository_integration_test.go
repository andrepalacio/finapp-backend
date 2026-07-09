//go:build integration

package repositories

import (
	"context"
	"testing"

	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T, repo *UserRepository) uuid.UUID {
	t.Helper()
	u, err := repo.Create(context.Background(), CreateUserParams{
		Email:        uuid.New().String() + "@test.com",
		PasswordHash: "hash",
		Name:         "Test User",
	})
	require.NoError(t, err)
	return u.ID
}

func TestWorkspaceRepository_CreateAndGetByID(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	ownerID := createTestUser(t, userRepo)

	ws, err := wsRepo.Create(context.Background(), CreateWorkspaceParams{Name: "Home", OwnerID: ownerID, Currency: "COP"})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, ws.ID)
	assert.Equal(t, "Home", ws.Name)
	assert.Equal(t, ownerID, ws.OwnerID)

	got, err := wsRepo.GetByID(context.Background(), ws.ID)
	require.NoError(t, err)
	assert.Equal(t, ws.ID, got.ID)
	assert.Equal(t, "Home", got.Name)
}

func TestWorkspaceRepository_GetByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	wsRepo := NewWorkspaceRepository(pool)

	_, err := wsRepo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestWorkspaceRepository_MembersAndIsMember(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	ownerID := createTestUser(t, userRepo)
	memberID := createTestUser(t, userRepo)

	ws, err := wsRepo.Create(context.Background(), CreateWorkspaceParams{Name: "Shared", OwnerID: ownerID, Currency: "COP"})
	require.NoError(t, err)

	// Owner is not auto-added as a member by Create; verify IsMember false until explicitly added.
	assert.False(t, wsRepo.IsMember(context.Background(), ws.ID, memberID))

	err = wsRepo.AddMember(context.Background(), ws.ID, memberID, "member")
	require.NoError(t, err)

	assert.True(t, wsRepo.IsMember(context.Background(), ws.ID, memberID))

	member, err := wsRepo.GetMember(context.Background(), ws.ID, memberID)
	require.NoError(t, err)
	assert.Equal(t, "member", member.Role)

	members, err := wsRepo.ListMembers(context.Background(), ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, memberID, members[0].UserID)

	err = wsRepo.UpdateMemberRole(context.Background(), ws.ID, memberID, "admin")
	require.NoError(t, err)
	member, err = wsRepo.GetMember(context.Background(), ws.ID, memberID)
	require.NoError(t, err)
	assert.Equal(t, "admin", member.Role)

	err = wsRepo.RemoveMember(context.Background(), ws.ID, memberID)
	require.NoError(t, err)
	assert.False(t, wsRepo.IsMember(context.Background(), ws.ID, memberID))
}

func TestWorkspaceRepository_ListByUser(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	ownerID := createTestUser(t, userRepo)
	otherOwnerID := createTestUser(t, userRepo)

	ws1, err := wsRepo.Create(context.Background(), CreateWorkspaceParams{Name: "A", OwnerID: ownerID, Currency: "COP"})
	require.NoError(t, err)
	_, err = wsRepo.Create(context.Background(), CreateWorkspaceParams{Name: "B", OwnerID: otherOwnerID, Currency: "COP"})
	require.NoError(t, err)

	require.NoError(t, wsRepo.AddMember(context.Background(), ws1.ID, ownerID, "owner"))

	list, err := wsRepo.ListByUser(context.Background(), ownerID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, ws1.ID, list[0].ID)
}

func TestWorkspaceRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	ownerID := createTestUser(t, userRepo)

	ws, err := wsRepo.Create(context.Background(), CreateWorkspaceParams{Name: "Old", OwnerID: ownerID, Currency: "COP"})
	require.NoError(t, err)

	updated, err := wsRepo.Update(context.Background(), ws.ID, "New Name", "USD")
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "USD", updated.Currency)
}

func TestWorkspaceRepository_Update_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	wsRepo := NewWorkspaceRepository(pool)

	_, err := wsRepo.Update(context.Background(), uuid.New(), "X", "COP")
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestWorkspaceRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	ownerID := createTestUser(t, userRepo)

	ws, err := wsRepo.Create(context.Background(), CreateWorkspaceParams{Name: "ToDelete", OwnerID: ownerID, Currency: "COP"})
	require.NoError(t, err)

	require.NoError(t, wsRepo.Delete(context.Background(), ws.ID))

	_, err = wsRepo.GetByID(context.Background(), ws.ID)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}
