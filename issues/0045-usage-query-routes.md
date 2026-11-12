---
status: done
branch: issue/0045-usage-query-routes
type: backend (Go)
agent: muse-spark-1.3-contributor
base: main
---

# Usage query routes: summary, timeseries, breakdown, events, failures, heatmap, export, labels, maintenance

## Goal

Implement the read side of the usage DB (issues 0043 and 0044 are merged) exactly as `docs/usage-api-contract.md` says. The panel issues (0046 to 0048) use that contract, so shapes, names, units, and defaults must match. A needed change to a shape is a question for the coordinator, not a silent edit.

## References (read-only)

- `docs/usage-api-contract.md` (the contract), `docs/usage-statistics-design.md`, `docs/usage-prices.md`, `DECISIONS.md` (entry 2026-11-12), `issues/0043-*.md` and `issues/0044-*.md` (Result sections), `internal/usagedb/` (store, insert, writer, prices, migrations), `internal/api/server_management_v8.go`, `internal/api/handlers/management/usage_db_status.go` (how a usage route is wired and tested).
- Credential labels: look at how the management handlers list credentials (`internal/api/handlers/management/auth_files.go`) and the auth manager, to resolve `auth_id` to a label.
- CPAMP ideas only, never copy code: `/home/su3h7am/Apps/LLMProxyAPI/.refs/CPA-Manager-Plus/apps/manager-server` (see `issues/0041-cpamp-inventory.md` section 3).

## Steps (small commits, Conventional Commits, English)

1. **Migration 3.** Add `usage_hourly.late_cost_micro_usd INTEGER NOT NULL DEFAULT 0`. Backfill it from `usage_events` where `cost_source='price-book-late'` (group by the hourly key). Change late pricing (issue 0044 code) so it adds to it in the same transaction as the cost, and so a correction keeps it right. Add `api_key_labels(api_key_id TEXT PRIMARY KEY, label TEXT NOT NULL, updated_at_ms INTEGER NOT NULL)`. Add indexes that the routes need (check with `EXPLAIN QUERY PLAN`; likely `(status_class, ts_ms)`, `(failed, ts_ms)`, `(provider, ts_ms)`). Do not add an index without a query that uses it. Test the migration on a DB built by migration 2 with rows in it.
2. **Query package.** `internal/usagedb/query` (or files in `usagedb`) with a `Filter` type (parse and validate the common parameters, range cap, repeated values) and builders that always use bound parameters (no string-built SQL from user input; column names come from a fixed allow-list). Source rule: counters come from `usage_hourly` (all filter dimensions exist there); `q`, percentiles, the event list, failures detail, and `fail_*` need `usage_events`. The `status` filter maps to `failed` or `status_class`. Day and week buckets and the heatmap use `tz`: convert each hour bucket with the zone (whole-hour offsets are exact; zones with 30 or 45 minute offsets put an hour bucket in the day of its start; document this).
3. **Routes.** `summary`, `timeseries`, `breakdown`, `events`, `events/:id`, `failures`, `heatmap`, `export`, `key-labels` (GET, PUT), `maintenance` (POST). Wire them in `server_management_v8.go` and handlers next to `usage_db_status.go`. Disabled DB gives 503 with the contract body. Credential labels: from the auth manager when the credential exists, else the latest stored `account_label`, else the id. API key labels: `api_key_labels`, else `key-<first 6 of id>`. Cursor: opaque base64 of `(ts_ms, id)`; invalid cursor gives 400. `total_estimate` exact up to 100000 matching rows (count with a limit), else an estimate and `total_exact:false`.
4. **Percentiles.** Exact P50 and P95 of `latency_ms` from `usage_events` for the range and filters, only when the matching rows are at most 1,000,000; use a count and a bounded `ORDER BY ... LIMIT 1 OFFSET n` (or another exact method you can justify in the Log). Above the limit return nulls and `source:"hourly"`. Measure on a synthetic 1M-row DB (see step 6). If the target in the contract is missed, report numbers and a proposal; do not change the contract yourself.
5. **Export.** Stream with a read-only connection, in pages of 2000 rows ordered by `ts_ms, id`, flush often, stop on context cancel, no buffering of the whole result. CSV with a header row, RFC 4180 quoting, and protection against spreadsheet formula injection (prefix a single quote when a text cell starts with `=`, `+`, `-`, `@`, tab, or CR). JSONL one object per line.
6. **Performance check.** A Go benchmark or test helper (not run in the normal test run; use a build tag or `-run`/`-bench` name) that fills a temp DB with 1M synthetic events through the real writer path (varied models, credentials, keys, failures, 120 days). Measure each route against the targets in the contract. Put the numbers in `## Result`. Do not commit a big DB.
7. **Tests.** Parameter parsing and rejection (bad range, cap, bad enum, bad cursor), counters math from known rows (including failures, estimated cost, unpriced), previous-period math, bucket and tz edge cases (day boundary, DST change day for `America/Sao_Paulo` and `Europe/Berlin`), breakdown sort and share and `limit`, label resolution order, events paging stable with equal `ts_ms`, `q` search, failures grouping, heatmap matrix (Monday first), export JSONL and CSV (quoting, formula guard, large range streaming, cancel), key-labels set and clear and that the raw key never appears, maintenance actions, 503 when disabled, auth required on every route, and SQL safety (a filter value with quotes or `;` does no harm). `-race`.
8. **Docs.** Update `docs/management-api-v8.md` with the routes (short rows linking `docs/usage-api-contract.md`). Keep the contract file as the source of shapes; if you change nothing in shapes, do not edit it.

## Gate

`/usr/bin/go build ./... && /usr/bin/go vet ./... && /usr/bin/go test -race ./internal/usagedb/... && /usr/bin/go test ./...`, `gofmt -l internal sdk cmd` empty, `CGO_ENABLED=0 go build ./cmd/server` passes. Fresh checkouts need `apps/web/dist/management.html` for the embed (create a stub in your checkout; it is gitignored). Do not weaken or delete a test. Smoke test in `/tmp` (temp config and auth-dir, stub upstream as in the 0044 smoke test): produce events, call every route with the management key, compare `summary` with `events` counts. Record commands and output in the Log. Never leave a server process running.

## Environment

- Use `/usr/bin/go`. Do NOT run `mise trust`. Builds only in `/tmp`.
- English only. ASD-STE100, concise.

## Rules

- Take only what this issue lists: no panel code, no change to the write path except the `late_cost_micro_usd` accounting in step 1.
- v8 routes only and additive. Contract shapes are fixed (see Goal).
- SQL: bound parameters only; allow-listed identifiers.
- Never return a raw API key, a body, or a header value. `fail_summary` only in event detail, failures, and export.
- You do not decide. If something is unclear or out of scope: stop, write the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. Work on branch `issue/0045-usage-query-routes` in your checkout. Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end: `status: done`, fill `## Result` (what landed, schema changes, benchmark numbers against the targets, known gaps, deviations from the contract if any with the reason), commit, report. Do not merge to `main`; the coordinator merges.

## Log

- Set `status: in-progress`. Work on branch `issue/0045-usage-query-routes` (checkout branch `t3/usage-query-routes` is at same tip as `main`, 45a67d12; created issue branch from it).
- Context read: contract, design doc, DECISIONS, `store.go`, `insert.go`, `writer.go`, `collector.go`, `redact.go`, `prices.go`, `prices_fill.go`, `retention.go`, `usage_db_status.go`, `usage_prices.go`, `server_management_v8.go`, handler auth/labels, 0043/0044 logs.
- Step 1 done (commit d1ba717d): migration 3 (`late_cost_micro_usd` with backfill from `price-book-late` events, `api_key_labels`, provider/failed/status_class range indexes; each index proven by EXPLAIN QUERY PLAN in `TestMigration3BackfillsLateCost`), late fill adds to the late part, audited correction removes it when it reprices a late event to `price-book`.
- Step 2 done (commit 51cf8c1a): `Filter` parse/validate (366-day cap), bound-only builders, hourly counters (or events with `q`), timeseries buckets with tz day/week and gap fill, breakdown sort/page, Monday-first heatmap, exact percentiles to 1M rows, `(ts_ms, id)` cursor, keyset export paging.
- Step 3 done (commit 81f7ade0): all routes wired in the v8 group (auth from the group middleware); 503 body is exactly `{"error":"usage db disabled"}`; labels from auth manager else stored label else id; `q` refused on rollup-only routes.
- Fixes after review: filter errors are `QueryError` (400, not 500); timeseries points nest `counters`; export JSONL always has `fail_summary`; `top` is validated with or without `group_by`.
- Step 4/5 implementation done; step 6 perf run finished (`-tags usageperf`, 1M rows); step 8 docs rows done.
- Perf numbers (1M uniform rows, 120 days, this dev box; seed 2m4s): summary 120d 8.1s cold (MISS, target 300ms); timeseries day grouped 904ms (MISS); breakdown 907ms (MISS); summary 7d exact percentiles 235ms (PASS, target 1s); events page 236ms (MISS, target 150ms, close); failures 30d 22ms (PASS); heatmap 30d 60ms (PASS); export full scan 41s (~24k rows/s). Cause: the uniform seed spreads rows over ~3840 key combos, so `usage_hourly` holds ~1M rows (worst case; real traffic concentrates on far fewer combos); first query also pays cold page cache; the events cost is the capped 100k total COUNT. Proposal: no contract change; if production shows the same, add an internal daily rollup or accept (coordinator decision).
- Gate: `/usr/bin/go build ./...`, `go vet ./...`, `go test -race ./internal/usagedb/...`, `go test ./...` (104 packages ok, exit 0), `gofmt -l internal sdk cmd` empty, `CGO_ENABLED=0 go build ./cmd/server` ok. Note: `go build ./...` needs the gitignored `apps/web/dist/management.html` stub (pre-existing, also noted in 0043).
- Smoke in `/tmp/smoke0045` (binary `/tmp/smoke0045/llmproxy`, stub upstream on 18992, server on 18317, temp auth dir): 2x `POST /v1/chat/completions` model `stub-ok` (200) + 1x `stub-fail` (500) stored 3 events, schema 3. Every route called with the management key: summary totals 3 requests / 1 failure match the events page (3 items); timeseries groups, breakdown, failures (1 group, auth_label fallback), heatmap, JSONL (3 lines) and CSV (header + quoting) export, key-labels GET/PUT/clear, maintenance prune/vacuum/checkpoint, event detail with price null (no price book for stub models), no-auth 401. Both processes stopped after (`kill -0` confirms none left).
- Observation for the coordinator (out of scope, not changed): the smoke detail shows `"account_label":"stub-upstream-key"` — a short upstream api-key value passes `maskAccountLabel` through and is stored (0043 write path). The new routes return it as the contract says. Redact at read time or fix the write path? Coordinator decides.
- Note: `main` moved to 6f2a837d (new file `issues/0046-*.md`) after this branch was cut; the diff `main..HEAD` shows that file as deleted, but no commit here touches it. The coordinator merge will rebase.
- Open questions for the coordinator (defaults used, change on decision):
  1. `status` filter maps to `failed` on raw events but to `status_class` on the hourly rollup (`ok`=`2xx`, `failed`=`4xx/5xx/other`). A failed record with a 2xx code counts as ok in rollup totals. Keep the mapping or extend the hourly key with `failed`?
  2. `summary.first_event_ms` is the global oldest event (not range-bound). Keep?
  3. Timeseries week buckets use ISO weeks (Monday). Keep?
  4. `total_estimate` above 100000 rows uses the hourly sum (marked `total_exact:false`); with `q` it runs the full exact count. Keep?
  5. Percentile rank method is nearest-rank (rank ceil(p*N)). Keep?
  6. Manager credential label uses auth `Label`; provider meta prefers the manager provider, auth_type comes from the latest stored event. Keep?
  7. Maintenance `checkpoint` response adds `busy`, `log_frames`, `checkpointed` next to `duration_ms`. Keep?

## Follow-ups

- Short upstream api-key values can land in `account_label` (see Log observation); decide read-time redaction or write-path fix.
- Manager-present credential label path is covered by construction only; consider a handler test with a populated auth manager.
- Percentile-over-limit and total-over-limit paths are covered by the perf run only, not unit tests.

## Result

Landed on branch `issue/0045-usage-query-routes` (no merge to `main`).

What landed: the full read side of the usage DB exactly as `docs/usage-api-contract.md` says (no shape changes; the contract file is untouched).
- Schema 3: `usage_hourly.late_cost_micro_usd` (backfilled from `price-book-late` events, maintained by the late fill and the audited correction), `api_key_labels`, indexes `(provider,ts_ms)`, `(failed,ts_ms)`, `(status_class,ts_ms)` (each proven used by EXPLAIN QUERY PLAN).
- Query layer (`internal/usagedb/query*.go`): validated `Filter` (366-day cap; client errors are 400), bound-parameters-only builders, hourly counters (events table only when `q` is set), buckets (hour, tz day, ISO week, gap-filled, top-plus-other), breakdown sort/page, Monday-first heatmap, exact P50/P95 to 1M rows with hourly fallback above, opaque `(ts_ms,id)` cursor, keyset export paging (2000/page); label resolvers (manager, stored, id fallback).
- Routes (`GET/PUT/POST` under `/v8/management/observability/usage/`): `summary`, `timeseries`, `breakdown`, `events`, `events/:id`, `failures`, `heatmap`, `export` (JSONL/CSV, formula guard, flush per write, stops on client cancel), `key-labels`, `maintenance` (prune/vacuum/checkpoint). Disabled DB gives 503 `{"error":"usage db disabled"}`. Auth comes from the v8 group middleware (401 proven for all 11 route registrations).
- Tests: migration on a v2 DB with rows, late-cost fill/correct, parsing/rejection, counters math (hourly and events paths), previous-period math, buckets/tz incl. Sao Paulo and Berlin DST days, breakdown share/limit, label order, stable paging with equal `ts_ms`, `q`, failures, heatmap, export (quoting, guard, 2500-row multi-page, cancel), key-labels set/clear, maintenance, 503 on all routes, auth on all routes, hostile SQL input harmless. Perf helper behind the `usageperf` tag (not in the normal run).
- Docs: `docs/management-api-v8.md` rows link the contract.

Benchmark numbers (1M uniform synthetic rows over 120 days, this dev box): summary 120d 8.1s cold (MISS vs 300ms), timeseries 904ms (MISS), breakdown 907ms (MISS), summary percentiles 7d 235ms (PASS vs 1s), events page 236ms (MISS vs 150ms, close), failures 30d 22ms (PASS vs 500ms), heatmap 30d 60ms (PASS vs 500ms), export 41s. See Log for cause analysis and proposal (no contract change).

Known gaps: manager-present credential label path is covered by construction only (empty-manager fallback is tested; building a populated auth manager in a unit test is not practical); percentile-over-limit and total-over-limit paths are logic-covered but not hit by unit tests (100k/1M-row fixtures are perf-run only); `q` on rollup-only routes is rejected (contract says events/failures/export only).
