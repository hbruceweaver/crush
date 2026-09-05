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

## 2026-09-05 17:47:10 EDT — Installed; graceful handoff in progress

- `eaa47762` records the pre-install verification journal and is the clean
  build source. `local/daily` was advanced using an expected-old-value ref
  update, without checking it out in any other worktree.
- Built with `CGO_ENABLED=0 GOEXPERIMENT=greenteagc go build -trimpath`,
  explicit Commit and BuildID set to `eaa47762e567217bacdb9af3b94339aed6236323`.
  Go build metadata reports `vcs.modified=false`. Binary version:
  `v0.91.12-0.20260905214136-eaa47762e567`.
- New binary SHA-256:
  `36e835df0d62518c4da2c91488b857406b10a793806725fe0488e56972467718`.
- Built to `/Users/hbruceweaver/go/bin/crush.restore-eaa47762.new`, verified
  its metadata and checksum, then atomically replaced `/Users/hbruceweaver/go/bin/crush`.
- Previous binary retained at
  `/Users/hbruceweaver/.local/share/crush/binary-backups/crush-20260905-0b65e3117618`.
- Started `crush update --graceful` with no timeout. The old server entered
  drain mode and is finishing one live turn, which continues to publish
  tool results. No forced stop or signal has been used.
- Before handoff, the attached workspace contained 449 sessions and 23,746
  messages; its queue and reply-obligation tables were empty.
- All eight pre-existing linked worktrees remain clean and at their original
  commits. Main checkout remains at `c69269bd` with only `.worktrees/` untracked.
  Provider configuration checksum is still identical.
- GitHub check confirms origin's default branch is `feature`; upstream
  PRs #28 and #29 remain open and #30/#31 remain drafts. No remote writes.

## 2026-09-05 17:47:59 EDT — Handoff and live validation passed

- Graceful update completed: the active turn finished, PID 43049 exited on
  its own, and new server PID 81629 started at 17:46:55 EDT.
- `/v1/version` confirms the installed binary and running server match:
  `v0.91.12-0.20260905214136-eaa47762e567`, commit/build ID
  `eaa47762e567217bacdb9af3b94339aed6236323`, protocol 2, darwin/arm64.
  `/v1/health` is healthy and no longer draining; normal work resumed.
- Live HTTP checks in the disposable workspace all passed:
  1. Default conversation fork left current files untouched.
  2. Restore rolled back the tracked file and removed an ordinary extra file,
     while preserving `.env`, ignored descendants beneath an absent parent,
     and `.worktrees/probe` content, including its `.git` file.
  3. Restore created a pinned safety ref, and restoring its commit hash
     recovered the exact pre-restore tracked/untracked file state.
- Detached the disposable workspace after verification. No user workspace
  was restored or forked for testing. The original workspace reconnected.
- All 449 original session IDs survived. The database now has 450 sessions
  and 23,790 messages (up from 23,746 as the live turn continued); SQLite
  `quick_check` returns `ok`. Queue preservation was verified by the race-tested
  handoff regression; the live queue was empty at the pre-handoff baseline.
- Provider configuration checksum and installed binary checksum still match
  the recorded values. Main checkout and all eight prior linked worktrees
  remain intact. No dependency changes, pushes, or worktree deletions.
- Durable local evidence copies are in `tmp/restore-integration-evidence/`
  (ignored). The probe helper and verifier are in `tmp/live-restore-probe/`.
- `local/daily` intentionally remains at the exact installed build `eaa47762`.
  The task branch's final journal commit records post-install results; its
  only difference from daily is this journal. No code remains unintegrated.
- Remaining optional work: publish the local daily/protection commits after
  user authorization; clean up the reviewed worktree candidates separately.
  The older swarm/model branch and the separate Fantasy Anthropic fixes
  remain untouched.

## 2026-09-05 17:55:10 EDT — Publication authorized

- The user approved pushing this integration to their GitHub fork.
- Rechecked the clean task worktree and the three local tips. The remote
  branches do not yet exist in `hbruceweaver/crush`; repository access is ADMIN.
- Publish `local/daily` at `eaa47762`, the original protection branch
  `fix/restore-honours-gitignore` at `c69269bd`, and this task's
  `codex/restore-protection-integration` branch with its final journal.
- Use one atomic push over HTTPS with gh credentials, disable push signing,
  then verify the advertised remote tips against the local refs.
- The default `feature` branch, upstream PRs, installed binary, provider
  configuration, and other worktrees are outside this publication step.

## 2026-09-05 17:56:04 EDT — Original fix published; workflow authorization pending

- GitHub rejected the atomic push because the stored gh OAuth token has
  `repo`, `read:org`, and `gist` scopes but lacks `workflow`, which is required
  for the CI changes included in the daily and integration branches. The
  atomic rejection initially left all three remote branches absent.
- Pushed the original restore fix separately: remote
  `fix/restore-honours-gitignore` now matches `c69269bdffbcf322594af9c2566aeafe531d9c5a`.
  Fetched and configured its origin tracking branch.
- Started the standard `gh auth refresh --hostname github.com --scopes workflow`
  device authorization and requested the user's approval on GitHub.
  Awaiting that authorization before retrying the remaining atomic push.
