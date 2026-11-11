---
status: done
branch: issue/0036-panel-locale-coverage
type: panel
agent: deepseek-v4.1-flash
base: integration/panel-bumpyclock
---

# Panel: fix missing locale keys, add a coverage guard, fix the repository link

## Goal

Fix 9 translation keys that code uses but no locale defines (5 of them show the raw key on screen), add a test that keeps this from coming back, remove unused keys, and make the System page repository link point to this fork.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0036-panel-locale-coverage integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Missing keys.** Add these keys in the 4 locales (en, zh-CN, zh-TW, ru) with correct text; find the English meaning from the caller and from the pre-import locale (`git show panel-pre-bumpyclock:apps/web/src/i18n/locales/en.json`): `common.clear` (`pages/LogsPage.tsx`), `providersPage.discovery.syncHint`, `providersPage.discovery.applyChanges` (`ModelDiscoveryPanel.tsx`), `notification.load_failed`, `notification.save_failed` (callers: `AuthFilesOAuthModelAliasEditPage.tsx`, `useAuthFilesModels.ts`, `AuthFilesOAuthExcludedEditPage.tsx`, `useConfigDocument.ts`), and the four keys that have a `defaultValue` and render fine today but should exist: `auth_files.delete_title`, `auth_files.delete_all_title`, `oauth_excluded.delete_title`, `oauth_model_alias.delete_title`. Locale JSON files contain a duplicated `errorRules` key under `providersPage`: do NOT re-dump the JSON (it collapses the duplicate); insert text lines only, and keep the files otherwise byte-identical.
2. **Coverage guard.** Add `tests/localeKeyCoverage.test.ts`: scan `src` for literal `t('namespace.key')` calls (and `i18n.t(...)`), skip dynamic keys, and fail if a key is missing in any of the 4 locales (accept plural forms `_one`/`_other`, and calls that pass a `defaultValue` only if you list them in a small explicit allow-list with a comment). The test must pass after step 1.
3. **Unused keys.** Remove keys that no code reads any more, from all 4 locales (text edits only), using this list from issue 0032 as the start: `nav.usage_statistics`, `nav_meta.usage_statistics`, `redesign.providers.setup*`, `dashboard.usage_*`, `dashboard.vitals_usage_differs*`. Before each removal grep `src` and `tests`, including dynamic key building (template strings). Keep any key that a kept-but-unreachable component still reads (`UsagePage.tsx`, `UsagePanel.tsx`, `features/dashboard/logic/usage.ts` stay for a later issue).
4. **Repository link.** The System page has a "WebUI Repository" link that names the reference repository. Make it point to `https://github.com/Su3h7aM/LLMProxyAPI` (find where the URL comes from in `SystemPage.tsx` and its constants). Do not change Go code or the `panel-github-repository` config key.
5. **Gate and tests.** Keep all tests green.
Do NOT do in this issue: other UI changes.

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

1. Create the branch as written above. Set `status: done` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current.
3. When the gate passes: `git rebase integration/panel-bumpyclock`, re-run the gate, set `status: done`, fill `## Result` (what landed, commit list, gate output summary, known gaps, follow-ups), commit.
4. Do NOT merge. Report: short summary, branch, commit range, follow-ups, surprises, unconfirmed claims.

## Log

- 2026-11-11: Set `status: in-progress`. Branch `issue/0036-panel-locale-coverage` from `integration/panel-bumpyclock`. Read `DECISIONS.md` (panel migration plan, one unit, design rule) and the 0032 steps for the key list.
- 2026-11-11: Step 1. Added the nine keys to en, zh-CN, zh-TW, ru as single inserted lines; no re-dump, so the duplicate `providersPage.errorRules` key stays. Text for `providersPage.discovery.syncHint` and `applyChanges` comes from the pre-import locale (`git show panel-pre-bumpyclock:apps/web/src/i18n/locales/<locale>.json`). `common.clear` is "Clear" (icon title and aria-label of the log search clear button). `notification.load_failed` / `save_failed` follow the existing `*_failed` pattern. The four `defaultValue` keys use the fallback text that the callers already show.
- 2026-11-11: Finding, not in the issue list. The new coverage guard found two more missing keys: `auth_login.login_another_account` and `auth_login.view_auth_files` exist in en, zh-CN and ru but not in zh-TW, so zh-TW showed raw keys on the OAuth page and the SIWC dialog. They came with the SIWC work (`91e633c1`). Added the two zh-TW strings (`4b261059`); the guard cannot pass without them and allow-listing them would hide a real defect.
- 2026-11-11: Step 2. `tests/localeKeyCoverage.test.ts` scans `src` for `t('namespace.key')` and `i18n.t(...)` with a static key; template keys and keys built in variables are skipped. A key counts as defined when the locale has it, or any i18next plural suffix (`_zero`, `_one`, `_two`, `_few`, `_many`, `_other`). Calls that pass `defaultValue` are skipped only for keys in `DEFAULT_VALUE_ALLOWED` (empty: every `defaultValue` key exists today). Self-tests cover a missing key, plural forms, blank values and the allow-list.
- 2026-11-11: Step 3. Removed 12 keys from all four locales (no code reads them): `nav.usage_statistics`, `nav_meta.usage_statistics`, `redesign.providers.setup{Title,Provider,OAuth,Client,Usage}`, `dashboard.usage_eyebrow`, `dashboard.usage_title`, `dashboard.usage_window_bounds`, `dashboard.vitals_usage_differs`, `dashboard.vitals_usage_differs_short`. Grepped `src` and `tests` for each key and for template or prefix building before removal. Keys that `UsagePanel.tsx` reads stay, as the issue requires. `tests/dashboardVitalsScope.test.ts` pinned the text of the two `vitals_usage_differs` keys and listed them; that case and those list entries go with the keys (the keys described the dashboard usage section that 0032 removed).
- 2026-11-11: Step 4. The System page "WebUI Repository" link (`system_info.link_webui_repo`) pointed to `router-for-me/Cli-Proxy-API-Management-Center`; it now points to `https://github.com/Su3h7aM/LLMProxyAPI`. The URL is a literal in `SystemPage.tsx`; no Go code and no config key changed. `tests/round2SystemPage.test.ts` pins it.
- 2026-11-11: Gate. `bun install --frozen-lockfile` and `bun run verify` green before and after the rebase: 2092 tests pass, 0 fail; `eslint`, `type-check`, token check and the one-file `vite build` pass. Rebase onto `integration/panel-bumpyclock` (`8f589ac6`, includes 0037) clean.

## Follow-ups

- The guard sees static keys only. Keys built from a template (`t(`dashboard.source_${id}`)`) still need the locale parity tests or a reviewer.
- `system_info.link_main_repo` still points to `router-for-me/LLMProxyAPI`. The issue named only the WebUI link; a later issue can decide the main repository link.
- The guard proves that a key exists, not that its text is right. Native-speaker review of the new ru and zh strings is still open.

## Result

- Landed: the nine missing keys in en, zh-CN, zh-TW, ru (`common.clear`, `providersPage.discovery.syncHint`, `providersPage.discovery.applyChanges`, `notification.load_failed`, `notification.save_failed`, `auth_files.delete_title`, `auth_files.delete_all_title`, `oauth_excluded.delete_title`, `oauth_model_alias.delete_title`); the two zh-TW `auth_login` keys the guard found; `tests/localeKeyCoverage.test.ts`; 12 unused keys removed; the System page WebUI repository link now points to the fork.
- Commits on `issue/0036-panel-locale-coverage` over `integration/panel-bumpyclock` (`2b64d406..a02fff92`): `89bdf86d` nine keys, `6eb184b7` zh-TW `auth_login`, `70fbef9b` coverage guard, `c74a9a2a` unused keys, `a02fff92` repository link. Rebase clean.
- Gate: `bun install --frozen-lockfile` and `bun run verify` green at the rebase head: 2092 tests pass, 0 fail; lint, type-check, token check pass; `vite build` writes one `dist/index.html` (3,469 kB).
- Known gaps: the guard skips dynamic keys and does not judge translation quality. Not merged, as ordered.
