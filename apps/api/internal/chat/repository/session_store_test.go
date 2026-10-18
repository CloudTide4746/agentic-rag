package repository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"teamflow/internal/chat"
	"teamflow/internal/faults"

	"github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openTestDB connects to the database named by TEST_DATABASE_DSN. The store
// tests are skipped unless it is set, so the default `go test ./...` run stays
// hermetic.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set; skipping database-backed store test")
	}
	driverConfig, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_DSN: %v", err)
	}
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{DSN: driverConfig.FormatDSN()}), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return db
}

// seedUser inserts a user and returns its id, so the owner FK is satisfied.
// Callers reset the namespace once per test before seeding, because a reset
// deletes every user under the prefix.
func seedUser(t *testing.T, db *gorm.DB, username string) uint {
	t.Helper()
	result := db.Exec(
		"INSERT INTO users (created_at, updated_at, username, email, password, nickname, avatar, status) VALUES (NOW(3), NOW(3), ?, ?, 'x', 'N', '', 1)",
		username, username+"@example.com",
	)
	if result.Error != nil {
		t.Fatalf("seed user: %v", result.Error)
	}
	var id uint
	if err := db.Raw("SELECT id FROM users WHERE username = ?", username).Scan(&id).Error; err != nil {
		t.Fatalf("read seeded user: %v", err)
	}
	return id
}

// resetNamespace deletes every row a test case owns: first the sessions that
// reference the namespaced users, then the users themselves. Matching sessions
// by owner_id (not by public id) is deliberate, because a public id is opaque
// and carries no guaranteed relationship to the user's name. Order matters
// because the owner FK is ON DELETE RESTRICT.
func resetNamespace(t *testing.T, db *gorm.DB, prefix string) {
	t.Helper()
	if err := db.Exec(
		"DELETE FROM chat_platform_sessions WHERE owner_id IN (SELECT id FROM users WHERE username LIKE ?)",
		prefix+"%",
	).Error; err != nil {
		t.Fatalf("reset sessions for %s: %v", prefix, err)
	}
	if err := db.Exec("DELETE FROM users WHERE username LIKE ?", prefix+"%").Error; err != nil {
		t.Fatalf("reset users for %s: %v", prefix, err)
	}
}

func TestSessionStoreLifecycle(t *testing.T) {
	db := openTestDB(t)
	store := NewSessionStore(db)
	ctx := context.Background()
	prefix := "store-lifecycle-"
	resetNamespace(t, db, prefix)
	owner := seedUser(t, db, prefix+"owner")
	other := seedUser(t, db, prefix+"other")

	created, err := store.Create(ctx, chat.Session{
		PublicID: prefix + "pub",
		OwnerID:  itoa(owner),
		Title:    "标题",
		Status:   chat.StatusCreating,
	}, prefix+"idem")
	if err != nil {
		t.Fatalf("create mapping: %v", err)
	}
	if created.PublicID != prefix+"pub" || created.Status != chat.StatusCreating {
		t.Fatalf("created = %+v, want the creating mapping", created)
	}

	// A different owner must not see the session.
	if _, err := store.GetByPublicID(ctx, prefix+"pub", other); !errors.Is(err, faults.ErrNotFound) {
		t.Fatalf("error = %v, want not found for a non-owner", err)
	}

	activated, err := store.MarkActive(ctx, prefix+"pub", "4242")
	if err != nil {
		t.Fatalf("mark active: %v", err)
	}
	// the returned mapping must reflect the committed row, not the input.
	if activated.Status != chat.StatusActive || activated.AgentSessionID != "4242" {
		t.Fatalf("activated = %+v, want active with agent id 4242", activated)
	}
	if !activated.UpdatedAt.After(created.UpdatedAt) && !activated.UpdatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("updated_at went backwards: created=%v activated=%v", created.UpdatedAt, activated.UpdatedAt)
	}
	active, err := store.GetByPublicID(ctx, prefix+"pub", owner)
	if err != nil {
		t.Fatalf("get after activate: %v", err)
	}
	if active.Status != chat.StatusActive || active.AgentSessionID != "4242" {
		t.Fatalf("active = %+v, want active with agent id 4242", active)
	}

	// Authorisation scope: the owner filter must be part of the query, not a
	// post-fetch check.
	if _, err := store.GetByPublicID(ctx, prefix+"pub", other); !errors.Is(err, faults.ErrNotFound) {
		t.Fatalf("error = %v, want not found after activation", err)
	}
}

func TestSessionStoreRecordsFailureStates(t *testing.T) {
	db := openTestDB(t)
	store := NewSessionStore(db)
	ctx := context.Background()
	prefix := "store-failure-"
	resetNamespace(t, db, prefix)
	owner := seedUser(t, db, prefix+"owner")

	for _, tc := range []struct {
		publicID string
		status   string
		mark     func(context.Context, string, string) error
	}{
		{prefix + "failed", chat.StatusFailed, store.MarkFailed},
		{prefix + "pending", chat.StatusPending, store.MarkPending},
	} {
		if _, err := store.Create(ctx, chat.Session{
			PublicID: tc.publicID, OwnerID: itoa(owner), Title: "T", Status: chat.StatusCreating,
		}, tc.publicID+"-idem"); err != nil {
			t.Fatalf("create %s: %v", tc.publicID, err)
		}
		if err := tc.mark(ctx, tc.publicID, "AGENT_TIMEOUT"); err != nil {
			t.Fatalf("mark %s: %v", tc.publicID, err)
		}
		got, err := store.GetByPublicID(ctx, tc.publicID, owner)
		if err != nil {
			t.Fatalf("get %s: %v", tc.publicID, err)
		}
		if got.Status != tc.status {
			t.Fatalf("status = %q, want %q", got.Status, tc.status)
		}
		if got.LastErrorCode != "AGENT_TIMEOUT" {
			t.Fatalf("last_error_code = %q, want AGENT_TIMEOUT", got.LastErrorCode)
		}
	}
}

func TestSessionStoreRejectsDuplicateIdempotencyKey(t *testing.T) {
	db := openTestDB(t)
	store := NewSessionStore(db)
	ctx := context.Background()
	prefix := "store-idem-"
	resetNamespace(t, db, prefix)
	owner := seedUser(t, db, prefix+"owner")

	if _, err := store.Create(ctx, chat.Session{
		PublicID: prefix + "a", OwnerID: itoa(owner), Title: "T", Status: chat.StatusCreating,
	}, prefix+"shared-key"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	// The unique index is what guarantees one Agent session per key.
	if _, err := store.Create(ctx, chat.Session{
		PublicID: prefix + "b", OwnerID: itoa(owner), Title: "T", Status: chat.StatusCreating,
	}, prefix+"shared-key"); err == nil {
		t.Fatal("duplicate idempotency key was accepted")
	}
}

func TestSessionStoreListIsOwnerScopedAndPaged(t *testing.T) {
	db := openTestDB(t)
	store := NewSessionStore(db)
	ctx := context.Background()
	prefix := "store-list-"
	resetNamespace(t, db, prefix)
	owner := seedUser(t, db, prefix+"owner")
	other := seedUser(t, db, prefix+"other")

	for _, id := range []string{prefix + "1", prefix + "2", prefix + "3"} {
		if _, err := store.Create(ctx, chat.Session{
			PublicID: id, OwnerID: itoa(owner), Title: "T", Status: chat.StatusActive,
		}, id+"-idem"); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	if _, err := store.Create(ctx, chat.Session{
		PublicID: prefix + "other", OwnerID: itoa(other), Title: "T", Status: chat.StatusActive,
	}, prefix+"other-idem"); err != nil {
		t.Fatalf("create other: %v", err)
	}

	items, total, err := store.ListByOwner(ctx, owner, 1, 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3 (only this owner's sessions)", total)
	}
	if len(items) != 2 {
		t.Fatalf("page size = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.OwnerID != itoa(owner) {
			t.Fatalf("leaked another owner's session: %+v", item)
		}
	}
}

func itoa(v uint) string { return strconv.FormatUint(uint64(v), 10) }
