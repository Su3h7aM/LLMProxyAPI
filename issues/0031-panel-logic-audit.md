---
status: in-progress
branch: issue/0031-panel-logic-audit
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel: restore provider-facing logic fixes that the BumpyClock base lacks

## Goal

The BumpyClock panel comes from an older base than the official panel that the fork used before. It lacks some non-UI fixes that the pre-import panel had. Example found: the Claude quota request header `User-Agent: claude-cli/...` is gone (`CLAUDE_REQUEST_HEADERS` in `src/utils/quota/constants.ts`), and the backend `POST /requests/api-call` adds no User-Agent. Find and restore such fixes.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0031-panel-logic-audit integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Diff.** Compare the non-UI logic of the pre-import panel (tag `panel-pre-bumpyclock`) with the integration branch: `src/utils/quota/*`, `src/features/quota/providers/*/data.ts` and request builders, quota parsers and validators, `src/services/api/*` request details (headers, bodies, timeouts, retries), `src/features/providers` model-discovery and test-request builders, `src/utils/*` helpers that touch upstream provider APIs. Use `git diff panel-pre-bumpyclock integration/panel-bumpyclock -- <path>` and read the history of the pre-import side (`git log panel-pre-bumpyclock -- <path>`; look for the official panel commit messages with "fix").
2. **Classify.** In `## Log`, make a table: file, delta, class. Class A = BumpyClock is newer or better, keep. Class B = the pre-import value is a fix for a provider API (header, user agent, URL, request body, response parsing, rate-limit or error handling) and BumpyClock lacks it. Class C = UI only or unclear: do not touch.
3. **Restore class B.** Apply only class B changes, one small commit per fix, with a test that pins the restored behavior. Keep the BumpyClock code style. Do not touch UI structure or styling.
4. **Report class C and doubtful items** in `## Follow-ups` for the coordinator, with file, delta, and your reason.
5. **Verify routes.** Write a small script in `/tmp` (do not commit) that extracts every `apiClient` path in `apps/web/src/services/api` and in feature modules (static and template paths) and matches it against the routes in `internal/api/server_management_v8.go`. Put the table (path, method, route found or not) in `## Log`. Report any call without a route; do not fix them unless it is a class B item.
Do NOT do in this issue: new features, cleanup, OpenCode Go (own issue).

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

## Follow-ups

## Result
