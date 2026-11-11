---
status: done
branch: issue/0025-panel-oauth-vertex
type: panel
agent: deepseek-v4.1-flash
base: integration/panel-bumpyclock
---

# Panel v8: OAuth login flows and Vertex import

## Goal

Port the OAuth page and the Vertex import to the v8 API.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0025-panel-oauth-vertex integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **OAuth API.** `src/services/api/oauth.ts` to v8: `GET`/`POST /oauth/auth-url?provider=<name>`, `GET /oauth/status`, `DELETE /oauth/session`, `POST /oauth/callback`. Use the pre-import `oauth.ts` and `features/oauth/*` (including `useOAuthFlows.ts`) as reference and check each route and body in the Go handlers (`internal/api/server_management_v8.go`, `internal/api/handlers/management/`).
2. **Providers list.** The OAuth page must list only providers that the fork backend can start (check the provider names in Go). Remove list entries and icons for providers with no backend (the `qoder` entry is already removed). Keep the sponsor-free list.
3. **Vertex import.** `src/services/api/vertex.ts` to `POST /oauth/import?provider=vertex` (multipart).
4. **Hints.** If the pre-import OAuth flow has provider hints or metadata handling that the backend change `482efbf5` supports (explicit OAuth hints, `oauth_provider`), bring it back in the BumpyClock OAuth page style.
5. **Tests.** Port pre-import OAuth tests; keep BumpyClock tests green.
Do NOT do in this issue: credentials list, quota, providers.

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

- 2026-11-11: Branch `issue/0025-panel-oauth-vertex` created from
  `integration/panel-bumpyclock` (35c2c884). Status in-progress.
- 2026-11-11: Routes checked in `internal/api/server_management_v8.go` and
  the handlers. v8 has `GET`/`POST /oauth/auth-url`, `GET /oauth/status`,
  `DELETE /oauth/session`, `POST /oauth/callback`,
  `POST /oauth/import?provider=vertex`. The auth-url GET reads
  `is_webui` from the query, so the web UI flag stays a query parameter.
- 2026-11-11: `oauth.ts` moved to the v8 routes. The UI key `anthropic`
  maps to the backend provider `claude` (path and callback body).
  Plugin providers use the same `/oauth/auth-url?provider=<id>` route.
  `vertex.ts` moved to `/oauth/import?provider=vertex`.
- 2026-11-11: Step 2 checked against the Go dispatch in `StartOAuthV8`:
  claude, codex, antigravity, kimi, kimi-ai, xai, devin, meta, plus
  plugin providers. The panel list has exactly these eight built-ins
  (`anthropic` is the UI name for `claude`), no qoder and no other
  provider without a backend. No entry removed.
- 2026-11-11: Step 4 checked: the pre-import OAuth flow has no provider
  hint or `oauth_provider` handling to port (`git grep` over
  `features/oauth` and `services/api/oauth.ts` on tag
  `panel-pre-bumpyclock` finds no such use). Commit `3b12afff`
  (`Upstream: router-for-me/CLIProxyAPI@482efbf5`) is backend-only. The
  panel side needs the target-credential re-login, which belongs to the
  auth files or providers area, not this issue.
- 2026-11-11: Tests: the v0 path assertions in `devinOAuthApi`,
  `kimiOAuth`, `metaOAuthProvider`, `pluginOAuthEntries`, and
  `oauthRequestCancellation` now expect v8 routes. New
  `tests/oauthV8Contract.test.ts` ports the pre-import assertions for the
  anthropic to claude mapping, the shared plugin route, and the Vertex
  import route.
- 2026-11-11: Gate passed on commit a197df89: `bun run verify` (1708
  tests, lint, build) and `bun run tokens:check`.

## Follow-ups

## Result

Landed:
- `apps/web/src/services/api/oauth.ts` uses the v8 routes:
  `GET /oauth/auth-url?provider=<p>[&is_webui=true]`,
  `GET /oauth/status`, `DELETE /oauth/session`,
  `POST /oauth/callback`. The UI key `anthropic` maps to the backend
  `claude` provider. Plugin providers share `/oauth/auth-url`.
- `apps/web/src/services/api/vertex.ts` uses
  `POST /oauth/import?provider=vertex` (multipart, `file` and optional
  `location`).
- Provider list checked against the Go dispatch: the eight built-ins
  (meta, kimi, kimi-ai, codex, anthropic, antigravity, xai, devin) all
  have a backend route. No entry or icon removed.
- Tests: `devinOAuthApi`, `kimiOAuth`, `metaOAuthProvider`,
  `pluginOAuthEntries`, and `oauthRequestCancellation` now assert v8
  routes. New `tests/oauthV8Contract.test.ts` ports the pre-import
  assertions for `anthropic` to `claude`, the shared plugin route, and
  the Vertex import route.

Commits (on top of `integration/panel-bumpyclock` 35c2c884):
- 0ac293d4 chore(panel): start issue 0025 OAuth and Vertex v8
- a197df89 feat(panel): move OAuth and Vertex import APIs to v8
- 8c0c684b docs(issues): log OAuth and Vertex v8 port
- (this commit) docs(issues): finish issue 0025

Gate: `bun install --frozen-lockfile`, `bun run verify` (1708 tests:
1697 pass, 11 skip, 0 fail; lint and build clean),
`bun run tokens:check` clean. Rebased on `integration/panel-bumpyclock`;
the branch had not moved.

Known gaps:
- Step 4 had no in-scope work: the pre-import OAuth flow has no provider
  hint or `oauth_provider` handling. The backend hint support
  (`3b12afff`, `Upstream: .../482efbf5`) serves target-credential
  re-login, which this panel does not have yet.
- SIWC bind (`/oauth/siwc/bind`) is a fork feature that BumpyClock does
  not have and this issue does not list. Not ported.
- The OAuth page still shows the BumpyClock 404 callback "upgrade"
  hint. It is a text, not a route probe; kept as is.

Follow-ups:
- Add target-credential re-login to the providers or auth files area,
  with the `oauth_provider` hint from the backend.
- Port the Codex SIWC bind flow (pre-import `siwcBinding.ts` and
  `SiwcBindDialog`) in its own issue.
- Decide whether to drop the 404 callback upgrade hint from the OAuth
  page.

