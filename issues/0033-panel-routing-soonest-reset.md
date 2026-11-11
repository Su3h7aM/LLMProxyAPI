---
status: done
branch: issue/0033-panel-routing-soonest-reset
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel: soonest-reset next credential and reset times on the routing page

## Goal

The routing page (issue 0029) has the `soonest-reset` strategy but does not show which credential is next and when each window resets. Add that, using data the panel already has (cached quota snapshots). Decision of the coordinator: no new upstream fan-out and no new backend route in this issue.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0033-panel-routing-soonest-reset integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Read first.** `issues/0029-*.md` (follow-ups) and `issues/0026-*.md` (quota reset schedule). Find the quota store (`src/stores/useQuotaStore.ts`) and the reset schedule logic (`features/quota/resetSchedule.ts`, `quotaTimelineModel.ts`, soonest-recovery sort) that decide the soonest reset today. Check the Go selector `sdk/cliproxy/auth/selector_soonest_reset.go` for the real rule (what counts as a window, ties, credentials without data go last) and make the page logic match it.
2. **Logic.** Write a small pure module that takes the credentials of the routing tiers and the cached quota snapshots and returns, per credential: soonest reset time (or none) and the rank, and the predicted next credential per tier. Test it, including: no data, exhausted quota, ties, and multiple windows.
3. **UI.** In the routing page, when the strategy is `soonest-reset`, show the next credential and the reset time per credential in BumpyClock style (tokens only; copy patterns of the quota page for time display, for example relative time with the absolute time as title). When there is no snapshot, show a short hint to open the Quota page to load reset times (add the string in the 4 locales: en, zh-CN, zh-TW, ru). Do not fetch quota from upstream on this page.
4. **Tests.** Logic tests plus a render test of the new part if the repo has a pattern for it.
Do NOT do in this issue: new backend routes, quota page changes, other pages.

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

- 2026-11-11: Set `status: in-progress`. Branch from `integration/panel-bumpyclock`. Read `DECISIONS.md`, issues 0029 (routing board, soonest-reset follow-up) and 0026 (quota reset schedule, timeline, soonest-row highlight).
- 2026-11-11: Inventory. Quota store (`src/stores/useQuotaStore.ts`): per-provider maps keyed by `getQuotaCacheKey(file)` (plain file name except Devin/plugin multi-identity keys). Reset logic: `features/quota/resetSchedule.ts` (`collectQuotaRowInstants`, `pickSoonestRowId`, `nextRecoveryMs`) over `getQuotaObservations(provider, quota)` windows (`resetAtMs`, `remainingPercent`, `recovers`, `periodHours`); all observed windows have `recovers: true`. Go selector `sdk/cliproxy/auth/selector_soonest_reset.go`: rank known-reset first, then no data, then exhausted; longest window decides the credential reset; elapsed resets skipped; ties on ID. Routing board members of kind `credential` carry the file name in `location.name` (`key: cred:<name>`); apiKey/compat members have no quota snapshots.
- 2026-11-11: Logic (`features/routing/soonestReset.ts` + `tests/routingSoonestReset.test.ts`, 7 pass). `predictSoonestReset` mirrors the Go rule from cached snapshots only: skip elapsed windows, any zero-remaining window puts the credential last (backend reads Codex `Used-Percent >= 100`; a rejected Claude window fails the quota read so the panel sees it as no data), longest `periodHours` decides, ties on reset then key. Returns board-order ranks, per-tier next, overall next (first tier in order with a credential). `resolveCredentialSnapshots` matches store entries by physical file name (covers multi-identity keys).
- 2026-11-11: UI (`components/SoonestResetSection.tsx`, wired in `PolicyPanel` + `RoutingPage`). Shown only for `soonest-reset`: next credential with reset time, per-credential rows (relative time with absolute title, quota page pattern), exhausted / no-reset states, Quota-page link hint when nothing is cached. No fetch on this page. Token-only SCSS (token guard green). Strings added in en, zh-CN, zh-TW, ru (pure insertions, verified zero deletions).
- 2026-11-11: Tests: 7 logic + 4 render (`tests/routingSoonestResetSection.test.ts`, `renderToStaticMarkup` + `MemoryRouter` pattern). `type-check`, `eslint`, token guard, `productionLocales` green.

## Follow-ups

- No live-backend check of predicted picks against the real selector (no credentials in this environment). A manual check with two quota-loaded credentials confirms the panel pick matches `SoonestResetSelector.Pick`.
- Per-tier next predictions exist in the module and tests but the UI shows only the overall next plus the rank-sorted list. Per-tier badges on the ladder are possible later.

## Result

- Landed: soonest-reset readout on the routing page from cached quota snapshots only (no fan-out, no new routes). Pure module `apps/web/src/features/routing/soonestReset.ts` mirrors `sdk/cliproxy/auth/selector_soonest_reset.go` (known first, no data next, exhausted last; longest window decides; ties on key) and returns ranks, per-tier next, overall next. UI section in the policy column (BumpyClock tokens, quota-page time pattern: relative with absolute title), shown only for `soonest-reset`; empty state links to `/quota`. Strings in en, zh-CN, zh-TW, ru.
- Commits on `issue/0033-panel-routing-soonest-reset` over `integration/panel-bumpyclock` (`db784257..5b7a8a1a` at rebase; this file closes it): `823c06bc` status, `408b9afd` logic+tests, `75966755` UI+locales, `5b7a8a1a` log. Rebase clean (base did not move).
- Gate: `bun install --frozen-lockfile`, `bun run verify` green — 2042 tests pass, 0 fail (11 new: 7 logic, 4 render); `type-check`, `eslint`, token guard, `productionLocales` green; `vite build` one `dist/index.html` (3,487 kB).
- Known gaps: none in scope. Not merged, as ordered.
