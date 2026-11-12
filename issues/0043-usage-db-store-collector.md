---
status: done
branch: issue/0043-usage-db-store-collector
type: backend (Go)
agent: muse-spark-1.3-contributor
base: main
---

# Usage DB: SQLite store, collector, redaction, retention, status route

## Goal

Persist every usage record in an embedded SQLite DB, in process. This is the base for the built-in statistics dashboard. Read `docs/usage-statistics-design.md` first (decisions 1 to 7, 11, and the data model). Prices, cost, and the query routes are other issues (0044, 0045). Do not do them here.

## References (read-only)

- `docs/usage-statistics-design.md`, `DECISIONS.md` (entry 2026-11-12), `issues/0042-triage-fork-usage-capabilities.md` (record fields, dispatcher facts, line numbers), `issues/0041-cpamp-inventory.md` section 2 (CPAMP storage, pragmas, redaction list). CPAMP source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/CPA-Manager-Plus/apps/manager-server` (git-ignored; read by absolute path; never copy its code, write your own).
- `sdk/cliproxy/usage/manager.go` (`Record`, `Plugin`, dispatcher), `internal/redisqueue/plugin.go` (an example of a built-in plugin and its registration), `internal/config` (config types), `internal/api/server_management_v8.go` (route table), `Containerfile`.

## Steps (small commits, Conventional Commits, English)

1. **Dependency.** Add `modernc.org/sqlite` (use the version in `apps/manager-server/go.mod`, v1.46.1, as a start) to the root `go.mod`. Run `go mod tidy`. Check `CGO_ENABLED=0 go build ./...` passes. Remove the stray `apps/manager-server` module (only `go.mod`) and its `go.work` line; say so in the commit body.
2. **Package.** Create `internal/usagedb`. Open the DB with pragmas: `journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`, `foreign_keys=ON`. One writer connection, a small read pool. Migrations: ordered, numbered, in a `schema_migrations` table; refuse to open a DB with a newer schema version. Tables and indexes as in the design (`usage_events`, `usage_hourly`, `settings`, `schema_migrations`; leave price tables to 0044). Columns for cost exist now and stay NULL until 0044.
3. **Redaction.** A pure function `RecordToRow(usage.Record)` that builds a row: `api_key_id` = first 16 hex of `sha256(APIKey)` (empty when no key); `account_label` and `source` masked (email becomes `ab***@domain`; long token-like strings are cut to `head...tail`); `fail_summary` cut to 4096 bytes; no raw key, no body, no headers. Table tests, including a record whose `Source` is the raw key: the row must not contain the key anywhere. `status_class` from the failure status (2xx ok, 4xx, 5xx, other).
4. **Writer.** A buffered channel (size from config, default 4096) and one goroutine. It takes up to 200 rows or 250 ms and inserts them in one transaction: `INSERT ... ON CONFLICT(request_id) DO NOTHING`, and for each inserted row an UPSERT into `usage_hourly` (only for rows that were inserted, so a duplicate never counts twice). When the channel is full, drop the oldest record and increase a drop counter (never block the caller). On a write error: log, keep a counter, retry the batch with backoff (max 30 s); after 5 failed tries drop the batch and count it. Stop flushes on shutdown with a timeout.
5. **Collector.** A built-in `usage.Plugin` that converts the record and sends it to the writer. Register it in the service start-up in the same place as the Redis queue plugin. It must not block `HandleUsage`. It runs when `observability.usage-db.enabled` is true (default true) and ignores `usage-statistics-enabled`.
6. **Config.** Section `observability.usage-db` with `enabled`, `path` (default `<auth-dir>/usage.db`), `retention-days` (default 90, 0 keeps all), `buffer-size`. Add it to `config.example.yaml`, the config types, the v8 config tree, and hot reload if the pattern exists (a path change needs a restart: say so in the comment). Check v0 config output does not break (tests).
7. **Retention.** A daily job (and once at start-up after 1 minute) deletes `usage_events` older than `retention-days`, in batches of 5000 rows with a short pause, never holding the write lock for long. Rollups stay.
8. **Status route.** `GET /v8/management/observability/usage/status`: `enabled`, `path`, `size_bytes`, `schema_version`, `events`, `oldest_ts_ms`, `newest_ts_ms`, `buffer_depth`, `buffer_capacity`, `dropped`, `write_errors`, `last_error`, `retention_days`, `last_prune_ts_ms`. Auth as the other v8 routes. Add it to the route table and route tests.
9. **Tests.** Store: open/migrate/reopen, newer-schema refusal, insert + dedup (same `request_id` twice gives one row and one count), hourly UPSERT sums, retention prune, writer batching, drop-oldest on a full buffer, shutdown flush, concurrent writers with `-race`. Collector: record in, row out. Status route.
10. **Container.** Check `Containerfile` and `compose*.yaml`: `usage.db` lives in the auth dir, which is a volume already. Add one line to the README or docs where the data dir is described. Check the image still builds statically (a `CGO_ENABLED=0` build of `./cmd/server` and `ldd` or `file` on the binary).

## Gate

`/usr/bin/go build ./... && /usr/bin/go vet ./... && /usr/bin/go test -race ./internal/usagedb/... && /usr/bin/go test ./...`, `gofmt -l .` empty. Do not weaken or delete a test to get green. Smoke test: build to `/tmp`, run with a temp config and `auth-dir` in `/tmp`, send a request to a stub or a failing upstream so one record is produced (or inject a record through a test hook), and read the status route. Record the commands and output in the Log.

## Environment

- Use `/usr/bin/go`. Do NOT run `mise trust`. Builds only in `/tmp`. Never leave a server process running.
- English only. ASD-STE100, concise.

## Rules

- Take only what this issue lists. No price tables, no cost, no query routes, no panel code.
- v8 routes only and additive. Do not change a released contract.
- Never store the raw API key, a request body, a response body, or a header value.
- You do not decide. If something is unclear or out of scope: stop, write the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. Work on branch `issue/0043-usage-db-store-collector` in your checkout. Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end: `status: done`, fill `## Result` (what landed, schema, config, known gaps), commit, report. Do not merge to `main`; the coordinator merges.

## Log

- Set `status: in-progress`. Created branch `issue/0043-usage-db-store-collector`.
- Step 1: `go get modernc.org/sqlite@v1.46.1`, `go mod tidy`, removed stray
  `apps/manager-server/go.mod` and its `go.work` line. `CGO_ENABLED=0 go build ./...`
  passes (needs gitignored `apps/web/dist/management.html` placeholder; baseline
  needs it too, pre-existing).
- Steps 2-5, 7: new package `internal/usagedb` (store, redact, writer,
  collector, retention). Pragmas `journal_mode=WAL`, `synchronous=NORMAL`,
  `busy_timeout=5000`, `foreign_keys=ON`; one writer conn plus 4-reader pool;
  ordered migrations with newer-schema refusal; `RecordToRow` redaction;
  batch writer (200 rows / 250 ms, drop-oldest, retry 5 with backoff max 30 s,
  flush on stop); `usage.Plugin` registered as `usage-db`; retention once
  after 1 min then daily, batches of 5000.
- Step 6: `observability.usage-db` config (`enabled` default true, `path`
  default `<auth-dir>/usage.db`, `retention-days` default 90, `buffer-size`
  default 4096), v8 tree mapping, example YAML, hot reload in
  `server_reload.go`, startup in `cmd/server/main.go`, flush in service
  shutdown. `internal/config` tests pass (v8 round-trip incl. example).
- Step 8: `GET /v8/management/observability/usage/status` in the v8 group
  (same auth), handler plus route test, docs row.
- Gate: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty,
  `go test -race ./internal/usagedb/...` ok, `go test ./...` ok.
- Smoke: built to `/tmp/usage0043/llmproxy`, stub upstream 500 on 18099,
  temp config port 18317 with auth-dir `/tmp/usage0043/auth`. `POST
  /v1/chat/completions` model `stub-model` gave upstream 500; status route
  then showed `events:1`, same oldest/newest ts, `dropped:0`,
  `write_errors:0`. Raw keys absent from `usage.db` and WAL bytes;
  `account_label` masked (`m:sk-u...fake`), `api_key_id` 16 hex,
  `status_class` 5xx. Server and stub stopped after.
- Question: in Home mode the usage manager still dispatches to the
  collector, so the local DB keeps local attempt rows while usage is also
  forwarded to Home. Keep per-config behavior, or skip Home mode? No code
  change made for this; coordinator decides.

## Follow-ups

- Home-mode collector behavior (see Log question).
- Price book and cost at insert (issue 0044); query routes (issue 0045).

## Result

Landed: embedded usage DB (`internal/usagedb`), `observability.usage-db`
config, and `GET /v8/management/observability/usage/status`.
Schema v1: `usage_events` (33 columns incl. nullable `cost_micro_usd`,
`cost_source` for 0044), `usage_hourly` keyed on hour/provider/model/alias/
auth/api-key/status/stream, `settings`, `schema_migrations`. Dedup on
`request_id`; hourly UPSERT in the same transaction. Redaction: hashed key
id, masked source/label, 4096 B fail cap with key removal, no bodies or
headers. Retention: 90-day default raw prune, rollups stay.
Known gaps: no price/cost logic (0044), no query routes (0045), retention
first prune runs 1 min after start, Home-mode behavior open (see Log).
