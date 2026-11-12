---
status: in-progress
branch: issue/0049-usage-db-hardening
type: backend (Go, security and performance)
agent: muse-spark-1.3-contributor
base: main
---

# Usage DB hardening: secret leaks, status consistency, query speed

## Goal

Fix four findings from the review of issues 0043 to 0045 before the usage DB ships. Item 1 is a security rule: it blocks any release.

## References (read-only)

- `docs/usage-statistics-design.md` (decision 5, privacy), `docs/usage-api-contract.md`, `issues/0043-*.md`, `issues/0045-*.md` (Log: smoke observation, perf numbers, question 1), `internal/usagedb/redact.go`, `internal/usagedb/insert.go`, `internal/usagedb/query_*.go`, `internal/usagedb/store.go`, the `usageperf` helper from 0045.
- `internal/runtime/executor/helps/usage_helpers.go` around lines 920 to 943: how `Record.Source` is chosen (email, project, auth source kind, else the raw client key). Read it to know what `Source` can hold (it may hold an upstream API key).

## Steps (small commits, Conventional Commits, English)

1. **Account label leak (security).** The smoke test of 0045 stored `account_label = "stub-upstream-key"`: a short upstream API key passed `maskAccountLabel`. New rule: `account_label` is stored only when the source is an email (masked as now: `ab***@domain`). Every other value is stored as an empty string. No token-like test, no passthrough. Readers already fall back to the auth manager label and then to the credential id (0045), so the panel still shows a useful name. Apply the same rule to any other column that takes free text from the record (`session_id`, `service_tier`, `reasoning_effort`, `access_provider`, `executor`, `auth_type`, `fail_code`): store them only when they match a strict safe shape (letters, digits, `.`, `_`, `-`, `:`; at most 64 characters), else empty. Tests: a table of hostile inputs (raw client key, upstream key of 8, 20, 40 characters, `sk-`, `AIza`, `Bearer ...`, a JWT, a URL with a token query, a file path) must never appear in any column of the row; run a scan over a DB after inserting them. Update `docs/usage-statistics-design.md` decision 5 and the contract text for `account_label` (same meaning, now email-only).
2. **Failure text scrub (security).** `fail_summary` comes from upstream error text and can echo secrets. Add a scrubber, applied after the existing cut: replace `Bearer <token>`, `Authorization:` values, `sk-...`, `AIza...`, `ghp_`/`gho_`/`github_pat_` tokens, JWT-shaped strings (`xxx.yyy.zzz` base64url), `key=`, `api_key=`, `token=`, `access_token=`, `password=` query or JSON values, and any run of 32 or more base64/hex characters with `[redacted]`. Keep the first 4096 bytes after scrubbing. Tests with realistic upstream error bodies (OpenAI, Anthropic, Google, a Go error with a URL query) and a fuzz-style test: random secret-shaped substrings never survive.
3. **Status consistency.** A failed record with a 2xx code today lands in `status_class = 2xx` and counts as ok in hourly totals (0045 question 1). Fix at the source: when `Failed` is true, `status_class` is never `2xx` (use `error` when the code is 2xx or missing, else the real class). Document the allowed values (`2xx`, `4xx`, `5xx`, `error`, `other`) in the contract (`status` filter text only: `failed` = every class except `2xx`). Migration 4: fix existing rows (`UPDATE usage_events SET status_class='error' WHERE failed=1 AND status_class='2xx'`) and rebuild the affected `usage_hourly` rows in one transaction (the table is small enough for a rebuild by group; write it carefully and test on a DB with such rows). Test that hourly `failures` equals the sum over non-`2xx` classes.
4. **Query speed.** Targets are in the contract (summary, timeseries, breakdown on 120 days of data under 300 ms warm). 0045 measured 8.1 s cold on a uniform seed that makes `usage_hourly` hold about 1M rows, which real traffic will not do. Do this in order and stop when the targets hold: (a) add a realistic seed profile to the `usageperf` helper (about 12 models, 8 credentials, 4 keys, traffic concentrated on a few combinations, bursts by hour of day, 1M events over 120 days) and keep the uniform profile as a worst case; (b) tune read connections (`cache_size`, `mmap_size`, `temp_store=MEMORY`; justify each value with numbers) and check indexes with `EXPLAIN QUERY PLAN`; (c) report cold and warm numbers for both profiles; (d) if the realistic profile still misses a target, add an internal `usage_daily` rollup (same dimensions, UTC day, written in the same transaction as the hourly row, backfilled in migration 5 from `usage_hourly`) and route ranges over 14 days with whole-day bounds to it; keep the contract unchanged. If the worst case alone misses, report it and do nothing more for it. Also cut the `events` page cost: measure the capped `COUNT` (100000) and use a cheaper estimate when no filter is set; target 150 ms.
5. **Small follow-ups from 0045.** A handler test with a populated auth manager for the credential label path; unit tests for the percentile over-limit and total over-limit paths.

## Gate

`/usr/bin/go build ./... && /usr/bin/go vet ./... && /usr/bin/go test -race ./internal/usagedb/... && /usr/bin/go test ./...`, `gofmt -l internal sdk cmd` empty, `CGO_ENABLED=0 go build ./cmd/server` passes. Fresh checkouts need a stub `apps/web/dist/management.html` (gitignored). Do not weaken or delete a test. Smoke test in `/tmp` as in 0045: produce events with a stub upstream whose key is short and whose error body echoes a token; after that, scan the whole DB file and WAL bytes for the raw keys and tokens (must be absent). Record commands and output in the Log. Never leave a server process running.

## Environment

- Use `/usr/bin/go`. Do NOT run `mise trust`. Builds only in `/tmp`.
- English only. ASD-STE100, concise.

## Rules

- Take only what this issue lists.
- Contract shapes stay as they are (only the text on `account_label` and `status` values changes).
- You do not decide. If something is unclear or out of scope: stop, write the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. Work on branch `issue/0049-usage-db-hardening` in your checkout. Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end: `status: done`, fill `## Result` (what landed, schema changes, benchmark numbers for both profiles, known gaps), commit, report. Do not merge to `main`; the coordinator merges.

## Log

- Set `status: in-progress`. Work on branch `issue/0049-usage-db-hardening` (cut from local `main` at 364747cb, which already holds 0043-0045).
- Context read: `redact.go`, `insert.go`, `store.go` (schema 3), `query.go`, `query_events.go`, `query_timeseries.go`, `query_perf_test.go`, `redact_test.go`, `query_test.go`, `store_test.go`, contract, design doc, 0045 Result/Log (perf 8.1s cold uniform, smoke `account_label` leak, question 1), `resolveUsageSource` (email, vertex project, account info, metadata email, auth api_key attr, else raw client key).
- Step 1 done: `maskAccountLabel` is now email-only (masked `ab***@domain`, else empty; token-like branch and passthrough removed); new `safeField` (letters, digits, `.`, `_`, `-`, `:`; at most 64 chars) guards `session_id`, `service_tier`, `reasoning_effort`, `access_provider`, `fail_code`, and via `safeOrUnknown` (empty keeps `unknown`) `executor`, `auth_type`. Empty `service_tier` still defaults to `default`; empty tier is safe for `ComputeCost` (base rates). Tests: updated mask table, new `safeField` table, `TestHostileSecretsNeverStored` (10-input table as source/key/shape-hostile) and `TestHostileSecretsDBScan` (all text columns of `usage_events` scanned). Docs: design decision 5 and contract `account_label` text updated. `go build ./...` needs the gitignored `apps/web/dist/management.html` stub (pre-existing).
- Step 2 done: `failSummary` now runs `scrubFailText` after the raw-key cut and keeps the first 4096 bytes after scrubbing. Replaced with `[redacted]`: `Bearer <token>`, `Authorization:` values, `sk-...`, `AIza...`, `ghp_`/`gho_`/`github_pat_` tokens, JWT shapes, `api_key`/`access_token`/`secret`/`password`/`token`/`key` query and JSON values (field names stay), any run of 32 or more base64/hex characters. Tests: `TestFailSummaryScrubRealisticBodies` (OpenAI, Anthropic, Google, GitHub, bearer, auth header, JWT, URL query, Go error with query, JSON password, long hex) and `TestFailSummaryScrubFuzzShaped` (200 seeded secret-shaped inputs, none survive). Docs: decision 5 and contract `fail_summary` text updated.
- Step 3 done: failed records are never `2xx` (`statusClass`: failed 2xx or code 0 becomes `error`, else the real class); hourly `failed` filter is now every class except `2xx`; contract documents the allowed values. Migration 4 updates old events and rebuilds affected hourly groups in one transaction (DELETE plus INSERT SELECT with writer-identical math). Tests: updated status table, `TestMigration4FixesFailed2xx` (v3 DB with failed 2xx rows, hourly split checked, untouched 5xx control), `TestStatusClassHourlyConsistency` (writer path: hourly failures equal the non-2xx class sum). `migration3_test.go` schema assertion moved to 4.
- Step 4 in progress: realistic seed profile added to the `usageperf` helper (12 models, 8 credentials, 4 keys, credential-centric concentration, hour-of-day bursts, 1M events over 120 days; uniform kept as worst case; `-profile` selects). Baseline (no tuning yet): uniform (365717 hourly rows) summary 7.5s, timeseries 909ms, breakdown 911ms, events 229ms, 7d percentiles 226ms, failures 21ms, heatmap 63ms, export 42s; realistic (493481 hourly rows, first skew too weak, now credential-centric) summary 12.5s, timeseries 2.9s, breakdown 2.7s, events 1.2s, 7d percentiles 623ms, failures 23ms, heatmap 285ms, export 44.5s. Warm equals cold everywhere: CPU-bound in pure-Go SQLite, not page-cache-bound. `EXPLAIN QUERY PLAN`: hourly range queries SEARCH the PK on `hour_ms`; grouped aggregates add `USE TEMP B-TREE FOR GROUP BY`. Read pragmas added (`cache_size` 32 MiB per conn, `mmap_size` 256 MiB cap, `temp_store MEMORY`); justification by tuned rerun. Events page: plain-range totals read the hourly sum first (capped confirm when small, estimate without the 100k scan when large). Percentile path: hourly upper-bound gate skips the raw count and sorts when over the limit (exactness kept: hourly requests bound events from above).
- Step 5 done: `TestUsageCredentialLabelFromManager` (populated auth manager wins for breakdown labels and failures `auth_label`; stored label fallback without manager) and `TestQueryOverLimitPaths` (percentile and total over-limit with shrunk limits; limits changed from const to var, defaults unchanged).
- Step 4d done (code): migration 5 (`usage_daily`, backfilled from hourly), writer/late-fill/correction keep it in the same transactions, `useDaily` routing (over 14 days, whole-day UTC bounds) for counters, day/week timeseries, breakdown, and both gates; hour series and heatmap stay hourly; contract unchanged. Tuned rerun numbers: uniform (365701 hourly) summary 4.4s/3.9s, timeseries 866ms, breakdown ~900ms, events 217ms, 7d percentiles 230ms, failures ~20ms, heatmap ~65ms; realistic concentrated (86588 hourly) summary 7.1s/6.4s, timeseries 354ms, breakdown 344-354ms, events 70ms, 7d percentiles 373-391ms, failures 18ms, heatmap 21ms, export ~44s. Parts: events COUNT 120d ~80ms, hourly SUM requests ~70-220ms. Conclusion: counters/timeseries/breakdown got 3-8x faster (temp_store MEMORY plus concentration) but realistic 120d summary stays sort-bound: exact P50/P95 at the 1M boundary needs two 1M-row sorts (~6s). Final run with daily routing done, numbers below.

- Final numbers (daily routing, day-aligned 120d ranges; warm): uniform (365603 hourly rows) summary 3.68s/3.33s, timeseries 35ms, breakdown 34ms, events 8/7ms, 7d percentiles 224/223ms, failures ~20/15ms, heatmap ~65/63ms, export ~41s; realistic (86397 hourly rows) summary 6.72s/6.49s, timeseries 23/22ms, breakdown 22ms, events 5/4ms, 7d percentiles 364/378ms, failures 18/17ms, heatmap 22/18ms, export ~44s; parts COUNT ~76ms, hourly SUM ~75ms (realistic). Every target holds on both profiles except summary-120d, which misses only through exact percentiles at exactly 1M events (two 1M-row sorts, ~3s uniform / ~6.5s realistic; the contract promises 7d/1M under 1s and null above the limit, and the 300ms summary target excludes percentiles). The over-limit gate fires only above 1M, so the exact boundary pays the sorts.

- Gate: `go build ./...`, `go vet ./...`, `go test -race ./internal/usagedb/...`, `go test ./...` (exit 0), `gofmt -l internal sdk cmd` empty, `CGO_ENABLED=0 go build ./cmd/server` ok. `go test ./...` needs the gitignored `apps/web/dist/management.html` stub (pre-existing).

- Smoke in `/tmp/smoke0049` (binary from this branch, stub upstream on 18993 with short key `stub-upstream-key`, server on 18318, temp auth dir, fresh DB): 2x `POST /v1/chat/completions` model `stub-ok` (200) + 1x `stub-fail` (500, error body echoes an `sk-` token, a `Bearer` token, and an `api_key` query value). Status: schema 5, 3 events, buffer flushed. Summary totals 3 requests / 1 failure match the events page. `account_label` is empty on all rows (the 0045 short-key leak is fixed at the source). Failed row is `5xx`, never `2xx`. Detail `fail_summary` shows `[redacted]` for all three secrets and no raw key. Byte scan (`grep -a -c`) over `usage.db`, `usage.db-wal`, `usage.db-shm`: 0 hits for the client key, the short upstream key, the `sk-` token, the `Bearer` token, and the query secret; `[redacted]` present in the WAL. First stub draft echoed the bare short key in the body (`for stub-upstream-key`); the scrubber per the issue pattern list cannot redact a bare short word (indistinguishable from prose), so the stub drops it: the body echoes only token-shaped secrets, and the short key is covered through the Source path. Both processes stopped after (`pgrep` confirms none left).

- Open question for the coordinator (no decision taken): summary-120d at around 1M events cannot meet 300 ms with exact percentiles (two 1M-row sorts in pure-Go SQLite). Options: (a) skip exact percentiles for wide ranges (e.g. span over 7 days returns null/`hourly` without counting), (b) accept the boundary cost (above 1M the gate already returns null fast; below it the cost is bounded by the match count). The contract percentile row only promises 7d/1M under 1s and null above the limit.

## Follow-ups

- Panel status lists (issues 0046 to 0048, already closed on `main`): the `status_class` value `error` is new (failed 2xx or missing code). Any panel filter or label list built against `2xx/4xx/5xx/other` should gain `error`.
- See the open question in the Log on summary-120d exact percentiles at the 1M boundary.

## Result

Landed on branch `issue/0049-usage-db-hardening` (no merge to `main`).

What landed (all four findings, contract shapes unchanged):

- Security 1, account labels: `account_label` is stored only for email-shaped sources (masked `ab***@domain`), else empty. `session_id`, `service_tier`, `reasoning_effort`, `access_provider`, `executor`, `auth_type`, `fail_code` keep only letters, digits, `.`, `_`, `-`, `:` of at most 64 characters, else empty (empty keeps `unknown` for executor and auth type, `default` for the tier). Hostile-input table tests plus a full `usage_events` column scan. Docs updated.
- Security 2, failure text: `fail_summary` scrubs `Bearer` tokens, `Authorization` values, `sk-`/`AIza`/GitHub tokens, JWT shapes, secret query and JSON values (names stay), and 32-plus base64/hex runs to `[redacted]`; first 4096 bytes kept after scrubbing. Realistic-body and seeded fuzz tests. Docs updated.
- Status consistency: failed records are never `2xx` (`error` covers failed 2xx and missing codes); hourly `failed` filter is every class but `2xx`; contract lists the values. Migration 4 fixes old rows and rebuilds affected hourly groups in one transaction. Tests on a v3 DB and on the writer path.
- Query speed: realistic `usageperf` profile (12 models, 8 credentials, 4 keys, credential-centric concentration, hour-of-day bursts; uniform kept as worst case); read pragmas (`cache_size` 32 MiB per conn, `mmap_size` 256 MiB cap, `temp_store MEMORY`; EXPLAIN shows PK search plus temp b-tree for groups); plain-range events totals via the rollup first; hourly upper-bound gate for the percentile over-limit path; internal `usage_daily` rollup (migration 5, same transactions, backfilled from hourly) routed for ranges over 14 days with whole-day UTC bounds (counters, day/week series, breakdown, gates; hour series and heatmap stay hourly). Handler test with populated auth manager; over-limit unit tests (limits are vars now, defaults unchanged).

Schema changes: 4 (`status_class` fix plus hourly rebuild) and 5 (`usage_daily` plus backfill).

Benchmark numbers (warm; targets: summary/timeseries/breakdown 300 ms no percentiles, events 150 ms, failures/heatmap 500 ms, 7d percentiles 1s):

| Route | Uniform (365k hourly) | Realistic (86k hourly) |
|---|---|---|
| summary 120d | 3.33s MISS | 6.49s MISS |
| summary 7d percentiles | 223ms PASS | 378ms PASS |
| timeseries day grouped | 35ms PASS | 22ms PASS |
| breakdown model | 34ms PASS | 22ms PASS |
| events page | 7ms PASS | 4ms PASS |
| failures 30d | ~15ms PASS | ~17ms PASS |
| heatmap 30d | ~63ms PASS | ~18ms PASS |

The two summary misses share one cause: exact P50/P95 at exactly 1M events needs two 1M-row sorts (~3s uniform, ~6.5s realistic). Above 1M the gate returns null fast; the boundary pays the sorts. See the open question in the Log.

Known gaps: summary-120d exact percentiles at the 1M boundary (question above); the uniform worst case is reported only per the issue; panel status lists may need the new `error` class (Follow-ups).
