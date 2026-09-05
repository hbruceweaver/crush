package checkpoint

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
	"github.com/taigrr/crush/internal/db"
)

// newServiceDB returns an in-memory sqlite connection with the snapshots
// schema and one session/message pair ("test-session"/"test-msg") so tests
// can create a real, database-backed snapshot.
func newServiceDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()

	conn, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(0)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	_, err = conn.ExecContext(t.Context(), `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY
		);
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS snapshots (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			parent_snapshot_id TEXT,
			git_commit_hash TEXT NOT NULL,
			description TEXT,
			created_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_snapshots_session_id ON snapshots(session_id);
		CREATE INDEX IF NOT EXISTS idx_snapshots_message_id ON snapshots(message_id);
	`)
	require.NoError(t, err)

	_, err = conn.ExecContext(t.Context(), "INSERT INTO sessions (id) VALUES ('test-session')")
	require.NoError(t, err)
	_, err = conn.ExecContext(t.Context(), "INSERT INTO messages (id, session_id) VALUES ('test-msg', 'test-session')")
	require.NoError(t, err)

	return conn, db.New(conn)
}

func TestServiceCreateSnapshot(t *testing.T) {
	t.Parallel()

	// Create temp dir for project
	projectDir := newProjectDir(t)

	// Create a test file
	testFile := filepath.Join(projectDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("hello world"), 0o644))

	conn, q := newServiceDB(t)

	// Create service
	svc, err := NewService(ServiceConfig{
		Enabled:    true,
		ProjectDir: projectDir,
	}, q, conn)
	require.NoError(t, err)
	require.True(t, svc.IsEnabled())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create snapshot
	snap, err := svc.CreateSnapshot(ctx, "test-session", "test-msg", "Test snapshot")
	require.NoError(t, err, "CreateSnapshot should succeed")
	require.NotNil(t, snap)
	require.NotEmpty(t, snap.ID)
	require.NotEmpty(t, snap.GitCommitHash)
	t.Logf("Created snapshot: ID=%s, GitCommit=%s", snap.ID, snap.GitCommitHash)

	// Verify it was persisted
	snaps, err := svc.ListSnapshots(ctx, "test-session")
	require.NoError(t, err)
	require.Len(t, snaps, 1)
	require.Equal(t, snap.ID, snaps[0].ID)
}

// safetyRefs returns the commit hashes pinned under refs/safety/ in the
// service's snapshot repository.
func safetyRefs(t *testing.T, svc Service) []string {
	t.Helper()
	repo := svc.(*service).repo
	refs, err := repo.repo.References()
	require.NoError(t, err)
	var hashes []string
	require.NoError(t, refs.ForEach(func(ref *plumbing.Reference) error {
		if strings.HasPrefix(string(ref.Name()), SafetyRefPrefix) {
			hashes = append(hashes, ref.Hash().String())
		}
		return nil
	}))
	return hashes
}

// TestServiceRestoreSnapshotTakesSafetySnapshot verifies that restoring
// through the service first pins the current work tree as a safety
// snapshot whose tree equals the pre-restore state, and that the safety
// commit hash can be restored directly to undo the operation.
func TestServiceRestoreSnapshotTakesSafetySnapshot(t *testing.T) {
	t.Parallel()

	projectDir := newProjectDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a.txt"), []byte("v1"), 0o644))

	conn, q := newServiceDB(t)
	svc, err := NewService(ServiceConfig{Enabled: true, ProjectDir: projectDir}, q, conn)
	require.NoError(t, err)

	ctx := t.Context()
	snap, err := svc.CreateSnapshot(ctx, "test-session", "test-msg", "baseline")
	require.NoError(t, err)
	require.Empty(t, safetyRefs(t, svc), "creating a message snapshot must not create safety refs")

	// Mutate the tree after the snapshot: this is the state the safety
	// snapshot must preserve.
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a.txt"), []byte("v2"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "b.txt"), []byte("new"), 0o644))

	require.NoError(t, svc.RestoreSnapshot(ctx, snap.ID, ""))

	// The restore itself happened.
	content, err := os.ReadFile(filepath.Join(projectDir, "a.txt"))
	require.NoError(t, err)
	require.Equal(t, "v1", string(content))
	_, err = os.Stat(filepath.Join(projectDir, "b.txt"))
	require.True(t, os.IsNotExist(err))

	// Exactly one safety snapshot exists and describes what it guards.
	safety := safetyRefs(t, svc)
	require.Len(t, safety, 1)
	repo := svc.(*service).repo
	commit, err := repo.repo.CommitObject(plumbing.NewHash(safety[0]))
	require.NoError(t, err)
	require.Contains(t, commit.Message, "pre-restore safety snapshot")
	require.Contains(t, commit.Message, snap.ID)

	// Its tree equals the pre-restore working tree.
	scratch := t.TempDir()
	require.NoError(t, repo.RestoreSnapshot(safety[0], scratch))
	content, err = os.ReadFile(filepath.Join(scratch, "a.txt"))
	require.NoError(t, err)
	require.Equal(t, "v2", string(content))
	content, err = os.ReadFile(filepath.Join(scratch, "b.txt"))
	require.NoError(t, err)
	require.Equal(t, "new", string(content))

	// One-step undo: the safety commit hash is accepted by RestoreSnapshot.
	require.NoError(t, svc.RestoreSnapshot(ctx, safety[0], ""))
	content, err = os.ReadFile(filepath.Join(projectDir, "a.txt"))
	require.NoError(t, err)
	require.Equal(t, "v2", string(content))
	content, err = os.ReadFile(filepath.Join(projectDir, "b.txt"))
	require.NoError(t, err)
	require.Equal(t, "new", string(content))
	require.Len(t, safetyRefs(t, svc), 2, "the undo is itself guarded by a safety snapshot")

	// Unknown ids are still rejected, hash-shaped or not.
	require.ErrorIs(t, svc.RestoreSnapshot(ctx, "does-not-exist", ""), ErrSnapshotNotFound)
	require.ErrorIs(t, svc.RestoreSnapshot(ctx, strings.Repeat("0", 40), ""), ErrSnapshotNotFound)
}

// TestServiceRestoreSnapshotAbortsWithoutSafetySnapshot verifies that when
// the pre-restore safety snapshot cannot be taken, the restore is refused
// and the work tree is left untouched.
func TestServiceRestoreSnapshotAbortsWithoutSafetySnapshot(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}

	projectDir := newProjectDir(t)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a.txt"), []byte("v1"), 0o644))

	conn, q := newServiceDB(t)
	svc, err := NewService(ServiceConfig{Enabled: true, ProjectDir: projectDir}, q, conn)
	require.NoError(t, err)

	ctx := t.Context()
	snap, err := svc.CreateSnapshot(ctx, "test-session", "test-msg", "baseline")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a.txt"), []byte("v2"), 0o644))

	// An unreadable directory makes the snapshot walk fail.
	locked := filepath.Join(projectDir, "locked")
	require.NoError(t, os.MkdirAll(locked, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(locked, "secret"), []byte("x"), 0o644))
	require.NoError(t, os.Chmod(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	err = svc.RestoreSnapshot(ctx, snap.ID, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "safety snapshot")

	content, err := os.ReadFile(filepath.Join(projectDir, "a.txt"))
	require.NoError(t, err)
	require.Equal(t, "v2", string(content), "work tree must be untouched when the safety snapshot fails")
	require.Empty(t, safetyRefs(t, svc))
}
