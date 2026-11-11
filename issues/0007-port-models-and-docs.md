---
status: done
branch: issue/0007-port-models-and-docs
type: port
agent: deepseek-v4.1-flash
---

# Port upstream: model registry entries and config docs

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
19a0a854 feat(models): add claude-fable-5-2 and gemini-omni-1.2-flash to model registry
84534584 docs(config): document X-Session-ID guidance for full-history compaction under session affinity
effbac16 feat(models): add Gemini 4 Carbon model entry to models.json
```

Triage notes: Triage: data and comments only; apply clean (test import order may differ). Extra cleanup, as its own commit `chore: remove unused sponsor image`: `assets/aicodemirror.png` is sponsor content. Run `grep -rn aicodemirror` over the repo (exclude `.refs`, `node_modules`). If nothing references the file, delete it. If something references it, do not delete; report it.

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

1. In your checkout create branch `issue/0007-port-models-and-docs` (rename your branch if T3 gave another name). Set `status: done` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

- 2026-11-11: Renamed branch `t3/port-models-docs` to
  `issue/0007-port-models-and-docs`. Created the stub
  `apps/web/dist/management.html` (not committed).
- 2026-11-11: Ported `19a0a854` clean with `git apply --3way`. Data only.
- 2026-11-11: Ported `84534584` with `patch -p1 -F3`. `git apply --3way`
  failed on `config.example.yaml` only because the fork inserts extra
  soonest-reset and weighted-round-robin comments before the touched lines
  (hunk 1 applied at offset -15, fuzz 1). `config_types.go` applied at offset
  +8. This is context-only, not semantic. Checked that the documented
  behaviour exists in the fork: `sdk/cliproxy/auth/selector.go` has
  `pickLCP` with LCP compaction matching, and `X-Session-ID` is a session
  header. The test helpers `yamlPath` and `findMapKeyIndex` already exist in
  `internal/config/config_v8.go`, so the new test compiles without edits.
- 2026-11-11: Ported `effbac16` clean with `git apply --3way`. Note: the
  subject names Gemini 4 Carbon only, but the commit also adds
  `gemini-4-carbon-high`, and `muse-ember-1.0` (owned_by meta). Ported all
  entries as upstream has them.
- 2026-11-11: Cleanup commit for the sponsor image. `grep -rn aicodemirror`
  (excluding `.refs`, `node_modules`, `.git`) finds: `issues/0002-*.md` and
  `issues/0007-*.md` (text), and `apps/web/tests/providerGroupsV8.test.ts`
  (the provider base URL `https://api.aicodemirror.ai`). None references the
  image file; the web test uses the domain, not the asset, so deletion does
  not break it. `assets/aicodemirror.png` is the AI CodeMirror sponsor logo
  (checked the image). Deleted in commit `chore: remove unused sponsor image`.
- 2026-11-11: `main` moved while this issue ran (issue 0003 merged, 11
  commits). Rebased onto `main` (now `53f611e2`) with no conflicts.
- 2026-11-11: Full gate on the rebased tree: `go build ./...` OK,
  `go test ./...` all packages pass, `go vet ./...` shows only the known
  baseline finding `internal/pluginhost/host_test.go:1637` (accepted by the
  DECISIONS.md vet rule). `gofmt -l internal/config/` is empty.
- No `Co-Authored-By` trailer was supplied by the harness, so the port
  commits end with the `Upstream:` trailer only.


## Follow-ups

- `assets/` still holds other partner/sponsor logos (`packycode*`, `poixeai`, `runapi`, and others). Only `aicodemirror.png` was in this issue scope. Decide separately if they should go.
- The `gofmt` and `go vet` findings on `main` that DECISIONS.md accepts (two executor test files, `host_test.go:1637`) still need their own small issue.

## Result

Ported three upstream commits, one fork commit each, plus the sponsor image
cleanup, on branch `issue/0007-port-models-and-docs` rebased onto `main`
`53f611e2`.

| Fork commit | Upstream commit | Content |
| --- | --- | --- |
| `98c18cf9` | `19a0a854` | `claude-fable-5-2` and `gemini-omni-1.2-flash` entries in `internal/registry/models/models.json` |
| `cb64bc47` | `84534584` | X-Session-ID and Merkle LCP compaction guidance in `config.example.yaml` and `RoutingConfig` comments, plus `TestV8ExampleDocumentsSessionAffinityCompactionGuidance` |
| `d7735573` | `effbac16` | `gemini-4-carbon`, `gemini-4-carbon-high`, `muse-ember-1.0` entries in `models.json` |
| `04d4ebfd` | none | deleted unused sponsor asset `assets/aicodemirror.png` |

Gate on the rebased tree:

- `go build ./...`: pass.
- `go test ./...`: pass, all packages.
- `go vet ./...`: only the known baseline finding
  `internal/pluginhost/host_test.go:1637`.
- `gofmt -l` on touched Go files: clean.

Deviations from upstream: none in content. `config.example.yaml` needed
`patch -F3` for context that the fork had changed (comment-only). The web
panel was not rebuilt or browser-checked: the changes are Go data and
comments, and no web source references the deleted asset.

Known gaps: the panel has no UI for the new model entries beyond what the
registry already serves. Unverified: that upstream's `muse-ember-1.0` entry
intent matches its placement (it came with the "Gemini 4 Carbon" commit).

