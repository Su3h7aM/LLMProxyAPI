# Built-in usage statistics: design

Status: accepted by the coordinator on 2026-11-12 (see `DECISIONS.md`). Evidence: `issues/0041-cpamp-inventory.md` (reference product CPA-Manager-Plus, CPAMP) and `issues/0042-triage-fork-usage-capabilities.md` (what the fork has).

## Goal

The proxy keeps its own usage database and serves its own statistics dashboard. No external container (CPAMP) and no Redis queue are needed. The dashboard has the depth of CPAMP and the look of the BumpyClock panel.

## What the fork already has

- One rich in-process record per upstream attempt: `sdk/cliproxy/usage.Record` (tokens with cache read/write and reasoning, latency, TTFT, failure, credential, key, model, alias, optional upstream cost).
- A dispatcher that calls `usage.Plugin.HandleUsage`. No persistence.
- bbolt (`kv.db`) in `<auth-dir>`. It is a key-value store, not suited for aggregates.
- A panel with an unrouted Usage page and no backend data.

## Decisions

| # | Topic | Decision | Reason |
|---|---|---|---|
| 1 | Engine | SQLite through `modernc.org/sqlite` (pure Go). One file `<auth-dir>/usage.db`, WAL mode. | Container build uses `CGO_ENABLED=0`. CPAMP proves this driver at scale. The auth volume already persists. |
| 2 | Source | A built-in `usage.Plugin` that consumes `usage.Record` in process. It has its own bounded buffer and one writer goroutine that inserts in batches. When the buffer is full it drops the oldest record and counts the drop. | The dispatcher queue is unbounded and a slow DB must not stall requests. |
| 3 | No Redis queue | The built-in DB does not use `internal/redisqueue`. The queue stays for external clients. | One unit, no abstraction layer. |
| 4 | Dedup | Unique `request_id` (one UUID per attempt). No import ledger. | Records come in process, not from a replayable queue. |
| 5 | Privacy | The DB never stores the raw client API key. It stores `api_key_id` = first 16 hex of `sha256(key)`. Labels for keys come from the config. `source` and email-like text are masked. No request or response bodies. Failure text is cut to 4096 bytes. | Record carries raw keys (0042). Non-negotiable security rule. |
| 6 | Aggregates | Raw table plus ONE hourly rollup table, updated in the same transaction as the insert (UPSERT). Dimensions: hour, provider, model, alias, auth id, api key id, status class, stream. No checkpoint families. | CPAMP needs 10 families because it reads a queue later. We write both at once. |
| 7 | Retention | Raw events: 90 days by default (config `retention-days`, 0 = keep). A daily job deletes older raw rows in batches. Rollups stay. | CPAMP default is off and the file grows without limit. |
| 8 | Cost | **History is frozen.** The price book is versioned by date: each row has `valid_from_ms` and `valid_to_ms` (open end for the current price). A sync adds a new version only when a price changed, and closes the old one. It never edits or deletes an old version. Cost is computed at insert with the version valid at the event time (`ts_ms`) and stored as integer micro-USD, with `price_id` of that version. Upstream cost, when present, wins. Unknown model stores NULL cost ("unpriced"), not 0. Rollups store summed cost and are never rebuilt from current prices. | A token used last month shows the price of last month. A price change affects only later events. |
| 9 | Price source | Sync with `https://models.dev/api.json` (public JSON, about 6 MB, USD per 1M tokens: `input`, `output`, `cache_read`, `cache_write`, `reasoning`, `tiers`, `context_over_200k`; 8993 models in 235 providers on 2026-11-12, 394 without cost). The binary embeds a snapshot as a fallback so the first start works offline. Sync runs at start-up and every 24 h (config: `price-sync.enabled`, `url`, `interval-hours`), and on demand from the panel. The fetch uses the configured proxy-url and the shared HTTP client rules. A user edit (override) is never replaced by a sync. A failed sync keeps the old prices and records the error. Price rows keep their source (`models.dev`, `manual`) and `synced_at`. | Developer order: prices must sync with models.dev. Embedded fallback and overrides keep the server useful offline. |
| 9a | Recompute | There is no automatic recompute of events that have a cost. Events with NULL cost are filled automatically when a price for their model appears (see "Price timing rules"), and a manual action "price unpriced events" does the same on demand. Neither touches an event that already has a cost. A forced correction (wrong manual price) is a separate explicit action: it needs a date range and a confirmation, writes an audit row (who, range, old and new totals), and fixes the hourly rollups in the same transaction. | History must not change by accident. |
| 10 | API | Additive v8 routes under `/v8/management/observability/usage/...`. Released routes do not change. v0 gets nothing new. | Contract rule. |
| 11 | Config | New section `observability.usage-db`: `enabled` (default true), `path`, `retention-days`, `buffer-size`. The flag `usage-statistics-enabled` stays for the Redis queue only. | Keeps released behavior. |
| 12 | Scope phase 1 | Dashboard usage section, Usage analytics page, request history table, model prices page, maintenance (retention, vacuum, stats, export). | Core and useful rows of CPAMP. |
| 13 | Not in phase 1 | Auto-disable on quota, auth-issue queue, Codex inspection, CPAMP data import, manager update, Home mode capture. | The fork has its own quota and routing. Home mode disables the local management API. |

## Price timing rules

- The first price version of a model starts at time 0, so events that exist before the first sync still get a price.
- A synced version starts at the time the sync detected the change. models.dev does not give the real change time. Events between the real change and the next sync keep the old price (at most the sync interval, default 24 h). A manual price can set an explicit effective date.
- A manual price with a past effective date does not change events that already have a cost. Use "price unpriced events" or the audited correction for them.
- **Late pricing (any price is better than none).** If no price exists for a model when an event arrives, the event is stored unpriced (cost NULL). When a price version for that model appears later (sync, import, or manual entry), the store fills cost for all unpriced events of that model automatically, in the same step that adds the version. Each event takes the version valid at its own time. If the event is older than the first version, it takes the nearest version, which is the first one. Such events get `cost_source=price-book-late`, so the panel can mark the cost as an estimate. Only events with NULL cost are filled. An event that already has a cost never changes (except by the audited correction).
- If a model has no price version at all, its events stay unpriced and the panel shows the count and the model names.
- The writer reads the price for an event by `(model, ts_ms)` from an in-memory index that reloads when a version is added.

## Data model (target)

`usage_events`: `id`, `request_id` (unique), `trace_id`, `ts_ms`, `provider`, `model`, `alias`, `executor`, `auth_id`, `auth_index`, `auth_type`, `account_label` (masked), `api_key_id`, `access_provider`, `session_id`, `stream`, `status_class`, `failed`, `fail_code`, `fail_summary`, `latency_ms`, `ttft_ms`, `input_tokens`, `output_tokens`, `reasoning_tokens`, `cache_read_tokens`, `cache_write_tokens`, `total_tokens`, `service_tier`, `reasoning_effort`, `cost_micro_usd` (nullable), `cost_source` (`upstream`, `price-book`, `price-book-late`, `none`), `price_id` (nullable, the price version used).

`usage_hourly`: key on (`hour_ms`, `provider`, `model`, `alias`, `auth_id`, `api_key_id`, `status_class`, `stream`); counters: `requests`, `failures`, `input`, `output`, `reasoning`, `cache_read`, `cache_write`, `cost_micro_usd`, `unpriced_requests`, `latency_sum_ms`, `latency_max_ms`, `ttft_sum_ms`, `ttft_count`.

`model_prices` (versioned: `valid_from_ms`, `valid_to_ms`, `source`, `synced_at`), `model_price_tiers`, `cost_corrections` (audit), `api_key_labels`, `settings`, `schema_migrations`.

Indexes on `usage_events`: `ts_ms`; `(auth_id, ts_ms)`; `(model, ts_ms)`; `(api_key_id, ts_ms)`; `request_id` unique.

## Query API (target, v8, additive)

All are GET unless noted. Query parameters: `from`, `to` (RFC 3339 or epoch ms), `tz`, and filters `provider`, `model`, `auth_id`, `api_key_id`, `status` (ok, failed), `stream`.

| Route | Returns |
|---|---|
| `/observability/usage/status` | DB state: path, size, row counts, oldest and newest, buffer depth, drops, writer errors, retention |
| `/observability/usage/summary` | Totals for the range: requests, failures, tokens by kind, cost, unpriced count, latency avg and P95 |
| `/observability/usage/timeseries` | Buckets (hour, day) of the summary counters; param `bucket` |
| `/observability/usage/breakdown` | Grouped counters; param `group_by` (model, provider, credential, api_key, status, alias); sort and limit |
| `/observability/usage/events` | Page of raw events with filters, search, cursor paging |
| `/observability/usage/events/:id` | One event |
| `/observability/usage/failures` | Failure groups by code and credential, recent failures |
| `/observability/usage/heatmap` | Hour-of-week counts |
| `/observability/usage/export` | JSONL or CSV stream for a range |
| `/observability/usage/prices` (GET, PUT, POST import) | Price book |
| `/observability/usage/prices/sync` (POST) | Sync now from models.dev; returns counts (added, updated, unchanged, skipped overrides) and the error if any |
| `/observability/usage/prices/price-unpriced` (POST) | Fill cost for events with NULL cost, by price version at event time. Never changes an existing cost |
| `/observability/usage/prices/correct` (POST) | Explicit forced correction for a date range, with confirmation field and audit row |
| `/observability/usage/prices/history` (GET) | Price versions of one model over time |
| `/observability/usage/maintenance` (POST) | Actions: prune now, vacuum |
| `/observability/usage/key-labels` (GET, PUT) | Label per `api_key_id` |

Latency percentiles: P95 comes from raw events for ranges up to 7 days and from the hourly sums (average and max only) beyond that. The response marks which one it used.

## Panel (target)

BumpyClock design system only. No raw colors or sizes (token guard).

- Dashboard: usage section (requests, tokens, cost, failure rate, sparkline, top models).
- Usage page: summary cards, timeline, model and provider share, credential and key tables, failure sources, hour-of-week heatmap, filter bar, events table with detail drawer, export.
- Prices page: price table with price history per model, add and edit (new version), import, sync now, price unpriced events, audited correction.
- System page: DB status and maintenance.

## Work plan

| Issue | Area | Depends on |
|---|---|---|
| 0043 | Go: SQLite store, schema and migrations, writer, collector plugin, redaction, retention, config, status route | none |
| 0044 | Go: price book, models.dev sync (embedded fallback, overrides, schedule), versioned price book, cost at insert by event time, price-unpriced, audited correction, prices routes | 0043 |
| 0045 | Go: summary, timeseries, breakdown, events, failures, heatmap, export routes, route tests, contract note | 0043, 0044 |
| 0046 | Panel: services, store, dashboard section | 0045 contract |
| 0047 | Panel: Usage page | 0046 |
| 0048 | Panel: prices page and maintenance | 0044, 0046 |

0046 can start from the contract in this file while 0045 is in progress, if the response shapes are written down first (0045 step 1).

## Gate (every issue)

Go: `/usr/bin/go build ./... && /usr/bin/go vet ./... && /usr/bin/go test ./...`, `gofmt -l` empty. Panel: `bun install --frozen-lockfile`, `bun run verify`, `bun run tokens:check`. Before a release: build the panel, copy to `management.html`, rebuild the binary, and check by hash that the served file is the new build.
