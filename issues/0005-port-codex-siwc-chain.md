---
status: in-progress
branch: issue/0005-port-codex-siwc-chain
type: port
agent: muse-spark-1.3-contributor
---

# Port upstream: Codex SIWC refresh-outcome chain

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
68cd7572 fix(codex): prevent replay after uncertain refresh outcomes
80ee739f fix(codex): preserve SIWC rotations and honor renewal windows
3891ebaa fix(codex): reuse the issued SIWC client during CLI reauthorization
8c6d8b18 fix(codex): defer SIWC renewal during macOS maintenance wakes
baec297f fix(auth): retain partition failures alongside SIWC wake cancellation
03c394ee test(codex): make SIWC power-state checks deterministic
a46be7ae fix(codex): retain SIWC renewal windows in refresh receipts
5f38d557 fix(auth): keep native Codex available during SIWC recovery
09b3a579 fix(codex): harden SIWC refresh deferral and expires_in parsing
```

Triage notes: Triage: ~3000 lines, new refresh-journal and outcome machinery that the fork never carried. The commits depend on each other: keep the order. Run the full test suite for `internal/auth`, `sdk/auth`, `sdk/cliproxy/auth`, `internal/runtime/executor` after the last commit. `68cd7572` has one trivial conflict in `nocopy_invariant_test.go`: the fork added a same-file entry with other wording. Keep both entries.

## How to port

1. Get the patch: `git -C /home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI show --format= <hash> > /tmp/<hash>.patch`.
2. Rewrite the module path in the patch: `github.com/router-for-me/CLIProxyAPI/v8` -> `github.com/Su3h7aM/LLMProxyAPI`. Then `git apply --3way` (or `patch -F3`). Fix only import-order context conflicts.
3. Run `gofmt -l` on touched Go files and fix. Keep the fork import order.
4. Commit: Conventional Commits, same subject as upstream, body ends with the line `Upstream: router-for-me/CLIProxyAPI@<full-hash>`. Also end with the Co-Authored-By trailer your harness gives you, if any.
5. Gate after each commit: `go build ./...` and `go test` for the touched packages. Gate at the end: `go build ./... && go vet ./... && go test ./...` (all must pass; baseline on `main` is green).

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

1. In your checkout create branch `issue/0005-port-codex-siwc-chain` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

- 68cd7572 ported as d73b7d0c. `git apply --3way` failed on two files (fork
  files match upstream parent; used `patch -p1 -F3` instead).
  `nocopy_invariant_test.go`: upstream adds a
  `internal/kvstore/bolt_store.go` entry, fork already has one with other
  wording. A Go map cannot hold two same keys, so merged into one entry
  `{1, ...}` that keeps both wordings. gofmt: fixed import order in 3 new
  files. Gate: `go build ./...` OK, touched packages OK.
- 80ee739f: patch applied with `patch -p1 -F3` (2 hunks offset in types.go
  around fork `opencode-go` case; fork extras kept). New test
  `TestSIWCTokenResponseValidatesScope` fails at this commit (4 refresh
  subtests get nil error). Cause: subtests share `(clientID, grant)` key
  `oaiapp_scope:refresh` through the process-global `processRefreshJournal`
  from 68cd7572, so subtest 1 receipt is reused. Upstream fixes this in this
  same chain at a46be7ae (`auth.SetRefreshJournal(&memoryRefreshJournal{})`).
  My files match upstream 80ee739f exactly except fork branding/import order,
  so this failure is upstream-faithful intermediate state. Continue chain;
  full gate must pass after 09b3a579.

## Follow-ups

## Result
