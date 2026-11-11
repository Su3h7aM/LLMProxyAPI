---
status: in-progress
branch: issue/0030-panel-opencode-go
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel v8: OpenCode Go as a first-class provider (keys, quota tab, credential filter)

## Goal

Bring the fork-only OpenCode Go support back in the panel, in the BumpyClock design: the key-only provider in the providers workbench, the quota tab, the credential filter, brand text and icon, and the config-to-UI data path. OpenCode Go is a first-class provider of this fork.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0030-panel-opencode-go integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Inventory first.** In `## Log`: list what commits `760d7f3a`, `c712e17f`, and `f3912994` changed in the panel (`git show <hash>` on `main`; the files are listed by `git show --stat`). Describe the user-visible behavior: provider entry (API key list, prefix, priority, excluded models), quota tab (what it shows and from which backend route), credential filter (`filter_opencode-go`), strings (`opencode_go_quota`, `providerNames.opencodeGo`), icons. Find the Go side: `internal/auth/opencode/usage.go`, `internal/api/handlers/management/plugin_quota.go` and the config types (`grep -rli opencode internal sdk`); read the real config path (`opencode-go-api-key`) and the quota route and response shape. Say how the current panel gets `openCodeGoApiKeys` (the v8 config tree) and what the v8 config reader in `src/services/api/transformers.ts` already reads.
2. **Provider entry.** Add OpenCode Go to the providers workbench (descriptors, adapters, brand logo, types, `useProviderWorkbench`, `providers.ts` key-list mapping to the v8 config tree) in BumpyClock style: find how another key-only provider (for example one with only API key, prefix, priority, and excluded models) is built, and copy its pattern. Include `types/config.ts` and the config store reading.
3. **Quota tab.** Add the OpenCode Go quota provider: `features/quota/providers/opencodeGo/{OpencodeGoQuotaBody.tsx,data.ts}`, registry and types, constants and validators, `useQuotaStore` support, built in the BumpyClock quota provider style (copy the structure of a similar provider body: windows, reset times, refresh). Use the correct v8 route from step 1; no invented routes. It must feed the reset schedule and soonest highlights like the other providers.
4. **Credential filter and strings.** `features/authFiles/constants.ts` filter `opencode-go` and its label; locale strings in the 4 locales (en, zh-CN, zh-TW, ru) with the brand text rule (LPA and LLM Proxy API).
5. **Tests.** Port the pre-import tests of commit `760d7f3a` that fit; add tests for the config path and the quota data parser; keep BumpyClock tests green.
Do NOT do in this issue: routing board, other providers, cleanup.

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

- 2026-11-11: Set `status: in-progress`. Branch from `integration/panel-bumpyclock`.
- 2026-11-11: Inventory. `760d7f3a` adds provider `opencodeGo` (key-only: API key, prefix, priority, excluded models; no URL/models/headers) on family `opencode-go-api-key` via v8 config groups, plus quota tab (`opencodeGo/OpencodeGoQuotaBody.tsx`, `data.ts`), `TYPE_COLORS['opencode-go']`, `OPENCODE_GO_USAGE_URL=https://opencode.ai/zen/go/v1/usage` with `x-opencode-session` headers, `filter_opencode-go`, `providerNames.opencodeGo`, `opencode_go_quota` strings, icons `opencode.svg`/`opencode-dark.svg`, and 2 tests. `c712e17f` adds 7-locale strings and LPA/LPAMC/LLM Proxy API brand text. `f3912994` maps `api-keys.opencode-go` groups to `openCodeGoApiKeys` in `transformers.ts` and marks config-backed keys `runtime_only=true` in Go synthesizer.
- 2026-11-11: Go side. Config key `opencode-go-api-key` (`internal/config/config.go:166`, `config_types.go:987`); v8 read via tree `PUT /config/*path` (`internal/api/server_management_v8.go`). Quota: `internal/auth/opencode/usage.go` (5h/weekly/monthly windows, session header); `plugin_quota.go:154` routes `opencode-go` auth to quota fetch; v8 `credentials/quota/fetch` removed (404 per `server_management_v8_test.go:119`). So panel uses `POST /requests/api-call` with fixed usage URL. Current panel already reads `api-keys.opencode-go` (`transformers.ts:384`), has `openCodeGoApiKeys`, family CRUD, `useConfigStore` case. Missing: descriptors, adapters, brandLogos, ProviderBrand, workbench mapping, dashboard count, authFiles icons/filter, quota provider, timeline/resetSchedule, locale strings (4 locales), icons.

## Result
