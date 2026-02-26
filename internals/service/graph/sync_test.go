// setup:feature:graph
package graph

import (
	"context"
	"os"
	"sync/atomic"
	"testing"

	"catgoose/go-htmx-template/internals/database"
	"catgoose/go-htmx-template/internals/domain"
	"catgoose/go-htmx-template/internals/logger"
)

func init() {
	os.Setenv("GO_ENV", "development")
	os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
}

func testUsers() []domain.GraphUser {
	return []domain.GraphUser{
		{AzureID: "aaa-111", DisplayName: "Alice"},
		{AzureID: "bbb-222", DisplayName: "Bob"},
	}
}

func setupTestCache(t *testing.T) *UserCache {
	t.Helper()
	db, err := database.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open in-memory SQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewUserCache(db)
}

func TestInitAndSyncUserCache_AfterSyncCalled(t *testing.T) {
	userCache := setupTestCache(t)
	users := testUsers()

	var callCount atomic.Int32
	var receivedUsers []domain.GraphUser

	afterSync := func(_ context.Context, u []domain.GraphUser) {
		callCount.Add(1)
		receivedUsers = u
	}

	err := InitAndSyncUserCache(
		context.Background(),
		userCache,
		3,
		func() ([]domain.GraphUser, error) { return users, nil },
		afterSync,
	)
	if err != nil {
		t.Fatalf("InitAndSyncUserCache: %v", err)
	}

	if got := callCount.Load(); got != 1 {
		t.Errorf("afterSync called %d times, want 1", got)
	}
	if len(receivedUsers) != len(users) {
		t.Errorf("afterSync received %d users, want %d", len(receivedUsers), len(users))
	}
}

func TestInitAndSyncUserCache_NilAfterSync(t *testing.T) {
	userCache := setupTestCache(t)
	users := testUsers()

	err := InitAndSyncUserCache(
		context.Background(),
		userCache,
		3,
		func() ([]domain.GraphUser, error) { return users, nil },
		nil,
	)
	if err != nil {
		t.Fatalf("InitAndSyncUserCache with nil afterSync: %v", err)
	}

	count, err := userCache.GetUserCount()
	if err != nil {
		t.Fatalf("GetUserCount: %v", err)
	}
	if count != len(users) {
		t.Errorf("user count = %d, want %d", count, len(users))
	}
}
