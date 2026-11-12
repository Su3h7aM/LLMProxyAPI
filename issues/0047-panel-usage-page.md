---
status: todo
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

## Follow-ups

## Result
