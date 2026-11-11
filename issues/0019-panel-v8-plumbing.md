---
status: in-progress
branch: issue/0019-panel-v8-plumbing
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel v8 plumbing: same-origin connection, v8 client, login, version

## Goal

Make the base of the panel v8-only and same-origin: prefix, API client, connection, auth store, login page, and version check. Delete the code that exists only for a separate or unknown server. Other API modules (config, credentials, logs, providers, quota, and so on) are NOT in this issue; they are ported by the next issues. After this issue only login, the System page, and the shared client work against the fork backend.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0019-panel-v8-plumbing integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **v8 prefix and constants.** `src/utils/constants.ts`: `MANAGEMENT_API_PREFIX` to `/v8/management`; remove `DEFAULT_API_PORT`, the minimum-version and support-plugin header key lists, and other constants that only serve version or capability negotiation. Keep the header key lists that the System page uses to show the server version and build date (display only).
2. **Same-origin connection.** `src/utils/connection.ts` and `src/stores/useAuthStore.ts`: the API base is always the page origin plus the v8 prefix. Delete API base detection, normalization, custom or remote base, scoped storage keys per base, and legacy `apiBase`/`apiUrl` migration. Keep the management key, remember-key option and its secure storage, auto-login with the stored key, and logout. Use the pre-import `apps/web/src/utils/connection.ts` and `useAuthStore.ts` as a v8 reference, then reduce them further to same-origin only (the pre-import ones still have a base readout and a Change toggle: do not take that part).
3. **Login page.** `src/pages/LoginPage.tsx`: key-only form (key input, show or hide, remember option, submit, error state). Remove the custom address checkbox and field, the auto-detect readout, and the remote connect hint. Keep the BumpyClock look. Remove the strings that become unused from the 4 locale files (custom connection, CORS, remote access, minimum version).
4. **Client.** `src/services/api/client.ts`: v8 base from step 2; Bearer key; 401 handling and logout event; port the connection-revision guard (`connectionRevision`, `getConnectionRevision`, `captureConnection`) from the pre-import `client.ts`; keep the response-header reading that feeds the System page version display. Delete the version negotiation, the support-plugin header gate, and any `x-cpa-*` based feature gating. Do not port `legacyBackendProbe.ts` and delete any BumpyClock equivalent (`error_legacy_backend` and similar).
5. **Shared v8 helpers.** Add `src/services/api/configValue.ts` and `src/services/api/configPatch.ts` from the pre-import panel (they are the v8 config tree helpers that other issues use). Port their tests. Do not wire them into pages: other issues do.
6. **Version check.** `src/services/api/version.ts` to `GET /server/latest-version` and the System page use of it; port the pre-import `tests/versionFailure.test.ts` (a 404 or 502 must not report a release). Check the Go handler `internal/api/handlers/management/config_basic.go` for the real response shape.
7. **Tests and cleanup.** Port or update tests for client, auth store, login form, connection; remove tests that pin removed behavior (say so in the commit body). List in `## Follow-ups` every remaining v0 path in `src/services/api/*` (file and path), so that the next issues can pick them up.


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

## Follow-ups

## Result
