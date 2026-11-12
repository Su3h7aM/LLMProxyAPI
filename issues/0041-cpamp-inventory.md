# CPAMP usage and observability inventory (reference pin `a67ed883`)

Scope: backend `apps/manager-server` (Go), UI `apps/web`, docs `apps/docs` and `docs/`. All paths below are relative to `.refs/CPA-Manager-Plus`. This file is evidence only. It proposes nothing.

## 1. Ingest: how usage data moves from CPA to CPAMP

CPAMP never reads CPA logs. CPA owns a Redis usage queue. CPAMP drains it with a bearer management key. Three transports exist. Mode `auto` (default) tries them in order: subscribe, then HTTP, then RESP poll.

| Item | Evidence |
|---|---|
| RESP poll | `internal/collector/collector.go:305` `runRESP`. Commands: `AUTH <key>`, then repeat `LPOP` or `RPOP <queue> <count>` (`internal/resp/client.go:63`, `:74`). Side from `PopSide` (default `right`). |
| HTTP poll | `internal/collector/collector.go:277` `runHTTP`. `GET {cpa}/v0/management/usage-queue?count=N` with `Authorization: Bearer <key>` (`internal/httpqueue/client.go:46`). 404/405/501 maps to `ErrUnsupported` and falls back to RESP (`collector.go:289`). |
| RESP subscribe | `internal/collector/collector.go:160` `runSubscribe`. `SUBSCRIBE <queue>` after `AUTH`, 30 s ping, 40 s read window (`collector.go:230`). Push path, no poll delay. |
| Batch size | `BatchSize`, default 100 (`internal/config/config.go:169`, `collector.go:681`). |
| Poll interval | `PollInterval`, default 500 ms (`internal/config/config.go:170`). Ticker wait when queue is empty (`collector.go:349`). |
| CPA side requirement | CPA must set `usageStatisticsEnabled=true`. CPAMP can set it via `PUT {cpa}/v0/management/usage-statistics-enabled` (`internal/service/cpa/client.go:87`). `ValidateCollectorConfig` fails when `pollIntervalMs > redis-usage-queue-retention-seconds * 1000` (`client.go:116`). |
| Record format | Each queue item is one JSON object (or JSON string of an object). `NormalizeRaw` accepts many key aliases (`timestamp`/`time`/`created_at`, `tokens.usage` nesting, `input_tokens`/`prompt_tokens`, and more) (`internal/usage/event.go:776`). Unknown keys do not fail. Non-object or invalid JSON goes to dead letters. |
| Control messages | Queue items `{"refresh":true}` clear the auth snapshot cache. `{"support_refresh":true}` is ignored (`collector.go:456`). |
| Auth snapshot enrichment | Events carry only `auth_index`. CPAMP resolves account, label, file, provider, account ID, project ID from CPA auth-file state and fills empty snapshot fields. Codex events need extra identity checks before fill (`collector.go:510`). |
| Retry | Any pop or insert error sets status `error` and retries with backoff 1 s, x2, max 30 s (`collector.go:641`, `:657`). No limit on retry count. |
| De-duplication | `event_hash` is `sha256(request_id|timestamp|endpoint|model|auth_index|source_hash|input|output|reasoning|cached|failed|latency?)` (`event.go:1326`). `usage_events.event_hash` is unique. `usage_event_identity_ledger` keeps each hash after raw delete, so re-imports stay skipped (`repository/usageevent/repository.go:286`). Result reports `inserted` and `skipped` counts. |
| Dead letters | Parse failures go to `dead_letter_events` with payload and error (`collector.go:426`). Count is in `/status`. |
| Restart and gap recovery | No local cursor exists. Unconsumed items wait in the CPA Redis queue up to `redis-usage-queue-retention-seconds`. After restart the collector resumes drain. Derived rollups resume from checkpoint tables (`usage_rollup_checkpoints`, `usage_monitoring_rollup_state`, `usage_pricing_rollup_state`, `usage_hourly_aggregate_state`). Items older than CPA retention are lost. I did not confirm CPA queue durability rules. The fork must check CPA source. |
| Fan-out after insert | Only new inserts go to quota snapshots and automation handlers (`collector.go:439`). Skipped duplicates do not retrigger automation. |

CPA endpoints CPAMP calls (upstream direction):

| Method and path | Use |
|---|---|
| `GET /v0/management/config` | Validate key, read `usageStatisticsEnabled` and retention (`service/cpa/client.go:26`) |
| `PUT /v0/management/usage-statistics-enabled` | Turn on CPA statistics |
| `GET /v0/management/usage-queue?count=N` | HTTP queue drain |
| RESP `AUTH`, `LPOP`/`RPOP`, `SUBSCRIBE` | RESP queue drain |
| `GET /v0/management/api-keys` | Key list for alias and model pages (`service/cpa/client.go:198`) |
| `GET /v1/models` | Runtime model list (`service/cpa/client.go:241`) |

## 2. Storage

| Item | Evidence |
|---|---|
| Engine | SQLite through `modernc.org/sqlite v1.46.1` (pure Go, no cgo) (`go.mod:7`, `repository/sqlite/db.go:12`). One file, default `/data/usage.sqlite`. |
| Pragmas | `journal_mode=WAL`, `synchronous=FULL`, `busy_timeout=5000`, `foreign_keys=ON` (`repository/sqlite/migrate.go:228`). |
| Migration style | `Migrate()` runs `CREATE TABLE IF NOT EXISTS` plus column and rebuild steps. No down migrations. Version gates reject unknown derived schema versions (`migrate.go:1131`). |
| Raw table | `usage_events` (~70 columns: identity, token buckets, normalized buckets, latency, failure, header quota fields, redacted `raw_json`, `fail_body`). `event_hash` unique. |
| Identity ledger | `usage_event_identity_ledger(event_hash PK, raw_event_id, timestamp_ms, bucket_ms, ...)`. Survives raw delete. |
| Precomputed tables | `usage_account_model_rollups`; `usage_dashboard_hourly_rollups`; `usage_hourly_aggregate_v1`; `usage_pricing_hourly_rollups_v1`; `usage_pricing_account_rollups_v1`; `usage_monitoring_account_daily_rollups_v1`; `usage_monitoring_api_key_daily_rollups_v1`; `usage_monitoring_selector_daily_rollups_v1`; `usage_monitoring_event_projection_v1`; `usage_monitoring_header_latest_v1`; plus checkpoint and state tables per rollup family. |
| Quota tables | `account_quota_observations`, `account_quota_windows`, `account_quota_window_activations`, `account_quota_cycles`, `account_quota_snapshots`. |
| Ops tables | `model_prices` + `model_price_context_tiers` + `model_price_service_tiers`; `api_key_aliases`; `account_action_candidates`; `codex_inspection_runs/results/logs/leases/disable_ownership`; `quota_cooldowns`; `dead_letter_events`; `settings`; archive tables (`usage_archive_runs/segments/event_refs/...`); `usage_data_migrations`. |
| Indexes on raw | `timestamp_ms`; covering `(timestamp_ms, latency_ms, ttft_ms)`; `request_id`; `model`; `auth_index`; `endpoint`; `header_quota_recover_at_ms`; `header_error_kind`; `header_trace_id` (`repository/sqlite/derived_cleanup.go:130`). Event projection has `(timestamp_ms, event_id)`, `(account_key, timestamp_ms, event_id)`, `(analytics_model, timestamp_ms, event_id)` indexes. |
| Retention and pruning | No time-based delete of `usage_events` exists. Pruning path is archive workflow: export segments to files under `usage-archives/`, verify, then delete raw. `CreateRetentionArchive` runs every 24 h when `USAGE_ARCHIVE_RETENTION_ENABLED=true` (default false, default 30 days) (`worker/usage_archive_retention.go:18`, `config.go:27`). |
| Rollup rebuild | Each rollup family tracks `backfill_last_event_id`, `coverage_event_id`, `target_event_id`, `status`. Workers catch up and rebuild after restart or version change. |
| Size per 1M requests | Measured synthetic data (850 B padding per row): 1M events = 1,657,166,064 bytes (~1.54 GiB, ~1.66 kB per event). Covering latency index adds ~1.25% (~20.7 MB per 1M) (`docs/performance/usage-latency-covering-index.zh-CN.md`, table A). Real size varies with `raw_json` length. I found no production size note in docs. |
| Redaction before storage | Secret keys by exact name or suffix (`*_api_key`, `*_management_key`, `*token`, `*secret`, `*password`, `*authorization`, `*cookie`, full list in `usage/persistence_security.go:69`); authorization and cookie headers; emails masked to `pre***@domain`; long secrets replaced with `m:head...tail`; API key and source stored only as `sha256`; source email masked; `fail_body` stays in DB but APIs and exports use 4096 B `fail_summary`; `client_ip` max 64 B, `x_forwarded_for` max 2048 B, `user_agent` max 1024 B; compatible payloads and JSONL exports omit request metadata (`event.go:41`, `:887`, `:1236`). |

## 3. Query API (manager server, all JSON)

Auth: `AuthorizePanel` accepts admin key, or CPA management key when the panel uses an external key (`middleware/auth.go:34`). `AuthorizeAdmin` needs the admin key only. `GET /health` and `GET /usage-service/info` need no auth. `GET /status` needs panel auth.

### Dashboard

| Method and path | Query and body | Response |
|---|---|---|
| `GET /v0/management/dashboard/summary` | `today_start_ms` (required), `now_ms`, `top_models`, `recent_failures` | today and rolling windows, top models, traffic timeline, hourly activity, request-health timeline, token mix, model cost rank, channel health, failure sources, recent failures (`service/dashboard/service.go:201`) |

### Monitoring

| Method and path | Query and body | Response |
|---|---|---|
| `POST /v0/management/monitoring/analytics` | `from_ms`, `to_ms` (required), `now_ms`, `time_zone`, `search_query`, `search_api_key_hash`, filters (models, providers, accounts, credential_ids, auth_files, auth_indices, api_key_hashes, source_hashes, project_ids, request_types, header_error_kinds/codes/plans/trace_ids, include_failed, failed_only, min_latency_ms, cache_status), include flags (summary, summary_profile, summary_percentiles, summary_comparison, timeline, hourly_distribution, model_share, channel_share, model_stats, failure_sources, account_stats, credential_stats, credential_timeline, api_key_timeline, api_key_stats, filter_options, filter_selectors, heatmap, anomaly_points, task_buckets, recent_failures, events_page, drilldown_preview, granularity) | Same sections plus coverage block (raw vs deleted ranges, fidelity limits) (`service/monitoring/service.go:117`, `:188`) |
| `POST /v0/management/monitoring/account-history` | Up to 200 accounts, each with row_key plus identity fields | Per-account history rows |
| `POST /v0/management/monitoring/account-window-usage` | Up to 400 windows with provider_window_id, time range, credential identity | Per-window usage rows |
| `GET /v0/management/monitoring/header-snapshots` | `days`, `limit` | Latest quota header snapshots per credential |

### Request history (compatible usage API)

| Method and path | Notes |
|---|---|
| `GET /v0/management/usage` | CPA-compatible aggregate payload (`BuildPayload`). Row cap is `USAGE_QUERY_LIMIT` (default 50000). |
| `GET /v0/management/usage/export` | Full export as NDJSON (`usage-events.jsonl`). Fail body excluded. |
| `POST /v0/management/usage/import` | One-shot import, max 64 MB body. Returns added, skipped, format warnings. |
| `POST /v0/management/usage/import-sessions` | Start chunked session (`filename`, `size_bytes`, `resume_key`). Defaults: 4 MB chunks, 16 GB disk quota, 2 sessions, 24 h TTL (`config.go:23`). |
| `PUT /v0/management/usage/import-sessions/{id}/chunk?offset=N` | Append bytes. `GET` reads session, `DELETE` cancels, `POST .../validate` checks prefix hash, `POST .../complete` starts processing. |
| `POST /v0/management/usage/archives/preview` | `cutoff_timestamp_ms`. Counts eligible events. |
| `POST /v0/management/usage/archives` | Create archive run. `GET` lists (limit max 100, status/mode/cursor). `GET /{id}` status. `POST /{id}/resume|verify|delete` continues workflow (`?wait=` or `?background=`). `POST /{id}/cancel` abandons before delete stage. |
| `GET /v0/management/usage/maintenance` | Admin only. Archive and maintenance status. |

### Usage analytics cost, credentials, quota

| Method and path | Notes |
|---|---|
| `GET /v0/management/model-prices` | Full price book with tiers |
| `PUT /v0/management/model-prices` | Replace book. User-set fields stay on sync (`preserved` list). |
| `POST /v0/management/model-prices/sync` | Fetch from `models.dev`, `litellm`, `openrouter`, or `multi` (`service/modelprice/service.go:28`) |
| `GET /v0/management/model-prices/runtime-models` | Live CPA models vs priced models |
| `GET /v0/management/model-prices/usage-summary` | Sampled model counts, capped by `USAGE_QUERY_LIMIT` |
| `GET /PUT /v0/management/api-key-aliases`, `DELETE .../{hash}` | Key-hash labels |
| `POST /v0/management/quota-snapshots`, `POST .../query` | Write and query quota window evidence (from response headers and inspection) |
| `GET /usage-service/quota-cooldowns`, `POST .../recover` | Active Codex cooldowns; manual recover (`authFileName`, `authIndex`, `provider=codex`) |

### Health, credentials, automation, settings

| Method and path | Notes |
|---|---|
| `GET /health` | `{ok, service}`. No auth. Used by Docker healthcheck. |
| `GET /status` | service, dbPath, event and dead-letter counts, collector status, migration and maintenance state. `?scope=database-maintenance` returns only maintenance. |
| `GET /usage-service/info` | Setup info. No auth check in handler. |
| `POST /setup` | Admin only. First connection setup. |
| `GET /usage-service/config`, `PUT` | Manager config. GET is open before setup, else panel auth. PUT needs admin. |
| `POST /v0/management/cpa-connection/validate` | Admin only. Server-side CPA check, ignores client key. |
| `GET /usage-service/account-processing-policy`, `PATCH` | Automation switches. GET panel auth, PATCH admin. |
| `POST /v0/management/account-action-candidates` = list (`status`, `limit`); `POST .../{id}/ignore|resolve|enable`; `DELETE .../{id}/auth-file` | Auth-issue queue |
| `POST /v0/management/codex-inspection/run`; `GET .../runs?limit`; `GET/POST .../runs/{id}[/actions][/cancel]` | Manual and scheduled credential inspection |
| `GET /usage-service/updates`, `POST .../check`, `POST .../notification`, `PUT .../channel`, `POST .../dismiss` | Release update check (channels auto, stable, rc, beta) |
| `GET /management.html`, `/favicon.ico`, `/apple-touch-icon.png`; `/` redirects to panel | Embedded web UI |
| `GET /v1/models`, `/models` | Proxied CPA model list |
| `/v0/management/*` proxy; CPA plugin paths; v8 api-keys config | Proxied CPA management (admin; plugin paths accept caller auth) |

## 4. Cost model

| Item | Evidence |
|---|---|
| Unit | USD per 1M tokens. models.dev values are already USD per 1M (`service/modelprice/service.go:692`). No currency setting exists. |
| Per-model fields | `prompt`, `completion`, `cache`, `cacheRead`, `cacheCreation`, each with `*Configured` flag; `source`, `sourceModelId`, `rawJson`; `contextTiers[]`; `serviceTiers[]` (`model/model_price.go:3`). |
| Formula | Uncached prompt x prompt price + legacy cached x cache price + cache-read x cache-read price + cache-create x cache-create price + output x completion price, all divided by 1M (`service/pricing/cost.go:117`). Cache-read price falls back to `max(cache, prompt*0.1)` when unset. Cache-create falls back to prompt price. |
| Context tiers | `model_price_context_tiers(model, threshold_tokens, ...)`. When a request band matches a tier, that tier price replaces base (`cost.go:54`). |
| Long-context premium (legacy) | Requests over 272,000 input tokens (`LongContextInputTokenThreshold`) for gpt-5.4 and gpt-5.5 families cost x2 input and x1.5 output on the long part (`cost.go:66`, `event.go:198`). Explicit context tiers replace this rule. |
| Service tiers | `model_price_service_tiers(model, mode, service_tier, ...)`. Explicit tier price wins. Else multiplier: `flex` and `batch` x0.5; `priority` and `fast` x2 to x2.5 for gpt-5.3-codex/5.4/5.5/5.6 families, x1 else (`cost.go:149`). |
| Unknown model | Cost is 0 when no price row matches (`cost.go:38`). Exception: built-in official prices for gpt-5.6 `luna` (1/6), `terra` (2.5/15), `sol` (5/30) apply even without a row (`cost.go:371`). |
| User overrides | `PUT /v0/management/model-prices` replaces the book. Sync keeps rows the user changed (`preserved[]` in `ModelPriceSyncResult`, `model_price.go:22`). |
| Price sources | `models.dev`, `litellm`, `openrouter`, `multi` chain (`service/modelprice/service.go:28`). Timeout 10 s per source. |

## 5. Dashboard features (web UI)

Routes from `apps/web/src/router/MainRoutes.tsx`. Usage-relevant rows first.

| Page (route) | Data shown | Source endpoint | Priority guess |
|---|---|---|---|
| Dashboard (`/dashboard`) | Traffic overview chart, usage metrics, collector status, health alerts, version | `dashboard/summary`, `status`, `updates` | core |
| Usage analytics (`/usage-analytics`) | Summary cards, timeline, hourly distribution, model share, channel share, model stats, failure sources, account and credential stats and timelines, API key stats and timeline, heatmap, anomaly points, task buckets, recent failures, events table with search and filters, drilldown preview, export | `monitoring/analytics`, `usage/export` | core |
| Monitoring center (`/monitoring`) | Accounts, API keys, realtime tabs; filter panel; health bar; account overview with quota windows; header snapshots; database maintenance hint | `monitoring/*`, `quota-snapshots/query`, `quota-cooldowns` | core |
| Model prices (`/model-prices`) | Price table, context and service tiers, sync status, runtime coverage | `model-prices/*` | core |
| Auth-issue queue (`/monitoring/account-actions`) | Candidate list with ignore, resolve, enable, delete-auth-file actions | `account-action-candidates/*` | useful |
| Usage maintenance (`/usage-maintenance`) | Archive runs, import sessions, retention and maintenance state | `usage/archives`, `usage/import-sessions`, `usage/maintenance` | useful |
| Codex inspection (redirects to Accounts) | Run history, results, logs, quota windows, disable ownership | `codex-inspection/*` | useful |
| Accounts (`/accounts`) | Credential cards, quota state, cooldown hint | `quota-snapshots/query`, `quota-cooldowns` | useful |
| System (`/system`) | Counts, collector state, migration state, maintenance | `status` | useful |
| Manager update (`/manager-update`) | Release check, channel, dismiss | `usage-service/updates/*` | optional |
| Config, AI providers, OAuth, Plugins, Logs, Login, Demo | CPA credential and plugin management, not usage scope | CPA proxy endpoints | optional |

## 6. Automation (acts, not only reads)

| Name | Trigger | Action | Config |
|---|---|---|---|
| Codex quota auto-disable | Usage event with 429-class quota signal and future reset time; Codex only; needs stable auth index | Disables auth file in CPA; writes `quota_cooldowns` row with `recover_at_ms` and CPAMP ownership; recovery worker re-enables at reset | `USAGE_QUOTA_COOLDOWN_ENABLED`, `account-processing-policy` (`worker/rate_limit_auto_disable.go:38`) |
| Manual cooldown recover | `POST quota-cooldowns/recover` | Re-enables one Codex file | panel auth, no extra flag |
| Auth-issue queue | Failed usage events classified by `credentialpolicy` (provider-normalized error signals) | Creates `account_action_candidates`; optional auto-disable of bad credential | `USAGE_ACCOUNT_ACTIONS_ENABLED`, `USAGE_ACCOUNT_ACTIONS_AUTO_DISABLE` (`worker/account_action_candidate.go:104`) |
| Codex inspection | Manual run, or scheduled tick (`CodexInspectionScheduleDue`); DB lease fences parallel runs | Probes auth files against CPA; results can disable, enable, delete, or reauth; ownership table stops fights | inspection settings in manager config |
| Quota snapshot writer | Each inserted usage event with header quota windows; each inspection result | Writes `account_quota_*` lifecycle rows (`quotaSnapshot.WriteUsageEvents`, called from `collector.go:440`) | always on |
| Usage rollups | Event insert; background catch-up loop | Fills hourly, pricing, monitoring, account-history aggregates from checkpoints | `USAGE_DASHBOARD_HOURLY_ROLLUP_ENABLED` (default true) |
| Archive retention | 24 h timer, when enabled | Creates retention archive for data older than N days, then deletes raw | `USAGE_ARCHIVE_RETENTION_ENABLED` (default false), `..._DAYS` (default 30) |
| Update check | Background `UpdateCheckService.Run`; manual check endpoint | Records release notice; UI shows upgrade hint. Never upgrades itself. | channel preference auto/stable/rc/beta |
| Cache accounting migrations | Startup workers | Backfills normalized token buckets and rollup formats across old rows | automatic, status in `/status` |

No webhook, alert, or outbound notice sender exists in `internal/`. A `grep` for `webhook` in backend Go files (outside tests) returns nothing. In-app notices use the web `useNotificationStore` only. I did not check CPA side hooks.

## 7. Auth and deployment

| Item | Evidence |
|---|---|
| Admin key | Format `cpamp_*`, generated when missing. Hash stored in DB via `security.NewAdminCredential`. Env `CPA_MANAGER_ADMIN_KEY` or `*_FILE` (default `/run/secrets/cpa_admin_key`). Reset CLI `command/adminreset` and `docs/reset-admin-key*`. |
| Panel auth | Accepts admin key. Accepts CPA management key only when the panel is in external-management-key mode (`middleware/auth.go:34`, `adminauth/service.go:32`). |
| CPA key | Env `CPA_MANAGEMENT_KEY` or file (default `/run/secrets/cpa_management_key`). Also stored in manager config or setup record after bootstrap. |
| Data key | `CPA_MANAGER_DATA_KEY` or file, path `CPA_MANAGER_DATA_KEY_PATH` (default `/data/data.key`). Encrypts sensitive DB fields. |
| Processes | One Go binary `cpa-manager-plus`. All workers (collector, rollups, retention, inspection, update check) run in-process as goroutines (`cmd/cpa-manager-plus/main.go:169`). Web UI is static files served from `PANEL_PATH` (`/management.html`). Optional separate pprof listener (`CPA_MANAGER_PPROF_ADDR`). No queue, cron, or worker process exists. |
| Ports | `18317` only (`HTTP_ADDR 0.0.0.0:18317`, `Dockerfile.manager-server:32`). Healthcheck `GET /health`. |
| Volumes and files | `/data`: `usage.sqlite` (+ WAL), `data.key`, `config.json`, `usage-archives/`, import staging. Docker volume `cpa-manager-plus-data` (`docker-compose.manager.yml`). |
| Env vars | `HTTP_ADDR`, `CPA_UPSTREAM_URL`, `CPA_MANAGEMENT_KEY[_FILE]`, `CPA_MANAGER_ADMIN_KEY[_FILE]`, `CPA_MANAGER_DATA_KEY[_FILE|_PATH]`, `USAGE_DATA_DIR`, `USAGE_DB_PATH`, `USAGE_COLLECTOR_MODE` (auto/http/resp/subscribe), `USAGE_RESP_QUEUE` (`usage`), `USAGE_RESP_POP_SIDE` (`right`), `USAGE_BATCH_SIZE` (100), `USAGE_POLL_INTERVAL_MS` (500), `USAGE_QUERY_LIMIT` (50000), `USAGE_CORS_ORIGINS` (`*`), `USAGE_RESP_TLS_SKIP_VERIFY`, `USAGE_QUOTA_COOLDOWN_ENABLED`, `USAGE_ACCOUNT_ACTIONS_ENABLED`, `USAGE_ACCOUNT_ACTIONS_AUTO_DISABLE`, `USAGE_DASHBOARD_HOURLY_ROLLUP_ENABLED`, `USAGE_IMPORT_*` (chunk 4 MB, quota 16 GB, sessions 2, TTL 24 h), `USAGE_ARCHIVE_RETENTION_ENABLED/DAYS`, `PANEL_PATH`, `CPA_MANAGER_PPROOF_ADDR`, `CPA_MANAGER_CONFIG`. |
| Parts that exist only because CPAMP is separate | CPA upstream connection (URL + key + queue config); CPA response-header quota parsing; CPA proxy passthrough (`/v0/management/*`, `/v1/models`, plugin resources, v8 API keys); external price sources (models.dev, LiteLLM, OpenRouter); GitHub release index for update check; admin key distinct from CPA key; setup and connection-validate flow. A built-in fork keeps only the price sources and release check external. |

## 8. Limits and known issues

| Item | Evidence |
|---|---|
| Long-window P95 cost | 120-day exact P95 summary over 3.2M events costs ~25 s after the covering index (was ~49 s). Sort cost stays. Not fit for interactive use on huge ranges (`docs/performance/usage-latency-covering-index.zh-CN.md`, section C). |
| Covering index trade-off | New `(timestamp_ms, latency_ms, ttft_ms)` index adds ~1.25% storage and extra write cost. Old time index stays for other queries. No forced index use. Only time and latency filters can be fully covered. |
| Hard API caps | `events_page.limit` max 50000; account history max 200 accounts; account-window usage max 400 windows; archive list max 100; import body max 64 MB; chunked import max 2 sessions and 16 GB staging. |
| No live tail | HTTP and RESP transports poll. Minimum interval is 1 ms in practice, default 500 ms. Only subscribe mode pushes. No websocket or SSE endpoint exists. |
| Retention default | Archive retention worker is off by default. DB grows until an operator archives. |
| Unknown-model cost | Unpriced models report 0 cost, which hides spend. Only gpt-5.6 families have built-in fallback prices. |
| No backend TODOs found | `grep` for `TODO|FIXME|XXX|HACK` in `internal/` Go files outside tests returns nothing at this pin. Open work lives in release notes and the `usage-analytics-implementation-plan` doc, which I did not score line by line. |
| Release history checked | `docs/release-notes` reaches v1.9.x pairs at this pin listing. I did not extract per-release known issues. The coordinator may want a pass over `apps/docs` troubleshooting and `migration-from-cpa-manager*` for extra limits. |

## Unconfirmed items (looked, not found)

| Claim | Where I looked |
|---|---|
| CPA Redis queue durability and retention unit | `service/cpa/client.go` reads `redisUsageQueueRetentionSeconds` from CPA config, but CPA source is outside this reference. Fork must confirm in CPA code whether unconsumed items survive CPA restarts. |
| Exact raw `usage_events` row bytes in production | Only synthetic benchmark exists (see section 2). No production dump in reference. |
| Outbound alerts or webhooks | No match in backend Go. UI notice store not traced to a sender. |
| Per-release known issues after v1.9.x | `docs/release-notes` listing only. Content not read. |
