---
status: done
branch: issue/0017-triage-panel-decoupling-and-shortcuts
type: triage (read-only, no code changes)
agent: muse-spark-1.3-contributor
---

# Triage panel migration C: decoupling code and shortcut opportunities

## Context

The fork dashboard (`apps/web`) will be replaced by the BumpyClock Management Center. Read `DECISIONS.md` (entries of 2026-11-11 about the dashboard and about "one unit") and `issues/0015-triage-panel-api-and-content.md` (merged result is on the 0015 branch; read it with `git show issue/0015-triage-panel-api-and-content:issues/0015-triage-panel-api-and-content.md` if it is not on `main` yet).

The panel and the proxy ship together: one container, one repository, one version. The panel can drop everything that exists only because it may talk to an unknown or separate server. Where the panel needs many calls or heavy client work only because the API is generic, a direct endpoint on the proxy is better. Limit: released `/v0` and `/v8` contracts never change or shrink; new endpoints must be additive.

References (read-only, never edit or commit there):

- BumpyClock panel: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center`
- Fork panel today: `apps/web` in your checkout
- Fork backend: your checkout (`internal/api`, `internal/api/handlers/management`, `sdk/cliproxy`)

## Tasks

1. **Decoupling code to delete.** In the BumpyClock source and in the fork panel (`apps/web`), list code that exists only because the panel is a separate client: backend probes, version negotiation and `x-cpa-*` / `x-server-version` header handling, capability checks that try an endpoint and fall back on 404, legacy backend detection (`legacyBackendProbe`), API base detection and normalization, custom/remote connect UI and storage scoped by base, v0 paths and v0-vs-v8 branching, CORS or proxy related workarounds, "minimum version" texts and i18n strings. Give file:line, what it does, and the safe deletion (what must stay, for example the management key login, 401 handling, ETag or revision guards that protect concurrent writes).
2. **Shortcut candidates.** Find places where the panel does client-side stitching that a single new endpoint could do: several round trips for one screen, merging or reshaping data in the browser, per-credential fan-out requests (for example quota or model fetch through `POST /api-call` to reach a provider), polling loops, repeated config reads. For each: current calls (file:line), what the browser computes, a proposed additive endpoint (method, path under `/v8/management`, request and response sketch), expected gain, and risk. Check the backend code to say if the data is already available inside the server (cite the Go function). Rank by value.
3. **Contract sharing.** Check if the Go management handlers and the TypeScript types can share one source of truth cheaply (for example a generated TS types file from Go structs or from an OpenAPI-like list). Say what exists today (cite files), what the cost is, and if it is worth doing now or later. Do not implement.
4. **Embedded delivery.** List what the server does today to serve the panel (`/management.html`, static path override, auto-update or download of the panel, `remote-management` options such as `disable-auto-update-panel`, `MANAGEMENT_STATIC_PATH`) and what of this is dead weight for a panel that is embedded in the binary (for example panel update and download logic). Give file:line and a safe removal list, marked as "released config key" if removing it would break an existing config file (config compat is mandatory: such keys must keep loading and be ignored).

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence and a proposal. The coordinator decides.
- Back each claim with a file path and line, a command, or its output. Mark each claim you could not confirm and say where you looked.
- Environment: use `/usr/bin/go`, not the `go` shim (do NOT run `mise trust`). bun is at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin/bun`. Do builds only in a scratch copy under `/tmp`.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0017-triage-panel-decoupling-and-shortcuts` in your checkout (rename your branch if T3 gave another name). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often (Conventional Commits).
3. When done: set `status: done`, fill `## Result` (tables per task, ranked shortcut list, risks, claims you could not confirm), commit. Do NOT merge to main. Do not create other issue files.
4. Report to the coordinator: short summary, branch, commit range, key findings, questions, surprises.

## Log

- 2026-11-11: Set `status: in-progress`. Read `DECISIONS.md`, issue 0015 result. Created branch.
- 2026-11-11: Ran parallel evidence pulls (decoupling grep, shortcut grep, contract sharing, embedded delivery). Verifying claims with file:line.
- 2026-11-11: Verified all key claims with direct reads (`connection.ts`, `useAuthStore.ts:55-150`, `LoginPage.tsx`, `server_management.go:317-340`, `config_types.go:518-533`, `updater.go`, `server_management_v8.go:21-70`, `quota.go:28-70`, `api_tools.go:273-294,819-869`). Wrote `## Result`. Set `status: done`. No code changed.

## Follow-ups

## Result

### Task 1: decoupling code to delete

"Fork" = `apps/web` in this checkout. "Bumpy" = `.refs/BumpyClock-Cli-Proxy-API-Management-Center` (read-only).

| # | Area | File:line | What it does | Proposal |
|---|------|-----------|--------------|----------|
| 1.1 | Legacy probe type | Fork `src/services/api/legacyBackendProbe.ts:5-10` | `LegacyBackendError` diagnostic, v0-yes/v8-no message | KEEP (sanctioned diagnostic). Delete only if legacy error text is dropped. |
| 1.2 | Legacy probe fetch | Fork `src/services/api/legacyBackendProbe.ts:12-30` | GETs `${base}/v0/management/config`, 3 s timeout, `maxRedirects:0`; 401/403/404/network → false | KEEP as diagnostic only. Never use v0 data. Only v0 touchpoint allowed. |
| 1.3 | Probe branch in login | Fork `src/stores/useAuthStore.ts:133-147` | On 404 with unchanged revision → `probeLegacyBackend()` → throw `LegacyBackendError` | Probe branch SAFE-TO-DELETE (generic 404 stays). Revision guards MUST STAY. |
| 1.4 | Probe i18n mapping | Fork `src/pages/LoginPage.tsx:16,34`, `src/i18n/locales/en.json:89-90` (+6 locales) | Maps probe error to `login.error_legacy_backend` / `error_not_found` v8 text | Delete with probe; else KEEP. |
| 1.5 | Probe test | Fork `tests/legacyBackendProbe.test.ts:4,24-59` | Pins probe URL and non-object rejection | Delete with probe. |
| 1.6 | Version headers | Fork `src/utils/constants.ts:19-23`, Bumpy same file `:19-23` | `VERSION_HEADER_KEYS=[x-cpa-version, x-server-version]`, build-date and plugin keys | Core MUST STAY. `x-server-*` fallback aliases SAFE-TO-NARROW to `x-cpa-*` only. |
| 1.7 | Header events | Fork `src/services/api/client.ts:68-117,140-163`, `src/stores/useAuthStore.ts:219-229,270-278` | Reads headers, dispatches `server-version-update` + `server-plugin-support-update` with `emitEvents` revision gate | MUST STAY (version display, plugin gating). Do not revert to Bumpy `client.ts:128-141` (no gate). |
| 1.8 | 404 capability gates | Fork `src/pages/AuthFilesOAuthExcludedEditPage.tsx:186,244`, `AuthFilesOAuthModelAliasEditPage.tsx:205,262`, `src/utils/quota/errors.ts:11-15`, `src/router/MainRoutes.tsx:18,53` | 404 → upgrade/unsupported message or plugin route gate | MUST STAY. These gate newer-v8 endpoints, not separate-client glue. Same in Bumpy. |
| 1.9 | Normalize helper | Fork `src/utils/connection.ts:3-12` | `normalizeApiBase`: trim, strip `/`, strip `/v8/management`, add `http://` | SAFE-TO-SIMPLIFY when base is fixed (keep for plugin-asset compat). |
| 1.10 | Base detection | Fork `src/utils/connection.ts:20-29`, `src/utils/constants.ts:16-17` | `detectApiBaseFromLocation()` from `window.location`, `DEFAULT_API_PORT=8317` fallback | Pure separate-client glue. SAFE-TO-DELETE, simplify to same-origin. Prefix `/v8/management` MUST STAY. |
| 1.11 | Subpath sniffing | Bumpy `src/utils/connection.ts:20-43`, `tests/connectionSubpath.test.ts:2-24` | `resolvePanelBasePath()` reverse-proxy mount detection | Already deleted in fork. Do NOT reintroduce. |
| 1.12 | Custom-base UI | Fork `src/pages/LoginPage.tsx:107-118,187,209`, `src/pages/login/LoginForm.tsx:7-12,61-102` | `detectedBase` memo, `showCustomBase` toggle, custom URL input | SAFE-TO-DELETE for same-origin target. Key input + remember MUST STAY. |
| 1.13 | Legacy base migration | Fork `src/stores/useAuthStore.ts:59-79,101-108` | Falls back to stored `apiBase`/`apiUrl` keys | SAFE-TO-DELETE after rollout. |
| 1.14 | Scoped storage | Bumpy `src/utils/constants.ts:33-35`, `src/services/storage/secureStorage.ts:121-250`, `src/stores/useAuthStore.ts:149-174` | Per-panel-mount + per-apiBase auth scoping | Already deleted in fork (`secureStorage.ts:1-75` plain, single `isLoggedIn`). KEEP fork shape. |
| 1.15 | CORS text | Fork `src/pages/LoginPage.tsx:87-90`, `src/i18n/locales/en.json:92` (+6 locales) | `cors/cross-origin` → `error_cors` | SAFE-TO-DELETE if same-origin-only; else KEEP. |
| 1.16 | Custom-connection i18n | Fork `src/i18n/locales/en.json:68-74` (+6 locales) | `connection_current/edit/done/auto_hint`, `custom_connection_*` | Delete with custom-base UI. |
| 1.17 | Must-stay guards | Fork `src/services/api/client.ts:23,42,55-66`, `src/components/providers/hooks/useProviderRecentRequests.ts:33-50`, `src/features/plugins/pluginResources.ts:27-34` | `connectionRevision` ABA guard, per-scope request cache, `resolvePluginAssetURL` | MUST STAY. Not decoupling. |

Safe deletion summary: 1.3 probe branch, 1.4-1.5 with probe, 1.10 detection, 1.12 custom-base UI, 1.13 legacy keys, 1.15-1.16 strings, narrow 1.6 aliases. Must stay: key login, 401 logout, revision guards, ETag writes, plugin-asset resolver, all 404 upgrade gates.

### Task 2: shortcut candidates (ranked)

All proposals are additive under `/v8/management`. Backend cites are `internal/api/handlers/management/`.

| Rank | Screen / waste | Current calls (file:line) | Browser computes | Proposed endpoint | Gain | Risk |
|------|----------------|---------------------------|------------------|-------------------|------|------|
| 1 | Quota per-credential fan-out via `POST /requests/api-call` | `useQuotaBatchLoader.ts:53-72` fan-out; per-cred multiplicity: claude `claude/data.ts:215-228`, codex `codex/data.ts:430-438,453`, xai `xai/data.ts:105-133,177-186,216-219`, kimi `kimi/data.ts:45-62`, antigravity `antigravity/data.ts:129-193`, opencodeGo `opencodeGo/data.ts:124-136` | Window/percent/reset/timeline math per credential, thrown away | `POST /v8/management/credentials/quota/batch` `{items:[{auth_index}]}` → `{results:[{auth_index, windows, plan, error}]}` partial-success envelope | N creds × 1-3 upstreams → 1 request; server reuses keepalive + token refresh | HIGH: upstream diversity, timeouts, PII (never log bodies); keep `api-call` fallback |
| 2 | Dashboard 4-way fan-out | `useDashboardOverview.ts:153-168` `Promise.allSettled([fetchConfig(true), loadAuthFiles(), loadModels(), refreshRecentRequests()])` | Dedup api-key-usage vs auth-files, accumulators, buckets `:172-284` | `GET /v8/management/views/dashboard-overview` → `{config_counts, credentials_health, traffic, models_count, observed_at}` | 4 → 1 per open/refresh | LOW-MED: define model-count semantics; return rollup not full files |
| 3 | Traffic duplicate polls | `useProviderRecentRequests.ts:173-178` 240 s poll → `GET /observability/usage/api-keys`; consumed by dashboard + `ProvidersWorkbenchPage.tsx:113` | `normalizeRecentRequestUsageEntry` + bucket merge per screen | `GET /v8/management/views/traffic-overview` pre-merged totals/by-provider/buckets | Kills duplicate polls + client bucket math; core of #2, can ship first | LOW-MED: bucket schema must stay stable |
| 4 | Providers workbench 3-way | `useProviderWorkbench.ts:282-286` `Promise.allSettled([fetchConfig(true), getVertexConfigs(), getOpenAIProviders()])`, merge `:290-324` | Vertex/compat adapters, group assembly | `GET /v8/management/views/providers-overview` → `{groups, counts, fetched_at}` | 3 → 1 per open/refetch | LOW: read-only assembly |
| 5 | Auth-files triple + model fan-out + 240 s poll | `AuthFilesPage.tsx:368-387` `Promise.all([loadFiles, loadExcluded(), loadModelAlias()])`; `useAuthFilesOauth.tsx:100-109` per-provider `getModelDefinitions`; edit pages `AuthFilesOAuthExcludedEditPage.tsx:158-161` triple | Almost nothing, pure round trips | `GET /v8/management/views/auth-files-overview?include=files,excluded,alias` (+ optional models map) | 3(+P) → 1 | LOW-MED: preserve 404/unsupported semantics; models opt-in (payload size) |
| 6 | Model discovery Gemini ≤20-page loop | `models.ts:286-334` paged loop + strip/dedupe; callers `useConnectivityTest.ts:289`, `useModelDiscovery.ts:95`; `models.ts:111-250` api-call wrappers | Endpoint/header stitching, pagination, dedupe | `GET /v8/management/views/available-models?provider=&auth_index=` server paginates | N keys × pages → 1 | MED: must match `api-call` credential/proxy semantics |
| 7 | Logs 1 s poll | `useLogStream.ts:252-262` `setInterval(1000)` → `loadLogs(true)` ≈ 60 req/min/viewer | Parse/filter/format | `GET /v8/management/observability/logs/stream` (SSE, cursor reuse) or cheaper `views/logs-batch` | Stream → 1 conn; batch kills extra error-log + config reads | MED: connection management; keep polling fallback |
| 8 | Plugin releases via api-call | `pluginReleaseVersions.ts:76-104` browser→server→github hairpin per dropdown | Slug parse, release normalize | `GET /v8/management/plugins/store/releases?repository=` server fetch + cache | Removes hairpin; central rate-limit | LOW: needs host allowlist + TTL |
| 9 | Global header 4-way | `MainLayout.tsx:738-745` `Promise.allSettled([fetchConfig(true), loadPluginResources(), loadAuthFilesCount(), triggerHeaderRefresh()])` | Merge for header | Reuse #2 response (or `views/shell-status` if header needs less) | 4 → 1 | LOW |

Backend availability (verified): `APICall` + `authByIndex` + `resolveTokenForAuth` with antigravity/meta/xai refresh (`api_tools.go:273-294,819-869`); `ResetQuota` (`quota.go:28-70`); `GetConfig` (`config_basic.go:26`); `ListAuthFiles` embeds quota/cooldowns/recent (`auth_files.go:106-177,654-693`); `GetAPIKeyUsage` (`api_key_usage.go:58-117`); cursor-incremental `GetLogs` (`logs.go:40-132,647-681,759-828`); `GetLatestVersion` server-side GitHub precedent (`config_basic.go:48-80`); v8 route table (`server_management_v8.go:21-70`). Build order proposal: #3 → #2 → #4 + #5 → #6 → #1 (flagged, per-provider) → #7, #8. Cross-cutting: `fetchConfig(true)` force-refresh bypasses TTL in dashboard/workbench/header callers; every `views/*` response carries `observed_at`/etag so panel stops force-refreshing config alongside.

### Task 3: contract sharing

| What | File:line | State |
|------|-----------|-------|
| Go handlers emit untyped `gin.H` maps, anonymous bind structs | `auth_files.go:176,221-230,672-686`, `quota.go:34-36,65-69`, `handler.go:481-510`, `plugin_quota.go:25-28`, `usage.go:24-43` | No DTO library, no schema |
| Rare typed struct, no annotations | `api_key_usage.go:12-16,58-116` | Typed but unspeced |
| Routes manual Gin, no spec generation | `server_management.go:18,209-239`, `server_management_v8.go:21-70` | No OpenAPI/Swagger |
| Spec/codegen deps | `go.mod:1-124`, `apps/web/package.json:17-49`, Bumpy `package.json:7-19` | None (`swaggo`, `oapi-codegen`, `orval`, `hey-api` all absent; verified) |
| TS side manual interfaces + hand normalizers | `apps/web/src/types/authFile.ts:23-92`, `src/types/config.ts:12-36`, `src/services/api/transformers.ts:339-386`, `src/services/api/authFiles.ts:268-360` | Normalizers bridge snake/kebab/camel drift |

Cost: type ~25-35 handler files to named DTOs first, then add Go + TS generator toolchains, CI drift gate, and `oneOf`/optional-map modeling for 30+ conditional auth-file keys. Proposal: LATER, not now. Drift risk today is field naming and optional presence, which codegen does not fix without the DTO refactor. Cheapest step now: extend `apps/web/tests/` normalizer contract tests.

### Task 4: embedded delivery

| # | Item | File:line | State | Proposal |
|---|------|-----------|-------|----------|
| 4.1 | Embed `dist/management.html` | `apps/web/panel.go:8-14`, `internal/managementasset/updater.go:21-25` | Core of embedded panel | MUST STAY |
| 4.2 | `/management.html` route | `internal/api/server_routes.go:57`, `server_management.go:317-340` (404 when `cfg==nil \|\| Home.Enabled \|\| DisableControlPanel`; `MANAGEMENT_STATIC_PATH` override wins; else embedded bytes) | Only serving logic | MUST STAY |
| 4.3 | `MANAGEMENT_STATIC_PATH` override | `updater.go:29-35,59-71` | Dev loop only (`README.md:67`, `check-panel.sh:25`, `server_test.go:1845-1898`) | KEEP (no runtime weight) |
| 4.4 | `StaticDir` non-override fallback | `updater.go:37-54` (`util.WritablePath()/static` else `<config-dir>/static`) | Unreachable from serve path (called only inside override branch) | Dead weight, safe to delete fallback; keep override branch |
| 4.5 | Auto-update / download | `updater.go:1-6` tombstone comment; grep for `StartPanel\|PanelSync\|DownloadPanel\|UpdatePanel` in `internal/` returns zero; `server.go:73-74,121-222` has no sync hook | Already gone | Nothing to remove |
| 4.6 | `allow-remote`, `secret-key`, `base-url`, `disable-control-panel` | `config_types.go:518-533` (`:520,:522,:531-532,:524`) | Live (auth gate, TUI `resolveManagementBaseURL`, serve gate `:319`) | MUST STAY |
| 4.7 | `disable-auto-update-panel` | `config_types.go:527`; comment `:525-526` claims updater enabled by default (STALE, no updater exists) | Dead runtime, **released config key** | Keep YAML loading, ignore value; fix stale comment |
| 4.8 | `panel-github-repository` | `config_types.go:530`; defaulting `parse.go:41,77-80`, `config_load.go:210,257-260`, `config_defaults.go:4`, pruning `config_yaml.go:483-484`, re-export `sdk/config/config.go:49` | Dead runtime, **released config key** | Keep loading, ignore; removal list = default constant + defaulting/pruning once plain-field compat shim is accepted |
| 4.9 | Watcher diff lines | `config_diff.go:536-550` (`:539-545` dead keys, `:537` live, `:547-550` base-url) | Cosmetic only | `:539-545` safe to delete; harmless to keep |
| 4.10 | Frontend dead-key UI | `useVisualConfig.ts:1450-1454,1709-1724`, `SectionConnectivity.tsx:139-154`, `searchIndex.ts:352-373`, `constants.ts:149-150`, `visualConfig.ts:165-166,268-269`, `disable_auto_update_panel*` i18n (7 locales), `check-panel.sh:19-20` stale fixture, `visualConfig-v8.yaml:12` fixture | Panel editors for dead keys | Dead UI weight; remove after backend ignores keys (frontend scope) |

Config compat is mandatory: 4.7 and 4.8 must keep loading and be ignored, never rejected.

### Risks

- Quota batch (#1) is the highest gain and highest risk: upstream timeout fan-in, token/PII handling, long-tail latency. Ship behind flag, per-provider rollout, partial-success envelope. Verify existing `credentials/quota/fetch` removal state first (`server_management_v8_test.go:119-123` asserts 404; route absent from `server_management_v8.go`).
- View endpoints (#2-#5, #9) risk payload bloat and overlapping shapes. Define one `views/` naming + `observed_at` convention first; do not create two overlapping views (#9 reuses #2).
- Logs stream (#7) adds connection/backpressure state. Keep polling fallback.
- Deleting custom-base UI removes the only remote-debug path. Keep the key login and 401 handling untouched.

### Claims not confirmed

- Exact per-provider upstream call counts on the quota page at runtime (needs a running server with real credentials; static code shows 1-3 calls per credential, Gemini up to 20 pages).
- Whether `POST /v8/management/credentials/quota/fetch` handler code still exists behind the removed route (route asserts 404; handler presence not checked).
- Bumpy `develop` branch state: all Bumpy cites are from the pinned reference clone; newer commits not checked.
