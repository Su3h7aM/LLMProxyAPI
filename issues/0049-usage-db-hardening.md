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

## Follow-ups

## Result
