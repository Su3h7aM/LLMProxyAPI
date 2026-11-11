---
status: done
branch: issue/0035-triage-panel-parity
type: triage (read-only, no code changes)
agent: deepseek-v4.1-flash
---

# Triage panel parity: pre-import panel vs integration branch

## Goal

Find what the old fork panel could do that the new panel (branch `integration/panel-bumpyclock`) cannot, and find calls without a backend route. This is read-only. Other agents still work on OpenCode Go, logic fixes, cleanup, the routing indicator, and CI: list their areas as "known, in progress" and do not count them as gaps.

## References (read-only)

- Old panel: tag `panel-pre-bumpyclock` (`git show panel-pre-bumpyclock:apps/web/<path>`, `git ls-tree -r panel-pre-bumpyclock --name-only apps/web`).
- New panel: branch `integration/panel-bumpyclock` (your checkout has `main`; read with `git show integration/panel-bumpyclock:apps/web/<path>` or create a scratch worktree under `/tmp`).
- Backend: Go code in this repo (`internal/api/server_management_v8.go`, `internal/api/handlers/management/`).
- Read `DECISIONS.md` (entries of 2026-11-11) and `issues/0015-*.md` to `issues/0029-*.md` (what was done and what is known).
- Known and decided (not gaps): sponsor content removed; Copilot and Qoder pages removed; Usage page and dashboard usage section to be hidden (no backend data); locales ko, tr, vi later; v0 and legacy support removed.

## Tasks

1. **Feature list.** For the old panel list every user-visible page, tab, action, setting, editor, dialog, and filter (use the router, nav items, page components, and locale keys). For the new panel do the same. Build a table: feature, old location, new location or MISSING, status (OK, MISSING, CHANGED with note, KNOWN-IN-PROGRESS with the issue number).
2. **Config coverage.** Compare the settings that the visual config page can edit in both panels (sections and fields). Compare with the fields of the Go config types (`internal/config/config_types.go`, `config_v8.go`): list Go settings that neither panel edits, and panel fields with no Go field.
3. **Route check.** Extract every API path that the new panel calls (static and template strings in `apps/web/src`; use a script in `/tmp`, do not commit it) and match it with the route table in `internal/api/server_management_v8.go` (and handlers for sub-paths). Table: path, method, caller file:line, route found (yes or no). Also list v8 routes that the old panel called and the new panel does not.
4. **Locale coverage.** For the 4 locales of the new panel: keys used in code (`t('...')`) but missing in a locale; keys with empty or copied English text in the zh and ru files for fork-only features.
5. **Quality checks.** Run in a scratch copy under `/tmp`: `bun install --frozen-lockfile`, `bun run verify`, `bun run tokens:check`; report results and the bundle size.

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence (file and line, command and output) and a proposal. Mark each claim you could not confirm and say where you looked.
- Use bun from `/home/su3h7am/.local/share/mise/installs/bun/latest/bin` and `/usr/bin/go`. Do NOT run `mise trust`. Do builds only in `/tmp`.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0035-triage-panel-parity` in your checkout (from `main`). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. When done: `status: done`, fill `## Result` (tables, proposed fix list ranked by user impact, claims you could not confirm), commit. Do NOT merge. Report: short summary, branch, commit range, key findings.

## Log

- Setup: worktree `/tmp/t0035-new` (branch `integration/panel-bumpyclock`,
  head `db784257`), old panel exported from tag `panel-pre-bumpyclock` to
  `/tmp/t0035-old`. Route extractors `/tmp/t0035-routes.js` and
  `/tmp/t0035-calls.js`; locale checker `/tmp/t0035-locale3.py`; Go v8 path
  dump in `/tmp/t0035-gorepo/cmd/dumpcfg` (copy of the repo: reflection over
  `config.Config` plus the `buildV8Paths` prefix table). No repository file
  changed except this issue.
- Routes: every management call of the new panel has a v8 route. No `/v0`
  path remains in `apps/web/src`.
- Config: the new panel field set is a superset of the old one (+3 fields:
  `codexIdentityConfuse`, `quotaSwitchProject`, `quotaSwitchPreviewModel`).
- Quality: `bun install --frozen-lockfile`, `bun run verify`,
  `bun run tokens:check` pass on the integration branch.

## Follow-ups

- Add a test that fails on a used `t('key')` with no key in the 4 locales.
  Current guard (`tests/productionLocales.test.ts`) checks a fixed list of
  sections only.
- Remove dead locale keys that no code uses: `auth_login.qwen_*` and
  `auth_login.iflow_cookie_*` (zh-TW only).
- Decide the status of the small UI differences in the quota bodies (xAI
  prepaid block, Claude stale badge, Meta window hours, Codex Pro Max label).

## Result

Headline: routes and config fields have parity. Four user-visible gaps
remain: 5 raw locale keys, no log-level filter, no API key names, and a few
cosmetic dashboard and OAuth differences.

New panel read at `integration/panel-bumpyclock` head `db784257`. Old panel
read at tag `panel-pre-bumpyclock`.

### Task 1 — feature list

Legend: OK, MISSING, CHANGED, KNOWN (known and in progress).

| Feature | Old location | New location | Status |
|---|---|---|---|
| Routes (`/`, `/dashboard`, `/ai-providers`, `/auth-files`, `/oauth`, `/routing`, `/quota`, `/config`, `/logs`, `/system`, `/plugins`, `/plugin-store`, `/plugin-pages/:id/:i`) | `src/router/MainRoutes.tsx` | same file | OK |
| `/usage` route | absent | `src/pages/UsagePage.tsx`, nav item at `MainLayout.tsx:435` | KNOWN 0027 (hide, no backend data) |
| Nav groups and order | `MainLayout.tsx:594-681` | `MainLayout.tsx:382-471` | OK |
| Login: key only, remember option, show/hide | `pages/LoginPage.tsx` | `pages/LoginPage.tsx` | OK (custom base removed on purpose, 0019) |
| Dashboard: hero verdict, vitals, throughput chart, quota summary, provider health, attention panel, credentials, runtime table, config panel | `features/dashboard/*` | same feature, 38 files | OK |
| Dashboard: 6 quick-link cards (`dashboard.cta_*`) | `DashboardPage.tsx:186-222` | no grid; inline links at `DashboardPage.tsx:633,653,710,713` | CHANGED (cosmetic) |
| Dashboard: sparkline of live requests | `components/LiveWire.tsx`, `Sparkline.tsx` | `components/ThroughputChart.tsx` | CHANGED |
| Dashboard: usage section | `/usage` reads | `UsagePanel.tsx` | KNOWN 0027 |
| Auth files: list, search, paging, select, upload, download, delete one/all, status, refresh | `features/authFiles/*` | same feature, 41 files | OK |
| Auth files: section nav (tabs) | absent | `components/AuthFilesSectionNav.tsx` | OK (new) |
| Auth files: cooldown + reset, policy fields, models dialog, cooldown hint | `AuthFilePolicyFields`, `authFileCooldowns.ts` | `AuthFileCooldownSection.tsx`, `authFilePolicyAdapter.ts` | OK |
| Auth files: OAuth excluded models editor | `pages/AuthFilesOAuthExcludedEditPage.tsx` | same path | OK |
| Auth files: OAuth model alias editor | `pages/AuthFilesOAuthModelAliasEditPage.tsx` | same path | OK |
| Providers: workbench groups, create/edit/delete sheets, key lists, openai-compatibility, vertex | `features/providers/*` | same feature, 36 files | OK (0021 landed) |
| Providers: policy editors (behavior, error rules, runtime, model entries, discovery) | `services/api/{errorRules,providerBehavior,runtimePolicy,modelEntries}.ts` | same modules | OK |
| Providers: model discovery panel | `useModelDiscovery.ts` | `sheets/forms/ModelDiscoveryPanel.tsx` | CHANGED (see Task 4: 2 keys missing) |
| Providers: OpenCode Go | `opencodeGo` provider + quota | partial | KNOWN 0030 |
| OAuth: provider list, search, flow dialog, URL copy, callback entry, polling, Devin, plugin OAuth | `features/oauth/*` (17 files) | `pages/OAuthPage.tsx` (1085 lines), `features/oauth/*` | OK |
| OAuth: SIWC bind dialog and status | `siwc*` | `features/oauth/components/SiwcBindDialog.tsx`, `siwcBinding.ts` | OK (0028) |
| OAuth: Vertex import | `components/VertexImportDialog.tsx` (drag and drop, "import another") | `OAuthPage.tsx:940-990` (file chooser) | CHANGED (no drop zone, no "import another") |
| Quota: card bodies for claude, codex, kimi, meta, xai, devin, antigravity, plugin | `features/quota/providers/*` | same | OK |
| Quota: Claude reset grants | `claudeResetGrants.ts` | `providers/claude/ClaudeResetGrants.tsx` | OK (0026) |
| Quota: Codex reset credits | `codex/CodexQuotaBody.tsx` | same | OK |
| Quota: OpenCode Go body | `providers/opencodeGo/*` | absent | KNOWN 0030 |
| Quota: xAI prepaid block, usage breakdown, resets label | `xai/XaiQuotaBody.tsx:169-211` | pay-as-you-go labels only | CHANGED (verify with design owner) |
| Quota: Claude stale snapshot badge | `claude/ClaudeQuotaBody.tsx:58` | absent | CHANGED (minor) |
| Quota: Meta window duration in hours | `meta/MetaQuotaBody.tsx:46` | absent | CHANGED (minor) |
| Quota: Codex "Pro Max" plan label | `codex/CodexQuotaBody.tsx:53` | absent | CHANGED (minor) |
| Quota: dev-only tier preview page | `preview/QuotaTierPreview.tsx` | absent | OK (dev only, not shipped) |
| Routing: board, strategy, session affinity, soonest reset | `features/routing/*` | same | OK (0029, 0033 in progress) |
| Logs: tail, request and error log tabs, error log open/download, request payload download by id, fullscreen, search, method/status/path filters | `features/logs/*` | `pages/LogsPage.tsx` + `pages/hooks/*` | OK |
| Logs: log-level filter (`logs.level_filter`) | `LogsPage.tsx:528-530` | none; `useLogFilters.ts` filters method, status, path only | MISSING |
| Logs: follow/pause read status, buffer eviction notice | `logs.read_status_*` | `logs.auto_refresh` toggle | CHANGED |
| Config: 7 visual sections, YAML source editor, diff modal, search, save bar, api-key cards, payload rules | `features/config/*` | same, plus `SectionQuota.tsx` | OK (0022) |
| Config: client API key names (local rename) | `apiKeyNames.ts`, `ApiKeysCardEditor.tsx:6-44` | none | MISSING |
| System: version, build, latest release check, stats, restart hints | `pages/SystemPage.tsx` | same | OK |
| Plugins: list, disable, delete, store, install, resource pages, quota, OAuth entries | `features/plugins/*` | same | OK |
| Theme, language (4 locales), notifications | shared | shared | OK (ko, tr, vi deferred by decision) |

### Task 2 — config coverage

Visual config field set: old 94 fields, new 97 fields (`fieldId` sets in
`features/config/searchIndex.ts`). Only in new: `codexIdentityConfuse`,
`quotaSwitchProject`, `quotaSwitchPreviewModel`. Only in old: none.
Sections: new adds `quota` (`SectionQuota.tsx`).

Proof: `comm` on the `fieldId` lists gives 0 old-only entries.

Go settings that neither panel edits. Universe: 189 v8 leaf paths, built by
reflection over `config.Config` with the `buildV8Paths` prefix table
(`/tmp/t0035-gorepo/cmd/dumpcfg`). A panel "edits" a path when its leaf key
appears in `apps/web/src`.

| Go setting group | Files | Note |
|---|---|---|
| `credentials.concurrency.*` (12 keys) | `internal/config/credential_concurrency.go` | tuning only |
| `credentials.in-flight.*` (10 keys) | same | tuning only |
| `upstream.claude.header-defaults.anthropic-beta-append`, `-strip`, `client-request-id`, `inject-diagnostics`, `native-entrypoints` | `config_types.go` | no UI |
| `upstream.claude.native-oauth-prompt-cache-1h`, `server-side-fallback`, `session-date-pin-days`, `tool-search` | same | no UI |
| `oauth.providers.codex.signin-with-chatgpt-identifier` | same | no UI |
| `upstream.codex.upstream-request-compression`, `-min-bytes`, `websocket-compression` | same | no UI |
| `server.max-request-body-bytes`, `requests.tls-handshake-timeout-seconds`, `requests.streaming.send-keepalive-before-ft`, `server.response-compression.min-bytes` | same | no UI |
| `observability.pprof.addr` | same | no UI |
| `oauth.providers.{antigravity,devin}.sensitive-words-replace` | same | no UI |

Panel fields with no Go field: none found. Each explicit v8 path in
`features/config` and `apps/web/src/services/api` maps to a Go path. Two
paths, `routing.session-affinity-subagents` and
`client.codex.optimize-multi-agent-v2`, map through `v8Aliases` /
`v8ClientPaths` (`internal/config/config_v8.go:27-58`), not through the
reflection prefix table. I verified both by runtime probe (Task 3).

Limit: `GetConfig` returns only keys that exist in the YAML file. A path
can be written even when `config_types.go` has no field, if
`NormalizeConfigLayout` keeps it. I did not test every Go-unknown path.

### Task 3 — route check

Extractor: `/tmp/t0035-routes.js` and `/tmp/t0035-calls.js` (scratch, not
committed). No caller uses `/v0`.

| Path (prefix `/v8/management`) | Method | Caller | Route |
|---|---|---|---|
| `/config` | GET | `services/api/config.ts:14`, `providers.ts:251`, `routing.ts:409` | yes |
| `/config` | PATCH | `services/api/configPatch.ts:254`, `routing.ts:474` | yes |
| `/config.yaml` | GET, PUT | `services/api/configFile.ts:9,20` | yes |
| `/config/api-keys/:family` | PUT | `services/api/providers.ts:277` | yes (`/config/*path`) |
| `/config/access/api-keys` | GET, PUT | `services/api/apiKeys.ts:9,23` | yes |
| `/config/oauth/excluded-models` | GET, PUT, DELETE | `services/api/authFiles.ts:461` | yes |
| `/config/oauth/model-alias` | GET, PUT, DELETE | `services/api/authFiles.ts:460` | yes |
| `/config/observability/logs/request-log` | PUT | `services/api/logs.ts:166` | yes |
| `/config/observability/usage/usage-statistics-enabled` | GET | `services/api/usage.ts:4` | yes |
| `/config/plugins/configs/:id`, `/enabled` | PUT | `services/api/plugins.ts:323,347` | yes |
| `/credentials` | GET, POST (multipart), DELETE | `services/api/authFiles.ts:548,596,606,614` | yes |
| `/credentials/download?name=` | GET | `authFiles.ts:617` | yes |
| `/credentials/models?name=` | GET | `authFiles.ts:672` | yes |
| `/credentials/status`, `/fields` | PATCH | `authFiles.ts:555,562` | yes |
| `/credentials/refresh` | POST | `authFiles.ts:566,573` | yes |
| `/credentials/siwc/status` | GET | `services/api/oauth.ts:69` | yes |
| `/oauth/siwc/status`, `/oauth/siwc/bind` | GET, POST | `oauth.ts:63,58,112` | yes |
| `/oauth/auth-url` | GET | `oauth.ts:80` | yes |
| `/oauth/callback` | POST | `oauth.ts:100` | yes |
| `/oauth/import?provider=vertex` | POST | `services/api/vertex.ts:23` | yes |
| `/oauth/status` | GET | `oauth.ts:87` | yes |
| `/oauth/session` | DELETE | `oauth.ts:93,113` | yes |
| `/observability/logs` | GET, DELETE | `services/api/logs.ts:137,146` | yes |
| `/observability/logs/errors` | GET | `logs.ts:149` | yes |
| `/observability/logs/errors/:name` | GET | `logs.ts:157` | yes |
| `/observability/logs/requests/:id` | GET | `logs.ts:160` | yes |
| `/observability/usage/api-keys` | GET | `services/api/apiKeyUsage.ts:8` | yes |
| `/observability/usage/queue` | GET | `features/dashboard/hooks/useDashboardOverview.ts:138`, `pages/usage-page-request-guard.ts:46` | yes |
| `/requests/api-call` | POST | `services/api/apiCall.ts:82` | yes |
| `/routing/cooldown/reset` | POST | `authFiles.ts:582` | yes |
| `/routing/model-definitions/:channel` | GET | `authFiles.ts:688` | yes |
| `/plugins` | GET | `services/api/plugins.ts:318` | yes |
| `/plugins/:id` | DELETE | `plugins.ts:326` | yes |
| `/plugins/store?query` | GET | `plugins.ts:354` | yes |
| `/plugins/store/:id/install` | POST | `plugins.ts:362` | yes |
| `/plugins/:id/quota` | POST | `features/quota/providers/plugin/data.ts:146` | yes |
| `/server/latest-version` | GET | `services/api/version.ts:8` | yes |

Old-panel routes that the new panel does not call: none. The old tree had
one extra path, `PUT /config/observability/logs/debug`, in a test only
(`tests/managementClientV8.test.ts:25`). New-only: the two SIWC status reads.

Runtime probe: I built `/tmp/t0035-server/llmproxy` from this checkout, ran
it with a `/tmp` config, and sent the panel writes with `If-Match`. All
returned 200: `routing/session-affinity-subagents`, `access/api-keys`,
`observability/logs/request-log`, `plugins/configs/demo/enabled`,
`api-keys/codex`, `oauth/excluded-models`.

v8 routes with no panel caller: `/credentials/verify`,
`/routing/model-definitions/refresh`, `/models/refresh`, `GET /oauth/callback`,
`GET /plugins/:id/quota`, `DELETE /plugins/:id/quota`.

### Task 4 — locale coverage

Checker: `/tmp/t0035-locale3.py`. 1729 `t('...')` keys with a namespace.
Keys that have no `_one`/`_other` form either are counted as missing.

Keys used in code and missing in all 4 locales (9):

| Key | Caller | Renders raw key |
|---|---|---|
| `common.clear` | `pages/LogsPage.tsx:1162-1163` | yes (title, aria-label) |
| `providersPage.discovery.syncHint` | `features/providers/sheets/forms/ModelDiscoveryPanel.tsx:124` | yes |
| `providersPage.discovery.applyChanges` | same file `:186` | yes |
| `notification.load_failed` | `pages/AuthFilesOAuthModelAliasEditPage.tsx:283`, `useAuthFilesModels.ts:99`, `AuthFilesOAuthExcludedEditPage.tsx:263` | yes (toast) |
| `notification.save_failed` | `features/config/hooks/useConfigDocument.ts:183,276` | yes (toast) |
| `auth_files.delete_title` | `useAuthFilesData.ts:325` | no (`defaultValue`) |
| `auth_files.delete_all_title` | `useAuthFilesData.ts:377` | no (`defaultValue`) |
| `oauth_excluded.delete_title` | `useAuthFilesOauth.tsx:200` | no (`defaultValue`) |
| `oauth_model_alias.delete_title` | `useAuthFilesOauth.tsx:248` | no (`defaultValue`) |

zh-TW only: `auth_login.login_another_account` (`OAuthPage.tsx:691`) and
`auth_login.view_auth_files` (`OAuthPage.tsx:864`) are missing and have no
`defaultValue`.

Keys in a locale but not in `en`: zh-CN 11, zh-TW 39, ru 51. Most are
plural variants (`_few`, `_many`) that en does not need. In zh-TW, 28 of
the 39 are dead OAuth keys: `auth_login.qwen_*` (11) and
`auth_login.iflow_cookie_*` (17). No code uses them. The other 11 are
plural base forms that also exist in zh-CN.

Copied English text for fork features in zh-CN, zh-TW, ru: none, except
technical terms `API Key`, `OAuth`, `{{share}}%`.

No test guards this. `tests/productionLocales.test.ts:4-15` checks a fixed
section list only, so `bun run verify` stays green with the 9 missing keys.

### Task 5 — quality checks

All commands ran in `/tmp/t0035-new/apps/web` on
`integration/panel-bumpyclock` (`db784257`), with bun from
`/home/su3h7am/.local/share/mise/installs/bun/latest/bin`.

| Command | Result |
|---|---|
| `bun install --frozen-lockfile` | pass, 272 packages |
| `bun run test` (inside verify) | 2031 pass, 0 fail, 244 files, 5.98 s |
| eslint (`bun run verify`) | pass |
| `tsc --noEmit` (app + node configs) | pass |
| `vite build` | pass, 1561 modules |
| `bun run tokens:check` | pass ("Design token output is up to date") |
| `bun run verify` | exit 0 |

Bundle: new `dist/index.html` 3,481.69 kB, gzip 1,001.88 kB. Old panel, same
commands on the tag tree: 3,416.99 kB, gzip 1,021.55 kB. New is 64.7 kB
larger on disk and 19.7 kB smaller as gzip.

### Proposed fix list, ranked by user impact

1. Add the 9 missing keys to the 4 locales, then add a test that checks
   every `t('key')` in `src` against the 4 locale files. Impact: 5 keys
   show as raw text now (`common.clear`, two discovery keys, two toast
   keys). Evidence: Task 4.
2. Add the log-level filter to the Logs page (`useLogFilters.ts` and
   `LogsPage.tsx` option row). Impact: users cannot hide debug or trace
   lines. Evidence: Task 1 row "Logs: log-level filter".
3. Add the client API key name editor to the config section
   (`ApiKeysCardEditor.tsx`). Impact: keys lose their labels after the
   import. Evidence: `apiKeyNames.ts` has no counterpart.
4. Add the zh-TW keys `auth_login.login_another_account` and
   `auth_login.view_auth_files`, or add `defaultValue`.
5. Decide on the quota body details (xAI prepaid, Claude stale badge, Meta
   window hours, Codex Pro Max label) with the design owner.
6. Cosmetic: dashboard quick-link grid, Vertex import drag-and-drop and
   "import another".

### Claims not confirmed

- All UI claims come from static code reading. I did not open the panel in
  a browser; the only runtime check was the HTTP probe on the `/tmp` binary.
- Quota display differences (item 5) may be intended design changes. I did
  not find a decision entry that covers them. Looked at `DECISIONS.md`
  entries of 2026-11-11 only.
- The old panel's behaviour for `logs.read_status_*` and buffer notices was
  not compared at runtime; I compared locale keys and component names only.
- The Go-unknown-path question in Task 2 is open for paths outside the 6
  probed ones.
