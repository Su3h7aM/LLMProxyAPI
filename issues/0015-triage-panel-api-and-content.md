---
status: done
branch: issue/0015-triage-panel-api-and-content
type: triage (read-only, no code changes)
agent: muse-spark-1.3-contributor
---

# Triage panel migration A: backend API compatibility and content audit

## Context

The fork dashboard (`apps/web`, vendored from the official Management Center,
pin `953fb24`) will be replaced by the BumpyClock Management Center. Read
`DECISIONS.md` (entry 2026-11-11 about the dashboard) and `UPSTREAM_BASELINE.md`.
The fork serves the panel from the proxy binary (`apps/web/panel.go` embeds
`dist/management.html`). The fork targets servers and containers, API only.

References (read-only, never edit or commit there):

- BumpyClock panel: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center`
- Upstream proxy: `/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI`
- Fork: your checkout. The official panel at the vendored pin is the baseline of `apps/web`.

## Goal

Find out what in the BumpyClock panel works against the fork backend, and what does not.

## Tasks

1. **Endpoint map.** List every HTTP call of the BumpyClock panel (all of `src/services/api/*`, plus any other `axios`/`fetch` use in `src`): method, path, file:line. For each, check the fork backend routes (`internal/api/server.go` and `internal/api/handlers/management/`, versions `/v0/management` and `/v8/management`): SERVED / MISSING / DIFFERENT (request or response shape differs; say how). Compare also with the official panel that the fork vendors (`apps/web/src/services/api`): which endpoints are new in BumpyClock, which were dropped.
2. **Dependent features.** For each MISSING or DIFFERENT endpoint list the pages, components, and stores that depend on it (for example `copilot.ts`, `kimiQuota.ts`, `devinQuota.ts`, `metaQuota.ts`, `antigravitySubscription.ts`, `plugins.ts`, `authFileCooldowns.ts`, `apiKeyUsage.ts`). Say if the feature looks CLIProxyAPIPlus-only (not in upstream CLIProxyAPI) or is in upstream and the fork backend already serves it.
3. **Sponsor and promo content** in the BumpyClock repo: files, assets, i18n keys, components, README. Give a removal list.
4. **Connection model.** Describe how the BumpyClock panel finds and connects to the backend: login page, API base detection, storage keys, management key handling, remote/other server connect flow, auto-login, version checks (`version.ts`, latest release check, minimum backend version). Then do the same for the fork panel (`apps/web`, `src/stores/useAuthStore.ts`, `src/utils/connection`). The target is same-origin: no API base URL setting and no remote server connect flow; management key login stays. List the BumpyClock files that must change for that, and what is the smallest change.
5. **Plus lineage.** The panel comes from a fork made for CLIProxyAPIPlus (`.cpamc-fork-upstream.env`). List code that clearly exists only for CLIProxyAPIPlus (provider names, endpoints, strings) so that the coordinator can decide.

## Result format

Tables per task (task 1 as one table: method, path, file:line, status, note). End with: proposed list of BumpyClock features to keep, hide, or remove (with the reason), risks, and a list of claims you could not confirm.


## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence and a proposal. The coordinator decides.
- Back each claim with a file path and line, a command, or its output. Mark each claim you could not confirm and say where you looked.
- Environment: use `/usr/bin/go`, not the `go` shim (mise config is untrusted; do NOT run `mise trust`). Use bun at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin/bun`. Do builds and installs only in a scratch copy under `/tmp` (for example `cp -r` the panel there). Do not leave `node_modules` or `dist` in your checkout.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0015-triage-panel-api-and-content` in your checkout (rename your branch if T3 gave another name). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often (Conventional Commits).
3. When done: set `status: done`, fill `## Result`, commit. Do NOT merge to main. Do not create other issue files.
4. Report to the coordinator: short summary, branch, commit range, key findings, questions, surprises.

## Log

- 2026-11-11: Set `status: in-progress`. Read `DECISIONS.md`, `UPSTREAM_BASELINE.md`.
- 2026-11-11: Ran 4 parallel workers: Bumpy endpoint pull, fork route pull, sponsor/lineage/connection pull, fork panel connection pull. Checked each claim with `grep` on both checkouts.
- 2026-11-11: Finding entries written below. Set `status: done`. No code changed.

## Follow-ups

## Result

### Task 1: endpoint map (Bumpy panel vs fork backend)

Prefix note: Bumpy uses `/v0/management` (`utils/constants.ts:17`,
`utils/connection.ts:14-18`). Fork panel uses `/v8/management`
(`apps/web/src/utils/constants.ts:17`). Fork backend still serves v0
(`internal/api/server_management.go`), so the table rates v0 first, then
the v8 target.

| Method | Path | Bumpy file:line | v0 | v8 target |
| --- | --- | --- | --- | --- |
| POST | `/api-call` | `apiCall.ts:80` | SERVED (`api_tools.go:104`) | DIFFERENT: `POST /requests/api-call` |
| POST form | `/vertex/import` | `vertex.ts:23` | SERVED (`server_management.go:194`) | DIFFERENT: `POST /oauth/import?provider=vertex` |
| GET | `/latest-version` | `version.ts:8` | SERVED (`config_basic.go:48`) | DIFFERENT: `GET /server/latest-version` |
| GET | `/copilot-quota?auth_index=` | `copilot.ts:88` | MISSING (no handler) | MISSING |
| GET | `/usage` | `usage.ts:14` | MISSING (only `/usage-queue`, `/usage-statistics-enabled`, `/api-key-usage`) | MISSING |
| GET | `/usage-statistics-enabled` | `usage.ts:21` | SERVED (`server_management.go:64-66`) | MISSING (no v8 route) |
| GET | `/api-key-usage` | `apiKeyUsage.ts:8` | SERVED (`api_key_usage.go:58`) | DIFFERENT: `GET /observability/usage/api-keys` |
| GET/PUT/PATCH/DELETE | `/api-keys` | `apiKeys.ts:9-18` | SERVED (`config_lists.go:158-167`) | MISSING (legacy; `server_management_v8_test.go:115`) |
| GET | `/config` | `config.ts:14`, `providers.ts:218` | SERVED (`config_basic.go:26`) | SERVED (same path, ETag writes) |
| PUT | `/request-log` | `config.ts:21` | SERVED (`server_management.go:110-112`) | MISSING direct (use `PUT /config/<tree-path>`) |
| GET raw / PUT | `/config.yaml` | `configFile.ts:9,20` | SERVED (`config_basic.go:129,185`) | SERVED (same path) |
| GET / DELETE | `/logs` | `logs.ts:67,71` | SERVED (`logs.go:40,135`) | DIFFERENT: `/observability/logs` |
| GET / GET raw | `/request-error-logs[/:name]` | `logs.ts:74,77` | SERVED (`logs.go:197,430`) | DIFFERENT: `/observability/logs/errors[/:name]` |
| GET raw | `/request-log-by-id/:id` | `logs.ts:83` | SERVED (`logs.go:261`) | DIFFERENT: `/observability/logs/requests/:id` |
| PUT | `/${section}` key lists | `providers.ts:220` | SERVED (per-family routes) | MISSING (fork uses `PUT /config/api-keys/<family>`, `providers.ts:277`) |
| DELETE | `/gemini\|interactions\|codex\|meta\|xai\|claude\|vertex-api-key` | `providers.ts:454-590` | SERVED | MISSING (same tree-config shift) |
| GET | `/meta-api-key`, `/vertex-api-key` | `providers.ts:497,565` | SERVED | MISSING |
| GET/PATCH/DELETE | `/openai-compatibility` | `providers.ts:593-623` | SERVED (`config_lists.go:866-1007`) | MISSING |
| GET | `/{codex,anthropic,antigravity,kimi,kimi-ai,xai,devin,meta}-auth-url` | `oauth.ts:56` | SERVED (`server_management.go:196-203`) | DIFFERENT: `GET/POST /oauth/auth-url?provider=` |
| GET | `/qoder-auth-url` | `oauth.ts:56` (`qoder` in union) | MISSING (no route; upstream also lacks) | MISSING |
| GET | `/get-auth-status?state=` | `oauth.ts:63` | SERVED (`server_management.go:204`) | DIFFERENT: `GET /oauth/status` |
| DELETE | `/oauth-session?state=` | `oauth.ts:69` | SERVED (`server_management.go:205`) | DIFFERENT: `DELETE /oauth/session` |
| POST | `/oauth-callback` | `oauth.ts:76` | SERVED (`server_management.go:25-26`) | DIFFERENT: `POST /oauth/callback` |
| GET/PATCH/DELETE | `/plugins`, `/plugins/:id/enabled`, `/plugins/:id` | `plugins.ts:316-324` | SERVED (`plugins.go:76,228,334`) | GET/DELETE served; `enabled` MISSING (fork uses `PUT /config/plugins/configs/:id/enabled`) |
| GET/PATCH | `/plugins/:id/config` | `plugins.ts:329-334` | SERVED (`plugins.go:174,292`) | MISSING (fork uses `GET/PUT /config/plugins/configs/:id`) |
| GET/POST | `/plugin-store`, `/plugin-store/:id/install` | `plugins.ts:339,354` | SERVED (`plugin_store.go:121,231`) | DIFFERENT: `/plugins/store[/:id/install]` |
| GET/POST form/DELETE | `/auth-files` (+`?all=true`, download) | `authFiles.ts:445-497` | SERVED | DIFFERENT: `/credentials[/download]` |
| PATCH | `/auth-files/status`, `/auth-files/fields` | `authFiles.ts:452-455` | SERVED (`auth_files_fields.go:30,263`) | DIFFERENT: `/credentials/status`, `/credentials/fields` |
| POST | `/auth-files/refresh` | `authFiles.ts:459` | SERVED (`auth_files_refresh.go:15`) | DIFFERENT: `POST /credentials/refresh` |
| GET/PUT/PATCH/DELETE | `/oauth-excluded-models` | `authFiles.ts:512-528` | SERVED (`config_lists.go:1224-1291`) | MISSING direct (fork uses `PUT /config/oauth/excluded-models`) |
| GET/PATCH/PUT/DELETE | `/oauth-model-alias` | `authFiles.ts:532-559` | SERVED (`config_lists.go:1315-1391`) | MISSING direct (fork uses `/config/oauth/model-alias`) |
| GET | `/auth-files/models` | `authFiles.ts:567` | SERVED (`auth_files.go:336`) | SERVED as `GET /credentials/models` |
| GET | `/model-definitions/:channel` | `authFiles.ts:582` | SERVED (`model_definitions.go:63`) | DIFFERENT: `GET /routing/model-definitions/:channel` |
| POST | `/quota/fetch` | `quota/providers/plugin/data.ts:91` | SERVED (`plugin_quota.go:98`) | MISSING (`v8_test.go:119-123` asserts 404) |
| direct axios / via `/api-call` | external `{base}/v1/models` etc | `models.ts:104-281` | N/A external; via call SERVED | N/A; via call DIFFERENT path |

Bumpy never calls `/reset-quota`, `/quota/reset`, `/quota/providers`
(no hit in `src`). Pure parsers with no HTTP:
`antigravitySubscription.ts`, `claudeUsage.ts`, `kimiQuota.ts`,
`devinQuota.ts`, `metaQuota.ts`, `authFileCooldowns.ts`, `apiError.ts`,
`transformers.ts`.

Fork panel file delta (`src/services/api/`): fork-only (7):
`authFilePolicyAdapter.ts`, `claudeResetGrants.ts`, `configPatch.ts`,
`configValue.ts`, `legacyBackendProbe.ts`, `providerModels.ts`,
`routing.ts`. Bumpy-only (2): `copilot.ts`, `usage.ts`. Index diff
confirms: fork drops `copilot`/`usage` exports, adds the 7 above.

### Task 2: dependent features for MISSING / DIFFERENT endpoints

- `GET /copilot-quota` MISSING: `quota/providers/copilot/data.ts:15`
  (`copilotApi.getQuota`), `CopilotQuotaBody.tsx`, `QuotaPage.tsx`,
  `constants.ts`, `registry.ts`, `types.ts`, `useQuotaStore.ts`,
  `authFiles/constants.ts`, `providerKeys.ts`, `quota/validators.ts`,
  `usageStats.ts`. Plus-only: upstream `.refs/CLIProxyAPI` has no
  `copilot-quota` route and fork backend has no Copilot provider. Hide or
  remove unless backend adds it.
- `GET /usage` MISSING: `UsagePage.tsx`,
  `dashboard/hooks/useDashboardOverview.ts`,
  `SponsorProviderForm.tsx`. Plus-only page data (upstream has no
  `/usage` management route). Usage page cannot work as-is; rewrite on
  `/observability/usage/*` or remove.
- `GET /qoder-auth-url` MISSING: `services/api/oauth.ts` type union,
  `quota/providers/qoder/*` (reads only auth-file snapshot, no HTTP),
  `OAuthPage.tsx`, `SystemPage.tsx`, `types/authFile.ts`,
  `assets/icons/qoder.svg`. OAuth start for qoder fails; quota snapshot
  still works. Plus-only provider flow.
- `GET /usage-statistics-enabled` (v8 MISSING): `usage.ts` flag reader
  has 404-null fallback already; keep as v0-tolerant probe or move to
  `GET /config` tree read.
- Kimi/devin/meta/antigravity quota parsers: fetch via `POST /api-call`
  (`kimi/data.ts:49`, `devin/requests.ts:2-9`, `meta/requests.ts:6`).
  SERVED on v0, path change on v8. Keep with path remap.
- Plugin quota tab (`POST /quota/fetch`, `plugin/data.ts:91`): SERVED v0,
  MISSING v8. Keep on v0; needs new v8 source or stays v0-only.
- Key-list editors (`providers.ts` PUT/DELETE per family): SERVED v0,
  MISSING v8 (fork uses `/config/api-keys/<family>`). Works on v0;
  v8 needs rewrite to tree config.
- Plugin config/enabled pages: SERVED v0, MISSING v8 (fork uses
  `/config/plugins/configs/:id`). Same: works on v0, rewrite for v8.
- OAuth per-provider URLs: SERVED v0 except qoder; v8 needs
  `/oauth/auth-url?provider=` remap. `OAuthPage.tsx` affected.
- Vertex import: SERVED v0; v8 remap to `/oauth/import?provider=vertex`.

### Task 3: sponsor and promo removal list

- `README.md:21-27`: `## Sponsor` APIMart block; assets
  `assets/apimart-en.png`, `assets/apimart-zh.png`; link
  `https://go.apimart.ai/gh-cli-proxy-api-management-center`.
- `src/features/config/sponsors.ts:1-20`: `SPONSORS` (BestProxy.com,
  `bestproxy.png`, `?keyword=ayh7otlb`); shown by
  `src/pages/SystemPage.tsx:99-148` (`OptionalResources`, sponsored
  `rel`, key `system_info.resources_sponsored` `en.json:1853`).
  Style: `SystemPage.module.scss:211,227`. Sidebar promo import:
  `MainLayout.tsx:56` (`hasApiKeyFunConfig`).
- Sponsor provider modules: `features/providers/sponsor.ts:4-14`
  (apikey.fan URLs, `aff=AKCPA`), `fennoAI.ts:4-6`
  (`aff=DQFAMNB6CBLY`), `qiniuCloud.ts:4-8`, `kimi.ts:4-13`
  (`aff=cliproxyapi`); registry `sponsorDefinitions.ts:76-135`;
  components `SponsorQuickStartPanel.tsx:8-42`,
  `sheets/forms/SponsorProviderForm.tsx`, `sponsorMutationRecovery.ts`,
  `useProviderWorkbench.ts:233-445`.
- i18n: `providersPage.sponsor.*` block (`en.json:1964-2034`, incl.
  `kimiPromo` `en.json:1969`), mirrored in `zh-CN.json:1944-2014`,
  `zh-TW.json:1970-2040`, `ru.json:1995-2065`.
- Tests that pin promos (update or drop with them):
  `tests/systemOptionalResources.test.ts:18-35`,
  `tests/configPresentation.test.ts:50-63`,
  `tests/providerQuickStartSponsor.test.ts:18-46`,
  `tests/sponsorAggregation.test.ts`, `tests/sponsorCustomEndpoint.test.ts`,
  `tests/sponsorMutationRecovery.test.ts`, `tests/kimiProvider.test.ts:55-64`,
  `tests/fennoProvider.test.ts:22`, `tests/round1ProvidersQuickStart.test.ts:45-56`.

### Task 4: connection model

Bumpy:
- Login: `src/pages/LoginPage.tsx:89-336`. Auto-detect readout
  (`detectApiBaseFromLocation`, `:113,247-251`), optional custom base
  checkbox (`:253-271`), key input `name="cpa-management-key"` (`:278`),
  remember-password (`:309-317`, reversible obfuscation only).
  Auto-login via `restoreSession()` (`:132-156`), 1500 ms splash.
- Base: `src/utils/connection.ts:1-58` (`normalizeApiBase`,
  `computeApiUrl = base + /v0/management`, `DEFAULT_API_PORT=8317`,
  `constants.ts:16-17`). Client `services/api/client.ts:19-150`
  (singleton, Bearer key `:105-118`, CPA version headers `:121-149`,
  401 event `:168-170`). Remote flow = type custom address
  (`LoginPage.tsx:263-271`) or open panel from that host; docs
  `README.md:67-89` (needs `allow-remote-management: true`). No server list.
- Store: `stores/useAuthStore.ts:1-249` (zustand persist,
  `STORAGE_KEY_AUTH='cli-proxy-auth'` + `:scope:`/`:selection:`/`:logged-in:`
  per panel-base+api-base, legacy plain keys migrated,
  `secureStorage.ts:1-250` `enc::v1::`). Key stored only if remember.
- Versions: passive `x-cpa-version` headers to System tiles
  (`SystemPage.tsx:199-212,417-454`); manual latest check
  `version.ts:7-8` + `SystemPage.tsx:336-367` (`compareVersions`
  `:83-95`). No minimum-version gate (floor `>= 7.2.147` docs-only,
  `README.md:17`).

Fork (`apps/web`): same shape, v8-first.
- `pages/LoginPage.tsx:107-239` + `pages/login/LoginForm.tsx:59-102`
  (read-only readout + Change toggle, no server list);
  `stores/useAuthStore.ts:39-278` (`enc::v2::`, no legacy migration,
  404 + `legacyBackendProbe.ts:1-30` gives `error_legacy_backend`);
  `utils/connection.ts:3-29` (strips `/v8/management`);
  `services/api/client.ts:39-163` (v8 base, revision guard, 401 logout);
  `vite.config.ts:39-77` (no proxy/base, single file);
  `panel.go:1-14` (embeds `dist/management.html`).
- Version: passive `x-cpa|x-server-version` headers; manual check
  `GET /server/latest-version` (`version.ts:8`,
  `SystemPage.tsx:243-274`); no numeric gate (route presence decides).

Same-origin change list (Bumpy files): `src/utils/constants.ts:16-17`
(prefix; drop `DEFAULT_API_PORT` use), `src/utils/connection.ts:1-58`
(drop custom-base normalize/detect-remote), `src/services/api/client.ts:19-150`
(fixed base, keep Bearer + 401), `src/stores/useAuthStore.ts:1-249` +
`src/services/storage/secureStorage.ts` (per-api-base scoping can go),
`src/pages/LoginPage.tsx:89-336` (remove custom-base branch `:253-271`,
keep key-only form), `src/services/api/version.ts:7-8` (v8 path), all
Table-1 v8 remaps, `README.md:67-89` connect docs. Smallest change: force
`computeApiUrl` to same-origin `/v8/management` (or `/v0/management`
first), delete the custom-base UI branch, remap Table-1 paths.

### Task 5: Plus lineage

- `.cpamc-fork-upstream.env:1-2`: `UPSTREAM_TAG=v1.24.2`,
  `UPSTREAM_COMMIT=4530da2`. `README.md:7-17`: maintained-fork for
  CLIProxyAPIPlus, `management.html` asset stream, `vX.Y.Z-N` tags
  (`README.md:143-149`), Upstream Sync workflow.
- No `isPlus` runtime branch found. CPA keys are compat, not Plus logic:
  `utils/constants.ts:19-23` (`x-cpa-version` etc),
  `services/api/client.ts:124-144`, CPAMC logos/alt
  (`LoginPage.tsx:209`, `SystemPage.tsx:413`, `MainLayout.tsx:961`),
  i18n `abbr: CPAMC` (`en.json:63`), upgrade strings
  (`en.json:863-865,913,962-964,1172-1174`).
- Clearly Plus-only runtime surface: `copilot-quota` endpoint + quota
  tab, `GET /usage` + Usage page, `qoder-auth-url` + qoder provider
  files (`quota/providers/qoder/*`, `assets/icons/qoder.svg`),
  sponsor provider set (task 3). Kimi/devin/meta/antigravity quota
  parsers are provider families, not Plus-only alone; fork backend
  serves their OAuth + `/api-call` paths.

### Proposal (coordinator decides): keep / hide / remove

- Keep (remap v0->v8 per Table 1): auth-files, OAuth (minus qoder),
  quota via `/api-call`, logs, config.yaml, api key usage, plugin
  list/store/install, version check, model definitions.
- Hide first: Copilot quota tab, Usage page, qoder provider + OAuth
  option. Reason: backend has no source (MISSING, upstream also lacks).
- Remove: sponsor block + sponsor providers + promo i18n + APIMart
  README/assets (task 3 list). Keep `kimi` non-affiliate provider
  definition if fork serves kimi (it does: `kimi-auth-url`); strip only
  affiliate URLs/promo strings.
- Fork extras to keep on top: OpenCode Go quota tab,
  `authFilePolicyAdapter`, `claudeResetGrants`, `configPatch/configValue`,
  `providerModels`, `routing`, soonest-reset sort, fork release check.
- Risks: v8 gaps (key-list CRUD, plugin config/enabled, `/quota/fetch`,
  `usage-statistics-enabled`, per-provider auth URLs) force either a v0
  compat window or rewrites to tree-config paths before those pages can
  move. Flat routes `PUT /request-log` etc have no direct v8 twin.

### Unconfirmed claims

- Neither panel was run (no build; builds only in `/tmp` scratch per
  rules; not needed for static map). Dynamic `PUT /${section}` union is
  read from `providers.ts:220` code, not runtime.
- `qoder` backend absence rests on route-file `grep`; a plugin could
  serve it via `pluginManagementNoRoute`
  (`server_management.go:260-296`) — only static routes were checked.
- Sponsor test list rests on file names + sampled asserts, not full runs.
- "No minimum-version gate" rests on `grep` for minimum/gate in `src`;
  docs-only floor per `README.md:17`.
