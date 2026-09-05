package fork

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/taigrr/crush/internal/checkpoint"
	"github.com/taigrr/crush/internal/db"
	"github.com/taigrr/crush/internal/message"
	"github.com/taigrr/crush/internal/session"
)

// forkFixture wires a real database, session/message services and a
// checkpoint service rooted at a temp project so Fork can be exercised
// end to end against a live working tree.
type forkFixture struct {
	projectDir  string
	svc         Service
	checkpoints checkpoint.Service
	sessionID   string
	messageID   string
	snapshotID  string
}

func newForkFixture(t *testing.T) *forkFixture {
	t.Helper()
	ctx := context.Background()

	projectDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "main.go"), []byte("v1"), 0o644))

	conn, err := db.Connect(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	q := db.New(conn)

	sessions := session.NewService(q, conn)
	messages := message.NewService(q)
	checkpoints, err := checkpoint.NewService(checkpoint.ServiceConfig{
		Enabled:    true,
		ProjectDir: projectDir,
	}, q, conn)
	require.NoError(t, err)

	src, err := sessions.Create(ctx, "source")
	require.NoError(t, err)
	msg, err := messages.Create(ctx, src.ID, message.CreateMessageParams{
		Role:  message.User,
		Parts: []message.ContentPart{message.TextContent{Text: "fork here"}},
	})
	require.NoError(t, err)

	snap, err := checkpoints.CreateSnapshot(ctx, src.ID, msg.ID, "at fork point")
	require.NoError(t, err)

	// Diverge the working tree after the snapshot; these are the changes a
	// fork must not destroy unless told to.
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "main.go"), []byte("v2"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "wip.txt"), []byte("uncommitted"), 0o644))

	return &forkFixture{
		projectDir:  projectDir,
		svc:         NewService(q, conn, sessions, messages, checkpoints, nil),
		checkpoints: checkpoints,
		sessionID:   src.ID,
		messageID:   msg.ID,
		snapshotID:  snap.ID,
	}
}

func (f *forkFixture) readFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(f.projectDir, name))
	require.NoError(t, err)
	return string(content)
}

// TestForkLeavesWorkingTreeAloneByDefault pins the fork default: forking a
// conversation is not a filesystem operation. Without an explicit opt-in the
// live working directory must be left byte-for-byte as it was, even though
// a snapshot for the fork point exists.
func TestForkLeavesWorkingTreeAloneByDefault(t *testing.T) {
	t.Parallel()
	f := newForkFixture(t)

	result, err := f.svc.Fork(context.Background(), ForkParams{
		SessionID: f.sessionID,
		MessageID: f.messageID,
		Title:     "fork",
	})
	require.NoError(t, err)
	require.NotNil(t, result.SourceSnapshot)
	require.Equal(t, f.snapshotID, result.SourceSnapshot.ID)
	require.Equal(t, "fork here", result.PrefillText)

	require.Equal(t, "v2", f.readFile(t, "main.go"), "fork must not rewrite the working tree by default")
	require.Equal(t, "uncommitted", f.readFile(t, "wip.txt"), "fork must not prune the working tree by default")
}

// TestForkRestoresWorkingTreeOnlyWhenAsked verifies the explicit opt-in
// path is the only one that touches the working directory.
func TestForkRestoresWorkingTreeOnlyWhenAsked(t *testing.T) {
	t.Parallel()
	f := newForkFixture(t)

	_, err := f.svc.Fork(context.Background(), ForkParams{
		SessionID:          f.sessionID,
		MessageID:          f.messageID,
		Title:              "fork",
		RestoreWorkingTree: true,
	})
	require.NoError(t, err)

	require.Equal(t, "v1", f.readFile(t, "main.go"))
	_, err = os.Stat(filepath.Join(f.projectDir, "wip.txt"))
	require.True(t, os.IsNotExist(err), "explicit restore prunes files not in the snapshot")
}

// TestForkWorktreeRequestWithoutWorktreesDoesNotRestoreInPlace guards the
// old fall-through: asking for a worktree when worktrees are unavailable
// used to silently restore into the live working directory instead.
func TestForkWorktreeRequestWithoutWorktreesDoesNotRestoreInPlace(t *testing.T) {
	t.Parallel()
	f := newForkFixture(t)

	result, err := f.svc.Fork(context.Background(), ForkParams{
		SessionID:      f.sessionID,
		MessageID:      f.messageID,
		Title:          "fork",
		CreateWorktree: true,
	})
	require.NoError(t, err)
	require.Nil(t, result.Worktree)

	require.Equal(t, "v2", f.readFile(t, "main.go"))
	require.Equal(t, "uncommitted", f.readFile(t, "wip.txt"))
}
