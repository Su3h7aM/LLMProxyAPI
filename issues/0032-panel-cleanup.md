---
status: done
branch: issue/0032-panel-cleanup
type: panel
agent: deepseek-v4.1-flash
base: integration/panel-bumpyclock
---

# Panel cleanup: hide Usage, remove dead toggle, leftovers, brand pass

## Goal

Clean the integration branch after the v8 port: hide screens with no backend data, remove a setting that the backend ignores, remove leftovers from the earlier issues, and finish the brand pass.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0032-panel-cleanup integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Hide Usage.** The backend has no aggregate usage data (see `issues/0027-*.md`). Remove the Usage page nav entry and route (`MainLayout.tsx`, `router/MainRoutes.tsx`, `features/providers/workbenchLayout.ts` link, dashboard links) and the usage section of the dashboard. Leave the components and logic files in place (a later issue may add the data), but make them unreachable, and keep the tests that cover them green (adjust only reachability tests; say so in the commit body). Do not touch the `/observability/usage/queue` route: the panel must never call it.
2. **Dead toggle.** Remove the `codexIdentityConfuse` setting from the config UI, the visual config types and parser, the search index, constants, locales, and tests: the Go config has no such field (`grep -rn identity internal/config` to confirm; read `issues/0022-*.md`).
3. **Leftovers.** From the follow-ups of earlier issues: the OAuth page 404 callback "upgrade" hint text (issue 0025); the stale Qoder text in `src/features/quota/README.md` and `isPluginQuotaFile` special cases for Copilot and Qoder (issue 0018); unrendered `QuickStartSteps`, `createLabel`, `workbenchLayout` code if nothing references it (check with grep and typecheck); `tests/visual-audit/*.mjs` stubs that use `/v0/management` (change to `/v8/management` or delete the stubs if the scripts are unused). Remove dead `providerNames` locale keys only if no code reads them (grep).
4. **Brand pass.** Search visible strings and docs in `apps/web` for old brand text: `grep -rniE "CPAMC|CLI Proxy API|\bCPA\b|cpa-manager|router-for" src tests index.html README.md AGENTS.md docs`. Change visible text to the fork brand (LLM Proxy API, LPA, LPAMC). Do not rename technical identifiers (header names, storage keys, class names, file names, URLs of external projects that must stay as they are, license notice text).
5. **Gate.** Everything green; list what you removed in `## Result`.
Do NOT do in this issue: new features.

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

- 2026-11-11: Branch from `integration/panel-bumpyclock`. Gate at each
  commit: `bun run type-check`, `bun run lint`, `bun test`.
- Step 1, Usage. Removed the nav entry (`MainLayout.tsx`), the `/usage`
  route (`MainRoutes.tsx`), the quick start step, the dashboard usage
  panel, and the "usage differs" caveat. Also stopped the dashboard
  polling the usage-statistics flag: nothing rendered its result, so the
  dashboard keeps only its four sources, and `DashboardPolledSourceId`
  (`Exclude<DashboardSourceId, 'usage'>`) types them. `attention.ts`
  keeps `'usage'` in `DashboardSourceId`, so its logic file stays whole.
  The runtime panel takes the full width with the existing
  `section + panel` pattern; the `.detailGrid` rule is gone. Usage page,
  guard, `UsagePanel`, `logic/usage.ts`, `types/usage.ts`, and the
  `dashboard.usage_*` locale keys stay. No test asserted the route, the
  nav entry, or the dashboard panel, so no test changed.
- Step 1 decision (assumption, not from the issue): `UsagePanel` keeps
  its two `to="/usage"` links. The component is unreachable, and the
  later issue that restores the page needs them. Say if the links must
  go now.
- Step 2, `codexIdentityConfuse`. Confirmed absent from the Go config
  (`grep -rn "identity-confuse" --include=*.go` finds nothing; issue
  0022 says the same). Removed UI, search index, field anchors, types,
  defaults, read and write paths, four locales, and the test
  `visualConfigCodexIdentityConfuse.test.ts` (it pins the removed
  behaviour).
- Step 3. OAuth 404 hint and `getErrorStatus` removed (issue 0025
  follow-up): the panel and the proxy are one unit, so no 404 branch.
  `isPluginQuotaFile` no longer lists `github-copilot` and `qoder`; the
  README sentence about Qoder is gone. `pluginQuota.test.ts` no longer
  lists those two providers (it pinned the removed special cases).
  Deleted the unreferenced `QuickStartSteps` component, its stylesheet,
  `QUICK_START_STEPS`, the unused `quickStart` sidebar icon entry.
  `createLabel.ts` stays: `ProviderResourcePanel` and
  `ProvidersWorkbenchPage` import `getProviderCreateLabel`.
  Deleted `tests/visual-audit/`: three manual browser scripts that stub
  the v0 management API (paths `/auth-files`, `/logs`, `/plugins`,
  `/usage`, `/latest-version`, ...). No task, CI job, or doc references
  them; nothing in the repo can run them against the v8 API.
  Locales: removed `apikeyFun`, `fennoAI`, `qiniuCloud` from
  `providerNames` (no reader; brand ids come from `ProviderBrand`).
  `kimi` stays because the backend serves Kimi.
- Step 4, brand. Changed `login.error_not_found` and
  `plugin_store.cli_proxy_api_source` in all four locales. Kept
  `CPA-<ShortID>` (the Go discovery code generates it), the
  `router-for-me` URLs and plugin org prefix, `x-cpa-*` headers,
  `cpa-management-key`, and code comments that say CPA for the backend.
- Known gaps left on purpose: `nav.usage_statistics`,
  `nav_meta.usage_statistics`, and `redesign.providers.setup*` locale
  keys now have no reader. Other issues may want them, and this issue
  does not list locale cleanup beyond `providerNames`.


## Follow-ups

- Delete the unused Usage and quick start locale keys, or restore them
  with the page.
- Decide the fate of `UsagePage`/`UsagePanel` if a v8 usage route
  appears later.
- The System quick link "WebUI Repository" still points at the
  reference repo; decide the fork target.
- `panel-github-repository` (Go default and the config placeholder) still
  names the reference repo.
- The brand pass left `CPA` in code comments and test names.

## Result

Landed on `issue/0032-panel-cleanup`:

- `feat(panel)!: hide the Usage page and the dashboard usage section`
  - no `/usage` route, no nav entry, no quick start step, no dashboard
    usage panel or usage caveat
  - the dashboard no longer polls the usage-statistics flag
- `fix(panel): remove the codexIdentityConfuse setting`
  - the toggle, its search entry, its types, defaults, YAML paths, and
    locales are gone; the backend never had the field
- `chore(panel): remove leftovers from the earlier panel issues`
  - OAuth 404 "upgrade" hint, Copilot and Qoder quota special cases,
    Qoder README text, `QuickStartSteps`, `tests/visual-audit`, three
    sponsor `providerNames` keys
- `chore(panel): finish the brand pass in visible text`
  - login not-found error and default plugin source label name
    `LLM Proxy API` in all four locales

Gate: from `apps/web`, `bun install --frozen-lockfile`, then
`bun run verify`: 2059 tests pass in 248 files, `eslint` clean,
`tsc` clean, vite build writes `dist/index.html`. `bun run tokens:check`
reports up to date. Rebased onto `integration/panel-bumpyclock`
(`ade773b1`); the only conflict was `isPluginQuotaFile`, resolved by
keeping the `opencode-go` adapter from issue 0030 and dropping only
`github-copilot` and `qoder`. No Go code changed.

Known gaps:

- `nav.usage_statistics`, `nav_meta.usage_statistics`,
  `redesign.providers.setup*`, `dashboard.usage_*`, and
  `dashboard.vitals_usage_differs*` locale keys now have no reader, or
  only an unreachable reader. Kept on purpose.
- `UsagePage`, its guard, `UsagePanel`, and `logic/usage.ts` stay, with
  no route.
- `UsagePanel` keeps its two `to="/usage"` links.
