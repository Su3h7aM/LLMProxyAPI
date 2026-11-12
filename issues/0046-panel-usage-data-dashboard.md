---
status: todo
branch: issue/0046-panel-usage-data-dashboard
type: panel
agent: deepseek-v4.1-flash
base: main
---

# Panel: usage data layer and dashboard usage section

## Goal

Give the panel a typed client, hooks, and formatters for the new usage API, and show a usage section on the dashboard from real data. The Usage page (0047) and the prices page (0048) build on this issue, so the data layer must be complete and tested. The backend routes are being built in issue 0045 in parallel: code against `docs/usage-api-contract.md`, with mocked responses. Price routes are in `docs/usage-prices.md`.

## References (read-only)

- `docs/usage-api-contract.md` (shapes, units, defaults), `docs/usage-prices.md`, `docs/usage-statistics-design.md`, `DECISIONS.md` (entry 2026-11-12), `apps/web/AGENTS.md`, `issues/0042-triage-fork-usage-capabilities.md` section 4 (what the panel has now).
- Existing code: `src/services/api/usage.ts` (only a flag read), `src/services/api/apiKeyUsage.ts`, `src/features/dashboard/` (`UsagePanel.tsx`, `logic/usage.ts`, `hooks/useDashboardOverview.ts`, `ThroughputChart.tsx`, `Sparkline.tsx`, `VitalsStrip.tsx`), `src/pages/UsagePage.tsx` (old, unrouted, based on a removed data format; 0047 replaces it), `src/types/usage.ts`, `src/utils/usageStats.ts`, `src/components/ui/TimeSeriesChart*.tsx`.
- Style source: BumpyClock panel `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (git-ignored, read by absolute path, never edit). Its dashboard and chart components are the style reference.

## Design rule

The BumpyClock interface, style, and design system do not change. New UI is built from its components, tokens, layout, and patterns, copying the structure of a similar existing page or component. No raw colors, sizes, or fonts: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing UI.

## Steps (small commits, Conventional Commits, English)

1. **Types.** `src/types/usageStats.ts` (new; do not reuse the old `types/usage.ts` shapes): TypeScript types for every response in the contract (counters, summary, timeseries, breakdown, events and event detail, failures, heatmap, key labels, maintenance, status from issue 0043, price items from issue 0044). Time and cost units as in the contract. Name the unit in field names exactly as the contract does.
2. **API client.** Extend `src/services/api/usage.ts` (keep `getStatisticsEnabled` only if something still uses it; if not, remove it and its tests, and say so in the commit body). Add one function per route under `/observability/usage/...` with query building (repeat parameters for multi-value filters; omit empty values), `AbortSignal` support, and the shared error handling (`apiClient`). Handle 503 `usage db disabled` as a typed error that the UI can show as "disabled". No v0 path, no fallback.
3. **Formatters and helpers** (pure, tested): micro-USD to a display string (currency formatting with the locale; show `<$0.01` for tiny non-zero values; keep full precision in a title value), tokens (compact), duration, percentage; delta between `totals` and `previous` (value, direction, and "no data" when previous is zero); range presets (last 1h, 24h, 7d, 30d, 90d, custom) to `from_ms` and `to_ms`; `tz` from the browser; stable filter-to-query mapping; cursor paging helper; an "estimate" flag when `estimated_cost_micro_usd > 0`; an "unpriced" flag when `unpriced_requests > 0`.
4. **Hooks and store.** A small `useUsageQuery`-style hook (or the polled-resource helper the dashboard already has, `hooks/usePolledResource.ts`) with: abort on change and unmount, request-race guard (latest response wins), loading and error states, manual refresh, and optional polling interval. A zustand store (follow the existing stores) for the shared usage filters (range preset, custom range, provider, model, credential, api key, status, stream), persisted in session storage only (not local storage).
5. **Dashboard section.** Rewire `UsagePanel` (and `logic/usage.ts`, `useDashboardOverview.ts` as needed) to the new API: cards for requests, tokens, cost (with estimate and unpriced marks), failure rate, each with the delta against `previous`; a sparkline or throughput chart from `timeseries` (hour buckets for 24h); top models from `breakdown` (group by model, limit 5). Range follows a compact selector on the section (24h, 7d, 30d). Empty state (no events yet) and disabled state (503) with a short hint. Show the section on the dashboard again (remove the "hidden" flag logic that issue 0032 added; find it with `grep -rn "usage" src/features/dashboard`). Add a link to the Usage page route `/usage` (the route comes in 0047: add the link only if the route exists; otherwise leave a TODO-free note in `## Follow-ups`).
6. **Locales.** Add all new strings in the 4 locales (`en`, `zh-CN`, `zh-TW`, `ru`) by text-only insertion (do not re-dump JSON; the files hold a duplicate key that a dump would collapse). The locale coverage test must pass.
7. **Tests.** Client (query building, errors, 503), formatters (rounding, tiny values, locales), delta, presets and tz, filter mapping, hook race and abort, store persistence, dashboard section render with mocked data (normal, empty, disabled, estimate, unpriced). Use the repo patterns in `tests/`.

Do NOT do in this issue: the Usage page (0047), the prices page, maintenance UI (0048), any Go change.

## Gate

`cd apps/web && export PATH=/home/su3h7am/.local/share/mise/installs/bun/latest/bin:$PATH`, then `bun install --frozen-lockfile`, `bun run verify` (test, lint, type-check, build), `bun run tokens:check`. Run `bun run type-check` and your area's tests after each commit. Do not delete or weaken a test to get green, except a test that pins behavior this issue removes on purpose (say which in the commit body).

## Environment

- Use bun at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin`. Do NOT run `mise trust`. Builds only inside your checkout or `/tmp`.
- Never commit `node_modules`, `dist`, or `apps/web/dist/management.html`.
- English only. ASD-STE100, concise.

## Rules

- API is v8 only. The panel and the proxy are one unit: no version check, no capability probe by 404, no fallback.
- Keep the management key login, 401 handling, and the connection-revision guard.
- Never display or store a raw API key; the API gives `api_key_id` and a label only.
- Take only what this issue lists.
- You do not decide. If a contract detail is missing or a UI choice is unclear: stop, write the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files, and do not edit `docs/usage-api-contract.md` (report contract problems).

## Workflow

1. Work on branch `issue/0046-panel-usage-data-dashboard` in your checkout. Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end: `status: done`, fill `## Result` (what landed, how to use the client and hooks, known gaps), commit, report. Do not merge to `main`; the coordinator merges.

## Log

## Follow-ups

## Result
