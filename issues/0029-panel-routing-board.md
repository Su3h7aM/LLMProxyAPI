---
status: in-progress
branch: issue/0029-panel-routing-board
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel v8: routing page and soonest-reset board

## Goal

BumpyClock has no routing page. The fork backend serves `/routing/*` routes and the pre-import panel has a routing board. Build the routing page in the BumpyClock design, with the fork-only `soonest-reset` strategy (use the account whose quota window resets first, to use the window better).

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0029-panel-routing-board integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Inventory first.** In `## Log`: describe the pre-import routing feature (`git show panel-pre-bumpyclock:apps/web/src/features/routing/...`: `PolicyPanel.tsx`, `logic.ts`, `services/api/routing.ts`, `types/routing.ts`, the `routing.*` locale keys, the nav entry and the `/routing` route, the dashboard link), what the user can do there, and which backend routes it uses (`internal/api/server_management_v8.go` lines about `/routing`, handlers in `internal/api/handlers/management/`). Check the `soonest-reset` parts of commit `ecc09878` (`git show ecc09878`).
2. **API and types.** Port `services/api/routing.ts` and `types/routing.ts` (v8). Read the Go handlers for the real request and response shapes.
3. **Page.** Build the routing page in BumpyClock style: find the closest BumpyClock page (for example quota, dashboard, or a config-like page with cards and a table) and copy its layout, components, tokens, and states (loading, empty, error). It must let the user see the routing strategy and per-credential ordering or state as the pre-import board did, and select the strategy including `soonest-reset`. For `soonest-reset`, show which credential is next by soonest reset and when each window resets (use the reset data that the quota code already has; if data is missing, say what is missing in `## Follow-ups`). The strategy select also exists in the config page (issue 0022 added the option): keep one source of truth for labels.
4. **Wiring.** Add the `/routing` route and the nav entry the way BumpyClock adds pages (router, `MainLayout` nav items, page title, search index if it has one). Add the strings in the 4 locales (en, zh-CN, zh-TW, ru).
5. **Tests.** Port the pre-import routing tests that fit (logic tests are pure and port well); add tests for the new page logic.
Do NOT do in this issue: quota page changes, dashboard, credentials.

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

- 2026-11-11: Set `status: in-progress`. Branch from `integration/panel-bumpyclock`. Read `DECISIONS.md` and triage results 0015/0016/0017.
- 2026-11-11: Inventory. Pre-import routing (`git show panel-pre-bumpyclock:apps/web/...`):
  - `features/routing/`: `RoutingPage.tsx` (426 lines), `logic.ts` (518, pure board arithmetic: tiers, moves, priorities, plan, route preview), `format.ts` (30), `useDragAutoScroll.ts` (107), `components/PolicyPanel.tsx` (183, strategy radio + affinity + retry counts), `TierLadder.tsx` (215), `MemberCard.tsx` (279), `WeightDialog.tsx` (92), `ChangeReviewDialog.tsx` (76), `RoutingSaveBar.tsx` (175, floating save bar). SCSS: `RoutingPage` (655), `MemberCard` (263), `RoutingSaveBar` (165) — all raw px values, must be rewritten with design tokens (guard `tests/designTokenGuard.test.ts` rejects raw values).
  - `services/api/routing.ts` (482): board reads `GET /config` + credential list via `authFilesApi`, model names via model-definitions; save writes credentials one by one, then one `PATCH /config` with the connection guard. No `/routing/*` board endpoint exists.
  - `types/routing.ts` (168): board/draft/plan/tier/member/settings types, `ROUTING_STRATEGIES` with `soonest-reset`.
  - 94-key `routing` locale block, `/routing` route (`MainRoutes.tsx`), nav entry in gateway group, dashboard strategy label.
  - User flow: view tier ladder (priority tiers, member cards, state chips, cooldown countdown, cooldown chips, weight badges, share preview per model filter), drag/keyboard reorder, weight edit, policy select, review dialog, save bar with conflict error on stale revision.
- 2026-11-11: Backend routes (`internal/api/server_management_v8.go`): `/routing/*` is only 3 routes — `POST /routing/cooldown/reset`, `GET /routing/model-definitions/:channel`, `POST /routing/model-definitions/refresh`. The board itself uses `/config` + `/credentials`. Commit `ecc09878`: `soonest-reset` is 4th strategy (parser, select option, board mark with outer dots filled, no fixed split in preview as for `fill-first`, search entry, dashboard label). Search entry, dashboard label, `SectionNetwork` option already in this tree via issue 0022.
- 2026-11-11: Current tree gaps: no `features/routing`, no `services/api/routing.ts`, no `types/routing.ts`, no `/routing` route or nav entry, no `routing` locale keys. All page deps exist here (`ui/Button`, `ToggleSwitch`, `AutocompleteInput`, `EmptyState`, `Skeleton`, `PageTransitionLayer`, `useRevealGroup`, `useUnsavedChangesGuard`, `createSharedClock`, `authFilesApi.list/patchFields/getModelDefinitions/getModelsForAuthFile`, `guardConfigConnection`, `isModelExcluded`, stores). Closest BumpyClock page pattern: `QuotaPage` workbench shell (`PageHeader` + cards); ladder itself is bespoke and keeps the pre-import structure with token SCSS.

## Follow-ups

## Result
