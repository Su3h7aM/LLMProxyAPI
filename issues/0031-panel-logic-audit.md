---
status: done
branch: issue/0031-panel-logic-audit
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel: restore provider-facing logic fixes that the BumpyClock base lacks

## Goal

The BumpyClock panel comes from an older base than the official panel that the fork used before. It lacks some non-UI fixes that the pre-import panel had. Example found: the Claude quota request header `User-Agent: claude-cli/...` is gone (`CLAUDE_REQUEST_HEADERS` in `src/utils/quota/constants.ts`), and the backend `POST /requests/api-call` adds no User-Agent. Find and restore such fixes.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0031-panel-logic-audit integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Diff.** Compare the non-UI logic of the pre-import panel (tag `panel-pre-bumpyclock`) with the integration branch: `src/utils/quota/*`, `src/features/quota/providers/*/data.ts` and request builders, quota parsers and validators, `src/services/api/*` request details (headers, bodies, timeouts, retries), `src/features/providers` model-discovery and test-request builders, `src/utils/*` helpers that touch upstream provider APIs. Use `git diff panel-pre-bumpyclock integration/panel-bumpyclock -- <path>` and read the history of the pre-import side (`git log panel-pre-bumpyclock -- <path>`; look for the official panel commit messages with "fix").
2. **Classify.** In `## Log`, make a table: file, delta, class. Class A = BumpyClock is newer or better, keep. Class B = the pre-import value is a fix for a provider API (header, user agent, URL, request body, response parsing, rate-limit or error handling) and BumpyClock lacks it. Class C = UI only or unclear: do not touch.
3. **Restore class B.** Apply only class B changes, one small commit per fix, with a test that pins the restored behavior. Keep the BumpyClock code style. Do not touch UI structure or styling.
4. **Report class C and doubtful items** in `## Follow-ups` for the coordinator, with file, delta, and your reason.
5. **Verify routes.** Write a small script in `/tmp` (do not commit) that extracts every `apiClient` path in `apps/web/src/services/api` and in feature modules (static and template paths) and matches it against the routes in `internal/api/server_management_v8.go`. Put the table (path, method, route found or not) in `## Log`. Report any call without a route; do not fix them unless it is a class B item.
Do NOT do in this issue: new features, cleanup, OpenCode Go (own issue).

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

- 2026-11-11: Set `status: in-progress`. Read `DECISIONS.md` and triage
  results 0015, 0016, 0017. Note: `git log panel-pre-bumpyclock -- <path>`
  yields no official-panel "fix" history; that side is squashed to the
  vendor commit `71171f19` plus fork commits, so classification below
  rests on content comparison only.
- 2026-11-11: Diffed pre-import tag vs integration branch over
  `src/utils/quota/*`, quota provider `data.ts` files, `src/services/api/*`
  request details, provider model-discovery/test builders, and `src/utils/*`
  upstream helpers. Class table (A = BumpyClock newer/better, keep;
  B = pre-import provider-API fix that BumpyClock lacks, restore;
  C = UI only or unclear, do not touch):

  | File | Delta (pre-import `-` vs integration `+`) | Class |
  | --- | --- | --- |
  | `utils/quota/constants.ts` | `User-Agent: claude-cli/2.1.280 (external, cli)` dropped from `CLAUDE_REQUEST_HEADERS` | B, restore |
  | `utils/quota/constants.ts` | Antigravity Hub and Grok version-alignment comments dropped (values same) | B-trivial, restore comments |
  | `utils/quota/constants.ts` | `XAI_USER_URL`/`XAI_SETTINGS_URL` dropped | C, needs deleted enrichment pipeline |
  | `utils/quota/constants.ts` | `OPENCODE_GO_*` dropped | Skip, issue 0030 owns |
  | `utils/quota/constants.ts` | `TYPE_COLORS` literals moved to design tokens | A, keep |
  | `utils/quota/builders.ts` | Antigravity label `description ?? window` fallback dropped | C, unclear benefit |
  | `utils/quota/builders.ts` | Kimi monthly `periodHours` 720 dropped, ratio clamped | A, current test pins `null`, keep |
  | `utils/quota/builders.ts` | `resolveXaiSubscriptionPlan`, `prepaidBalanceCents` dropped | C, tied to removed enrichment |
  | `utils/quota/planTier.ts` | Elite Pro 500 (`promax`) changed to Pro 20x (`pro`) | C, semantic conflict, current test pins new |
  | `utils/quota/validators.ts` | `isOpencodeGoFile`/`BUILT_IN_*` dropped, copilot/qoder added | Skip, issue 0030 / Plus surface |
  | `quota/providers/claude/data.ts` | `resolveClaudeMaxTier` (Max 5x/20x) dropped, tests deleted, i18n keys orphaned | B, restore resolver and tests |
  | `quota/providers/claude/data.ts` | Dollar-denominated `iguana_necktie` handling dropped | C, types changed too |
  | `quota/providers/codex/data.ts` | Redeem `code` check and `resetCooldown` orchestration dropped | C, fork-backend orchestration |
  | `quota/providers/kimi/data.ts`, `services/api/kimiQuota.ts` | `fetchKimiPlanName`/`parseKimiPlanName` dropped | C, needs deleted enrichment |
  | `quota/providers/xai/data.ts` | Subscription-plan `enrichQuota` dropped | C, needs deleted enrichment |
  | `quota/providers/meta,devin,antigravity,plugin/data.ts` | Only `getObservations` added or generic rewrite | A, keep |
  | `services/api/models.ts` | Base-URL normalizer, signal/timeout added | A, keep |
  | `services/api/logs,plugins,oauth,client.ts` | `setRequestLog`, store refresh, SIWC, `x-server-*` narrowing | A or other issues, keep |
  | `features/providers` discovery/test builders | No logic diff (icon sizes only) | None |
  | `utils/connection,format,constants`, `cardTier.ts`, `quotaEnrichment.ts` | Decoupling, formatting, deleted enrichment | Skip or C |

- 2026-11-11: Route check with `/tmp/opencode/route_check_0031.py`
  (static plus template `apiClient` paths in `services/api` and feature
  modules, matched against `internal/api/server_management_v8.go`).
  Result: every backend-facing call has a v8 route. Script false
  positives, all verified served: `/plugins/store` plus `?refresh=true`
  query, `/plugins/store/:id/install` via template, `/oauth/callback`
  (engine-level POST with prefix), all `/config/*` tree paths via the
  `Handle /config/*path` wildcard, form posts `/credentials` and
  `/oauth/import?provider=vertex`. Upstream calls via `api-call`
  (Claude/XAI/Kimi/Codex/Antigravity/Meta/Devin quota, model discovery)
  need no backend route. No call without a route; nothing to fix.

## Follow-ups

- C1: xAI subscription plan (`XAI_USER_URL`, `XAI_SETTINGS_URL`,
  `resolveXaiSubscriptionPlan`, `planLabel`/`planTier`,
  `prepaidBalanceCents`) with `tests/xaiQuotaEnrichment.test.ts`.
  Needs the deleted `quotaEnrichment.ts` pipeline; the BumpyClock side
  replaced it with `canObserve`/`observation`/`healthTest`. Restore
  only with a new observation, not by reverting.
- C2: Kimi plan name (`fetchKimiPlanName`, `parseKimiPlanName`,
  `tests/kimiPlan.test.ts`). Same deleted-pipeline reason as C1.
- C3: Codex reset flow (redeem `code` must be `reset`/`already_redeemed`,
  else `reset_not_confirmed`; then `resetCooldown` with connection
  guards; `tests/codexQuotaReset.test.ts`). Fork-backend
  orchestration; the integration side simplified reset to
  consume-then-fetch. Note: unknown `code` values now count as
  success. Needs a decision before any change.
- C4: Claude dollar-denominated `iguana_necktie` handling
  (`isDollarDenominatedWindow`, cloud-session-credits row,
  `limit_dollars` types). Types changed on the integration side;
  needs a decision.
- C5: Kimi monthly `periodHours`. Pre-import invented 720; the
  integration side pins `null` ("without inventing a fixed period").
  Kept as is.
- C6: `planTier.ts` elite conflict: pre-import Pro 500 (`promax`),
  integration Pro 20x (`pro`). Current `quotaPlanTier.test.ts` pins
  the new mapping. Needs upstream evidence; not changed.
- C7: Antigravity `labels.ts` machinery (bucket label keys, group
  models with "Gemini Argon") and the bucket `description ?? window`
  label fallback. UI/i18n feature tied to bodies; not restored.
- C8: Backend version drift: panel pins Antigravity Hub `2.23.0`
  while `internal/misc/antigravity_version.go` fallback is now
  `2.29.0` (floor statement still `2.23.0`, so no break). Watch
  only; coordinator decides if the panel pin should move.
- Unconfirmed: whether Anthropic still accepts `claude-cli/2.1.280`
  (restored value is the last known-good fork value; backend proxy
  default is `2.1.70`); whether Grok still accepts versions above
  `1.0.13` (comment claim, not re-verified live); dynamic quota
  call counts at runtime (static analysis only, no live backend).

## Result

- Landed: two class-B restores, one small commit each, both with
  pinning tests. (1) `fix(panel): send claude-cli User-Agent on
  Claude quota requests` (`1219a8b0`): `User-Agent:
  claude-cli/2.1.280 (external, cli)` back in
  `CLAUDE_REQUEST_HEADERS` plus the Hub/Grok alignment comments,
  with new `tests/claudeRequestHeaders.test.ts`.
  (2) `fix(panel): separate Claude Max 5x from Max 20x by
  rate-limit tier` (`1c869d9e`): `resolveClaudeMaxTier` back in
  `quota/providers/claude/data.ts` plus the deleted
  `claudePlanType.test.ts` cases (`plan_max20`/`plan_max5` i18n
  keys were orphaned). BumpyClock style kept; no UI changed.
- Gate: `bun install --frozen-lockfile`, `bun run verify` passes
  (2042 tests, 245 files, lint clean, `dist/index.html` 3.48 MB),
  `bun run type-check` clean. Area tests
  (`claudeRequestHeaders`, `claudePlanType`, `claudeOAuthUsage`,
  `claudeFableQuota`: 46 pass) green after each commit.
- Route check: every backend-facing `apiClient` call in the area
  has a v8 route; no missing route. Script kept out of the repo
  at `/tmp/opencode/route_check_0031.py`.
- Known gaps: none in the restored behavior. Open points are all
  in `## Follow-ups` (C1-C8): xAI plan, Kimi plan name, Codex
  reset flow, Claude dollar windows, Kimi monthly period, planTier
  elite conflict, Antigravity labels, Hub version drift.
