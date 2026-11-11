---
status: todo
branch: issue/0022-panel-config
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel v8: config page, config file, access keys, routing strategy

## Goal

Port the config area of the panel to the v8 API: the config store, the visual config page, the raw YAML editor, access API keys, and the routing strategy select with the fork-only `soonest-reset` option.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0022-panel-config integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **API modules.** `src/services/api/config.ts`, `configFile.ts`, `apiKeys.ts`, `apiKeyUsage.ts`, `transformers.ts`, and `src/stores/useConfigStore.ts` to v8: `GET /config` with ETag, tree writes with the `configPatch.ts` helpers (already on the branch), `GET`/`PUT /config.yaml`, request log flag via `PUT /config/observability/logs/request-log` (check the pre-import `config.ts`), access keys through the config tree (the pre-import visual config uses `access.api-keys`), API key usage via `GET /observability/usage/api-keys`. Use the pre-import modules as reference and check every route in Go (`internal/api/server_management_v8.go`).
2. **Visual config.** Fix `useVisualConfig` and its types so the 11 skipped tests in `tests/configPatch.test.ts` pass (remove the skips; do not edit the expectations unless the Go config really differs: then explain in the commit body).
3. **Missing sections.** The pre-import panel has config sections that BumpyClock lacks: `SectionDiscovery`, `SectionOAuthBehavior` (and fields for v8 settings that exist in the Go config types). Add them in BumpyClock section style (copy the structure of its existing sections and fields components). Port their tests.
4. **Routing strategy.** Add `soonest-reset` as the fourth strategy in the visual config parser, the type union, the `SectionNetwork` select (label and help text in the 4 locales: spends the account whose quota window resets first), the config search index entry, and the preview text (no fixed split). Reference: commit `ecc09878` on `main` (`git show ecc09878`). The routing board (page) is another issue.
5. **Tests.** Port pre-import tests for these modules; keep BumpyClock tests green.
Do NOT do in this issue: the routing board page, provider lists, credentials, logs.

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
