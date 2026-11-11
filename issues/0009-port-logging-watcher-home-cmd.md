---
status: done
branch: issue/0009-port-logging-watcher-home-cmd
type: port
agent: muse-spark-1.3-contributor
---

# Port upstream: logging, watcher, Home liveness, login-proxy flag

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
0561334f fix(home): preserve fork kind in home dispatch
bebd92c4 feat(cmd): add -login-proxy flag for CLI OAuth logins
4b3f2c7e fix(logging): skip request error dumps for unrouted 404 responses
628e54ab fix(watcher): update active auth directory and fsnotify watch on runtime auth-dir changes
c9dad847 fix(home): only renew liveness on heartbeat pong
ece3f52a fix(logging): preserve bounded upstream responses in error logs
```

Triage notes: Triage: all apply clean except import order in tests (`628e54ab`: `watcher.go` import order, use `patch -F3`). `ece3f52` is the largest commit (39 files, 8 test files): its `request_logging.go` hunks need `4b3f2c7e` first (that is why both are here). `bebd92c4` adds a `-login-proxy` flag: `cmd/server/main.go` differs in the fork (XDG paths, no managementasset); keep the fork lines. Run the logging, watcher, home, `sdk/auth`, and `cmd` tests.

## How to port

1. Get the patch: `git -C /home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI show --format= <hash> > /tmp/<hash>.patch`.
2. Rewrite the module path in the patch: `github.com/router-for-me/CLIProxyAPI/v8` -> `github.com/Su3h7aM/LLMProxyAPI`. Then `git apply --3way` (or `patch -F3`). Fix only import-order context conflicts.
3. Run `gofmt -l` on touched Go files and fix. Keep the fork import order.
4. Commit: Conventional Commits, same subject as upstream, body ends with the line `Upstream: router-for-me/CLIProxyAPI@<full-hash>`. Also end with the Co-Authored-By trailer your harness gives you, if any.
5. Gate after each commit: `go build ./...` and `go test` for the touched packages. Gate at the end: `go build ./... && go vet ./... && go test ./...`. Build and tests must pass. `go vet` has one known old finding on `main` (`internal/pluginhost/host_test.go:1637`, until issue 0013 fixes it): no NEW finding is allowed. `gofmt -l` must be clean on the files you touch.

## Environment

- Use `/usr/bin/go` (1.27.3). Plain `go` is a mise shim and fails here (untrusted mise config). Do NOT run `mise trust` or change anything outside this checkout.
- `go build ./...` needs `apps/web/dist/management.html` (git ignores it). Create a stub: `mkdir -p apps/web/dist && echo '<html></html>' > apps/web/dist/management.html`. Never commit it.
- Code, docs, comments, commit messages: English only. Use ASD-STE100 Simplified Technical English in all your text, be concise.

## Rules

- Take only what the commits change. Do not refactor, rename, or "improve" anything else.
- Fork features stay. Protect: Go module `github.com/Su3h7aM/LLMProxyAPI`, vendored panel in `apps/web`, OpenCode Go provider/quota support, routing panel and `soonest-reset`, session affinity with strikes (`SessionAffinitySelector`, `AffinityPersister`), no sponsor/promo content, no Plus code. Upstream config, auth files, and `kv.db` must keep loading.
- You do not decide. If a conflict is semantic (not import order), if a test fails and you cannot fix it by a small adaptation inside the upstream commit's intent, or if anything is unclear: stop, write the question in `## Log`, commit, and report to the coordinator. Do not skip a commit or weaken a test to get green.
- Do not edit `DECISIONS.md` or other issue files. List follow-ups under `## Follow-ups`.

## Workflow

1. In your checkout create branch `issue/0009-port-logging-watcher-home-cmd` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

- Ported all 6 commits in listed order (one fork commit per upstream commit), with `bebd92c4` ported last after a scope reversal (see below).
- `0561334f` (home fork kind): `git apply --3way` clean. Gate: `sdk/cliproxy/auth` tests pass.
- `4b3f2c7e` (skip 404 error dumps): applied clean. Gate: middleware and logging tests pass.
- `628e54ab` (watcher auth-dir reload): strict apply failed on import context (`watcher.go`, `watcher_test.go`, `config_reload.go`); applied with `patch -p1 -F3` as triage notes allow. One manual fix: `watcher_test.go` new `pluginapi` import moved into the fork SDK import group (upstream single sorted group vs fork module-first order). Gate: watcher tests pass.
- `c9dad847` (heartbeat pong liveness): applied clean except gofmt import order in new `client_heartbeat_deadline_test.go` (fixed with `gofmt -w`: fork `Su3h7aM` sorts before `redis`). Gate: home tests pass.
- `ece3f52a` (bounded upstream error logs, 39 files): applied with `patch -p1 -F3`. One hunk failed (`xai_websockets_executor_test.go` imports: fork has extra `sdktranslator` import and no `gin`); hand-applied the three new imports (`sync/atomic`, `internal/logging`, `gin`) in fork import order. Two import blocks fixed with `gofmt -w` (`request_logging_test.go`, `codex_executor_execute_usage_test.go`). A `patch` backup pair (`*.orig`) was first committed by accident, then removed by recommitting the port as one clean 39-file commit (+2598/-156, same as upstream). Gate: build OK; logging, middleware, executor, util, handlers tests pass.
- Scope change and reversal: coordinator first ordered `bebd92c4` skipped (local CLI feature, out of scope); fork commit `208a8291` was dropped via `git rebase --onto 208a8291^ 208a8291` and the branch was rebased onto new `main` with no conflicts. Coordinator then reversed: local features may be ported (low priority, keeps fork near upstream). Re-ported `bebd92c4` as the last code commit on top of the rebased branch: all files applied clean except `sdk/auth/manager.go` (direct application after 3-way fallback, same as the first attempt; result matches upstream intent). Fork `cmd/server/main.go` lines kept (only additions). Gate: `sdk/auth` and `internal/cmd` tests pass.
- Final gate on rebased branch: `go build ./...` pass; `go vet ./...` fully clean (the old `pluginhost` finding was fixed on `main` by issue 0013); `go test ./...` no failures; `gofmt -l` clean on all touched files. `main` advanced twice during this issue (0012, then 0013 housekeeping, then 0010 oauth_provider); final `git rebase main` applied with no conflicts. Note: a `git stash`/`pop` cycle picked up a foreign stash entry from issue 0011 (`cc95661c attempt`) because stash refs are shared across worktrees of this repo; the contamination was removed with `git reset --hard HEAD` (commits were safe) and the foreign stash entry was left untouched. The uncommitted issue-file text lost in that cycle was rewritten from scratch.
- `status: done`. No merge to `main` (coordinator merges).

## Follow-ups

- `gofmt -l` flags `internal/runtime/executor/claude_thinking_replay_test.go` and `codex_stream_bootstrap_buffering_test.go` on old bases, but current `main` already fixed them (issue 0013 housekeeping); this branch is clean after the final rebase.
- Shared-repo caution: `git stash` in one worktree can surface another issue's stash entries. Prefer no stash, or `stash push -m` with a unique message and verify `stash list` before popping.

## Result

Ported 6 commits (upstream -> fork):
- `0561334f` -> `6e88c99b` fix(home): preserve fork kind in home dispatch
- `4b3f2c7e` -> `72efe2f0` fix(logging): skip request error dumps for unrouted 404 responses
- `628e54ab` -> `344ccc49` fix(watcher): update active auth directory and fsnotify watch on runtime auth-dir changes
- `c9dad847` -> `ba0f458b` fix(home): only renew liveness on heartbeat pong
- `ece3f52a` -> `5bdb5fa7` fix(logging): preserve bounded upstream responses in error logs
- `bebd92c4` -> `6fde5064` feat(cmd): add `-login-proxy` flag for CLI OAuth logins (ported last after scope reversal; fork `cmd/server/main.go` lines kept)
- Deviations from upstream: module path rewrite only; import order kept in fork style (`watcher_test.go`, `client_heartbeat_deadline_test.go`, `request_logging_test.go`, `codex_executor_execute_usage_test.go`, `xai_websockets_executor_test.go`). No semantic changes.
- Gate (after final `git rebase main`): `go build ./...` pass; `go vet ./...` fully clean, zero findings; `go test ./...` pass (no FAIL lines); `gofmt -l` clean on all touched files. Known gaps: none found.
