---
status: todo
branch: issue/0006-port-claude-executor-fixes
type: port
agent: muse-spark-1.3-contributor
---

# Port upstream: Claude executor and header fixes

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
ea0b4712 fix(claude): align OAuth authorization state and scopes
27762819 fix(claude): keep mid-session responses system items in place to preserve prompt cache
2c5b01f4 fix(claude): disable default upstream connection reuse and revert post-stream draining
542a525f fix(claude): recognize native client user-agent extensions
59304330 fix(claude): retain request fields with stabilized client profiles
beb43359 fix(claude): materialize top-level cache_control in cloaked requests
b4e7dbd2 fix(claude): bump Claude Code CLI fingerprint baseline to 2.1.321
6db7d4b8 fix(claude): extend preserved thinking and per-turn timing gates to Claude Fable 5.2
f0c9c2d0 fix(claude): embed tool schemas with rejected property names in descriptions
6876a2eb fix(claude): treat cli-bg x-app header as native claude code client
```

Triage notes: Triage: apply clean. `b4e7dbd` changes the emulated Claude Code CLI fingerprint (wire-visible): port as upstream, list it in `## Result`. `2c5b01f` reverts upstream `ce38c75` (stream drain), which the fork does NOT take. If `2c5b01f` does not apply because `ce38c75` is absent, port only its net effect: `git diff ce38c75^ 2c5b01f -- <paths>` (module path rewritten) as ONE commit, and log it. If unsure, stop and report.

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

1. In your checkout create branch `issue/0006-port-claude-executor-fixes` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

## Follow-ups

## Result
