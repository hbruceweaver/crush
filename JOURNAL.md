# Restore protection integration journal

## 2026-09-05 17:36 EDT — Baseline and integration

- Task authorization: integrate the committed restore/fork safeguards into
  `local/daily`, test, install atomically, and use the graceful server update.
  Local commits are authorized; pushing and worktree deletion are out of scope.
- Read repository and UI `AGENTS.md`. The Codex ancestor `AGENTS.md` is empty.
- Main checkout remains on `fix/restore-honours-gitignore` at `c69269bd`.
  Baseline daily tip is `0b65e311`; installed binary and server report
  `v0.91.12-0.20260904221444-0b65e3117618+dirty` (server PID 43049).
- Created `codex/restore-protection-integration` in this isolated worktree
  from `local/daily`, then fast-forwarded to existing fix `c69269bd`.
- Reviewing the four upstream-only commits and refreshing remote refs.
  Preserve provider configuration and the Fantasy dependency version.
- All commits created here disable GPG signing per task instructions.

## 2026-09-05 17:41:36 EDT — Integration and verification complete

- Fetched upstream `feature` over HTTPS: still `b3b389fb`. Refreshed origin's
  `feature` ref through HTTPS/gh credentials: still `a5a12287`. No push.
- `541eb686` merges all four upstream updates (`0e3a19f4`, `cb276bf4`,
  `74679677`, `b3b389fb`). Only README conflicted; preserved local sysadmin,
  swarm lineage, and graceful-update documentation within upstream's layout.
- `6688df06` closes two additional reproducible protection gaps: recursive
  cleanup preserves ignored/excluded descendants beneath absent parents,
  and snapshots/restores refresh ignore rules. Restore uses the destination's
  rules and also avoids overwriting files newly ignored since an older snapshot.
  Added regressions failed before the changes and pass afterward.
- `cdf45ef3` repairs two existing test fixture failures: shorten reconnect
  grace for the shutdown test, and ignore `.crush/` in the merge fixture,
  matching runtime setup. Test repository commits disable signing.
- Full `CGO_ENABLED=0 GOEXPERIMENT=greenteagc go test ./... -count=1` passed
  (64 packages with tests; signing disabled through Git environment overrides).
- Race tests passed for checkpoint, fork, journal, and server, including
  `TestE2E_DrainHandsOffQueueToNextServer` (active turn completes; two queued
  prompts replay after handoff). Race tests use CGO_ENABLED=1 as Go requires.
- `go vet` passed for checkpoint, fork, server, UI model, and UI dialog.
  Go changes formatted with gofumpt. Diff whitespace, log capitalization,
  and upstream lint shell syntax checks passed. golangci-lint is not installed.
- Test logs are `/tmp/crush-restore-integration-full.log`,
  `/tmp/crush-restore-integration-race.log`, and
  `/tmp/crush-restore-integration-vet.log`.
- Fantasy remains `github.com/taigrr/fantasy v0.27.0-fork`; go.mod and go.sum
  unchanged. Provider configuration baseline SHA-256:
  `c6176a6539d74efc1c537cd3f1ea67f40fd8d90c42673b9a5848b31856241367`.

### Worktree review (no deletions)

- `feat/graceful-update`, `feat/steer-midturn`, and
  `fix/sidebar-busy-visibility` are merged cleanup candidates.
- `fix/tool-call-repair`, `fix/accepted-run-busy`, and
  `port/upstream-feature` have no unmatched patches according to `git cherry`.
- `research/claude-code-parity`: range-diff maps all 16 commits to daily's
  port, with changes adapting to queue journaling and renamed APIs, plus
  three subsequent fixes on daily. Candidate for archival after retaining
  its branch/reference; it is not an ancestor by commit identity.
- `feat/swarm-lineage`: lineage and pinned working-directory behavior are
  ported, but this branch also retains the older per-session model-column
  and orchestrator design that daily replaced with model_ref/role handling.
  Keep as a reference; do not treat it as a simple merged cleanup candidate.
- The restore feature branch remains unchanged in the main checkout and will
  become an ancestor of the updated daily line.

### Installation preparation

- A disposable fixture was seeded using the normal session/message/checkpoint
  services in `tmp/live-restore-probe/main.go` (ignored task helper). Its
  metadata is `/tmp/crush-restore-live-probe.json`. This will verify the live
  server's fork default, ignored-file preservation, safety refs, and undo.
- Build next from this committed tree into a new binary filename, retain the
  old installed binary, atomically rename, then run `update --graceful` with
  no timeout so no active turn is forcibly cancelled.
