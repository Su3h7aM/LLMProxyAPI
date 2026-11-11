---
status: done
branch: issue/0004-port-pluginhost-store
type: port
agent: deepseek-v4.1-flash
---

# Port upstream: plugin host and plugin store

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
b5b724d0 fix(pluginhost): reuse content-addressed plugin shadow copies across restarts
f69db99d feat(plugin-store): support opt-in release cache refresh on store listing
00748974 feat(pluginhost): expose token modality details in usage.handle
14e57717 fix(pluginhost): support credential-scoped rate limit errors from plugin executors
```

Triage notes: Triage: apply clean. `14e57717` has context from upstream `68cd7572` (issue 0005). If a hunk fails for that reason, stop and report. The panel has no control for the new `?refresh=true` store query: do not add one (note it as follow-up).

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

1. In your checkout create branch `issue/0004-port-pluginhost-store` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

- 2026-11-11: Created branch `issue/0004-port-pluginhost-store`, stub
  `apps/web/dist/management.html`, status in-progress.
- 2026-11-11: Ported all four commits in order; no semantic conflict. Each
  patch first failed on import-order context only (fork puts
  `github.com/Su3h7aM/...` before third-party imports). Fixed by hand, then
  compared each touched file against the upstream tree at the commit, with the
  module path rewritten: all equal, except the fork-only `GOWORK=off` line in
  `loader_unix_test.go` and the known `~/.llm-proxy-api` path renames.
- 2026-11-11: `14e57717` applied without the `68cd7572` context, because the
  hunk offsets do not overlap. The fork still lacks upstream
  `refresh_outcome_unknown` in `conductor_cooldown.go`; this commit does not
  touch those lines.
- 2026-11-11: Finding, confirmed: `go vet ./internal/pluginhost/` fails on
  `main` at `host_test.go:1637` before this branch. Proven with
  `git archive main` into a temp tree. Cause on `main`: fork commit
  `6e103599`. `DECISIONS.md` on `main` (commit `53f611e2`) already allows
  this one finding. No other vet finding. No new finding from this port.
- 2026-11-11: Rebased onto `main` (no conflict). Gate re-run after rebase.

## Follow-ups

- The panel has no control for the new `?refresh=true` store query. The store
  page always lists without refresh. Add a control only on request.
- `sdk/pluginapi` and `sdk/pluginabi` changes are not shown in the plugin SDK
  docs (`docs/sdk-*.md`). No doc update was in scope.
- The old vet finding at `internal/pluginhost/host_test.go:1637` still needs
  its own small issue, as `DECISIONS.md` states.

## Result

Ported four upstream commits, one fork commit each:

| Fork commit | Upstream commit | Subject |
| --- | --- | --- |
| `8a6d26cc` | `b5b724d06df01748055259512d163605fd2f482d` | fix(pluginhost): reuse content-addressed plugin shadow copies across restarts |
| `b3d12c33` | `f69db99d5ebb9f0a68ea2e6741257968eb27536d` | feat(plugin-store): support opt-in release cache refresh on store listing |
| `3d1ce8f1` | `0074897481fee1dcd595c1caa1d57e6a379b6be6` | feat(pluginhost): expose token modality details in usage.handle |
| `4a3d5e2a` | `14e577174a37b339c6b6862b4a08543f2e10a15e` | fix(pluginhost): credential-scoped rate limit errors from plugin executors |

Also `e50961b2` for the issue status start.

Gate after final rebase onto `main` (`de6975ea`):

- `go build ./...`: pass.
- `go vet ./...`: one finding, `internal/pluginhost/host_test.go:1637`, the
  known `main` finding that `DECISIONS.md` allows. No new finding.
- `go test ./...`: pass.
- `gofmt -l` on all touched files: clean.

Deviations from upstream: import order only, to keep the fork convention.
The fork-only `GOWORK=off` line in `loader_unix_test.go` and the
`~/.llm-proxy-api` path renames stay. No test was changed or weakened.

Known gaps: see Follow-ups.

