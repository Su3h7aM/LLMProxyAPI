---
status: done
branch: issue/0012-port-claude-cloak-sensitive-words
type: port
agent: muse-spark-1.3-contributor
---

# Port upstream: sensitive-words-replace, Home cloak options, server-side fallback opt-in

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
9a7e1e6b feat(executor): add sensitive-words-replace and harden Antigravity obfuscation
65de937c feat(claude): consume Home credential_options cloak settings
fa11a573 fix(claude): make server-side fallback opt-in and surface serving models
```

Triage notes: Triage: `9a7e1e6` is large (26 files). It changes the Antigravity obfuscation character from U+200B to CGJ (wire-visible) and adds config keys. In `config_normalization.go` near line 379 the fork added `SanitizeOpenCodeGoKeys`: keep BOTH the fork call and the new upstream call. `65de937` needs `9a7e1e6` (uses `SensitiveWordsReplace`); `fa11a57` needs `9a7e1e6` and `65de937`. Import-order conflict in `claude_signing.go`. Keep the order. List the wire-visible changes in `## Result`.

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

1. In your checkout create branch `issue/0012-port-claude-cloak-sensitive-words` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

- Renamed branch to `issue/0012-port-claude-cloak-sensitive-words`. Set status in-progress.
- Ported `9a7e1e6b`: 25 of 26 files via `git apply --3way` after module-path rewrite. `internal/config/config_normalization.go` conflicted (fork `SanitizeOpenCodeGoKeys` sits where upstream inserts new code). Kept fork code and inserted upstream `NormalizeSensitiveWordsReplace`, `SanitizeAntigravity`, `SanitizeDevin`, plus the `SensitiveWordsReplace` line in `NormalizeCloakConfig`. Touched files gofmt-clean. Build and tests for touched packages pass.
- Ported `65de937c`: 6 of 7 files via apply. `internal/runtime/executor/claude_signing.go` had the expected import-order conflict. Kept fork import order, added `claudeauth` import in sorted place and the Home cloak block in `resolveClaudeKeyCloakConfig`. Build and tests pass.
- Ported `fa11a573`: all 16 files applied cleanly. Build and tests pass.
- Full gate: `go build ./...` passes. `go vet ./...` shows only the known old finding (`internal/pluginhost/host_test.go:1637`). `go test ./...` passes. `git rebase main` was clean (branch already up to date).

## Follow-ups

- None found. No semantic conflict met.

## Result

Ported commits (fork hashes):

- `8b6da2e6` feat(executor): add sensitive-words-replace and harden Antigravity obfuscation. Upstream `9a7e1e6b412747729dedecefd7e23bd2e3ee1d3d`. 26 files.
- `1b855da0` feat(claude): consume Home credential_options cloak settings. Upstream `65de937cf19b88436ee8519caa42f0fcb66e2307`. 7 files.
- `eeb2ca3a` fix(claude): make server-side fallback opt-in and surface serving models. Upstream `fa11a573b252b30256bef4de952a95feee3c8c9d`. 16 files.

Gate: `go build ./...` passes. `go vet ./...` shows only the known old finding on main (`internal/pluginhost/host_test.go:1637`). `go test ./...` passes. `gofmt -l` clean on all touched files. Two files flagged by `gofmt -l` repo-wide (`claude_thinking_replay_test.go`, `codex_stream_bootstrap_buffering_test.go`) are pre-existing and untouched by this work.

Deviations from upstream: module path rewrite only, plus import-order keeps (fork order in `claude_signing.go`; fork `sort`/`strings` imports already present in `config_normalization.go`). Fork `SanitizeOpenCodeGoKeys` kept next to new upstream sanitize calls.

Wire-visible changes:

- Antigravity obfuscation character changed from U+200B to CGJ (U+034F).
- New `sensitive-words-replace` map config for cloak, Antigravity, and Devin sections (literal target to replacement text).
- Claude Home `credential_options` cloak settings now feed cloaked requests.
- New opt-in `server-side-fallback` flag (default false). When fallback occurs, translated OpenAI Chat and Responses payloads report the actual serving model with fallback metadata.
- Known gaps: none.
