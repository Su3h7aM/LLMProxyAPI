---
status: done
branch: issue/0010-port-management-oauth-provider
type: port
agent: deepseek-v4.1-flash
---

# Port upstream: oauth_provider in credential list and OAuth hints

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
0559fa3c feat(management): expose oauth_provider in credential list
482efbf5 feat(auth): enhance OAuth provider handling with explicit hints and metadata preservation
```

Triage notes: Triage: `0559fa3` applies clean (additive v8 field; the frozen v0 API must not change). `482efbf` has an import-order conflict in `auth_files_provider_oauth.go` (`gin` import moved in the fork). It uses `host.HasMultipleAuthProviders` and `ojson`; both exist in the fork. Compare the fork body of `HasMultipleAuthProviders` with the upstream baseline `e5e6ad1e` and report any difference. The panel (`apps/web`) may not read the new field: do not change the panel, list it as follow-up.

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

1. In your checkout create branch `issue/0010-port-management-oauth-provider` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

- 0559fa3c ported as `69e280ab` (hash after rebase onto main). Patch applied with `git apply --3way`. The new
  test file needed one gofmt fix: in the fork the module path sorts before
  `github.com/gin-gonic/gin`, so `gin` goes last in the third-party group.
- 482efbf5 ported as `3b12afff` (hash after rebase onto main). Only import-order context conflicts, as triage
  said:
  - `auth_files_provider_oauth.go` and `auth_provider.go`: the patch adds the
    `github.com/MoeForge/moego/ojson` import in an upstream import block. I
    dropped those two import hunks and added the import in fork order.
  - `auth_files_oauth_provider_test.go`: the whole file is rewritten by the
    patch, so the context of the gofmt-reordered import block did not match. I
    took the upstream file at 482efbf5, rewrote the module path, and ran
    `gofmt -w`. Content equals upstream, except import order.
- Checked the two commits against upstream 482efbf5 file by file. Only
  pre-existing fork lines differ: `gin` import position, the OpenCode Go
  `supports_quota`/`quota_provider` entry in `auth_files.go`, and
  `claude.GenerateState()` in `auth_files_provider_oauth.go`.
- Question in the issue: compare the fork body of `HasMultipleAuthProviders`
  with upstream baseline `e5e6ad1e`. Result: the function does not exist in
  either tree before 482efbf5. `git log -S` on upstream shows 482efbf5 as the
  only commit that adds it, so there is no fork version to compare and no
  semantic conflict. The fork has `callRecordAuthProviderIdentifiers`,
  `recordCurrent`, and `ojson` already, so the new code compiles unchanged.
- Gate: `go build ./...` OK; `go vet ./...` reports only the known
  `internal/pluginhost/host_test.go:1637` finding; `go test ./...` all pass;
  `gofmt -l` clean on the eight touched files.
- Rebased onto main (`6deacf71`, which includes issue 0013). After the rebase
  `go vet ./...` prints nothing, because 0013 removed that finding. Gate re-run
  after rebase: build OK, vet clean, all tests pass.

## Follow-ups

- The panel (`apps/web`) does not read the new `oauth_provider` field. Show it
  in the credential list if the product wants it.

## Result

Ported both upstream commits, one fork commit each:

| Upstream | Fork commit | Subject |
| --- | --- | --- |
| `0559fa3cfd5722cf1755a7dc3a90573ede3835f6` | `69e280ab` | feat(management): expose oauth_provider in credential list |
| `482efbf5a777771523c1237ed7aaee965451c51d` | `3b12afff` | feat(auth): enhance OAuth provider handling with explicit hints and metadata preservation |

The v8 credential list now returns `oauth_provider` when the credential has an
OAuth provider hint. The v0 route keeps its frozen shape; a test locks this.
Plugin auth parsing, login polling, refresh, and file synthesis now read the
hint from storage JSON too, stamp it for multi-entry plugins, and preserve it
across refresh.

Deviations from upstream: import order only (`ojson` import position,
`gin` last in the third-party group of the new test file). No logic change. The
port does not change `apps/web`.

Gate at the end, after the rebase onto `6deacf71`: `go build ./...` OK,
`go vet ./...` clean (issue 0013 is on main already), `go test ./...` all pass,
`gofmt -l` clean on the touched files.
