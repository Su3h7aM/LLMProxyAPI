---
status: todo
branch: issue/0037-panel-logs-level-apikey-names
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel: log level filter and client API key names

## Goal

Bring back two features of the pre-import panel that the new panel lacks: a log-level filter on the Logs page, and local display names for client API keys in the config editor. Build both in the BumpyClock design.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0037-panel-logs-level-apikey-names integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Log level filter.** Pre-import: `features/logs` (`git show panel-pre-bumpyclock:apps/web/src/pages/LogsPage.tsx`, lines around the `logs.level_filter` key). New panel: `src/pages/LogsPage.tsx` and `src/pages/hooks/useLogFilters.ts` filter by method, status, and path only. Add a level filter (for example all, info, warn, error; use the levels that the Go log lines carry: check the log format in `internal/logging` and the parser in the Logs page) in BumpyClock style: copy how the existing filters are built and placed. Add strings in the 4 locales (en, zh-CN, zh-TW, ru; text insertion only, the locale files have a duplicated `errorRules` key that a JSON re-dump would collapse). Add tests for the filter logic.
2. **Client API key names.** Pre-import: `src/features/config/apiKeyNames.ts` and `ApiKeysCardEditor.tsx` (lines about 6 to 44). A user can give a local display name to a client API key; the name is stored in the browser only and keyed by a hash of the key (it uses `@noble/hashes`). Port this into the BumpyClock config API keys editor (find the component that edits the access keys, `features/config/components/...`, written in issue 0022) in BumpyClock style. The dependency `@noble/hashes` is not in `apps/web/package.json`: add it with the same version range as the pre-import `package.json`, using `bun add` so that `bun.lock` stays consistent, then check `bun install --frozen-lockfile` still passes. Show the names where the pre-import panel showed them (also check the API key usage display `services/api/apiKeyUsage.ts` consumers). Add strings in the 4 locales and port the pre-import tests that fit.
3. **Gate.** `bun run verify` green, `bun run tokens:check` green.
Do NOT do in this issue: other config fields, other pages.

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
