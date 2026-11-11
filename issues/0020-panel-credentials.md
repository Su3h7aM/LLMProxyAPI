---
status: done
branch: issue/0020-panel-credentials
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel v8: credentials (auth files), policy fields, cooldowns

## Goal

Port the credentials area of the panel to the v8 API and bring back the v8 credential features of the pre-import panel, in the BumpyClock design.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0020-panel-credentials integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **API module.** `src/services/api/authFiles.ts` and `authFileCooldowns.ts` to v8: list, upload, delete (one and all), download, status, fields, refresh, models (`/credentials*`). OAuth excluded models and OAuth model alias go through the config tree (`/config/oauth/excluded-models`, `/config/oauth/model-alias`) with the `configPatch.ts` and `configValue.ts` helpers. Model definitions use `GET /routing/model-definitions/:channel`. Take the exact paths, bodies, and shapes from the pre-import `services/api/authFiles.ts` and `authFilePolicyAdapter.ts`, and check them in the Go handlers.
2. **Transformers and types.** Update `types` and the auth-file normalizers for the v8 shapes (cursor paging, `auth-index`, and the `oauth_provider` field of the credential list; check `internal/api/handlers/management/auth_files*.go`).
3. **Policy features.** Bring back from the pre-import panel: auth-file policy fields (`authFilePolicyAdapter.ts`, `credentialPolicy.ts`, `manualRefresh.ts`, `AuthFilePolicyFields`, `AuthFileRefreshResults`). Build the UI in BumpyClock style inside its auth-files page (find the matching place: edit sheet, details, refresh action). Show the OAuth provider of a credential where the BumpyClock list has a provider label.
4. **Pages.** Fix the auth files page, the OAuth excluded models page, and the OAuth model alias page so they work on v8 (loading, editing, saving, error states).
5. **Tests.** Port pre-import tests for these modules (for example `authFileCooldowns.test.ts`; note the pre-import test at line 228 failed before the import because `auth_files.cooldown_reset_hint` was not rendered: check if the BumpyClock UI should render a cooldown reset hint and tell me in `## Follow-ups`). Keep BumpyClock tests green.
Do NOT do in this issue: the `opencode-go` filter (own issue), Claude reset grants (quota issue), provider key lists, config page.

## Gate

In `apps/web` with `export PATH=/home/su3h7am/.local/share/mise/installs/bun/latest/bin:$PATH`: `bun install --frozen-lockfile`, `bun run verify` (test, lint, build) must pass at the end, and `bun run type-check` plus the tests of your area after each commit. 
Tests: port the pre-import tests of your area (`git show panel-pre-bumpyclock:apps/web/tests/<name>`) when they fit the BumpyClock components; update BumpyClock tests that assert v0 paths. Do not delete or weaken a test to get green, except a test that pins behavior that this issue removes on purpose (say which in the commit body).

## Environment

- Use bun at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin` (put it on `PATH`; the token plugin runs `bun` at build time) and `/usr/bin/go`. Do NOT run `mise trust`.
- Never commit `node_modules`, `dist`, or `apps/web/dist/management.html`.
- English only. Use ASD-STE100 Simplified Technical English in all your text, be concise.

## Rules

- API is v8 only. No v0 path, no fallback to an older backend, no version check, no capability probe by 404. The panel and the proxy are one unit.
- Take only what this issue lists. Other areas are other issues.
- Keep the management key login, 401 handling, and the connection-revision guard on config read then write.
- Fork-only features stay (OpenCode Go, reset system, soonest-reset); they get their own issues unless this issue lists them.
- You do not decide. If a v8 route does not exist or differs from what the panel needs, if a UI choice is unclear, or if something is out of scope: stop, write the question in `## Log`, commit, and report. Do not invent endpoints and do not change Go code unless this issue says so.

- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. Create the branch as written above. Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current.
3. When the gate passes: `git rebase integration/panel-bumpyclock`, re-run the gate, set `status: done`, fill `## Result` (what landed, commit list, gate output summary, known gaps, follow-ups), commit.
4. Do NOT merge. Report: short summary, branch, commit range, follow-ups, surprises, unconfirmed claims.

## Log

- 2026-11-11: Set `status: in-progress`. Read `DECISIONS.md`, issues 0015/0016/0017/0019, current area files, pre-import reference, Go v8 handlers.
- 2026-11-11: Committed v8 API module (`authFiles.ts`, `authFilePolicyAdapter.ts` verbatim; `RequestScopedErrorRule` type).
- 2026-11-11: Committed v8 credential types (siwc fields, `quotaPluginId`; kept `qoder` member).
- 2026-11-11: Committed policy files (`credentialPolicy`, `manualRefresh`, `errorRules`, `ErrorRulesEditor`, `AuthFilePolicyFields`, `AuthFileRefreshResults`, token-safe scss), locale keys, minimal `Select.ariaInvalid` and `ModelEntriesEditor.oauthAliasOnly` props.
- 2026-11-11: Committed UI wiring (editor policy patch, key-based refresh/status, refresh-all modal, cooldown reset, oauth_provider label, VaultHeader button). Ported 6 pre-import tests; updated 5 v0-path tests.
- 2026-11-11: Gate green, rebased onto `integration/panel-bumpyclock`, gate green again. No merge.

## Follow-ups

- Cooldown reset hint: yes, the BumpyClock UI now renders it. The reset button in `AuthFileCooldownSection.tsx` carries `title={t('auth_files.cooldown_reset_hint')}`, and the key exists in all 4 locales. This answers the pre-import `authFileCooldowns.test.ts:228` question.
- SIWC is NOT ported (coordinator order). Places seen in the pre-import panel that a later SIWC issue needs: `features/authFiles/components/AuthFileCard.tsx` + `.module.scss` (`SiwcStatusBadge` use), `features/authFiles/components/AuthFileDetailsSheet.tsx` (`use_siwc` toggle, `supportsAuthFileSiwc`, `readAuthFileUseSiwc`), `features/authFiles/constants.ts` (siwc helpers), `features/authFiles/hooks/useAuthFilesPrefixProxyEditor.ts` (`useSiwc` field, patch, preview), `features/oauth/OAuthPage.tsx`, `features/oauth/siwcBinding.ts`, `features/oauth/components/SiwcBindDialog.*` and `SiwcStatusBadge.*`, `services/api/oauth.ts` (siwc calls), locale `siwc*` keys. Backend serves `POST /oauth/siwc/bind`, `GET /oauth/siwc/status`, `GET /credentials/siwc/status` (`server_management_v8.go:57-59`). Carried but UI-unused in this issue: `AuthFileItem` siwc fields, normalizer siwc mapping, `AuthFileFieldsPatch.use_siwc`, `auth_files.use_siwc_*` locale keys skipped on purpose.
- Skipped locale keys (other issues own the features): `auth_files.filter_opencode-go` (opencode-go filter issue), `auth_files.use_siwc_hint/label` (SIWC issue).
- Shared deps ported verbatim that overlap issue 0021 (providers): `features/providers/errorRules.ts`, `features/providers/sheets/forms/ErrorRulesEditor.*`, `RequestScopedErrorRule` type, `providersPage.errorRules` locale block, `ModelEntriesEditor.oauthAliasOnly` prop, `Select.ariaInvalid` prop. If 0021 ports the same files, content should merge cleanly.
- Not ported (other issues own them): `opencode-go` filter, Claude reset grants, provider key lists, config page, OAuth/SIWC flows, `filter_opencode-go`.
- Note: Go list uses `page`/`page_size`, not cursor paging. The panel loads the full list; no paging params are sent. The `oauth_provider` list field passes through the index signature and is now shown beside the provider badge when it differs.

## Result

Landed on `issue/0020-panel-credentials` (5 commits, no merge):

- `cdf91867` API module to v8 (`authFiles.ts`, `authFilePolicyAdapter.ts`, `RequestScopedErrorRule`).
- `a9b35cce` credential types (siwc fields, `quotaPluginId`).
- `eed19540` policy components, token-safe styles, locale keys, minimal shared props.
- `254b2bbd` UI wiring, ported tests, v0-path test updates.
- Gate: `bun run verify` green after rebase (1750 pass, 11 skip — pre-existing 0019 skips — 0 fail; lint, type-check, build `dist/index.html` 3.23 MB, tokens up to date).
- Pages fixed for v8: auth files page (list, upload, delete one/all, download, status, fields incl. policy, refresh one/all, models, cooldown reset), OAuth excluded-models and model-alias pages (config-tree reads/writes, `/routing/model-definitions` catalog).
- Known gaps: none in this area. SIWC UI intentionally absent (see Follow-ups).
