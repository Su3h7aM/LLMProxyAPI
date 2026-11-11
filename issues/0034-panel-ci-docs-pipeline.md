---
status: done
branch: issue/0034-panel-ci-docs-pipeline
type: panel
agent: deepseek-v4.1-flash
base: integration/panel-bumpyclock
---

# Panel pipeline: CI job, container build check, docs

## Goal

Make the new panel part of the repository checks, and update docs. Today no workflow runs on push or pull request; the release workflows run only on version tags, so a broken panel would show only at release time.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0034-panel-ci-docs-pipeline integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **CI workflow.** Add `.github/workflows/ci.yml`, triggered on `push` to `main` and on `pull_request`, with two jobs: (a) Go: setup Go `1.27.3`, create the stub `apps/web/dist/management.html`, run `go build ./...`, `go vet ./...`, `go test ./...`, and fail if `gofmt -l $(git ls-files '*.go')` prints anything; (b) panel: setup bun `1.4.4` (`oven-sh/setup-bun`), in `apps/web` run `bun install --frozen-lockfile` and `bun run verify`. Use the same action versions as the existing workflows in `.github/workflows/` where they exist. You cannot run GitHub Actions: check the YAML with a linter if one is installed, and say that it is unconfirmed.
2. **Container build check.** Read `Containerfile` and `.containerignore`. Check by reading and by running the same commands in a scratch copy under `/tmp` that the panel stage works with the new tree: the files copied before `bun install --frozen-lockfile` must include everything the build needs (`package.json`, `bun.lock`, and for example `bunfig.toml`, `scripts/design-tokens/`, `src/design-system/` if the build reads them early), the token plugin needs `bun` on `PATH`, and the output is `dist/index.html` copied to `management.html`. If `podman` or `docker` is available, try `podman build` or `docker build` of the panel stage only (`--target panel`); if not, say so. Fix the `Containerfile` only if something is wrong.
3. **Scripts.** `scripts/build-web.sh` and `scripts/install.sh` should use `bun install --frozen-lockfile`. Check `scripts/check-panel.sh` still matches the title `LLM Proxy API Management Center` (issue 0018 changed it).
4. **Docs.** Update the root `README.md` panel section (what the panel is: BumpyClock Management Center, embedded in the binary, same origin, management key login, how to build with mise tasks and with bun). Update `apps/web/README.md` and `apps/web/AGENTS.md` to match the final state (v8 only, same-origin, no legacy, design tokens rule, how to run `bun run verify`, how to add a page). Update `UPSTREAM_BASELINE.md`: the BumpyClock row and notes say that the panel is now vendored from it (pin `ce5f6179`), that the old official panel (pin `953fb24`) is no longer the source, and list what the fork changed on top (v8 API layer, same-origin login, removed sponsor and Plus-only pages, fork features: OpenCode Go, routing page, soonest-reset, SIWC, Claude reset grants). Do not edit `DECISIONS.md`.
Do NOT do in this issue: panel code changes.

## Gate

In `apps/web` with `export PATH=/home/su3h7am/.local/share/mise/installs/bun/latest/bin:$PATH`: `bun install --frozen-lockfile`, `bun run verify` (test, lint, build) must pass at the end, and `bun run type-check` plus the tests of your area after each commit. 
Tests: port the pre-import tests of your area (`git show panel-pre-bumpyclock:apps/web/tests/<name>`) when they fit the BumpyClock components; update BumpyClock tests that assert v0 paths. Do not delete or weaken a test to get green, except a test that pins behavior that this issue removes on purpose (say which in the commit body).

## Environment

- Use bun at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin` (put it on `PATH`; the token plugin runs `bun` at build time) and `/usr/bin/go`. Do NOT run `mise trust`.
- Never commit `node_modules`, `dist`, or `apps/web/dist/management.html`.
- English only. Use ASD-STE100 Simplified Technical English in all your text, be concise.

## Rules

- API is v8 only. No v0 path, no fallback to an older backend, no version check, no capability probe by 404. The panel and the proxy are one unit.
- Take only what this issue lists. Other areas are other issues.
- Keep the management key login, 401 handling, and the connection-revision guard on config read then write.
- Fork-only features stay (OpenCode Go, reset system, soonest-reset); they get their own issues unless this issue lists them.
- You do not decide. If a v8 route does not exist or differs from what the panel needs, if a UI choice is unclear, or if something is out of scope: stop, write the question in `## Log`, commit, and report. Do not invent endpoints and do not change Go code unless this issue says so.

- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. Create the branch as written above. Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current.
3. When the gate passes: `git rebase integration/panel-bumpyclock`, re-run the gate, set `status: done`, fill `## Result` (what landed, commit list, gate output summary, known gaps, follow-ups), commit.
4. Do NOT merge. Report: short summary, branch, commit range, follow-ups, surprises, unconfirmed claims.

## Log

- 2026-11-11: Branch created from `integration/panel-bumpyclock` (`db784257`). Baseline `bun run verify` passes.
- CI: added `.github/workflows/ci.yml`. Jobs `go` (setup-go 1.27.3, stub
  `apps/web/dist/management.html`, gofmt check, build, vet, test) and
  `panel` (setup-bun 1.4.4, `bun install --frozen-lockfile`,
  `bun run verify`). Triggers: push to `main`, all pull requests.
  Unconfirmed: GitHub Actions cannot run here. YAML parsed with `yq`;
  `actionlint` is not installed. The Go job commands were run locally
  with `/usr/bin/go` and pass.
- Container: read `Containerfile` and `.containerignore`. `podman build
  --target panel` with a clean context passes and produces
  `dist/index.html` plus `dist/management.html`. The bun base image has
  `bun` on `PATH`, so the design token plugin works. Found one problem:
  the build context had no ignore for `apps/web/node_modules` and
  `apps/web/dist`, so `COPY apps/web/ ./` could copy a host
  `node_modules` over the installed one. Added both to
  `.containerignore`. No `Containerfile` change was needed.
- Scripts: `build-web.sh` and `install.sh` now run
  `bun install --frozen-lockfile`. `check-panel.sh` already checks the
  title `LLM Proxy API Management Center`; no change.
- Docs: root `README.md` panel section, `apps/web/README.md`,
  `apps/web/AGENTS.md`, and the panel notes in `UPSTREAM_BASELINE.md`
  now describe the final state (BumpyClock pin `ce5f6179`, v8 only,
  same origin, no legacy, fork changes). `DECISIONS.md` untouched.
- Open questions: none.

## Follow-ups

## Result

Status: done. No merge (coordinator merges).

What landed:

- `.github/workflows/ci.yml`: two jobs. `go` runs on Go 1.27.3: it
  writes a stub `apps/web/dist/management.html`, fails on any output of
  `gofmt -l`, then runs `go build ./...`, `go vet ./...`, `go test ./...`.
  `panel` runs on bun 1.4.4: `bun install --frozen-lockfile` and
  `bun run verify` in `apps/web`. Triggers: push to `main` and all pull
  requests. Actions: `actions/checkout@v4` (same as the other workflows),
  `actions/setup-go@v5`, `oven-sh/setup-bun@v2`.
- `.containerignore`: ignore `apps/web/node_modules` and
  `apps/web/dist`, so `COPY apps/web/ ./` cannot copy a host tree over the
  container install. No `Containerfile` change was needed.
- `scripts/build-web.sh`, `scripts/install.sh`: `bun install
  --frozen-lockfile`. `scripts/check-panel.sh` already checks the title
  `LLM Proxy API Management Center`.
- Docs: root `README.md` panel section (BumpyClock panel, embedded,
  same origin, management key login, mise and bun builds),
  `apps/web/README.md` and `apps/web/AGENTS.md` (v8 only, same origin,
  tokens rule, verify, add a page), `UPSTREAM_BASELINE.md` panel notes
  (BumpyClock `ce5f6179` is the source, official `953fb24` is not, list of
  fork changes). `DECISIONS.md` not touched.

Commits:

- `5714780d` docs(issues): start issue 0034 panel pipeline
- `63962d08` ci: run Go and panel checks on push and pull request
- `31c8d94f` build: ignore panel build output in container context, freeze web installs
- `566edab9` docs: describe the BumpyClock panel, its build, and the fork changes
- `ad4ea843` docs(issues): log issue 0034 progress
- final commit: docs(issues): mark 0034 done with result

Gate (after rebase onto `bedf8bde`):

- `bun install --frozen-lockfile` ok; `bun run tokens:check` ok;
  `bun run verify` exit 0: 2042 pass, 0 fail, lint ok, type-check ok,
  build writes `dist/index.html`.
- Go: `gofmt -l` empty; `go build ./...`, `go vet ./...`,
  `go test ./...` all pass (103 packages ok).
- Container: `podman build --target panel` passes; the panel stage
  contains `dist/index.html` and `dist/management.html`.

Known gaps and follow-ups:

- GitHub Actions is unconfirmed: no run was possible here. The YAML was
  parsed with `yq`; `actionlint` is not installed. The Go commands were
  run locally and pass.
- CI runs `go test ./...` on the whole tree. That is what the issue
  asks; it does not split the slow packages.
- `apps/web` does not proxy the API in `bun run dev`, so the panel works
  end to end only through the proxy. This is the current code state, not
  a new gap.

