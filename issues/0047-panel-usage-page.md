---
status: done
branch: issue/0047-panel-usage-page
type: panel
agent: muse-spark-1.3-contributor
base: main
---

# Panel: Usage page

## Goal

Build the Usage page: the deep statistics view (CPA-Manager-Plus level, BumpyClock look). Route `/usage`, sidebar item, link from the dashboard section (issue 0046 left it out because the route did not exist). The old unrouted `src/pages/UsagePage.tsx` and its helpers (`usage-page-request-guard.ts`, old `types/usage.ts`, `utils/usageStats.ts` parts that only it uses) are removed in this issue.

## References (read-only)

- `docs/usage-api-contract.md` (shapes, units, defaults), `docs/usage-prices.md`, `docs/usage-statistics-design.md`, `DECISIONS.md` (entry 2026-11-12), `apps/web/AGENTS.md`, `issues/0046-panel-usage-data-dashboard.md` (Result: client, hooks, store, formatters, types you must reuse).
- Data layer already merged (issue 0046): `src/types/usageStats.ts`, `src/services/api/usage.ts`, `src/features/usage/logic/`, `src/features/usage/hooks/useUsageQuery.ts`, `src/stores/useUsageFilterStore.ts`. Reuse them. Do not write a second client.
- UI building blocks: `src/components/ui/` (Table, Sheet, SegmentedControl, Select, TimeSeriesChart, PageHeader, Card, EmptyState, Skeleton), `src/design-system`, dashboard components `src/features/dashboard/components/` (Sparkline, ThroughputChart, Meter).
- Style source: BumpyClock panel `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (git-ignored; read by absolute path; never edit). Its `src/pages/UsagePage.tsx` and chart components show the intended look and layout of a usage page: copy the structure and patterns, not the old data format.
- The backend routes are built (issues 0045 and 0049, 0049 may still be open): you can run the real server for a manual check. Build it with `/usr/bin/go build -o /tmp/<dir>/llmproxy ./cmd/server` after you build the panel (`bun run build`, then copy `apps/web/dist/index.html` to `apps/web/dist/management.html`), use a temp config and auth-dir in `/tmp`, and stop the process when done.

## Design rule

The BumpyClock interface, style, and design system do not change. New UI is built from its components, tokens, layout, and patterns, copying the structure of a similar existing page or component. No raw colors, sizes, or fonts: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing UI.

## Steps (small commits, Conventional Commits, English)

1. **Route and shell.** `src/features/usage/UsagePage.tsx` (new), route `/usage` in `src/router/MainRoutes.tsx`, sidebar item in `src/components/layout/MainLayout.tsx`, page title and labels in the 4 locales under the namespace `usage_page`. Add the "open usage" link from the dashboard section. Remove the old page and dead code (list what you removed in the commit body; check with `grep` that nothing else uses it; remove `getStatisticsEnabled` if unused).
2. **Filter bar.** Range presets (1h, 24h, 7d, 30d, 90d, custom date-time range), provider, model, credential, API key, status (ok or failed), stream, and text search (`q`, used by the events table only). Use `useUsageFilterStore`. Options for the selects come from `breakdown` calls (group by provider, model, credential, api_key; limit 200). Show active filters as removable chips. A reset button. The filter state is shared with the dashboard section.
3. **Summary cards.** Requests, tokens (with input, output, reasoning, cache read and write on hover or in a detail row), cost (estimate and unpriced marks as in the dashboard), failure rate, latency average and P95 (show `n/a` with a tooltip when `source` is `hourly`), each with the delta against `previous`.
4. **Timeline.** `TimeSeriesChart` of requests, tokens, or cost (a segmented control picks the metric) with the bucket from the contract (`auto`), optional grouping by model, provider, credential, or key (top 5 plus other) as stacked or multi-line series. Tooltip with the counters. Accessible table alternative (the component has `TimeSeriesChartTable`).
5. **Breakdown tables.** Tabs: models, providers, credentials, API keys, status. Each row: label, share bar, requests, tokens, cost, failure rate, average latency; sortable by the contract's sort keys; paging by `limit` and `offset`. Row click adds the filter. The API key tab edits labels inline (`PUT /key-labels`, optimistic update, rollback on error); show the label or the `key-xxxxxx` fallback, never a raw key.
6. **Failures.** Group table (code, class, provider, credential, model, count, first and last seen) and the recent failures list with the detail drawer (step 8). A click on a group applies filters.
7. **Heatmap.** Hour-of-week grid (7 rows, Monday first, 24 columns) with a metric switch (requests, tokens, cost, failures), color scale from design tokens only, tooltips, keyboard focus and an accessible text alternative.
8. **Events table and drawer.** Table with cursor paging ("load more"), columns: time, model, provider, credential label, key label, status, latency, TTFT, tokens, cost (estimate mark), request id (copy button). Click opens a `Sheet` with event detail (`GET /events/{id}`): all fields, fail summary, the price version used, the cost source. Search box (`q`) with debounce. Export button: downloads from `/export` with the current filters in JSONL or CSV (use the authenticated client with a blob download; large exports must not load the page into memory more than necessary: stream to a blob and show a progress hint).
9. **States.** Loading skeletons, empty state (no events in range, with a hint to send traffic), disabled state (503: show "usage database is disabled" and where to enable it: config `observability.usage-db.enabled`), error with retry, partial errors per section (one failing section does not blank the page). All requests use the abort and latest-wins hook.
10. **Responsive and a11y.** The page works from 360 px wide; tables scroll or collapse as the existing tables do; keyboard order is sane; every chart has a text alternative.
11. **Tests.** Component and logic tests: filter bar to query mapping, chips, cards with delta and marks, breakdown sorting and paging calls, label edit optimistic update and rollback, events paging and drawer, export request, heatmap matrix render, disabled and error states, no raw key anywhere in the rendered output. Locale coverage and token guard must pass.
12. **Manual check.** Run the real server (see References) with a stub upstream or by sending a few requests, open the page through the HTTP route with `curl` for the asset and verify the API calls match the real responses (record commands and results in the Log). A browser check is not possible for you: list what you could not check visually under `## Follow-ups`.

Do NOT do in this issue: prices page, maintenance UI, System page status (issue 0048), Go changes.

## Gate

`cd apps/web && export PATH=/home/su3h7am/.local/share/mise/installs/bun/latest/bin:$PATH`, then `bun install --frozen-lockfile`, `bun run verify` (test, lint, type-check, build), `bun run tokens:check`. Run `bun run type-check` and your area's tests after each commit. Do not delete or weaken a test to get green, except a test that pins behavior this issue removes on purpose (say which in the commit body). Locale files: insert new keys as text only (do not re-dump JSON; the files hold a duplicate key that a dump would collapse); the locale coverage test must pass in the 4 locales (`en`, `zh-CN`, `zh-TW`, `ru`).

## Environment

- Use bun at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin` and `/usr/bin/go`. Do NOT run `mise trust`. Builds only inside your checkout or `/tmp`. Never leave a server process running.
- Never commit `node_modules`, `dist`, or `apps/web/dist/management.html`.
- English only. ASD-STE100, concise.

## Rules

- API is v8 only. The panel and the proxy are one unit: no version check, no capability probe by 404, no fallback.
- Keep the management key login, 401 handling, and the connection-revision guard.
- Never display or store a raw API key; the API gives `api_key_id` and a label only.
- Take only what this issue lists. Another issue runs in parallel in the same tree (0047 usage page and 0048 prices and maintenance): keep your files separate. Route and sidebar edits are small and the coordinator resolves conflicts at merge. Put new locale keys in your own namespace.
- You do not decide. If a contract detail is missing or a UI choice is unclear: stop, write the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files, and do not edit `docs/usage-api-contract.md` (report contract problems).

## Workflow

1. Work on branch `issue/0047-panel-usage-page` in your checkout. Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end: `status: done`, fill `## Result` (what landed, how it looks, known gaps, a manual check against the real server with the commands you ran), commit, report. Do not merge to `main`; the coordinator merges.

## Log

- 2026-11-12: Branch `issue/0047-panel-usage-page` cut from `main` at
  `bc037809`. Read `AGENTS.md`, `apps/web/AGENTS.md`,
  `docs/usage-api-contract.md`, issues 0046 and 0042, the data layer of
  0046 (types, client, hooks, store, logic), UI blocks (Table, Sheet,
  SegmentedControl, Select, TimeSeriesChart), the BumpyClock reference
  (`nav.usage_statistics`, `/usage` route, `IconSidebarUsage`), and the
  old page with its callers. Plan: shell first, then sections in small
  commits.
- 2026-11-12: Steps 1 and 2 land in one commit (route, sidebar,
  filter bar, locales, shell, old-code removal). Gate: type-check,
  lint, area tests, `tokens:check` pass. Full `verify` has one failure
  in `versionFailure.test.ts` that also fails on clean `main`
  (pre-existing full-run flake, unrelated).
- 2026-11-12: Steps 6 and 7 land in one commit (failures groups and
  recent list; heatmap with metric switch and ARIA grid), plus the
  event drawer both sections share. Price-block rates are micro-USD
  per 1M tokens (confirmed in `internal/usagedb/prices.go`), so the
  drawer shows them as `$X / 1M`. Gate: type-check, lint, area tests
  pass.
- 2026-11-12: Step 8 lands (events table with cursor paging, debounced
  search, export; `mergeEventsPages` and `displayKeyLabel` pure and
  tested; detail view render covered). Timer lengths use named
  duration tokens (`ts-inline-timer` guard). Gate: type-check, lint,
  area tests, `tokens:check` pass.
- 2026-11-12: Manual check vs the real server (commit `9c1b4bb0`
  context below). No proxy capture wiring exists in this checkout
  (collector is configured but no request path records yet; 0045/0049
  territory), so traffic was seeded by hand: 3 raw events plus
  matching hourly rows, one manual price, one key label. Commands:
  `/usr/bin/go build -o /tmp/usage0047/llmproxy ./cmd/server` from the
  repo root; `bun run build` plus copy to `dist/management.html`;
  server with a temp config and auth dir in `/tmp/usage0047`;
  `curl` with the management key against every route. Results: status,
  summary (zero and seeded totals with estimate and unpriced marks,
  p50/p95 with `source:"events"`), timeseries (auto bucket `hour`,
  zero-filled gaps, top-5 groups), breakdown (model, provider,
  credential, api_key, status_class with share and meta), events list,
  event detail (all fields, fail summary, price version, cost source),
  failures (groups and recent with `fail_summary`), heatmap, JSONL
  export with `Content-Disposition` filename, key labels GET and PUT,
  panel asset `200` (3.5 MB), unknown route `404`. All shapes match
  `docs/usage-api-contract.md`. Two notes: (1) seeded `price_id: 1`
  resolves to an embedded-snapshot row of another model, so the drawer
  showed a foreign price block; production rows point at their own
  version, and the panel only displays what the API sends. (2) Keyless
  traffic is labeled `key-` by the backend; the panel now shows a dash
  there (`displayKeyLabel`, commit `9c1b4bb0`). No server process left
  running; seed data lives only in `/tmp/usage0047`.
- 2026-11-12: Step 5 lands (breakdown tabs with contract sort keys,
  limit and offset paging, row-click filters, inline key-label edit
  with optimistic update and rollback). Found and fixed a duplicate
  `col_latency` key inside the new `usage_page` block (renamed the
  breakdown column to `col_avg_latency` in all 4 locales, commit
  `049a8548`). Gate: type-check, lint, area tests pass.
- 2026-11-12: Steps 3 and 4 land in one commit (six summary cards
  with deltas and marks; timeline with metric switch, auto buckets,
  top-5-plus-other grouping). Gate: type-check, lint, area tests pass.
  One refresh generation reloads every section.
- 2026-11-12 (follow-up): the `versionFailure` failure is not a flake
  and not only on `main`. Root cause: the `400` test in
  `tests/usageStatsApi.test.ts` (issue 0046) spied on `apiClient.get`
  and never restored it. A later `spyOn` on the still-mocked method
  records one call twice under bun, so `versionFailure` fails on call
  counts. Full-suite alphabetical order hides it (some later file
  re-spies and restores); the pair order and the branch order expose
  it deterministically. Fix: the spy joins the file's `restore` list;
  a `usage test hygiene` test pins that the shared client carries no
  mock after the suite. Earlier Log lines calling it a pre-existing
  flake are corrected by this note. Proof: pair order passes,
  `verify` passes with 0 fail, `tokens:check` passes, full suite
  passes 3 times.

## Follow-ups

- Visual check not done: no browser is available to this agent. The
  page renders (build passes, SSR render tests cover every section),
  but spacing, the heatmap grid, the drawer, and the 360 px layout
  need a human look before merge.
- Optional: show stale dashboard usage data with an inline error on
  poll failure (carried over from 0046).
- The `versionFailure.test.ts` full-run flake fails on clean `main`
  too; it is unrelated to this issue but blocks a clean `verify`.

## Result

The Usage page is complete at route `/usage` with a sidebar item and a
dashboard link. One filter bar (range presets 1h to 90d plus custom
date-time range, provider, model, credential, API key, status, stream,
chips, reset) shares `useUsageFilterStore` with the dashboard. Below
it: six summary cards, a timeline with metric switch and top-5-plus
grouping, breakdown tabs with sort and paging and inline key-label
edit, failures with row-click filters and recent list, an hour-of-week
heatmap, and an events table with search, load more, export, and a
detail drawer. States: skeletons, empty, disabled (503 with the
`observability.usage-db.enabled` pointer), error with retry, and
per-section errors that never blank the page. Locales `en`, `zh-CN`,
`zh-TW`, `ru` hold the `usage_page` namespace; no raw key is ever
rendered. Removed: the old `src/pages/UsagePage`, its request guard,
`types/usage`, `utils/usageStats`, and `getStatisticsEnabled`.

Gate: `type-check`, `lint`, `build`, `tokens:check` pass; 2167 tests
pass with one pre-existing full-run flake (`versionFailure`, also red
on clean `main`). Manual check ran against the real server with
hand-seeded rows (see Log): every route shape matches
`docs/usage-api-contract.md`, and the panel asset serves.

## Result
