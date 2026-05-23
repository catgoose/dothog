// setup:feature:graph
package graph

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"catgoose/dothog/internal/logger"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("GO_ENV", "development")
	_ = os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
	os.Exit(m.Run())
}

func testUsers() []GraphUser {
	return []GraphUser{
		{AzureID: "aaa-111", DisplayName: "Alice"},
		{AzureID: "bbb-222", DisplayName: "Bob"},
	}
}

func setupTestDirectory(t *testing.T) *Directory {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "graph_cache.db")
	dir, err := OpenDirectory(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open Graph directory: %v", err)
	}
	t.Cleanup(func() { _ = dir.Close() })
	return dir
}

func TestInitAndSyncDirectory_AfterSyncCalled(t *testing.T) {
	directory := setupTestDirectory(t)
	users := testUsers()

	var callCount atomic.Int32
	var receivedUsers []GraphUser

	afterSync := func(_ context.Context, u []GraphUser) {
		callCount.Add(1)
		receivedUsers = u
	}

	err := InitAndSyncDirectory(
		context.Background(),
		directory,
		3,
		func(context.Context) ([]GraphUser, error) { return users, nil },
		afterSync,
	)
	if err != nil {
		t.Fatalf("InitAndSyncDirectory: %v", err)
	}

	if got := callCount.Load(); got != 1 {
		t.Errorf("afterSync called %d times, want 1", got)
	}
	if len(receivedUsers) != len(users) {
		t.Errorf("afterSync received %d users, want %d", len(receivedUsers), len(users))
	}
}

func TestInitAndSyncDirectory_FetchErrorWithExistingSnapshot_KeepsCacheAndProceeds(t *testing.T) {
	directory := setupTestDirectory(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	seed := testUsers()
	if err := directory.ReplaceUsers(ctx, seed); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	fetchErr := errors.New("graph unavailable")
	err := InitAndSyncDirectory(
		ctx,
		directory,
		3,
		func(context.Context) ([]GraphUser, error) { return nil, fetchErr },
		nil,
	)
	if err != nil {
		t.Fatalf("InitAndSyncDirectory must keep startup running on existing snapshot, got: %v", err)
	}

	count, err := directory.UserCount(ctx)
	if err != nil {
		t.Fatalf("UserCount: %v", err)
	}
	if count != len(seed) {
		t.Errorf("snapshot wiped after failed fetch: count = %d, want %d", count, len(seed))
	}
}

func TestInitAndSyncDirectory_FetchErrorWithNoSnapshot_ReturnsError(t *testing.T) {
	directory := setupTestDirectory(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	fetchErr := errors.New("graph unavailable")
	err := InitAndSyncDirectory(
		ctx,
		directory,
		3,
		func(context.Context) ([]GraphUser, error) { return nil, fetchErr },
		nil,
	)
	if err == nil {
		t.Fatal("InitAndSyncDirectory must error when fetch fails with no usable snapshot")
	}
	if !errors.Is(err, fetchErr) {
		t.Errorf("returned error must wrap fetch error, got: %v", err)
	}
}

func TestInitAndSyncDirectory_NilAfterSync(t *testing.T) {
	directory := setupTestDirectory(t)
	users := testUsers()

	err := InitAndSyncDirectory(
		context.Background(),
		directory,
		3,
		func(context.Context) ([]GraphUser, error) { return users, nil },
		nil,
	)
	if err != nil {
		t.Fatalf("InitAndSyncDirectory with nil afterSync: %v", err)
	}

	count, err := directory.UserCount(context.Background())
	if err != nil {
		t.Fatalf("UserCount: %v", err)
	}
	if count != len(users) {
		t.Errorf("user count = %d, want %d", count, len(users))
	}
}
