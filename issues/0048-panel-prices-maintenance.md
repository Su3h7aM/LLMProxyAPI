---
status: done
branch: issue/0048-panel-prices-maintenance
type: panel
agent: deepseek-v4.1-flash
base: main
---

# Panel: prices page and usage maintenance

## Goal

Build the model prices page and the usage database status and maintenance section. Route `/usage/prices` (reachable from a link on the Usage page header and the sidebar group of the usage item, if the sidebar has groups; else a tab or link on the page), and a usage database card on the System page.

## References (read-only)

- `docs/usage-api-contract.md` (shapes, units, defaults), `docs/usage-prices.md`, `docs/usage-statistics-design.md`, `DECISIONS.md` (entry 2026-11-12), `apps/web/AGENTS.md`, `issues/0046-panel-usage-data-dashboard.md` (Result: client, hooks, store, formatters, types you must reuse).
- Data layer already merged (issue 0046): `src/types/usageStats.ts`, `src/services/api/usage.ts`, `src/features/usage/logic/`, `src/features/usage/hooks/useUsageQuery.ts`, `src/stores/useUsageFilterStore.ts`. Reuse them. Do not write a second client.
- UI building blocks: `src/components/ui/` (Table, Sheet, SegmentedControl, Select, TimeSeriesChart, PageHeader, Card, EmptyState, Skeleton), `src/design-system`, dashboard components `src/features/dashboard/components/` (Sparkline, ThroughputChart, Meter).
- Style source: BumpyClock panel `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (git-ignored; read by absolute path; never edit). Its `src/pages/UsagePage.tsx` and chart components show the intended look and layout of a usage page: copy the structure and patterns, not the old data format.
- The backend routes are built (issues 0045 and 0049, 0049 may still be open): you can run the real server for a manual check. Build it with `/usr/bin/go build -o /tmp/<dir>/llmproxy ./cmd/server` after you build the panel (`bun run build`, then copy `apps/web/dist/index.html` to `apps/web/dist/management.html`), use a temp config and auth-dir in `/tmp`, and stop the process when done.

## Design rule

The BumpyClock interface, style, and design system do not change. New UI is built from its components, tokens, layout, and patterns, copying the structure of a similar existing page or component. No raw colors, sizes, or fonts: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing UI.

## Steps (small commits, Conventional Commits, English)

1. **Prices client.** Add the price calls to `src/services/api/usage.ts` if the 0046 client lacks any: list, history, put (manual price), delete manual, sync, status, price-unpriced, correct, per `docs/usage-prices.md` and the route examples in `issues/0044-usage-prices-cost.md` (Result). Types in `src/types/usageStats.ts`. Rates on the wire are micro-USD per 1M tokens (list) and USD per 1M tokens (PUT body): convert in one place with tests.
2. **Prices page.** `src/features/usage/PricesPage.tsx` (new), route `/usage/prices`, labels in the 4 locales under the namespace `usage_prices`. Table: provider, model, input, output, cache read, cache write, reasoning (USD per 1M, formatted), source badge (`models.dev`, `manual`, `embedded`), valid from, synced at; filter by provider and source, text search, paging. Row actions: edit (opens a `Sheet` with the form: rates, effective-from date-time default now, note; validation: non-negative numbers, required input and output), history (a `Sheet` listing all versions of the model, newest first, with valid ranges), remove manual (confirm dialog; explain that the synced price applies again from now). "Add price" button for a model not in the list (provider and model fields).
3. **Sync and status.** A header block with the last sync time, last error (if any), counts by source, and a "Sync now" button (shows the result counts: added, updated, unchanged, skipped overrides; error text on failure). Show the price-sync config state if the config tree exposes it (`observability.usage-db.price-sync`; read-only here).
4. **Unmatched models.** A list from `prices/status` `unmatched` (provider, model, events). Each row has an "Add price" action that opens the form prefilled. After a price is added, show a toast that unpriced events of the model are filled in the background, and refresh the list.
5. **Actions with consequences.** "Price unpriced events" button (optional range and model; shows counts). "Correct costs" in an advanced section: needs a date range (max 366 days), optional model, a reason text, and a typed confirmation (the user types the word `correct` or its localized equivalent); a warning text says history is frozen and this rewrites cost for the range, with an audit entry; show the result (events updated, old and new totals).
6. **Usage database card on the System page.** In `src/pages/SystemPage.tsx` (or the feature part it uses) add a card from `GET /status`: enabled, path, size, schema version, events, oldest and newest event, buffer depth and capacity, dropped, write errors and last error, retention days, last prune. Buttons: prune now, checkpoint, vacuum (confirm: it blocks writes for a short time), each calling `POST /maintenance` and showing the result. Show a warning state when `dropped` or `write_errors` is above zero. Disabled state when the DB is disabled.
7. **Tests.** Rate conversion, form validation, table filters and paging, history sheet, remove-manual confirm, sync result and error rendering, unmatched list and prefill, correct flow (typed confirmation, range cap), status card states (ok, warning, disabled), maintenance calls. Locale coverage and token guard must pass.
8. **Manual check** against the real server as in the References (record commands and results in the Log): add a manual price by `curl` through the same client paths, sync against a local fixture server, check that the page data matches. List what you could not check visually under `## Follow-ups`.

Do NOT do in this issue: the Usage page (issue 0047), Go changes.

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

1. Work on branch `issue/0048-panel-prices-maintenance` in your checkout. Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end: `status: done`, fill `## Result` (what landed, how it looks, known gaps, a manual check against the real server with the commands you ran), commit, report. Do not merge to `main`; the coordinator merges.

## Log

- 2026-11-12: Read `AGENTS.md`, `apps/web/AGENTS.md`, this issue, `docs/usage-api-contract.md`, `docs/usage-prices.md`, the 0044 Result, the price Go handlers, and the 0043 status handler. Branch `issue/0048-panel-prices-maintenance` on `main` (bc037809). Status in-progress.
- 2026-11-12: Data layer (0046) reused: `usageApi` gained the seven price calls and `postMaintenance`. No second client.
- 2026-11-12: Logic in `features/usage/logic/prices.ts` (rate conversion, list query and paging, form validators, correction and unpriced validators, read-only sync config, source tone) and `logic/usageDb.ts` (card state). Tests in `tests/pricesLogic.test.ts` and `tests/usageDbLogic.test.ts`.
- 2026-11-12: `PricesPage` at `/usage/prices`, with `PriceSyncPanel`, `PricesTable`, `UnmatchedPrices`, `PriceFormSheet`, `PriceHistorySheet`, `PriceUnpricedPanel`, `CorrectionPanel`. Sidebar item in the `observe` group. `UsageDbCard` on `SystemPage`.
- 2026-11-12: Locale keys `usage_prices` and `usage_db` in the four locales, inserted as text.
- 2026-11-12: Removed a leftover `tests/versionFailure.test.ts` edit from an earlier session: the full suite passes without it, so the original test stands unchanged.
- 2026-11-12: Correction form is a collapsed `Collapsible` ("advanced") inside its card. Source badge has a tone per source (manual warning, synced success, else neutral).
- 2026-11-12: Gate green: `bun install --frozen-lockfile`, `bun run verify` (2183 tests, 0 fail; lint; type-check; build), `bun run tokens:check`.
- 2026-11-12: Manual check on the real server, see Result. All processes stopped, ports free.
- Note for the coordinator: the "link on the Usage page header" belongs to issue 0047, so this issue adds only the sidebar entry (same `observe` group) plus this page. No conflict expected.

## Follow-ups

- No DOM test harness in `apps/web` (tests render static markup), so the remove-manual confirm, the sync button, and the maintenance buttons are covered at logic and render level only, not by a click test.
- The `unmatched` list was not seen with real events: the temp server had no upstream credential, so no event was stored. Route shape (empty list) checked. Needs a stub upstream to see the list and the background fill toast.
- Visual layout and the two sheets were not checked in a browser.
- Price tiers (`context` and `service`) are received in the list but not shown; no route adds them yet.

## Result

Landed on `issue/0048-panel-prices-maintenance` (no merge to `main`).

- `src/features/usage/PricesPage.tsx` + `PricesPage.module.scss`: route `/usage/prices`; sync and status header, price table with provider/source/text filters and paging, unmatched list, price-unpriced panel, collapsed advanced correction panel, add/edit/history/remove-manual actions.
- `src/features/usage/components/`: `PricesTable`, `PriceSyncPanel`, `UnmatchedPrices`, `PriceFormSheet`, `PriceHistorySheet`, `PriceUnpricedPanel`, `CorrectionPanel`, `UsageDbCard`, `pricePanels.module.scss`, `usageDbCard.module.scss`.
- `src/features/usage/logic/prices.ts`, `logic/usageDb.ts`: one place for the micro-USD list rates and the USD PUT rates, form validators, correction and unpriced validators, read-only `observability.usage-db.price-sync` state, card state.
- `src/services/api/usage.ts`: list, history, put, delete manual, sync, status, price-unpriced, correct, maintenance. `src/types/usageStats.ts`: price and maintenance shapes.
- `MainRoutes.tsx`: `/usage/prices`. `MainLayout.tsx`: sidebar item in `observe`. `SystemPage.tsx`: `UsageDbCard`.
- Locales: `usage_prices` and `usage_db` in `en`, `zh-CN`, `zh-TW`, `ru`.

Manual check (`/tmp/lpamc-0048`; fixture `api.json` with `stubprovider`; server on 18318, price source on 18991):

```
/usr/bin/go build -o /tmp/lpamc-0048/llmproxy ./cmd/server
curl -X POST .../prices/sync        -> {"added":2,...,"error":""}
GET .../prices/status               -> {"counts_by_source":{"embedded":323,"models.dev":2},"unmatched":[]}
GET .../prices?provider=stubprovider&q=stub -> input 2000000, output 8000000, source models.dev
PUT .../prices {"input":3,"output":9,"cache_read":0.75} -> input 3000000, output 9000000, source manual
GET .../prices/history?provider=stubprovider&model=stub-model -> 2 versions, synced closed
DELETE .../prices/manual -> {"restored":true}; new models.dev version open
POST .../prices/price-unpriced {}   -> {"priced":0,"skipped_no_price":0}
POST .../prices/correct (400 days)  -> 400; without confirm -> {"error":"confirm must be true"}
POST .../prices/correct (valid)     -> {"correction_id":1,"events_updated":0,...}
POST .../maintenance prune|checkpoint|vacuum -> action, duration_ms, and per-action counts
```

Disabled state: second instance with `observability.usage-db.enabled: false` gave `status` 200 with `enabled:false` and `prices` 503, which the page shows as the disabled card. `/management.html` answered 200 and the built `dist/index.html` holds the page and the route. Units on the wire: list rates in micro-USD per 1M (2 USD = 2,000,000), PUT body in USD per 1M.

Known gaps: see Follow-ups.
