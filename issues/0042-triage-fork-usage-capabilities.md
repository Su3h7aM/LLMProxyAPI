---
status: done
branch: issue/0042-triage-fork-usage-capabilities
type: triage (read-only, no code changes)
agent: deepseek-v4.1-flash
---

# Triage: what the fork already has for usage data

## Goal

The fork needs a built-in usage DB and statistics dashboard (see issue 0041 for the reference product). This issue records the in-process hooks, storage, and panel code that the fork already has, and the gaps. It is read-only. The coordinator writes the design.

## References (read-only)

- Go code in this repo: `internal/redisqueue`, `internal/kvstore`, `internal/store`, `internal/pluginhost`, `sdk/cliproxy`, `sdk/api/handlers` (usage reporting), `internal/api/server_management_v8.go`, `internal/api/handlers/management/`, `internal/config`.
- Panel: `apps/web/src` (pages, services, stores). The Usage page and the dashboard usage section are hidden because the backend has no aggregate data. Find them with `grep -ri usage apps/web/src`.
- Upstream: `.refs/CLIProxyAPI` (how upstream publishes usage records; compare with the fork).
- `DECISIONS.md`, `issues/0018-*.md` to `issues/0040-*.md` (grep for "usage").

## Tasks

1. **Usage records.** Where the proxy creates a usage record per request. Give the Go type (file and line), every field (name, type, meaning, can it be empty), and when it is emitted (success, error, stream end, cancel). Is there token data for cache read, cache write, reasoning? Is the credential (auth id, provider, account label), API key (hashed?), model (requested and routed), latency, status, and request id in the record?
2. **Consumers.** Every consumer of these records today (redisqueue, plugins, logs, quota code). Is there an in-process subscriber API that a built-in DB can use? What is the cost of a slow subscriber (blocking, drop, buffer)?
3. **Storage.** What persistent stores exist (bbolt in `kvstore`, files in auth-dir). Which Go SQL or embedded DB modules are already in `go.mod` or the module cache? Is CGO used in the build? Is the build static for the container (check `Containerfile`)? List options for an embedded DB that needs no CGO, with module name and version, from `go.mod`/`go.sum` evidence only (do not add dependencies).
4. **Panel.** What the panel has now for usage: pages, components, services, types, locale keys, hidden flags, tests. Which backend routes does it call or expect? List the dead code that will be reused or removed.
5. **Management API.** Existing `/v8/management/observability/...` routes: method, path, handler, response. Which ones carry usage data today (`usage` per API key, request log, etc.)?
6. **Pricing and models.** Does the fork have model metadata (context size, prices) in `internal/registry`, `internal/modelconfig`, or the model definitions? List fields and sources.
7. **Container.** Data directory and volume layout (`Containerfile`, `compose*.yaml`, config): where a new DB file would live and what must persist.

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence (file and line, command and output) and a proposal. Mark each claim you could not confirm and say where you looked.
- Use `/usr/bin/go`. Do NOT run `mise trust`. Builds only in `/tmp`.
- English only. Use ASD-STE100 Simplified Technical English, be concise. Use tables.

## Workflow

1. Create branch `issue/0042-triage-fork-usage-capabilities` in your checkout (from `main`). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end set `status: done`, fill `## Result`, commit, and report to the coordinator. Do not merge to `main`; the coordinator merges.

## Log

- 2026-11-11: Set `status: in-progress`. Read `AGENTS.md`, this issue,
  `DECISIONS.md`, issues 0027 and 0032 (panel usage port/hide), and the
  Go usage path. Read-only; only this file changes.
- 2026-11-11: Task 1. The record is `sdk/cliproxy/usage.Record`
  (`manager.go:26`). The plugin view `pluginapi.UsageRecord`
  (`sdk/pluginapi/types.go:1643`) is lossy. Emission is once per
  `UsageReporter`, one reporter per executor attempt
  (`internal/runtime/executor/helps/usage_helpers.go:95`,
  `sdk/api/handlers/handlers_execution.go:239`). Retries therefore emit
  one record per attempt.
- 2026-11-11: Task 2. One consumer path only: the global usage manager
  dispatches to registered plugins (`manager.go:802`). The built-in
  `internal/redisqueue` plugin (`plugin.go:22`) is gated by
  `Enabled` and `UsageStatisticsEnabled`; pluginhost adapters
  (`adapters_usage_translation.go:20`) are not gated by that flag. No
  in-process DB, no usage persistence. The manager queue is an
  unbounded slice and `NewManager(buffer)` ignores `buffer`
  (`manager.go:646,655`).
- 2026-11-11: Task 3. Persistent stores are bbolt `kv.db`
  (`internal/kvstore/bolt_store.go:105`), auth files, logs, and optional
  Postgres/object/git auth stores. Root `go.mod` has bbolt v1.5.0
  (indirect, used) and pgx v5.9.2 (direct). No root sqlite dependency.
  The stray `apps/manager-server/go.mod` names `modernc.org/sqlite`
  v1.46.1 but has no Go packages or `go.sum`. Container build uses
  `CGO_ENABLED=0` (`Containerfile:31`), so dynamic plugins are not
  built in (`internal/pluginhost/loader_unix.go:1`).
- 2026-11-11: Task 4. The panel Usage page is not routed
  (`router/MainRoutes.tsx:19-44`) and not in the sidebar
  (`components/layout/MainLayout.tsx:372-466`). `UsagePanel` is not
  rendered (`DashboardPage.tsx` has no import). Only
  `GET /observability/usage/api-keys` is live (dashboard traffic and
  providers). The flag is read only by the dead Usage page.
- 2026-11-11: Task 5. v8 observability routes are six:
  logs GET/DELETE, error logs list/download, request log, API-key usage,
  usage queue. Only API-key usage and the destructive queue carry usage
  data. `/credentials` also returns `success`, `failed`, and
  `recent_requests` (`auth_files.go:687-689`).
- 2026-11-11: Task 6. `internal/registry.ModelInfo` has context and
  modality metadata, no price fields. Cost exists only as an optional
  upstream-reported `CostDetails` in the record. Task 7: the only
  persistent mount is the auth dir; `kv.db` lives there by default.
- 2026-11-11: `## Result` written. No code changed.
- 2026-11-11: Verification pass on every table row. Confirmed: `Record`
  (`manager.go:26`), `Failure` (98), `Detail` (108), `Plugin` (629),
  `Manager` (639), `NewManager` (655, `buffer` ignored), the append site
  (779), `dispatch` (802), `safeInvoke` (818), `NewManager(512)` (827); the
  `once` guard at `helpers:593`; 52 `NewExecutorUsageReporter` call sites in
  built-in executors; `CGO_ENABLED=0` at `Containerfile:31`; bucket `cpa_kv`
  at `bolt_store.go:31`; the seven v8 observability routes
  (`server_management_v8.go:37-43`); `GetAPIKeyUsage` (58) and
  `GetUsageQueue` (24); `apps/manager-server` holds only `go.mod` (no
  packages, not in `go.sum`); root `go.mod` and `go.sum` have no `modernc`
  or `sqlite` entry; no file imports `UsagePage` (only tests do), and
  `UsagePanel` is imported only by `dashboardRendering.test.ts`.
- 2026-11-11: Corrected six drifted line numbers (API handlers, `Plugin`,
  `helpers:142`, `PublishAdditionalModel`, dashboard hook, config types).
  Added the plugin-executor nesting caveat: "one record per attempt" is exact
  for built-in executors and approximate for plugin executors, because the
  handler makes one reporter and nested calls are suppressed. Status set to
  `done`. No code changed.

## Follow-ups

## Result

Read-only triage. No code changed. All paths are relative to the repository
root. Line numbers are from branch
`issue/0042-triage-fork-usage-capabilities` at the time of writing.

### 1. Usage records

The proxy already creates one rich usage record per upstream attempt. There is
no storage layer behind it.

| Item | Evidence |
| --- | --- |
| Core type | `sdk/cliproxy/usage.Record`, `sdk/cliproxy/usage/manager.go:26` |
| Token detail | `sdk/cliproxy/usage.Detail`, `manager.go:108`; subtypes in `sdk/cliproxy/usage/detail.go:6-120` |
| Failure | `sdk/cliproxy/usage.Failure`, `manager.go:98` |
| Plugin ABI view | `sdk/pluginapi.UsageRecord`, `sdk/pluginapi/types.go:1643`; `UsageDetail`, `types.go:1720` |
| Producer | `helps.UsageReporter`, `internal/runtime/executor/helps/usage_helpers.go:31` |
| Token accounting contract | `sdk/cliproxy/usage/accounting.go:8-60` (`TokenBreakdown`, schema version 2) |

The plugin ABI view is lossy. It drops `Fail.Code`, `TokenBreakdown`,
`CacheDetails`, `CostDetails`, `ToolUsageDetails`, `Iterations`,
`AccessTokenSHA256`, and `HomeSelection` (`internal/pluginhost/adapters_usage_translation.go:602-680`).
A built-in database must consume `sdk/cliproxy/usage.Record`, not the plugin ABI
view.

#### 1.1 Core record fields

"Empty?" means the field can be empty or zero in a delivered record.

| Field | Type | Meaning | Empty? |
| --- | --- | --- | --- |
| `RequestID` | string | UUID v4 for this execution attempt. Filled by `Publish` when empty (`manager.go:733`). | No |
| `TraceID` | string | Parent inbound HTTP request ID, 8-character hex. | Yes, when no inbound ID |
| `SelectionID` | string | Scheduler selection decision ID. | Yes |
| `HomeSelection` | `*HomeSelection` | Home deferred-accounting data. | Yes outside Home |
| `Provider` | string | Upstream provider / executor identifier. | Yes, if caller passes empty |
| `BaseURL` | string | Configured upstream base URL. | Yes |
| `ExecutorType` | string | Concrete executor name. | Yes on the plugin-executor path: `helpers:95` does not set it |
| `Model` | string | Routed/upstream model passed to the executor. | Expected non-empty, not enforced |
| `Alias` | string | Client-requested model name. Defaults to `Model`. | Yes if both empty |
| `APIKey` | string | Client API key principal. **Raw value, not hashed.** | Yes when no access provider |
| `IsNativeKey` | bool | Key came from the native config provider. | Always set |
| `AccessProvider` | string | Client auth provider ID (`config-inline`, plugin ID, ...). | Yes |
| `SessionID` | string | Session ID from client metadata. | Yes |
| `ParentSessionID` | string | Parent session ID. | Yes |
| `AuthID` | string | Selected credential ID. | Yes when `auth == nil` |
| `AuthIndex` | string | Selected credential index. | Yes when `auth == nil` |
| `AccessTokenSHA256` | string | Hash of the upstream OAuth access token version. Not the client key. | Yes for API-key auths |
| `AuthType` | string | Credential kind. | Yes when `auth == nil` |
| `Source` | string | Account label or source: email, Vertex project, auth source kind, else raw client key (`helpers:920-943`). | Yes; may contain the raw key or PII |
| `ReasoningEffort` | string | Requested reasoning effort. | Yes |
| `ServiceTier` | string | Client-requested tier; `default` when absent (`manager.go:289`). | No, defaults |
| `EffectiveServiceTier` | string | Tier after translation and payload defaults. | Yes |
| `ResponseServiceTier` | string | Tier reported by the upstream response. | Yes |
| `ResponseModel` | string | Model reported by the upstream response. | Yes |
| `Generate` | `*bool` | Generation requested. Reporters always set it; nil means true. | No |
| `Stream` | bool | Streaming request. | Always set |
| `RequestedAt` | `time.Time` | Time at reporter creation inside the executor, not request receipt. | No |
| `Latency` | `time.Duration` | Time from reporter creation to publish. Excludes auth, routing, and retry wait. | No |
| `TTFT` | `time.Duration` | Time to first token for streams. | Zero when not measured |
| `Failed` | bool | Attempt failed. | Always set |
| `Fail` | `Failure` | `{StatusCode int, Body string, Code string}`. Zero on success. | Yes |
| `Detail` | `Detail` | Token and cost counters. | Zero values when unknown |
| `ResponseHeaders` | `http.Header` | Selected upstream response header snapshot. | Yes |

Notes on this table:

- `RequestedAt` and `Latency` do not match their field comments. The comment
  says "time the request was received" (`manager.go:88-91`). The value is set
  in `NewUsageReporter` (`helpers:142`), which runs inside `Executor.Execute`
  or `Executor.ExecuteStream`. The dashboard p50/p95 therefore covers executor
  time only.
- `APIKey` is the raw principal from `internal/access/config_access/provider.go:95`
  through `internal/api/server_middleware.go:521`. It is not hashed. `Source`
  can contain the same raw key (`helpers:940-942`).
- The record has no success status code. Success is `Failed == false` with a
  zero `Fail`. The redisqueue payload synthesizes 200
  (`internal/redisqueue/plugin.go:resolveFail`).
- The record has no credential label. Use `AuthID` or `AuthIndex` and join to
  `GET /credentials`.

#### 1.2 Detail fields

| Field | Type | Meaning | Empty? |
| --- | --- | --- | --- |
| `InputTokens`, `OutputTokens`, `TotalTokens` | int64 | Prompt, completion, total. | 0 means unknown or true zero |
| `ReasoningTokens` | int64 | Reasoning/thinking tokens. | 0 as above |
| `CachedTokens` | int64 | Total cached tokens. | 0 as above |
| `CacheReadTokens` | int64 | Cache read. | 0 as above |
| `CacheCreationTokens` | int64 | Cache write. | 0 as above |
| `TokenBreakdown` | struct | Canonical non-overlapping v2 buckets: input uncached/read/write, output reasoning/non-reasoning, unclassified. `Quality` is `complete`, `inconsistent`, or `unclassified`. | Zero value is invalid until `EnsureTokenBreakdownForProvider` runs |
| `CacheCreation5m/30m/1h/12hTokens` | `*int64` | Cache write TTL buckets. | nil |
| `Text/Image/Audio/Video/FileInputTokens` | `*int64` | Input modality split. | nil |
| `CachedText/Image/AudioTokens` | `*int64` | Cached modality split. | nil |
| `Text/Image/AudioOutputTokens` | `*int64` | Output modality split. | nil |
| `AcceptedPredictionTokens`, `RejectedPredictionTokens` | `*int64` | Prediction accounting. | nil |
| `CacheDetails` | `*CacheDetails` | Cache mode, TTL, retention, per-TTL map. | nil |
| `WebSearchRequests`, `ImageCount` | `*int64` | Tool and image counters. | nil |
| `RequestedImageDetails`, `ImageDetails` | `*ImageDetails` | Image size, quality, format, ... | nil |
| `MediaCounts` | `*MediaCounts` | Media and audio duration counters. | nil |
| `ToolUsageDetails` | `*ToolUsageDetails` | Web search, fetch, MCP, code execution, ... counters. | nil |
| `Iterations` | `*[]IterationUsage` | Per-iteration usage. | nil |
| `CostDetails` | `*CostDetails` | Optional upstream-reported cost (for example OpenRouter). | nil |
| `Speed`, `InferenceGeo`, `UpstreamServiceTier` | string | Upstream hints. | Yes |

Cache read, cache write, and reasoning data are present. Cache write has both a
total and TTL buckets. Numeric zeros cannot distinguish "unknown" from "zero".
The canonical `TokenBreakdown.Quality` is the only quality signal.

#### 1.3 When a record is emitted

| Case | Evidence | Record content |
| --- | --- | --- |
| Non-stream success | `sdk/api/handlers/handlers_execution.go:239-240` | `Publish(detail)` then `EnsurePublished`; first call wins (`helpers:593-600`) |
| Non-stream error | `handlers_execution.go:230`; `internal/runtime/executor/codex_executor_execute.go:286-353` | `PublishFailure` or `PublishFailureWithDetail`; `Failed=true` |
| Stream end, success | `sdk/api/handlers/handlers_stream.go:205-216` | Buffered detail, else zero-token success via `EnsurePublished` |
| Stream error | same defer | `PublishFailure` with buffered detail when present |
| Client cancel | `handlers_stream.go:223-231` | `Failed=true`, status 499 (`internal/clienterror/client_error.go:16,57`) |
| Upstream response has no usage fields | `helpers:622-628` | Zero-token record; keeps request counts |
| Image/tool side model | `helpers:365`, `helpers:555-580` | Extra record with a new `RequestID`; bypasses the `once` guard |
| Failure before an executor runs | `handlers_execution.go` auth/pick errors | No record |
| Internal source request | `handlers_execution.go:222` | No reporter, no record |
| Home upstream 401 without executor | `sdk/cliproxy/auth/home_result.go:57` | Zero-token failed record, status 401 |

Two consequences:

1. The unit is the upstream attempt, not the client request. A credential
   rotation or retry creates a new executor call and a new record. Traffic
   totals count failed attempts too.
2. A request can create more than one record only through
   `PublishAdditionalModel` (image generation side models). It gets a new
   `RequestID` and `HomeSelection.Additional=true`.

Two precision notes on consequence 1:

- Built-in executors make one reporter per attempt: 52 call sites of
  `NewExecutorUsageReporter` (`internal/runtime/executor/`, for example
  `codex_executor_execute.go:133,444`). Verified by count.
- The plugin-executor path makes one reporter in the handler
  (`handlers_execution.go:223`) and suppresses nested plugin calls
  (`nestedTracker.hasNestedExecution`). A nested plugin call does not add a
  record. So "one record per attempt" is exact for built-in executors and
  approximate for plugin executors.

### 2. Consumers

| Consumer | Registration / path | Filter | Effect |
| --- | --- | --- | --- |
| Usage manager | `sdk/cliproxy/usage/manager.go:639` | none | Owns the queue and dispatches to plugins |
| redisqueue plugin | `internal/redisqueue/plugin.go:16-22`; registered in `init` | `redisqueue.Enabled()` and `UsageStatisticsEnabled()` | JSON marshal, then in-memory queue |
| RESP / management queue | `internal/api/redis_queue_protocol.go:42,239-252`; `handlers/management/usage.go:24` | none | External RESP subscribers; destructive pop |
| Home forwarder | `sdk/cliproxy/service_home.go:349-360` | Home mode | `LPushUsage` to the Home server |
| Plugin usage adapters | `internal/pluginhost/adapters_usage_translation.go:20-31,553` | plugin declares `UsagePlugin` | Delivers to each plugin; not gated by `usage-statistics-enabled` |
| Request logs | `internal/logging/request_logger_format.go:448` | none | No structured token fields |
| Quota / health | `sdk/cliproxy/auth/conductor_cooldown.go:923` (`MarkResult`) | none | Separate path; success/failed counters and cooldowns |

There is no consumer that persists usage.

#### 2.1 In-process subscriber API

Yes. `sdk/cliproxy/usage.Plugin` has one method, `HandleUsage(ctx, Record)`
(`manager.go:629-631`). Register with `RegisterPlugin` or
`RegisterNamedPlugin` (`manager.go:832-836`). The default manager starts in
`sdk/cliproxy/service_lifecycle.go:59`. `internal/redisqueue` uses this API
from `init`.

The API has no channel, no filter, no topic, no replay, and no per-sink queue.
All plugins share one dispatcher goroutine.

#### 2.2 Cost of a slow subscriber

| Property | Evidence | Consequence |
| --- | --- | --- |
| Single dispatcher | `manager.go:784-836` | One slow plugin delays every later record for every plugin |
| Unbounded queue | `manager.go:646` (`queue []queueItem`); `Publish` appends without a limit (`manager.go:772-780`) | Memory grows; no drop and no backpressure |
| `buffer` argument ignored | `manager.go:655-659`; `NewManager(512)` at `manager.go:827` | The "buffered queue" comment is not true |
| Whole context retained | `queueItem.ctx` (`manager.go:634`); `Publish` stores the context | A record can retain the gin context and request data until dispatch |
| No timeout, no drop | `dispatch` calls plugins directly (`manager.go:802-825`) | A hung plugin stops usage delivery |
| Panic isolation only | `safeInvoke` recovers (`manager.go:817-825`) | A failing plugin does not stop the rest, but does not unblock a hang |
| Plugin-host backpressure | `adapters_usage_translation.go:134,454-520` | Per-plugin pending limit 4096; then blocks the shared dispatcher |
| Subscriber drop (post-plugin) | `internal/redisqueue/queue.go:12,141-160` | Full 256-slot subscriber channels are closed and removed |

A built-in database sink must buffer in its own goroutine and batch writes. It
must not write synchronously in `HandleUsage`.

### 3. Storage

#### 3.1 Existing persistent stores

| Store | Path / code | Contents | Usage data? |
| --- | --- | --- | --- |
| bbolt KV | `internal/kvstore/bolt_store.go:105`; default `<auth-dir>/kv.db` (`internal/config/config_types.go:458-489`); config `server.kv-store` (`config.example.yaml:94-98`) | One bucket `cpa_kv`, TTL envelopes, caches, affinity snapshots | No |
| Auth files | `internal/config` `auth-dir`; JSON files per credential | Credentials and quota state | No |
| Logs | `internal/logging/global_logger.go:187`; `logs` dir | Application and request logs | No structured tokens |
| Postgres store | `internal/store/postgresstore.go:21-34` | Config, auth, cooldown tables; `pgx` stdlib | No |
| Object store | `internal/store/objectstore.go:32` | Auth/config objects (S3/MinIO) | No |
| Git store | `internal/store/gitstore.go:37` | Auth/config snapshots | No |
| Auth counters | `sdk/cliproxy/auth/types.go:146-148,156` (`json:"-"`) | `Success`, `Failed`, 20 in-memory buckets | In-memory only; reset on restart |

The 20 recent-request buckets are 10 minutes each, so the window is 200
minutes (`types.go:319-320,462-481`). They are not persisted.

#### 3.2 Go SQL / embedded DB modules in the repository

| Module | Version | Evidence | State |
| --- | --- | --- | --- |
| `go.etcd.io/bbolt` | v1.5.0 | `go.mod` require (indirect); `go.sum:241-242`; imported by `internal/kvstore/bolt_store.go:20` | Already used. Pure Go |
| `github.com/jackc/pgx/v5` | v5.9.2 | `go.mod` direct; `go.sum:103-104` | Client-server SQL. Not embedded |
| `modernc.org/sqlite` | v1.46.1 | `apps/manager-server/go.mod:6` only. No Go packages in that directory; no `go.sum` in the fork; root `go.sum` has no `modernc` entry | Not a fork dependency. Pure Go. CPAMP uses it (`/home/su3h7am/Apps/LLMProxyAPI/.refs/CPA-Manager-Plus/apps/manager-server/go.mod:7`) |
| `github.com/redis/go-redis/v9` | v9.19.0 | `go.mod` direct | Redis client. Not storage for this purpose |

No Badger, Pebble, DuckDB, or `mattn/go-sqlite3` entry exists in `go.mod` or
`go.sum`.

The Go module cache at `/home/su3h7am/go/pkg/mod/modernc.org` holds
`sqlite@v1.46.1` plus newer versions up to `v1.61.1` (directory listing). This
is cache evidence only, not a repository dependency. Marked **unconfirmed as a
dependency decision**.

#### 3.3 CGO and static build

| Fact | Evidence |
| --- | --- |
| Container build disables CGO | `Containerfile:31`: `CGO_ENABLED=0 GOOS=linux go build ...` |
| Runtime image is Alpine | `Containerfile:34-35` |
| Dynamic plugin loading needs CGO | `internal/pluginhost/loader_unix.go:1`: `//go:build cgo && (linux || darwin || freebsd)` |
| Non-CGO loader is unsupported | `internal/pluginhost/loader_unsupported.go:1-8` |
| Plugin support value is `0` without CGO | `internal/pluginhost/support_nocgo.go:1-5` |

So the shipped container binary is static, and it cannot load dynamic
`.so` plugins. A built-in usage database must be compiled into the binary.

#### 3.4 Embedded DB options (evidence only, no decision)

| Option | Module and version | Evidence | Strengths | Limits |
| --- | --- | --- | --- | --- |
| bbolt | `go.etcd.io/bbolt` v1.5.0 | in `go.mod` and `go.sum`; open code at `internal/kvstore/bolt_store.go:105-123` | No new dependency; pure Go; single file; file mode 0600; simple buckets | No SQL; single writer; manual indexes and retention; no ad-hoc aggregation |
| SQLite (pure Go) | `modernc.org/sqlite` v1.46.1 | only in the stray `apps/manager-server/go.mod`; CPAMP uses it; cache has it | SQL, indexes, `GROUP BY`, WAL, easy retention; pure Go; proven by CPAMP | New root dependency and `go.sum`; larger binary; one writer |
| PostgreSQL | `github.com/jackc/pgx/v5` v5.9.2 | root `go.mod` and `go.sum`; `internal/store/postgresstore.go:21-34` | Already present; good for cluster | External service; not embedded; not available in simple deployments |
| NDJSON files | standard library | no dependency | Trivial append | No indexes, no safe retention, slow queries |

### 4. Panel

#### 4.1 State: live, hidden, dead

| Item | File | State |
| --- | --- | --- |
| Usage page | `apps/web/src/pages/UsagePage.tsx` | Not routed (`src/router/MainRoutes.tsx:19-44`); not in nav (`src/components/layout/MainLayout.tsx:372-466`) |
| Usage page guard | `apps/web/src/pages/usage-page-request-guard.ts` | Used only by the dead page and its tests |
| Usage page styles | `apps/web/src/pages/UsagePage.module.scss` | Used only by the dead page |
| Usage snapshot types | `apps/web/src/types/usage.ts` | Reused by dead page, dead panel, dead logic |
| Snapshot reshape | `apps/web/src/utils/usageStats.ts` | Used only by the dead page |
| Dashboard usage section | `apps/web/src/features/dashboard/components/UsagePanel.tsx` | Not rendered; `DashboardPage.tsx` has no import |
| Dashboard usage logic | `apps/web/src/features/dashboard/logic/usage.ts` | Used by `UsagePanel` and tests |
| Dashboard usage source | `apps/web/src/features/dashboard/hooks/useDashboardOverview.ts:17-52` | `usage` is excluded from polled sources; `UsageSourceData` remains |
| Source ID | `src/features/dashboard/logic/attention.ts:5` | `'usage'` remains in `DashboardSourceId` |
| API-key usage | `src/services/api/apiKeyUsage.ts:7-10` | Live. Dashboard traffic and provider pages |
| Flag reader | `src/services/api/usage.ts:4,14` | Used only by the dead Usage page |
| Config flag UI | `src/hooks/useVisualConfig.ts:1358,1611-1621`; `src/features/config/searchIndex.ts:347` | Live |
| Locale keys | `src/i18n/locales/*.json` | `usage_stats.*`, `redesign.usage.*`, `dashboard.usage_*` remain; `nav.usage_statistics` is gone |
| Chart height token | `src/design-system/tokens.ts:603` (`chart.height.usage: 220`) | Kept for the dead page |
| Tests | `tests/usagePageRedesign.test.ts`, `tests/usagePageConcurrency.test.ts`, `tests/dashboardUsage.test.ts`, `tests/dashboardRendering.test.ts`, `tests/runtimeReview.test.ts` | Cover dead code; `productionLocales.test.ts` checks locale parity |

Issue 0032 hid the page and the dashboard section on purpose and kept the code
for a later data source (`issues/0032-panel-cleanup.md:34,73-86`).

#### 4.2 Routes the panel calls or expects

| Route | Caller | State |
| --- | --- | --- |
| `GET /observability/usage/api-keys` | `src/services/api/apiKeyUsage.ts:8` | Live |
| `GET /credentials` | `src/features/dashboard/hooks/useDashboardOverview.ts:90`; includes `success`, `failed`, `recent_requests` | Live |
| `GET /config/observability/usage/usage-statistics-enabled` | `src/services/api/usage.ts:4` | Read only by the dead page; config UI uses the shared config store |
| `GET /observability/logs` and related | `src/services/api/logs.ts:137-166` | Live; no token data |
| `GET /observability/usage/queue` | none | Not called. The guard comment forbids it (`usage-page-request-guard.ts:45-47`) |
| `GET /usage` | none | Not called; no backend route |

#### 4.3 Snapshot shape the dead code expects

`apps/web/src/types/usage.ts` and `utils/usageStats.ts` define this shape. A new
summary route can return it, and the hidden page plus dashboard section become
usable again.

| Field | Type | Consumer |
| --- | --- | --- |
| `total_requests`, `success_count`, `failure_count`, `total_tokens` | numbers | Usage page stats; dashboard vitals |
| `apis` | map API name to `{total_requests, total_tokens, failure_count, models}` | Usage page tables |
| `apis[*].models` | map model to `{total_requests, total_tokens, failure_count, details[]}` | Top models; API breakdown; dashboard table |
| `details[]` | `{timestamp, latency_ms, source, auth_index, tokens{input,output,reasoning,cached,total}, failed}` | p50/p95, recent failures |
| `requests_by_day`, `tokens_by_day` | map `YYYY-MM-DD` to number | Daily charts |
| `requests_by_hour`, `tokens_by_hour` | map hour label to number | Defined but not rendered |

The panel masks the API-name map key before display
(`utils/usageStats.ts:42-56`) and the test asserts that a raw secret is absent
(`tests/usagePageRedesign.test.ts:200-215`). A new backend should return a
hashed key ID, not the raw key.

### 5. Management API

#### 5.1 v8 observability routes

All handlers are in `internal/api/handlers/management/`. Route table:
`internal/api/server_management_v8.go:37-43`. Documented in
`docs/management-api-v8.md:106-113`.

| Method | Path | Handler | Response | Usage data |
| --- | --- | --- | --- | --- |
| GET | `/v8/management/observability/logs` | `GetLogs` (`logs.go:40`) | `{lines[], line-count, latest-timestamp, next-cursor, reset}` | No structured tokens |
| DELETE | `/v8/management/observability/logs` | `DeleteLogs` (`logs.go:135`) | `{success, message, removed}` | No |
| GET | `/v8/management/observability/logs/errors` | `GetRequestErrorLogs` (`logs.go:197`) | `{files[]}` | No |
| GET | `/v8/management/observability/logs/errors/:name` | `DownloadRequestErrorLog` (`logs.go:430`) | File download | No |
| GET | `/v8/management/observability/logs/requests/:id` | `GetRequestLogByID` (`logs.go:261`) | Request log text | Raw bodies only |
| GET | `/v8/management/observability/usage/api-keys` | `GetAPIKeyUsage` (`api_key_usage.go:58`) | `{provider: {"baseURL|apiKey": {success, failed, recent_requests[]}}}` | Counts and 200-minute buckets |
| GET | `/v8/management/observability/usage/queue` | `GetUsageQueue` (`usage.go:24`) | JSON array of raw payload objects | Full record fields; **destructive pop** |

v0 equivalents exist: `/v0/management/usage-statistics-enabled` GET/PUT,
`/v0/management/api-key-usage`, `/v0/management/usage-queue`
(`internal/api/server_management.go:64-65,92-93`).

#### 5.2 API-key usage details

`GetAPIKeyUsage` (`api_key_usage.go:58-113`):

- Includes only auths whose `AccountInfo()` kind is `api_key`
  (`api_key_usage.go:75-78`). OAuth credentials are excluded.
- Groups by lowercase provider, or `compat_name` when present
  (`api_key_usage.go:43-55`).
- Composite key is `base_url + "|" + api_key` (`api_key_usage.go:91`). The raw
  API key is the map key.
- Returns `success`, `failed`, and 20 buckets of 10 minutes. Each bucket is
  `{time: "HH:MM-HH:MM", success, failed}` (`sdk/cliproxy/auth/types.go:333-337`).
- No model, token, latency, cost, or request ID.
- Counters reset on restart (`json:"-"`).

`GET /credentials` returns the same `success`, `failed`, and `recent_requests`
per credential (`auth_files.go:687-689`).

#### 5.3 Usage queue details

`GetUsageQueue` (`usage.go:24-46`):

- Query `count`, positive integer, default 1, no maximum.
- Pops up to `count` items with `redisqueue.PopOldest`.
- Default retention is 60 seconds, maximum 3600
  (`internal/redisqueue/queue.go:11-14`; `config.example.yaml:1291-1293`).
- Items are the JSON payload built in `internal/redisqueue/plugin.go:168-235`.
  The payload includes request/execution/trace IDs, tokens, model, provider,
  auth index, access-token hash, session, endpoint, failure, cost, tool and
  image details.
- It is a transport endpoint. It is not a history source and must not be
  polled by the panel.

### 6. Pricing and model metadata

| Item | Evidence | Result |
| --- | --- | --- |
| Model type | `internal/registry.ModelInfo`, `model_registry.go:41-109` | Context and capability metadata |
| Context fields | `InputTokenLimit`, `OutputTokenLimit`, `ContextLength`, `MaxContextLength`, `MaxCompletionTokens` | Present |
| Other fields | `Object`, `Created`, `OwnedBy`, `Type`, `DisplayName`, `Name`, `Version`, `Description`, `SupportedGenerationMethods`, `SupportedParameters`, input/output modalities, `SupportsWebSearch`, `Thinking`, `Config`, `UserDefined`, `IsCompat` | Present |
| Static catalog | `internal/registry/models/models.json` (embedded), `devin_models.json`, `codex_client_models.json` | Context and capability only; no price keys |
| Remote refresh | `internal/registry/model_updater.go:18-22,69`; 3-hour interval | Same schema |
| Resolution | `internal/modelconfig/model_info.go:12-32` | Static info plus config overrides |
| Model config | `ModelConfig` has `OverrideHeader` only (`model_registry.go:112-116`) | No price |
| Price fields | repository grep for `pricing`, `input_price`, `output_price`, `per_million` | Only a price sentence in a video model `Description` (`models.json:5072`); no price field |
| Per-request cost | `usage.CostDetails`, `sdk/cliproxy/usage/detail.go:98-104` | Optional upstream cost only (for example OpenRouter); not computed locally |

The fork has no price table. Cost calculation needs a new source.

### 7. Container and persistence

| Item | Evidence | Value |
| --- | --- | --- |
| Work dir | `Containerfile:41` | `/LLMProxyAPI` |
| Binary | `Containerfile:31,45` | `/LLMProxyAPI/llmproxy` |
| User and HOME | `Containerfile:35-38` | `llmproxy`, `HOME=/home/llmproxy` |
| Data dir default | `internal/config/config_xdg.go:27-35` | `$XDG_DATA_HOME/llm-proxy-api`, else `~/.local/share/llm-proxy-api` |
| auth-dir default | `config.example.yaml:1018-1020` | Empty means the data dir above |
| kv.db default | `internal/config/config_types.go:458-489` | `<auth-dir>/kv.db` |
| Compose auth volume | `compose.yaml:26` | `${LLM_PROXY_AUTH_PATH:-./auths}:/home/llmproxy/.local/share/llm-proxy-api` |
| Compose log volume | `compose.yaml:27` | `${LLM_PROXY_LOG_PATH:-./logs}:/LLMProxyAPI/logs` |
| Compose plugin volume | `compose.yaml:28` | `${LLM_PROXY_PLUGIN_PATH:-./plugins}:/LLMProxyAPI/plugins` |
| Cluster volume | `compose.cluster.yaml:17` | `${LLM_PROXY_HOME_PATH:-./home}:/home/llmproxy/.local/share/llm-proxy-api` |
| Log directory | `internal/logging/global_logger.go:187-203` | `logs` under config dir or auth dir |

A new DB file under `<auth-dir>` persists with the existing auth volume. A file
under `/LLMProxyAPI` or the current work dir does not persist unless a new
volume is added. In Home mode the local management API is disabled
(`docs/management-api-v8.md:12`), and usage is forwarded to Home
(`service_home.go:349`), so a local DB would not see the Home records.

### 8. Proposal (not a decision)

1. Add one built-in in-process sink that implements
   `sdk/cliproxy/usage.Plugin` and registers with `RegisterNamedPlugin`.
   Do not use the plugin ABI record: it loses fields.
2. The sink must own an internal buffered channel and batch writer. The shared
   dispatcher is single-threaded and unbounded; a synchronous DB write blocks
   all usage delivery.
3. Store one row per record. Keep `RequestID` unique, index `RequestedAt`,
   `(Provider, Model, RequestedAt)`, `AuthID`, and a hashed client key ID.
   Do not store the raw `APIKey` or the raw `Source` key.
4. Add an additive v8 summary route that returns the existing panel
   `UsageSnapshot` shape from SQL/bucket queries. Do not reuse
   `/observability/usage/queue` for display.
5. Put the file at `<auth-dir>/usage.db` by default and add an explicit config
   path. The existing auth volume then persists it. Honor Home mode: forward
   only, or store only after Home returns authoritative accounting.
6. Choose the engine on query needs. bbolt needs no new dependency but needs
   manual indexes and aggregation. Pure-Go SQLite needs one new root
   dependency and gives SQL, WAL, and retention. Both are CGO-free.

### 9. Unconfirmed claims

| Claim | Status | Where I looked |
| --- | --- | --- |
| `modernc.org/sqlite` is available as a fork dependency | **Not confirmed.** Only a stray `apps/manager-server/go.mod` names it; root `go.mod` and `go.sum` do not | `go.mod`, `go.sum`, `apps/manager-server/`, module cache directory |
| A DB file in `<auth-dir>` is writable by the container user | **Not confirmed by a container run.** Existing `kv.db` uses the same directory, so the path is plausible | `Containerfile`, `compose.yaml`, `kvstore/path.go`; no container started |
| All upstream providers populate every token counter | **Not confirmed.** Zero values are common and can mean unknown | `helpers:552-730`, `detail.go`; no per-executor audit |
| The `Source` field always identifies an account label | **Not confirmed.** Code falls back to the raw client key | `helpers:920-943` |
| `GET /observability/usage/queue` is safe to call concurrently with Home forwarding | **Not confirmed.** It pops the same queue that the Home forwarder pops | `usage.go`, `service_home.go:349` |

### 10. Follow-ups

- Remove or complete the stray `apps/manager-server/go.mod`. It is in
  `go.work`, has no packages, and names an unused SQLite dependency. Decide
  whether it is a real CPAMP server stub or leftover.
- Decide whether `usage-statistics-enabled` should gate a built-in DB. Today
  it gates only the redisqueue plugin; pluginhost usage delivery ignores it.
- Fix the `Record.RequestedAt` and `Latency` comments or start the clock at
  request receipt if the dashboard needs end-to-end latency.
- Add a stable client-key identifier. The raw key in `APIKey` and `Source`
  must not enter a database or a UI payload.
- Delete or reuse the dead panel usage code after the summary route lands:
  `UsagePage`, `usage-page-request-guard`, `UsagePage.module.scss`,
  `utils/usageStats.ts`, dashboard `UsagePanel` and `logic/usage.ts`, the
  `dashboard.usage_*`/`redesign.usage.*`/`usage_stats.*` locale keys, and the
  `chart.height.usage` token. Keep the tests only if the code is reused.
- Confirm whether the queue pop must be restricted to the transport client in
  Home mode and in multi-client deployments.
