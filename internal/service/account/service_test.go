package account

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	accountRepoPG "github.com/neo-vai/go-events/internal/repository/account/postgres"
	"github.com/neo-vai/go-events/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAccountService(t *testing.T) (*AccountService, *accountRepoPG.AccountRepositoryPG) {
	t.Helper()
	deps := testutil.SetupIntegrationTest(t)
	repo := accountRepoPG.NewAccountRepositoryPG(deps.DB)
	hasher := NewBcryptHasher(10) // lower cost for tests
	service := NewAccountService(repo, hasher)
	return service, repo
}

func TestCreateAccount_Success(t *testing.T) {
	svc, repo := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{
		ID:    uuid.New(),
		Email: "new@example.com",
	}
	err := svc.CreateAccount(ctx, acc, "StrongPass123")
	require.NoError(t, err)

	// Verify account was created with default role and active=true
	retrieved, err := repo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", retrieved.Email)
	assert.Equal(t, "user", retrieved.Role)
	assert.True(t, retrieved.Active)
	assert.NotEmpty(t, retrieved.PasswordHash)

	// Password hash should not equal plain password
	assert.NotEqual(t, "StrongPass123", retrieved.PasswordHash)

	// Verify password comparison works
	valid, err := svc.VerifyPassword(ctx, "new@example.com", "StrongPass123")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestCreateAccount_DuplicateEmail(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	acc1 := &account.Account{ID: uuid.New(), Email: "duplicate@example.com"}
	err := svc.CreateAccount(ctx, acc1, "Pass123")
	require.NoError(t, err)

	acc2 := &account.Account{ID: uuid.New(), Email: "duplicate@example.com"}
	err = svc.CreateAccount(ctx, acc2, "Pass123")
	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestVerifyPassword_InvalidPassword(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "test@example.com"}
	err := svc.CreateAccount(ctx, acc, "CorrectPass")
	require.NoError(t, err)

	valid, err := svc.VerifyPassword(ctx, "test@example.com", "WrongPass")
	assert.ErrorIs(t, err, ErrInvalidPassword)
	assert.False(t, valid)
}

func TestVerifyPassword_AccountNotFound(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	valid, err := svc.VerifyPassword(ctx, "nonexistent@example.com", "any")
	assert.ErrorIs(t, err, ErrAccountNotFound)
	assert.False(t, valid)
}

func TestVerifyPassword_InactiveAccount(t *testing.T) {
	svc, repo := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "inactive@example.com"}
	err := svc.CreateAccount(ctx, acc, "Pass123")
	require.NoError(t, err)

	// Manually deactivate account
	acc.Active = false
	err = repo.Update(ctx, acc)
	require.NoError(t, err)

	valid, err := svc.VerifyPassword(ctx, "inactive@example.com", "Pass123")
	assert.ErrorIs(t, err, ErrAccountInactive)
	assert.False(t, valid)
}

func TestUpdateAccount_EmailUpdate(t *testing.T) {
	svc, repo := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "old@example.com"}
	err := svc.CreateAccount(ctx, acc, "Pass123")
	require.NoError(t, err)

	acc.Email = "new@example.com"
	err = svc.UpdateAccount(ctx, acc)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", updated.Email)
	// Role and active should remain unchanged (preserved by service)
	assert.Equal(t, "user", updated.Role)
	assert.True(t, updated.Active)
}

func TestUpdateAccount_EmailConflict(t *testing.T) {
	svc, repo := setupAccountService(t)
	ctx := context.Background()

	acc1 := &account.Account{ID: uuid.New(), Email: "first@example.com"}
	err := svc.CreateAccount(ctx, acc1, "Pass123")
	require.NoError(t, err)

	acc2 := &account.Account{ID: uuid.New(), Email: "second@example.com"}
	err = svc.CreateAccount(ctx, acc2, "Pass123")
	require.NoError(t, err)

	acc2.Email = "first@example.com"
	err = svc.UpdateAccount(ctx, acc2)
	assert.ErrorIs(t, err, ErrEmailAlreadyExists)

	// Verify second account email unchanged
	acc2After, _ := repo.GetByID(ctx, acc2.ID)
	assert.Equal(t, "second@example.com", acc2After.Email)
}

func TestUpdateAccount_NotFound(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	ghost := &account.Account{ID: uuid.New(), Email: "ghost@example.com"}
	err := svc.UpdateAccount(ctx, ghost)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestDeleteAccount_Success(t *testing.T) {
	svc, repo := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "delete@example.com"}
	err := svc.CreateAccount(ctx, acc, "Pass123")
	require.NoError(t, err)

	err = svc.DeleteAccount(ctx, acc.ID.String())
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, acc.ID)
	assert.Error(t, err) // should be not found
}

func TestDeleteAccount_InvalidID(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	err := svc.DeleteAccount(ctx, "not-a-uuid")
	assert.ErrorIs(t, err, ErrInvalidAccountID)
}

func TestDeleteAccount_NotFound(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	err := svc.DeleteAccount(ctx, uuid.New().String())
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestGetByID_Success(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "get@example.com"}
	err := svc.CreateAccount(ctx, acc, "Pass123")
	require.NoError(t, err)

	retrieved, err := svc.GetByID(ctx, acc.ID.String())
	require.NoError(t, err)
	assert.Equal(t, acc.Email, retrieved.Email)
}

func TestGetByID_InvalidID(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "invalid-uuid")
	assert.ErrorIs(t, err, ErrInvalidAccountID)
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, uuid.New().String())
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestGetByEmail_Success(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "email@example.com"}
	err := svc.CreateAccount(ctx, acc, "Pass123")
	require.NoError(t, err)

	retrieved, err := svc.GetByEmail(ctx, "email@example.com")
	require.NoError(t, err)
	assert.Equal(t, acc.ID, retrieved.ID)
}

func TestGetByEmail_NotFound(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	_, err := svc.GetByEmail(ctx, "missing@example.com")
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

// Admin operations

func TestUpdateAccountAdmin_ChangeRoleAndActive(t *testing.T) {
	svc, repo := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "user@example.com"}
	err := svc.CreateAccount(ctx, acc, "Pass123")
	require.NoError(t, err)

	updates := map[string]interface{}{
		"role":   "admin",
		"active": false,
	}
	updated, err := svc.UpdateAccountAdmin(ctx, acc.ID.String(), updates)
	require.NoError(t, err)
	assert.Equal(t, "admin", updated.Role)
	assert.False(t, updated.Active)

	// Verify in DB
	fromDB, _ := repo.GetByID(ctx, acc.ID)
	assert.Equal(t, "admin", fromDB.Role)
	assert.False(t, fromDB.Active)
}

func TestUpdateAccountAdmin_InvalidRole(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	acc := &account.Account{ID: uuid.New(), Email: "user2@example.com"}
	err := svc.CreateAccount(ctx, acc, "Pass123")
	require.NoError(t, err)

	updates := map[string]interface{}{
		"role": "superadmin",
	}
	_, err = svc.UpdateAccountAdmin(ctx, acc.ID.String(), updates)
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestUpdateAccountAdmin_NotFound(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	_, err := svc.UpdateAccountAdmin(ctx, uuid.New().String(), map[string]interface{}{"active": true})
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestListAccounts_PaginationAndFilters(t *testing.T) {
	svc, _ := setupAccountService(t)
	ctx := context.Background()

	// Create 5 accounts
	for i := 0; i < 5; i++ {
		acc := &account.Account{ID: uuid.New(), Email: "user" + string(rune('0'+i)) + "@example.com"}
		err := svc.CreateAccount(ctx, acc, "Pass123")
		require.NoError(t, err)
	}
	// Make one admin
	adminAcc := &account.Account{ID: uuid.New(), Email: "admin@example.com", Role: "admin"}
	err := svc.CreateAccount(ctx, adminAcc, "Pass123")
	require.NoError(t, err)

	// List first page (limit 3)
	accounts, total, err := svc.ListAccounts(ctx, 0, 3, "created_at", "ASC", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(6), total)
	assert.Len(t, accounts, 3)

	// Filter by role=admin
	filters := map[string]interface{}{"role": "admin"}
	accounts, total, err = svc.ListAccounts(ctx, 0, 10, "created_at", "ASC", filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, accounts, 1)
	assert.Equal(t, "admin@example.com", accounts[0].Email)

	// Filter by active=false (none yet)
	filters = map[string]interface{}{"active": false}
	accounts, total, err = svc.ListAccounts(ctx, 0, 10, "created_at", "ASC", filters)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, accounts)

	// Search by email (q)
	filters = map[string]interface{}{"q": "admin"}
	accounts, total, err = svc.ListAccounts(ctx, 0, 10, "created_at", "ASC", filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "admin@example.com", accounts[0].Email)
}
