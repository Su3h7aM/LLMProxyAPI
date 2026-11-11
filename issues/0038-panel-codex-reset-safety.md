---
status: todo
branch: issue/0038-panel-codex-reset-safety
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel: Codex quota reset must confirm the redeem result

## Goal

Fix a safety gap found in issue 0031 (follow-up C3): in the new panel the Codex quota reset treats an unknown redeem result as success. The pre-import panel accepted only the codes `reset` and `already_redeemed`, and then cleared the credential cooldown. Restore that, so the panel never shows or applies a reset that did not happen.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0038-panel-codex-reset-safety integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Read.** `issues/0031-*.md` (follow-up C3), the pre-import flow (`git show panel-pre-bumpyclock:apps/web/src/features/quota/providers/codex/CodexQuotaBody.tsx` and the related data and API code; search `panel-pre-bumpyclock` for `reset_not_confirmed` and `resetCooldown`), and its test `tests/codexQuotaReset.test.ts`. Find the same flow in the new panel (`features/quota/providers/codex/*`, `services/api/authFiles.ts` `resetCooldown` at `/routing/cooldown/reset`). Read the Go handler of the redeem call that the panel uses (via `/requests/api-call` to the Codex backend) only as far as the panel needs: the panel must treat only `reset` and `already_redeemed` as confirmed; every other `code`, a missing `code`, or a non-2xx answer is a failure.
2. **Fix.** On an unconfirmed result: show an error (use the existing `reset_not_confirmed` string or add it in the 4 locales; text insertion only), do not call `resetCooldown`, and do not refresh the card as if reset succeeded. On a confirmed result: keep the current consume-then-fetch behavior and call `resetCooldown` with the connection-revision guards as before.
3. **Tests.** Port `tests/codexQuotaReset.test.ts` (adapt to the new components) and add cases: unknown code, missing code, non-2xx, `already_redeemed`.
4. **Gate.** `bun run verify` green.
Do NOT do in this issue: other quota providers.

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
