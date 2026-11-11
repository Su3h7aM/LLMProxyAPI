---
status: in-progress
branch: issue/0027-panel-dashboard-usage
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel v8: dashboard and Usage page on observability routes

## Goal

The dashboard and the Usage page of the BumpyClock panel read `GET /usage`, which the fork backend does not serve. Move them to the v8 observability routes, keeping the BumpyClock UI.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0027-panel-dashboard-usage integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Inventory first.** In `## Log`: list the v8 observability and usage routes that the Go backend serves (`internal/api/server_management_v8.go` and `internal/api/handlers/management/`; look for `/observability/usage/*`, usage queue, `usage-statistics-enabled` in the config tree). List what the pre-import dashboard (`git show panel-pre-bumpyclock:apps/web/src/features/dashboard/...`) reads and from which route. List every data point that the BumpyClock dashboard and Usage page need (`features/dashboard/hooks/useDashboardOverview.ts`, `pages/UsagePage.tsx`, `features/dashboard/logic/*`) and mark each as: available on v8 (route and field), available after a client-side reshape, or not available.
2. **API.** Replace `src/services/api/usage.ts`: no `/usage`, no `/usage-statistics-enabled`. The usage statistics flag is read and written through the config tree (check the pre-import code and `internal/config`). Point the dashboard hooks and the Usage page to the v8 routes.
3. **Reshape.** Where the v8 data has another shape, write the reshaping in a small pure module with tests. Keep the charts and cards of BumpyClock as they are.
4. **Gaps.** If a chart needs data that no v8 route gives, do not invent a route and do not remove the chart silently: stop on that item, write the exact need in `## Log` and `## Follow-ups` (route sketch: method, path, response), keep the rest of the issue moving, and show an empty state for that chart in BumpyClock style.
5. **Tests.** Port pre-import dashboard tests that fit; keep BumpyClock tests green.
Do NOT do in this issue: quota page, routing board, credentials.

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

- 2026-11-11: Set `status: in-progress`. Read `DECISIONS.md`, triage 0015-0017,
  `server_management_v8.go`, `handlers/management/usage.go`, `api_key_usage.go`,
  `config/config_v8.go:116`, pre-import `features/dashboard/hooks/useDashboardOverview.ts`,
  BumpyClock `useDashboardOverview.ts`, `UsagePage.tsx`, `logic/usage.ts`,
  `UsagePanel.tsx`, `utils/usageStats.ts`.
- 2026-11-11: Inventory (step 1). v8 routes served:
  (a) `GET /v8/management/observability/usage/api-keys` (`GetAPIKeyUsage`):
  `{provider: {base_url|api_key: {success, failed, recent_requests[]}}}`.
  Panel already reads it (`apiKeyUsageApi`, `useProviderRecentRequests`, fleet).
  (b) `GET /v8/management/observability/usage/queue` (`GetUsageQueue`): POPS
  raw queue records (`redisqueue.PopOldest`; relay transport pushed to a remote
  client in `sdk/cliproxy/service_home.go:349`). Destructive internal plumbing,
  never a display source. The panel must not call it.
  (c) Flag `usage-statistics-enabled`: v8 tree
  `observability.usage.usage-statistics-enabled` (`config_v8.go:116`); read
  `GET /config/observability/usage/usage-statistics-enabled`, write via tree
  PUT (config page already does both: `useVisualConfig.ts:1367,1634`).
  Pre-import dashboard reads only credentials + api-key-usage + config + models.
  Pre-import has no Usage page, no `usage.ts`, no usage source at all.
  BumpyClock data needs: 1. api-key traffic: available (a). 2. credential
  list/health: available (`GET /credentials`). 3. config counts: available
  (`GET /config`). 4. model list: available (models store via api-call).
  5. statistics flag: available after client reshape (c) (was flat
  `GET /usage-statistics-enabled`, v0-only and absent from v8).
  6. aggregate `UsageSnapshot` (per-api/model totals, tokens, latency details,
  daily/hourly buckets): NOT available. `GET /usage` exists on no backend
  (fork v0 lacks it, v8 lacks it; Plus-only). `RecentRequestBucket` is
  `{time, success, failed}` only: per-model/token/latency/daily history cannot
  be rebuilt from (a) or auth-file snapshots. Gap item, see Follow-ups.
  7. daily/hourly history series: NOT available (same gap).
- 2026-11-11: Rendering consequence (no new copy, BumpyClock style kept):
  dashboard `UsagePanel` keeps its `usage_unsupported` block ("This server
  does not report per-model usage...") and shows it when the flag is on but no
  snapshot exists; vitals `usageDiffers` banner is gated on snapshot presence
  so it cannot fire on the permanent null snapshot; Usage page treats
  enabled-without-snapshot as its existing `empty` status (header only, no
  numbers or charts; `showData` stays false), which matches the codebase rule
  that missing fields stay unreported instead of manufacturing zero metrics.

## Follow-ups

## Result
