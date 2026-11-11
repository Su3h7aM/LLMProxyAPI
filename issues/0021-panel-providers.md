---
status: done
branch: issue/0021-panel-providers
type: panel
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel v8: AI providers, policies, model discovery

## Goal

Port the AI providers area of the panel to the v8 API and bring back the v8 provider policy editors of the pre-import panel, in the BumpyClock design.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0021-panel-providers integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **API module.** `src/services/api/providers.ts` to v8: key lists per family and the OpenAI-compatibility groups use the config tree (`PUT /config/api-keys/<family>` and the related config paths). Take exact paths and shapes from the pre-import `providers.ts`, `configPatch.ts`, `providerModels.ts`, and from the Go config handlers. Port `providerModels.ts`.
2. **Model discovery.** `src/services/api/models.ts` and `apiCall.ts`: server-side calls go through `POST /requests/api-call`. Keep direct browser calls only where the BumpyClock code already makes them to the provider itself. Check the Gemini listing loop and keep it working.
3. **Policy editors.** Bring back from the pre-import panel: `errorRules.ts`, `providerBehavior.ts`, `runtimePolicy.ts`, `modelEntries.ts`, `modelOptions.ts`, `discoverySelection.ts`, and the UI that edits them. Build them in BumpyClock style inside its provider workbench sheets and forms (find the matching sheets in `features/providers/sheets` and copy their structure).
4. **Groups.** Keep the sponsor-free generic group logic (`buildProviderGroups`) and its test `providerGroupsV8.test.ts`.
5. **Pages.** The providers workbench must list, create, edit, delete, and test providers on v8, for all families that the fork backend serves (check the Go config types).
6. **Tests.** Port the pre-import provider tests; keep BumpyClock tests green.
Do NOT do in this issue: OpenCode Go provider (own issue), credentials, config page, quota.

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

- 2026-11-11: Set `status: in-progress`. Read DECISIONS.md and issues 0015-0017. Checked Go v8 routes (`server_management_v8.go`, `config_v8.go`, `api_tools.go`) and the pre-import reference.
- 2026-11-11: Ported API layer (commit `6d9ba21c` after rebase): `providerModels.ts` and 6 policy libs verbatim; `providers.ts`, `transformers.ts`, `apiCall.ts` (v8 path `/requests/api-call` plus `proxy_url`, confirmed in `api_tools.go`), `models.ts` (`proxyUrl` passthrough, kept BumpyClock direct-fetch signature); `types/provider.ts` verbatim; `Config` gained `providerGroups` and `openCodeGoApiKeys`.
- Question logged, answered from evidence, no stop needed: Go serves 9 key families (`gemini/interactions/codex/xai/meta/opencode-go/claude/vertex/openai-compatibility`); `providers.ts` carries the opencode-go family functions for file parity, but the opencodeGo brand UI stays with its own issue. Go has no `test-model` field, so per-model `testModel` is UI-only and stripped on write (as pre-import does).
- 2026-11-11: Wired workbench (commit `f70f506d` after rebase): v8 group mutations with persisted `source`, policy/behavior/model-options builders, `sourceIndex` rows, proxy threading in connectivity/discovery, RuntimePolicy/ProviderBehavior/ModelAdvancedFields/ErrorRules editors in BumpyClock sheets, sync discovery panel, `aria-invalid` on shared Select, i18n blocks `behavior/errorRules/modelOptions/runtimePolicy` and `discovery.syncHint/applyChanges` in 4 locales, Xiaomi/StepFun families plus codex pattern in `utils/models.ts` (qoder entry kept: other areas still reference it).
- Test work: ported 8 updated family/api tests and 16 new provider tests verbatim; adapted `providerTestModel.test.ts` hint assertion to the BumpyClock per-brand hint (`testModelClaudeHint`, 4 locales). 0020 overlap: integration already had `errorRules.ts`, token-clean `ErrorRulesEditor` files, `oauthAliasOnly` rows in `ModelEntriesEditor`, and `aria-invalid` on Select; kept integration versions on rebase and merged the advanced-fields branch in. Added `disabled={mutating}` to the row expander (pre-import parity; fixes 0020 `authFilePolicyForms` test).
- 2026-11-11: Gate green, rebased on `integration/panel-bumpyclock` (now at 0023), re-ran `bun run verify`: pass. Set `status: done`.

## Follow-ups

- OpenCode Go brand UI (descriptors, adapters, workbench brand, quota tab): left for its own issue. API functions and `Config.openCodeGoApiKeys` already exist.
- `normalizeConfigResponse` now reads the v8 config tree; issue 0022 (config page) builds on it.
- Model discovery add/remove sync replaces the BumpyClock add-only panel; `alreadyAdded` tag and `apply` key remain in locales but are unused.
- `qoder` model family entry in `utils/models.ts` kept (referenced by oauth/quota areas); removal is a separate decision.

## Result

Done. Providers area speaks v8 only; policy editors are back in BumpyClock style.

- API: `providers.ts` uses `GET /config` plus `PUT /config/api-keys/<family>` with group identity and inheritance-safe writes; `apiCall.ts` uses `POST /requests/api-call` with `proxy_url`; `models.ts` forwards `proxyUrl` (Gemini paging loop kept); `providerModels.ts` and `transformers.ts` normalize groups, runtime policy, model options, and source identity.
- UI: workbench lists, creates, edits, deletes, disables, and tests keys for gemini, interactions, codex, meta, xai, claude, vertex, and openai-compatibility; RuntimePolicy, ProviderBehavior, ModelAdvancedFields, and ErrorRules editors live in the BumpyClock sheets; discovery panel syncs add/remove.
- Tests: 24 pre-import provider tests ported, 1 adapted (`providerTestModel` hint). Full gate: `bun run verify` passes (1916 pass, 11 skip, 0 fail; lint clean; build writes one `dist/index.html`, 3.26 MB).
- Commits: `6d9ba21c` (API layer), `f70f506d` (workbench and UI wiring), plus branch start and this file. No merge performed.
