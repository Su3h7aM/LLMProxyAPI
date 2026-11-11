---
status: done
branch: issue/0028-panel-siwc
type: panel
agent: deepseek-v4.1-flash
base: integration/panel-bumpyclock
---

# Panel v8: Codex SIWC status and bind

## Goal

Bring back the Codex SIWC features of the pre-import panel in the BumpyClock design: SIWC status on the credential card and details, and the SIWC bind flow.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0028-panel-siwc integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Inventory first.** In `## Log`: list the pre-import SIWC code (`git grep -il siwc panel-pre-bumpyclock -- apps/web/src apps/web/tests`): `features/oauth/siwcBinding.ts`, `components/SiwcBindDialog.*`, `components/SiwcStatusBadge.*`, the SIWC rows in `AuthFileCard.tsx` and `AuthFileDetailsSheet.tsx`, `features/authFiles/constants.ts`, `useAuthFilesPrefixProxyEditor.ts`, `OAuthPage.tsx`, locale keys (`use_siwc_*`). Also issue 0020 `## Follow-ups` lists the SIWC places it saw.
2. **API.** Add the SIWC calls in v8: `POST /oauth/siwc/bind`, `GET /oauth/siwc/status`, `GET /credentials/siwc/status` (read the Go handlers `StartSIWCBind`, `GetSIWCStatus` for request and response shapes; do not guess). Make sure the credential normalizer carries the SIWC fields (for example `siwcExpiresAt`).
3. **UI.** Add the status badge and the expiry row to the BumpyClock auth-file card and details sheet, and the bind dialog and entry on the OAuth page, in BumpyClock style: use its Badge, Modal or Sheet, Button, and spacing tokens (look at how the card shows other states and how other dialogs are built). No raw values.
4. **i18n.** Add the strings in the 4 locales (en, zh-CN, zh-TW, ru; check the pre-import locale keys for the English text).
5. **Tests.** Port the pre-import SIWC tests that fit; keep BumpyClock tests green.
Do NOT do in this issue: other credential fields, quota, providers.

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

### 2026-11-11: Inventory (step 1)

Pre-import SIWC code (from `git grep -il siwc panel-pre-bumpyclock -- apps/web/src apps/web/tests`):

- API: `services/api/oauth.ts` (`SiwcBindResponse`, `oauthApi.startSiwcBind`, `captureSiwcBindingApi`); `services/api/authFiles.ts` normalizer SIWC fields; `types/authFile.ts` SIWC fields.
- Logic: `features/oauth/siwcBinding.ts`: `supportsSiwcBinding`, `listSiwcTargets`, `getSiwcBindingStatus`, `isSiwcCallbackValid`, `createSiwcStart`.
- UI: `features/oauth/components/SiwcBindDialog.tsx` + `.module.scss`; `features/oauth/components/SiwcStatusBadge.tsx` + `.module.scss`; `features/authFiles/components/AuthFileCard.tsx` + `.module.scss` (status badge + expiry row); `features/authFiles/components/AuthFileDetailsSheet.tsx` (`use_siwc` toggle); `features/oauth/OAuthPage.tsx` (entry + dialog).
- Helpers: `features/authFiles/constants.ts` (`supportsAuthFileSiwc`, `readAuthFileUseSiwc`); `features/authFiles/hooks/useAuthFilesPrefixProxyEditor.ts` (`useSiwc`, patch, preview).
- i18n: `siwc.*` (32 keys), `auth_files.use_siwc_label`, `auth_files.use_siwc_hint`.
- Tests: `tests/siwcBinding.test.ts`; SIWC parts of `tests/oauthPage.test.ts`, `tests/managementClientV8.test.ts`, `tests/authFileWeight.test.ts`.

State on `integration/panel-bumpyclock` (after 0018 to 0027): the normalizer and `AuthFileItem` already carry `siwcBindSupported`, `useSiwc`, `siwcBound`, `siwcExpiresAt`, and `AuthFileFieldsPatch.use_siwc` exists (issue 0020). Missing: all SIWC UI, `siwcBinding.ts`, the API calls, the locale keys.

Backend shapes (checked in Go, not guessed):

- `POST /v8/management/oauth/siwc/bind` body `{name}` -> `{status:"ok", url, state, flow:"siwc_bind", expires_in}`.
- `GET /oauth/siwc/status` and `GET /credentials/siwc/status` use the same handler `GetSIWCStatus`; query `name` or `auth_index`; response `{status:"off"|"valid"|"invalid"}`.
- Callback: `POST /oauth/callback` `{provider:"codex", redirect_url}`; cancel: `DELETE /oauth/session?state=`.

### 2026-11-11: Decisions taken inside the issue, and one deviation

- No `unsupported` phase and no 404 probe in the bind dialog: the panel and the proxy are one unit, and the rules forbid a capability probe by 404. Any start error shows the generic error view. The `siwc.unsupported` locale key is not ported (31 keys, not 32).
- BumpyClock has no Badge primitive; the SIWC badge is a small pill built from the BumpyClock tokens, like `regionBadge` and `state` in `OAuthPage.module.scss` and `AuthFileCard.module.scss`.
- The OAuth page entry is a `Card` inside the existing "Other login methods" section, next to Vertex, and it opens the bind `Modal`. This follows the BumpyClock page, which has no provider-tile grid.
- Status endpoints: both v8 GET paths hit the same Go handler. The API module gets both wrappers, covered by a test. The card and the sheet use the list fields (`siwc_bound`, `siwc_expires_at`), as the pre-import panel did, so no extra request is sent per credential.

### 2026-11-11: Work log

- `b9fb1dfd` API: `oauthApi.startSiwcBind`, `getSiwcStatus`, `getCredentialSiwcStatus`, `captureSiwcBindingApi`; `features/oauth/siwcBinding.ts`; `supportsAuthFileSiwc` and `readAuthFileUseSiwc`.
- `545f135a` UI: card pill (`SiwcStatusBadge`) in the credential subline, SIWC expiry in the details column, `use_siwc` switch and SIWC pill plus expiry in the details sheet, editor `useSiwc` field and patch, locale blocks.
- `cc7acd03` UI: bind `Card` in "Other login methods" and `SiwcBindDialog` (pick, authorize, paste callback, poll, cancel on close, restart in place).
- `2a4add25` tests: `tests/siwcBinding.test.ts` and the `use_siwc` cases in `tests/authFileWeight.test.ts`.

No Go change. No v0 path. The normalizer and `AuthFileItem` already carried the SIWC fields (issue 0020), so no change was needed there.

## Follow-ups

- The SIWC status calls (`oauthApi.getSiwcStatus`, `oauthApi.getCredentialSiwcStatus`) have no UI consumer yet: the card and the sheet read the binding state from the credential list, as the pre-import panel did. A later issue can use them for a live refresh per credential.
- The bind dialog is covered by source contracts and lifecycle unit tests, not by a browser test. A browser check of the paste-callback path needs a live Codex authorization.

## Result

Landed on `issue/0028-panel-siwc`, no merge.

What landed:

- v8 API: `POST /oauth/siwc/bind`, `GET /oauth/siwc/status`, `GET /credentials/siwc/status`, plus `captureSiwcBindingApi` for a same-connection start and cancel.
- Logic: `features/oauth/siwcBinding.ts` (capability, target list, status, callback validation, start lifecycle).
- Credential card: SIWC pill in the credential subline, SIWC expiry entry in the details column.
- Details sheet: "Use SIWC (use_siwc)" switch, SIWC pill and expiry; the switch writes `use_siwc` and preserves the binding data.
- OAuth page: "Bind SIWC" card in "Other login methods" that opens a modal: pick a Codex credential, open the authorization URL, paste the loopback callback, wait for the server to save the binding, restart in place on failure.
- 4 locales: `siwc.*` (31 keys) and `auth_files.use_siwc_label/hint`.

Commits: `76ad487b`, `2d22ec02`, `d453a73a`, `3fb92a27`, `e9f91840`, `605b3bff`, `de9a50d9`.

Gate (after `git rebase integration/panel-bumpyclock`), in `apps/web`:

- `bun install --frozen-lockfile`: no changes.
- `bun run tokens:check`: output up to date.
- `bun run verify`: 2031 pass, 0 fail (244 files); eslint clean; `tsc` clean; build wrote `dist/index.html` (3.48 MB).

Known gaps:

- No 404 capability probe and no "unsupported" view: the panel and the proxy are one unit. The `siwc.unsupported` locale key is not ported.
- No live SIWC status request per credential; see Follow-ups.
- The bind flow is not exercised against a real Codex authorization in this environment.

Unconfirmed: the SIWC pill placement in the narrow status column was avoided on purpose (it sits in the credential subline), but the real look on a narrow screen was not checked in a browser.
