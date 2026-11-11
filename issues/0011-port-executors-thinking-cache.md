---
status: in-progress
branch: issue/0011-port-executors-thinking-cache
type: port
agent: deepseek-v4.1-flash
---

# Port upstream: executor, usage, and replay-cache fixes

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
f36b195e perf(antigravity): skip reasoning replay KV for symmetric gemini/interactions protocols
eb6e25ab fix(devin): count cache-write tokens in total input
cc95661c feat(gemini): support asymmetric thoughtSignature replay KV across Gemini executors
00c6ae82 perf(cache): eliminate double base64 encoding for reasoning replay KV arrays
13fdcc5e fix(thinking): recognize claude thinking.display updates in summary pipeline
949b43b3 fix(xai): classify non-stream terminal SSE errors and empty or incomplete streams
f77a3c10 fix(meta): pass through explicit client prompt_cache_retention
```

Triage notes: Triage: `f36b195e`, `eb6e25ab` (test import order), `949b43b3`, `13fdcc5e`, `00c6ae8` apply clean. `f77a3c10` has comment-context differences in `config.example.yaml` and context from upstream `ece3f52` (issue 0009, may or may not be on `main` yet): if a hunk fails for that reason, stop and report. `cc95661` needs `f36b195e` first (its test hunk edits the symmetric-skip test that `f36b195e` adds): that is why both are here. Keep the order.

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

1. In your checkout create branch `issue/0011-port-executors-thinking-cache` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

### 2026-11-11: branch setup and first two commits

Branch `issue/0011-port-executors-thinking-cache`. Stub `apps/web/dist/management.html`
made (not committed). Patches from `/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI`,
module path rewritten with `sed`.

- `f36b195e` -> `05077546`: `git apply --3way` reported a failed hunk at
  `antigravity_reasoning_replay.go:22` and then applied all hunks. Diff stat
  matches upstream (2 files, +90). `go build ./...` OK, `go test
  ./internal/runtime/executor/` OK, `gofmt -l` clean.
- `eb6e25ab` -> `e8bb77b7`: import hunk of `devin_executor_test.go` failed
  (fork has extra `google/uuid` import line after `sdktranslator`). Applied with
  `patch -F3`; the import landed in sorted order, no edit needed. Build OK,
  executor tests OK, `gofmt -l` clean.

### 2026-11-11: BLOCKER, order of `cc95661c` and `00c6ae82` (needs a decision)

`cc95661c` applies to the fork without conflict, but it does not build:

```
internal/runtime/executor/antigravity_reasoning_replay.go:608:44: undefined:
internalcache.PeekAntigravityReasoningReplayItemsWithSnapshotRequired
```

Evidence:

- The fork diff of `cc95661c` matches upstream exactly (10 files, +866/-123).
- `PeekAntigravityReasoningReplayItemsWithSnapshotRequired` is added by
  `00c6ae82` (next commit in the list) in
  `internal/cache/antigravity_reasoning_replay_cache.go`.
- This is not a fork defect. I extracted the upstream tree at `cc95661c` with
  `git archive` to `/tmp/upcc` and ran `go build ./internal/runtime/executor/`:
  the same undefined symbol. `git grep "func Peek" cc95661c -- internal/cache/`
  finds nothing. So the upstream commit `cc95661c` does not compile alone;
  upstream fixed it in the later `00c6ae82`.

So the listed order cannot meet the gate "build and tests must pass after each
commit". Two ways out, both deviate from the issue text:

- A (recommended): swap, port `00c6ae82` first, then `cc95661c`. Probed on a
  temporary branch from `e8bb77b7`: `00c6ae82` applies clean, builds, `go test
  ./internal/cache/` OK. Then `cc95661c` applies clean, builds, `go test
  ./internal/runtime/executor/ ./internal/cache/` OK. `f36b195e` still comes
  before `cc95661c`, so the stated reason for the order holds. Cost: the fork
  history has `00c6ae82` before `cc95661c`, and `00c6ae82`'s commit message
  says "perf(cache)" before the feature that uses it.
- B: keep the order, and move the two `Peek...` helpers (+ the
  `reserveOnMiss` split of `getAntigravityReasoningReplayItemsWithSnapshot`)
  from `00c6ae82` into the `cc95661c` commit. `cc95661c` then builds, but it
  carries part of a later upstream commit, which breaks "one fork commit per
  upstream commit".

Rest of the list probed on the temporary branch, all fine after option A:
`13fdcc5e` clean, tests of `internal/thinking` and
`internal/runtime/executor/helps` pass; `949b43b3` applies with an import-order
fuzz hunk in `xai_executor_test.go`, executor tests pass; `f77a3c10` applies
clean including `config.example.yaml` (no comment-context conflict here),
executor tests pass.

Question for the coordinator: use A or B? Work is stopped at `e8bb77b7`.

## Follow-ups

## Result
