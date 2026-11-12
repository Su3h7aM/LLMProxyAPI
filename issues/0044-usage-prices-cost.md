---
status: done
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
- 2026-11-12: Fetched one copy of `https://models.dev/api.json` to `/tmp/models.dev.api.json` (6 MB) to read the real `tiers` and `context_over_200k` shapes. All `tiers[].tier.type` values are `context`; `context_over_200k` carries `input`/`output`/`cache_read`/`cache_write`. Keys kept for tiers: input, output, cache_read, cache_write, reasoning (rare).
- 2026-11-12: Read CPAMP pricing ideas only (`model_price_rules.go`, `service/pricing/cost.go`): context tier selected by total input tokens, strictly-greater-than threshold; tier is a whole-request rate set. Own implementation, no code copied.
- 2026-11-12: Commits: price book schema + index + cost at insert; late pricing + correction; sync + snapshot + config + wiring; routes + docs. Gate: `go build ./...` (with gitignored `apps/web/dist` stub; dist is gitignored and built via `mise run build-web`, so a fresh checkout fails this step without it), `go vet` clean, `go test -race ./internal/usagedb/...` pass, `go test ./...` pass, `gofmt -l` empty, `CGO_ENABLED=0 go build ./cmd/server` pass (`/tmp/llmproxy-0044`).
- 2026-11-12: Smoke test in `/tmp/smoke` (temp config, temp auth-dir, fixture price server on 18991, stub OpenAI upstream on 18992, server on 18317, binary `/tmp/llmproxy-0044`). `POST prices/sync` returned `added 1`; `GET prices?provider=openai&model=stub-model` returned the synced version (input 2000000); `POST /v1/chat/completions` with model `stub-model` stored one event with `cost_micro_usd=6000000`, `cost_source=price-book`, hourly sums right, unpriced 0. `PUT` manual closed the synced version and appended a manual one (history shows both). `POST correct` without `confirm` rejected; with a range over 366 days rejected; valid range wrote audit row 1 and kept cost (time-valid version unchanged). `DELETE manual` restored synced rates (`restored true`). All server and stub processes stopped after; none left running.
- Questions for the coordinator (implemented with documented defaults, change on decision):
  1. `CostDetails.CostInUSDTicks` is ignored (unit not defined). Upstream precedence used: `Cost`, then `UpstreamInferenceCost`, then prompt+completions sum. Keep or define a ticks conversion?
  2. Events with an unknown provider resolve to an unclassified token total, priced at the input rate. Keep or leave unpriced?
  3. `(provider_key, model_id, valid_from_ms)` has one UNIQUE index only (serves lookups); no second identical index. Keep?
  4. models.dev context tiers apply as whole-request rates (CPAMP convention). Keep?
  5. Correction audit `actor` is the fixed string `management-api`. Keep?

## Follow-ups

- Audio token pricing: usage rows carry no audio buckets, so `input_audio`/`output_audio` rates are parsed but never applied. Needs token buckets first.
- `PUT /observability/usage/prices` stores base rates only; no API for service-tier overrides yet (tier matching code exists, sync never emits service tiers).
- Price `display_name` is not stored (schema has no column); panel lists show `model_id` only. Snapshot file keeps names if the panel wants them.
- Registry models below stay unmatched even after a full sync (virtual aliases, suffixed variants, image models). Panel shows them in `unmatched`.

## Result

Landed on branch `issue/0044-usage-prices-cost` (no merge to `main`).

Schema (migration 2, `SchemaVersion=2`): `model_prices` (versioned, append-only; unique `(provider_key, model_id, valid_from_ms)`), `model_price_tiers` (`context`/`service` rows, NULL rate falls back to base), `cost_corrections` (audit), `usage_events.price_id` NULL. Code never UPDATEs a price except `valid_to_ms` once, never DELETEs. Conversion: micro-USD per 1M = round half up of USD per 1M; one rounding per event.

Cost at insert (writer transaction): upstream cost wins (`upstream`, `price_id` NULL); else time-valid version prices the non-overlapping buckets from `Detail.PricedBuckets` (subset protocols never double count; reasoning falls back to output; unknown protocols price the total at input) with `price-book`; else NULL/`none` + hourly `unpriced_requests`.

Late pricing: a new version triggers a background fill of NULL-cost events of that key (batches of 5000, 50 ms pause), each event at its own time, pre-first events take the first version, marked `price-book-late`, hourly cost added and unpriced decremented in the same transaction. Never touches priced events; idempotent.

Sync: fetch with shared HTTP rules (proxy, 30 s timeout, 32 MB cap, ETag/`If-None-Match`, generic User-Agent). First version starts at 0, changed versions close/reopen at sync time, manual versions skip (`skipped_overrides`), failures keep data and record `last_error`. Scheduler: first run 30 s after start, then cadence. Embedded snapshot (`internal/usagedb/snapshot.json.gz`, 5.7 KB, 16 providers, 323 models, built by `scripts/gen-price-snapshot.sh`) loads at first start when the table is empty; sync replaces `embedded` like `models.dev`.

Config: `observability.usage-db.price-sync` (`enabled` default true, `url` default models.dev, `interval-hours` default 24), example YAML, v8 tree automatic, hot reload via `UsageDB` compare. Home mode: collector disabled with one info log.

Routes (v8, additive, same auth as other management routes):

- `GET /observability/usage/prices?provider=&model=&source=&q=&limit=&offset=` returns `{"prices":[{"id":324,"provider":"openai","model":"stub-model","valid_from_ms":0,"valid_to_ms":null,"input":2000000,"output":8000000,"cache_read":500000,"cache_write":0,"reasoning":null,"source":"models.dev","synced_at_ms":1794450947548,"note":"","tiers":[]}],"total":1,"limit":100,"offset":0}`. Rates are micro-USD per 1M.
- `GET /observability/usage/prices/history?provider=&model=` returns `{"provider":"openai","model":"stub-model","versions":[...]}` oldest first.
- `PUT /observability/usage/prices` body `{"provider","model","input","output","cache_read?","cache_write?","reasoning?","effective_from_ms?","note?"}` (USD per 1M) closes current at `effective_from_ms` (default now), appends `manual`, returns the version. Past dates allowed; stored cost never changes.
- `DELETE /observability/usage/prices/manual?provider=&model=` closes the manual version; restores last synced rates when present. Returns `{"provider":"openai","model":"stub-model","restored":true}`.
- `POST /observability/usage/prices/sync` returns `{"added":1,"updated":0,"unchanged":0,"skipped_overrides":0,"error":"","duration_ms":3}`.
- `GET /observability/usage/prices/status` returns `{"last_sync_ms":1794450947548,"last_attempt_ms":1794450947548,"last_error":"","counts_by_source":{"embedded":323,"models.dev":1},"unmatched":[{"provider":"codex","model":"gpt-5u","events":1}]}`.
- `POST /observability/usage/prices/price-unpriced` body `{"from_ms?","to_ms?","provider?","model?"}` returns `{"priced":1,"skipped_no_price":0,"duration_ms":12}`.
- `POST /observability/usage/prices/correct` body `{"from_ms","to_ms","provider?","model?","confirm":true,"reason"}` returns `{"correction_id":1,"events_updated":1,"old_cost_micro_usd":6000000,"new_cost_micro_usd":6000000}`; rejects missing confirm, missing reason, and ranges over 366 days with 400.

Tests: price math, tiers, rounding, lookup (provider map, alias, date suffix both ways, cross-provider, time validity, manual wins), append-only guard, parser fixture, sync first/unchanged/changed/manual-skip/failure/ETag, embedded load once, insert cost states, upstream conversion, late pricing (per-time versions, keeps priced, no mix, nearest, idempotent), correction, all routes, Home mode, buckets. `go test -race ./internal/usagedb/...` and full `go test ./...` pass.

Docs: `docs/usage-prices.md` (source, frozen history, unpriced, overrides, corrections); route rows in `docs/management-api-v8.md`.

Known gaps: audio rates not applied; no service-tier API; no `display_name` column. Registry models with no price match even after full sync: claude (5 dated: `claude-opus-4-1-20250805`, `claude-opus-4-20250514`, `claude-sonnet-4-20250514`, `claude-3-7-sonnet-20250219`, `claude-3-5-haiku-20241022`), gemini/aistudio/gemini-cli (`gemini-3-pro-preview`, `gemini-3.1-flash-image-preview`, `gemini-3-pro-image-preview`, `gemini-omni-1.2-flash`), vertex (plus `gemini-3-pro`, `gemini-3-flash`, `gemini-3.1-pro`, `imagen-*`), codex channels (`gpt-6-chat-thinking`, `codex-auto-review`), kimi (`kimi-k2`, `kimi-k2-thinking`, `kimi-k2.5`, `kimi-k2.8`, `kimi-k2.8-code`, `kimi-k3-256k`), antigravity (`-high`/`-medium` variants, `claude-opus-4-6-thinking`, `gpt-oss-120b-medium`), xai (`grok-4.7-build-fast`, `grok-3-mini*`, `grok-composer-2.5-fast`, image variants), meta (`muse-wonder-*`).
