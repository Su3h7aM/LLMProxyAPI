---
status: in-progress
branch: issue/0044-usage-prices-cost
type: backend (Go)
agent: muse-spark-1.3-contributor
base: main
---

# Usage prices: versioned price book, models.dev sync, cost at insert

## Goal

Add cost to the usage DB (issue 0043 is merged). Read `docs/usage-statistics-design.md` first: decisions 8, 9, 9a, the section "Price timing rules", and the data model. The key rule: **history is frozen**. Cost of an event uses the price version valid at the event time. A price change never rewrites old cost. There is no automatic recompute.

## References (read-only)

- `docs/usage-statistics-design.md`, `DECISIONS.md` (entry 2026-11-12), `issues/0043-usage-db-store-collector.md` (what exists), `internal/usagedb/` (store, writer, insert, redact, migrations), `sdk/cliproxy/usage/manager.go` (`Record.Detail`, `CostDetails`, token buckets), `internal/registry` and `internal/modelconfig` (model ids and providers of the fork).
- models.dev: `https://models.dev/api.json`. Shape (checked 2026-11-12): top object `provider id -> {id, name, models: {model id -> {id, name, limit{context,output}, cost{input, output, cache_read, cache_write, reasoning, input_audio, output_audio, tiers[], context_over_200k{...}}}}}`. Prices are USD per 1M tokens (floats). About 6 MB, 235 providers, 8993 models, 394 models without `cost`. `opencode-go`, `openai`, `anthropic`, `google`, `xai` are keys. Save one copy to `/tmp` for your tests and read its real `tiers` shape; do not commit the 6 MB file.
- CPAMP pricing code for ideas only (read by absolute path, never copy code): `/home/su3h7am/Apps/LLMProxyAPI/.refs/CPA-Manager-Plus/apps/manager-server` (see `issues/0041-cpamp-inventory.md` section 4).

## Steps (small commits, Conventional Commits, English)

1. **Migration 2.** Add to `internal/usagedb`: `model_prices` (`id`, `provider_key`, `model_id`, `valid_from_ms`, `valid_to_ms` NULL = current, `input`, `output`, `cache_read`, `cache_write`, `reasoning` as micro-USD per 1M tokens integers (convert floats with rounding; document the rule), `source` (`models.dev`, `manual`, `embedded`), `synced_at_ms`, `note`), `model_price_tiers` (context threshold and service tier rows, shape from the real models.dev `tiers` and `context_over_200k` data), `cost_corrections` (audit: id, ts, range, scope, old and new totals, reason), and `ALTER TABLE usage_events ADD COLUMN price_id INTEGER NULL`. Unique `(provider_key, model_id, valid_from_ms)`. Index `(provider_key, model_id, valid_from_ms)`. Version rows are append-only: code must never UPDATE a price row except to set `valid_to_ms` once, and never DELETE one.
2. **Price index.** An in-memory index: `Lookup(provider, model, tsMs) -> price version or none`, reloaded when a version is added. Matching rules, in order: (a) provider map from fork provider to models.dev provider id (put the map in one table in code with tests; start with claude to anthropic, codex and openai to openai, gemini and vertex to google, xai to xai, opencode-go to opencode-go; add others you find in `internal/registry`); (b) exact model id in that provider; (c) model id with a trailing date suffix (`-YYYYMMDD`, `-YYYY-MM-DD`) removed, both ways; (d) exact model id in any other provider, picking by a fixed provider preference list; else no match. Use the routed `Model` first, then `Alias`. A manual price for `(provider_key, model_id)` wins over models.dev for the time it is valid.
3. **Cost at insert.** In the writer transaction, before insert: if the record has an upstream-reported cost (`CostDetails`), store it with `cost_source=upstream`. Else look up the price at `ts_ms` and compute micro-USD from the token buckets: input (non-cached), output, cache read, cache write, reasoning (priced as output when no reasoning price). Apply context and service tier rules from the data. Round once per event (round half up). Set `price_id`, `cost_source=price-book`. No price: `cost_micro_usd` NULL, `cost_source=none`, and the hourly row counts `unpriced_requests`. Add the cost to the hourly UPSERT in the same transaction. Test: an event at a time before and after a price version change gets two different costs; a duplicate `request_id` does not double count.
3a. **Late pricing.** When a price version is added for `(provider_key, model_id)` by any path (sync, embedded load, manual entry), fill cost for the events of that model that have NULL cost, in batches (5000 rows per transaction, short pause), choosing the version valid at each event time; an event older than the first version takes the first version (nearest). Set `cost_source=price-book-late` and `price_id`. Update `usage_hourly`: add the cost and decrease `unpriced_requests` in the same transaction as each batch. Never change an event that already has a cost. The match uses the same lookup rules as step 2 (the event model and alias against the new version), and it must be idempotent. Run it in the background after the version is added, not inside the request that adds the version. Tests: events inserted with no price, then a price appears (cost filled, hourly sums right, unpriced count drops, estimate source set); an event that already has cost is untouched; an event older than the first version takes the first version; two models do not mix; re-run changes nothing.
4. **models.dev sync.** Package `internal/usagedb/prices` (or in `usagedb`): fetch `url` (config, default the models.dev URL) with the shared HTTP client rules of the repo (proxy-url, timeouts, size limit 32 MB, `If-None-Match` with ETag when present). Parse. For each model with cost: if no current version, add one (`valid_from_ms` = 0 for the first version of this `(provider, model)`, else sync time); if the current version has equal prices, do nothing; if it differs and the current version's source is `models.dev`, close it (`valid_to_ms` = sync time) and add the new one at sync time; if the current version is `manual`, skip and count `skipped_overrides`. Run in one transaction per provider batch. Schedule: at start-up (after 30 s), then every `interval-hours` (default 24). On failure keep old prices, store `last_error` and `last_attempt_ms` in `settings`. Config `observability.usage-db.price-sync`: `enabled` (default true), `url`, `interval-hours`. Update config types, v8 tree, example YAML, hot reload.
5. **Embedded snapshot.** A generator `scripts/gen-price-snapshot.sh` (or a Go `go:generate` tool) that fetches models.dev and writes `internal/usagedb/prices/snapshot.json.gz` with only the fields needed (cost, ids, names) and only the providers the fork uses plus a small set of common ones (list them in the tool). Target under 400 KB gzip. Commit the snapshot. At first start with an empty price table, load it as `source=embedded`, `valid_from_ms=0`. A later sync replaces `embedded` versions the same way as `models.dev` ones. The build and the tests never call the network.
6. **Routes (v8, additive, auth as others).** `GET /observability/usage/prices` (current versions; filters `provider`, `model`, `source`, `q`; paging), `GET /observability/usage/prices/history?provider=&model=` (all versions), `PUT /observability/usage/prices` (add or edit a manual price: closes the current version at `effective_from_ms`, adds a manual one; `effective_from_ms` default now), `DELETE /observability/usage/prices/manual?provider=&model=` (closes the manual version so the models.dev one applies again from now; never deletes rows), `POST /observability/usage/prices/sync` (run now; returns added, updated, unchanged, skipped_overrides, error, duration), `GET /observability/usage/prices/status` (last sync, last error, counts by source, unmatched models seen in usage: models with events but no price), `POST /observability/usage/prices/price-unpriced` (the same fill as step 3a, on demand: events with NULL cost, by the version valid at each event time or the nearest; never touches an event that has a cost; body: optional `from_ms`, `to_ms`, `model`; returns counts), `POST /observability/usage/prices/correct` (forced correction for a range and optional model: body needs `from_ms`, `to_ms`, `confirm: true`, `reason`; recomputes cost for the events in the range with the version valid at each event time, fixes `usage_hourly` in the same transaction, writes a `cost_corrections` row with old and new totals; reject a range over 366 days or without confirm). Response shapes: write them in `## Result` as JSON examples, because the panel issues use them.
7. **Home mode.** Decision: in Home mode the collector does not store (usage forwards to Home and the local management API is off). Skip registering the `usage-db` collector when Home is enabled; log one info line. Test it.
8. **Tests.** Price math (each bucket, tiers, rounding), time-valid lookup (before, at, after a change), models.dev parser against a small fixture file you write from the real shape, sync (first, unchanged, changed, manual override skipped, failure keeps old data, ETag 304), embedded load, append-only guard (a test that tries to change an old row through the public API and fails), price-unpriced (does not change existing cost), correction (audit row, rollup fix, confirm required, range cap), routes (auth, shapes), `-race`.
9. **Docs.** One short section in `docs/` for operators: where prices come from, how overrides work, that history is frozen, how to correct a wrong price.

## Gate

`/usr/bin/go build ./... && /usr/bin/go vet ./... && /usr/bin/go test -race ./internal/usagedb/... && /usr/bin/go test ./...`, `gofmt -l .` empty, `CGO_ENABLED=0 go build ./cmd/server` passes. Do not weaken or delete a test. Smoke test in `/tmp` (temp config, temp auth-dir): start the server with price sync pointed at a local test file or a local `httptest`-like server (no real models.dev call is needed), produce events with a stub upstream, read the price routes and check cost on events. One real fetch from models.dev by hand is fine to check the parser (once, output to `/tmp`). Record commands and output in the Log. Never leave a server process running.

## Environment

- Use `/usr/bin/go`. Do NOT run `mise trust`. Builds only in `/tmp`.
- English only. ASD-STE100, concise.

## Rules

- Take only what this issue lists: no query routes for summary, timeseries, breakdown, events (that is 0045), no panel code.
- v8 routes only and additive. Do not change a released contract.
- Price rows are append-only. Cost of an existing event changes only through the audited `correct` action.
- No secrets in logs or in outbound requests (no keys, no client data; the fetch sends no identifying header beyond a generic User-Agent).
- You do not decide. If something is unclear or out of scope: stop, write the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. Work on branch `issue/0044-usage-prices-cost` in your checkout. Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end: `status: done`, fill `## Result` (what landed, schema, config, route JSON examples, known gaps, list of models seen in the registry with no price match), commit, report. Do not merge to `main`; the coordinator merges.

## Log

- 2026-11-12: Read issue, design doc, `store.go`, `insert.go`, `writer.go`, `collector.go`, `redact.go`, `detail.go`. Rebased onto `main` (775b5908, late pricing rule). Created branch `issue/0044-usage-prices-cost`. Status in-progress.

## Follow-ups

## Result
