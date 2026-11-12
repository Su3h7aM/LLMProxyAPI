---
status: todo
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

## Follow-ups

## Result
