---
status: done
branch: issue/0016-triage-panel-fork-features-and-build
type: triage (read-only, no code changes)
agent: deepseek-v4.1-flash
---

# Triage panel migration B: fork feature inventory, stack, and build pipeline

## Context

The fork dashboard (`apps/web`, vendored from the official Management Center,
pin `953fb24`) will be replaced by the BumpyClock Management Center. Read
`DECISIONS.md` (entry 2026-11-11 about the dashboard) and `UPSTREAM_BASELINE.md`.
The fork serves the panel from the proxy binary (`apps/web/panel.go` embeds
`dist/management.html`). The fork targets servers and containers, API only.

References (read-only, never edit or commit there):

- BumpyClock panel: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center`
- Upstream proxy: `/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI`
- Fork: your checkout. The official panel at the vendored pin is the baseline of `apps/web`.

## Goal

List what the fork panel has that must survive the replacement, how the stack differs, and what the build and embed pipeline needs.

## Tasks

1. **Fork feature inventory.** The fork changed the vendored official panel in these commits (read them with `git show <hash>` in your checkout): `760d7f3a` (OpenCode Go provider and quota tab), `c712e17f` (OpenCode Go localization and rebrand text), `c40624fc` (rebrand console chrome), `f3912994` (load OpenCode Go keys and quota from config), `ecc09878` (soonest-reset in the routing panel), `21f7ad81`, `21b6926f`, `4d1b8e47`, `4f0ed0ce` (sponsor and promo removal), `2dd22671` (latest release check on the fork repo; check which files are web and which are Go), `5247f14c` (vendored panel docs and CI). For each: files changed under `apps/web`, the behavior to keep, and where the equivalent place is in the BumpyClock source tree (`src/features/*`, `src/pages`, `src/services/api`, `src/i18n`, `src/stores`, `src/components`). If BumpyClock has no equivalent place, say so. Also list any other fork-only difference between `apps/web` and the official panel at the pin that you find (the official pin is in `UPSTREAM_BASELINE.md`; you may fetch it from https://github.com/router-for-me/Cli-Proxy-API-Management-Center into `/tmp` and diff).
2. **Stack delta.** Compare `apps/web` and the BumpyClock root: `package.json` (dependencies, scripts, `packageManager`), `bun.lock`, `vite.config.ts` (single file output? base path? plugins), TypeScript config, eslint and prettier config, test runner and test count, design-token scripts (`scripts/design-tokens`), `src/design-system`, `docs`, `plans`. Say what is new in BumpyClock. Note the version pins (bun, vite, typescript, react).
3. **Build check in `/tmp`.** Copy the BumpyClock panel to `/tmp`, run `bun install --frozen-lockfile`, `bun run build`, `bun test`, `bun run lint` (report time and results; if something fails say why). Report the output files and sizes, and check that the output is one HTML file that can be embedded as `management.html`. Report the HTML `<title>`.
4. **Pipeline impact.** List every file in the fork that depends on the panel layout or tooling and what would change: `apps/web/panel.go`, `scripts/build-web.sh`, `scripts/stage-panel.sh`, `scripts/check-panel.sh` (it checks the `<title>`), `Containerfile` (bun image, file copies), `.github/workflows/*`, `.mise/`, `.gitignore`, `package.json` at the root, docs that mention the panel. Do not change them.
5. **Import plan proposal.** Propose how to bring the BumpyClock source into `apps/web`: replace the tree, what fork-only files in `apps/web` to keep (for example `panel.go`, `AGENTS.md`, license files), what files from BumpyClock not to bring (`.github`, `plans`, sponsor assets, `.cpamc-fork-upstream.env`, `CHANGELOG.md` if it is theirs, etc.), and an ordered list of small steps, each step small enough for one issue.

## Result format

Tables per task. End with: proposed step list, risks, and a list of claims you could not confirm.


## Rules

- Do not change code. Your only edits: this issue file (`## Log

- 2026-11-11: Branch renamed to `issue/0016-triage-panel-fork-features-and-build`. Status in-progress.
- Pin data confirmed: `UPSTREAM_BASELINE.md:9-33` gives official panel `953fb24` and BumpyClock `ce5f6179`. Fork vendored the official panel in `71171f19`; therefore `git diff 71171f19 HEAD -- apps/web` is the exact fork-only panel delta (109 files: 12 added, 22 deleted, 75 modified).
- Read the 11 fork commits with `git show`. Read the BumpyClock clone (read-only, 1275 commits) and compared it to official `953fb24` with `diff -rq` in `/tmp` (712 differences: 228 only in BumpyClock, 152 only in official, 332 modified files).
- Scratch builds in `/tmp` only: `/tmp/t16/bumpy` (BumpyClock tree, no `.git`), `/tmp/t16/forkweb/web` (current `apps/web` copy), `/tmp/t16/official` (official panel clone at the pin). No `node_modules` or `dist` added to the checkout.
- Main finding: the BumpyClock tree is a **v0** panel (`src/utils/constants.ts:17` = `/v0/management`) and it is older than the official pin in feature coverage (no `features/routing`, no `features/oauth`, no `features/logs`, fewer config sections, 4 locales). The import is therefore a UI replacement **plus** a v0-to-v8 API port **plus** a re-port of the fork features. Evidence in `## Result`.
- Build of BumpyClock passes in `/tmp`: install, build (one 3.49 MB HTML), 1740 tests, lint clean, design tokens up to date.
- Fork panel (`apps/web`) baseline in `/tmp`: 1885 tests, 1 failure that already exists (`tests/authFileCooldowns.test.ts:228`, `auth_files.cooldown_reset_hint` is not rendered).
- Also found: `scripts/check-panel.sh:37` expects `<title>CLI Proxy API Management Center</title>` but `apps/web/index.html:8` is `LLM Proxy API Management Center`, so `mise run check-panel` must fail today. Not changed (read-only issue).
- Wrote `## Result`.

## Follow-ups

- `scripts/check-panel.sh:37` does not match the rebranded `<title>`; the check fails today.
- `apps/web/tests/authFileCooldowns.test.ts:228` fails on the current tree.
- `.mise/config.toml` has no bun pin although `scripts/build-web.sh` needs bun.
- Three locales (ko, tr, vi) exist only in the vendored official panel; decide before the tree is replaced.
- BumpyClock's Copilot and Qoder quota providers need backend routes that this fork does not serve.

## Result

Read-only triage. No code changed. Scratch copies: `/tmp/t16/bumpy`, `/tmp/t16/forkweb/web`, `/tmp/t16/official`.
Fork panel baseline for every "delta" below is `git diff 71171f19 HEAD -- apps/web` (109 files).
BumpyClock reference is `/.refs/BumpyClock-Cli-Proxy-API-Management-Center` at `ce5f6179`.

### Summary of the finding that changes the plan

| Claim | Evidence |
| --- | --- |
| The BumpyClock tree speaks the **v0** Management API, the fork requires **v8** only. | `src/utils/constants.ts:17` in both trees: BumpyClock `'/v0/management'`, fork `'/v8/management'`. Endpoint sets differ: `comm -23 bumpy-endpoints.txt official-endpoints.txt` gives `/auth-files`, `/api-keys`, `/api-call`, `/logs`, `/request-log`, `/usage`, `/plugin-store`, `/latest-version`; official has `/credentials`, `/config/...`, `/observability/...`, `/requests/api-call`, `/server/latest-version` (see `internal/api/server_management_v8.go:12-60`). |
| BumpyClock is **older** than the official pin in feature coverage, not newer. | `diff -rq` of the two trees: `Only in official/src/features: logs`, `oauth`, `routing`; `Only in official/src/features/config/components/sections: SectionDiscovery.tsx, SectionOAuthBehavior.tsx`; BumpyClock has 4 locales, official has 7 (`Only in official/src/i18n/locales: ko.json, tr.json, vi.json`). |
| The import therefore has three jobs: replace the UI tree, port the API layer v0 to v8, and re-port the fork features. | Tables 1a, 2, 4, 5. |
| BumpyClock's API client has no connection-revision guard. | `diff -u official/src/services/api/client.ts bumpy/src/services/api/client.ts`: BumpyClock omits `connectionRevision`, `getConnectionRevision`, `captureConnection`. |
| Two quota providers in BumpyClock need backend routes that the fork does not have. | BumpyClock `src/services/api/copilot.ts` calls `/copilot-quota`; `providers/qoder/data.ts` uses a snapshot. `grep -rn "copilot\|qoder" --include=*.go internal` finds no quota route. |

### Task 1a — Fork panel commits

| Commit | Files under `apps/web` | Behavior to keep | BumpyClock place, or gap |
| --- | --- | --- | --- |
| `760d7f3a` OpenCode Go provider and quota tab | 27: `features/providers/{descriptors,adapters,types,brandLogos,useProviderWorkbench}.ts`, `features/quota/{constants,logic,resetSchedule,quotaTimelineModel}.ts`, `features/quota/providers/{index,types}.ts`, new `features/quota/providers/opencodeGo/{OpencodeGoQuotaBody.tsx,data.ts}`, `features/authFiles/constants.ts`, `services/api/providers.ts`, `stores/{useConfigStore,useQuotaStore}.ts`, `types/{config,quota}.ts`, `utils/quota/{constants,validators}.ts`, icons, 2 tests | Key-only provider `opencodeGo` (API key, prefix, priority, excluded models; no URL or headers) on the v8 key family `opencode-go-api-key`; quota tab that reads the fixed Go usage route through api-call with a session id, 5-hour/weekly/monthly windows, timeline lane, auth-file quota section | All targets exist: `src/features/providers/*`, `src/services/api/providers.ts`, `src/stores/useConfigStore.ts`, `src/features/authFiles/constants.ts`, `src/utils/quota/*`. Quota needs one extra registration point (`src/features/quota/providers/registry.ts:13-26`, `providers/index.ts` `withBody`) and must emit `observations` (`providers/observations.ts`). Provider side must land after the v8 key API port (BumpyClock uses `/api-keys`, not v8 config groups). |
| `c712e17f` OpenCode Go localization and rebrand | 7 locale files | `opencode_go_quota` block, `providerNames.opencodeGo`, `auth_files.filter_opencode-go`; brand text CPA→LPA, CPAMC→LPAMC, CLI Proxy API→LLM Proxy API | `src/i18n/locales/{en,ru,zh-CN,zh-TW}.json` only. ko/tr/vi do not exist. Code-side brand strings also in `src/main.tsx`, `src/pages/{LoginPage,SystemPage}.tsx`, `src/pages/OAuthPage.tsx:585`, `src/components/layout/MainLayout.tsx`. |
| `c40624fc` Rebrand console chrome | 9: `index.html`, `src/main.tsx`, `components/layout/MainLayout.tsx`, `pages/{LoginPage,SystemPage}.tsx`, `pages/login/DotMatrixHero.tsx`, `features/authFiles/components/AuthFileModelsModal.tsx`, `features/oauth/hooks/useOAuthFlows.ts`, `tests/loginForm.test.ts` | Title, `document.title`, login brand name, logo alt, about links | Present for `index.html:8` (`<title>CLI Proxy API Management Center</title>`), `main.tsx:7`, `MainLayout.tsx`, `LoginPage.tsx`, `SystemPage.tsx`. Gap: no `pages/login/DotMatrixHero.tsx` (no `pages/login` dir), no `features/oauth/hooks/useOAuthFlows.ts` (same default string lives in `pages/OAuthPage.tsx:585`). |
| `f3912994` Load OpenCode Go keys from config | `services/api/transformers.ts` (+3) and `internal/watcher/synthesizer/config.go` (Go) | `openCodeGoApiKeys` reaches the UI from the v8 config tree | `src/services/api/transformers.ts` and `src/types/config.ts` exist, but BumpyClock's transformers are v0-shaped (`diff` shows `normalizeProviderGroups`, `providerModels` imports removed). Target only after the v8 API port. Go side is unaffected by the panel import. |
| `ecc09878` Soonest-reset in the routing panel | 18: `features/routing/{components/PolicyPanel.tsx,logic.ts}`, `features/config/{searchIndex.ts,components/sections/SectionNetwork.tsx}`, `features/dashboard/DashboardPage.tsx`, `hooks/useVisualConfig.ts`, 7 locales, `services/api/routing.ts`, `types/{routing,visualConfig}.ts`, 2 tests | `soonest-reset` as a fourth strategy: parser, select option, board mark, no fixed split in the preview, search entry, dashboard label | Partial. `src/hooks/useVisualConfig.ts:443-447` (parser), `src/types/visualConfig.ts:3` (union), `src/features/config/components/sections/SectionNetwork.tsx:132-146` (select), `searchIndex.ts` exist. **Gap:** no `features/routing` and no `services/api/routing.ts` or `types/routing.ts`, so the routing board and its preview have no home. `tests/routingBoard.test.ts` cannot port; `tests/visualConfigRoutingStrategy.test.ts` can. |
| `21f7ad81` Sponsor removal, OAuth and config | 12: `features/oauth/*` (SponsorCard and SCSS, `_sponsor.scss`), `features/config/sponsors.ts`, `features/config/components/fields/sharedFields.tsx`, `features/oauth/providers.ts`, tests | No sponsor cards, no sponsor links, no sponsor-provider list | BumpyClock has no `features/oauth` (page is `src/pages/OAuthPage.tsx`), but it still has `src/features/config/sponsors.ts` (20 lines) and `src/features/config/components/fields/sharedFields.tsx`. Removal must be redone on the new tree. |
| `21b6926f` Sponsor promo links and campaign | 9: `components/layout/MainLayout.tsx`, `features/providers/ProvidersWorkbenchPage.tsx`, `components/ProviderResourcePanel.tsx`, `SponsorCampaignCard.*`, `SponsorQuickStartPanel.*`, `router/MainRoutes.tsx`, test | No campaign card, no quick-start sponsor panel, no sidebar promo | BumpyClock still has `src/features/providers/components/SponsorQuickStartPanel.{tsx,module.scss}`. `SponsorCampaignCard` is absent. Removal must be redone. |
| `4d1b8e47` Sponsor-matching endpoints stay in generic groups | `features/providers/useProviderWorkbench.ts`, `tests/providerGroupsV8.test.ts` | Sponsor-looking endpoints must not vanish from `openai-compatibility` groups | `src/features/providers/useProviderWorkbench.ts:360` `buildProviderGroups` exists; `tests/providerGroupsV8.test.ts` is absent in BumpyClock (only `providerQuickStartSponsor.test.ts`). Port the logic and recreate the test. |
| `4f0ed0ce` Sponsor providers out of the dashboard | 39: `providers/{adapters,descriptors,brandLogos,sponsor,sponsorDefinitions,sponsorMutationRecovery,fennoAI,kimi,qiniuCloud,aiCodeMirror}.ts`, `sheets/forms/{SponsorProviderForm,useSponsorUsageCheck}`, `ProviderResourceTable`, `ResourceDetailView`, 7 locales, tests | Provider list holds only backend-served brands | All those files still exist in BumpyClock (`providers/sponsor.ts` 196 lines, `SponsorProviderForm.tsx`, `sponsorDefinitions.ts`). Removal must be redone, and BumpyClock tests that assert sponsor behavior drop out (`sponsorAggregation`, `sponsorCustomEndpoint`, `sponsorMutationRecovery`, `providerQuickStartSponsor`, `round1ConfigCopySponsor`). |
| `2dd22671` Latest release on the fork repository | `apps/web/tests/versionFailure.test.ts` (17 lines); the rest is Go (`internal/api/handlers/management/config_basic.go:22`) | Release check must hit the fork repo; a 404 or 502 must not report a release | BumpyClock `src/services/api/version.ts:8` calls `'/latest-version'`; fork calls `'/server/latest-version'` (`tests/versionFailure.test.ts` asserts it). Change with the API port; the test can port. |
| `5247f14c` Vendored panel docs and CI | `apps/web/{AGENTS.md,README.md,README_CN.md,.github/workflows/{ci,release}.yml}` | Fork-owned panel guidance and docs | Fork-authored. `AGENTS.md` must be rewritten for the new tree (BumpyClock's own `AGENTS.md` says v0 and 4 locales). Nested `apps/web/.github/workflows` has no runtime effect; BumpyClock brings 5 more workflows (`runs-on: boat`, bun 1.3.14) that must not be copied. |

### Task 1b — Other fork-only differences beyond those commits

`git diff --name-status 71171f19 HEAD -- apps/web` gives 12 A, 22 D, 75 M. The 12 additions are the OpenCode Go pair, 2 icons, 2 tests, `versionFailure.test.ts`, and the 5 `5247f14c` docs/CI files. The 22 deletions are the sponsor files listed above. No other fork-only panel change exists. The only fork panel change outside `apps/web` in this set is `internal/api/handlers/management/config_basic.go` plus its test (`2dd22671`).

| Area | Fork panel | BumpyClock | Note |
| --- | --- | --- | --- |
| Routes | `/dashboard, /providers (ai-providers, quick-start), /auth-files, /oauth, /quota, /config, /logs, /system, /routing` | same minus `/routing`, plus `/usage` | `src/router/MainRoutes.tsx` in both |
| API modules | 30 files, 5033 lines, 51 `apiClient` calls | 25 files, 3560 lines, 63 calls | v8 vs v0 |
| Provider policy editors | `errorRules.ts`, `providerBehavior.ts`, `runtimePolicy.ts`, `modelEntries.ts`, `modelOptions.ts`, `discoverySelection.ts`, `services/api/{configPatch,configValue,providerModels}.ts` | absent | The fork backend has these v8 fields; the new panel would lose the editors. |
| Auth-file policy | `features/authFiles/{credentialPolicy.ts,manualRefresh.ts,components/AuthFilePolicyFields,AuthFileRefreshResults}`, `services/api/authFilePolicyAdapter.ts` | absent | Same concern. |
| Claude reset grants | `features/quota/providers/claude/resetGrant*`, `services/api/claudeResetGrants.ts` | absent | Same concern. |
| Locales | 7 | 4 | ko, tr, vi lost |
| Test files | 191 items | 225 items | not a like-for-like count |

### Task 2 — Stack delta

| Item | Fork `apps/web` | BumpyClock root | New in BumpyClock |
| --- | --- | --- | --- |
| `packageManager` | `bun@1.4.3` | `bun@1.3.14` | lower pin; the root `Containerfile:1` uses `oven/bun:1.4.4` and `bun install --frozen-lockfile` succeeds with 1.4.4 (tested) |
| Scripts | `build: tsc && vite build`, `type-check: tsc --noEmit` | `build: bun run type-check && vite build`, `type-check` over `tsconfig.app.json` + `tsconfig.node.json`, plus `tokens`, `tokens:check`, `tokens:migrate` | yes |
| Dependencies added | — | `recharts 3.10.1`, `react-is 19.2.7` | yes |
| Dependencies dropped | `@noble/hashes ^2.4.0` | — | fork-only (`apps/web/src/features/config/apiKeyNames.ts:1`) |
| Version pins | react `^19.3.1`, vite `^8.3.5`, typescript `^6.0.3`, axios `1.20.0` | react `^19.2.7`, vite `^8.1.4`, typescript `^6.0.3`, axios `1.18.1` | older |
| `vite.config.ts` | react + `vite-plugin-singlefile` (`removeViteModuleLoader`), `assetsInlineLimit`/`chunkSizeWarningLimit` 1e8, `cssCodeSplit:false`, `rolldownOptions.output.codeSplitting:false`, alias `@`, SCSS inject `@/styles/variables.scss` | same plus `designTokens(__dirname)` plugin | yes; the plugin calls `execFileSync('bun', …)` on `buildStart`, so `bun` must be on `PATH` |
| TypeScript | one `tsconfig.json` (+ `tsconfig.node.json`) | `tsconfig.json` with project references, `tsconfig.app.json`, `tsconfig.node.json` | yes |
| eslint / prettier | `eslint.config.js`, `.prettierrc` | same, plus `react-hooks/set-state-in-effect: off`, `.prettierignore` for generated tokens | small |
| Test runner | `bun test` | `bun test` + `bunfig.toml` preload `tests/helpers/timezone.ts` | yes |
| Test count | 1885 tests in 187 files, 1 failure | 1740 tests in 215 files, 0 failures | different, not comparable |
| Design system | `src/styles/{variables,themes,global,components,layout,mixins,reset}.scss` | `src/design-system/{tokens.ts (1051), index.ts (149), generated/_tokens.scss, generated/_themes.scss}` + `src/styles/variables.scss` that only forwards tokens + `scripts/design-tokens/` (9 files: generate, guard, classify, migrate, scanner, css-normalize, vite) + `tests/designTokenGuard.test.ts` that rejects raw values | yes, major |
| Docs | none in `apps/web` before `5247f14c`; then `README.md`, `README_CN.md`, `AGENTS.md` | `docs/design-tokens.md`, `docs/workspace-redesign.md`, `docs/screenshots/` | yes |
| Plans | none | `plans/260710-0828-management-center-issue-13/`, `plans/reports/` | yes |
| Changelog | none | `CHANGELOG.md` (BumpyClock's own) | yes |
| CI | `apps/web/.github/workflows/{ci,release}.yml` (inert) | `.github/workflows/{ci,pr-test-build,release,sync-release-tag,upstream-sync}.yml`, all `runs-on: boat`, bun 1.3.14 | yes, do not copy |
| Fork metadata | none | `.cpamc-fork-upstream.env` (`UPSTREAM_TAG=v1.24.2`, `UPSTREAM_COMMIT=4530da27`), `assets/apimart-*.png`, sponsor section in `README.md` | yes, do not copy |
| Git ignore | root `.gitignore` covers `node_modules/`, `apps/web/dist/` | own `.gitignore` ignores `AGENTS.md`, `CLAUDE.md`, `skills`, `design-review/`, `api.md`, `style.md` | copy with care: an `AGENTS.md` entry makes new guidance untracked |
| Size | 498 files, 84,631 lines under `src` | 515 files, 88,495 lines under `src` | similar size, different content |

### Task 3 — Build check in `/tmp`

Host: bun 1.4.4 at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin/bun`. Copy: `/tmp/t16/bumpy` from `cp -r` of the clone, `.git` and `node_modules` removed.

| Step | Command | Result | Time |
| --- | --- | --- | --- |
| Install | `bun install --frozen-lockfile` | 272 packages, no error (warm cache) | 0.26 s |
| Build | `bun run build` | `dist/index.html 3,489.89 kB`, gzip 1,095.28 kB, `built in 2.16s` | 9.89 s total (2 × `tsc --noEmit`, then vite; 1530 modules) |
| Test | `bun test` | 1740 pass, 0 fail, 12,572 `expect()` calls, 215 files | 4.12 s |
| Lint | `bun run lint` | no output, exit 0 | 14.48 s |
| Tokens | `bun run tokens:check` | `Design token output is up to date.` | — |

| Artifact check | Result |
| --- | --- |
| Files in `dist/` | one: `index.html` (3,489,899 bytes) |
| Inline script count | 2 `<script` matches; `vite:singlefile` logs `Inlining: index-*.js` and `style-*.css` |
| Embeddable as `management.html` | yes: `cp dist/index.html dist/management.html` is all that `scripts/stage-panel.sh:6` needs |
| `<title>` | `<title>CLI Proxy API Management Center</title>` (`index.html:8`) |
| `__APP_VERSION__` in the copy | `dev`, because the copy has no `.git`; inside the fork, `git describe --tags` returns the fork CalVer tag |
| Side effect | `designTokens` rewrites `src/design-system/generated/*` on every build; output was byte-identical to the clone |

### Task 4 — Pipeline impact

| File or area | Current dependency | What the import changes | Must change? |
| --- | --- | --- | --- |
| `apps/web/panel.go:8` | `//go:embed dist/management.html` | none | no |
| `scripts/build-web.sh:4-6` | `cd apps/web`, `bun install` (not frozen), `bun run build` | keep; `bun` must be on `PATH` for the token plugin; a frozen install is better for CI | small |
| `scripts/install.sh:4-5` | `cd apps/web`, `bun install` | same | small |
| `scripts/stage-panel.sh:6` | `cp dist/index.html dist/management.html` | none | no |
| `scripts/check-panel.sh:37` | `test "$TITLE" = "<title>CLI Proxy API Management Center</title>"` | BumpyClock title matches today; **the fork already fails this check** because `apps/web/index.html:8` is `LLM Proxy API Management Center` | yes (rebrand) |
| `Containerfile:1` | `oven/bun:1.4.4-debian` | lockfile came from bun 1.3.14; frozen install with 1.4.4 works | recommended: pin one bun everywhere |
| `Containerfile:9-16` | copy `package.json` and `bun.lock`, `bun install --frozen-lockfile`, copy the rest, `bun run build && cp dist/index.html dist/management.html` | new `bunfig.toml`, `scripts/design-tokens/`, `src/design-system/generated/` are covered by the second copy; `bun` is in `PATH` in the image | no change needed, verify once |
| `.github/workflows/container-image.yml` | builds only on `v*` tags | no web verify job exists; a broken panel only appears at release | recommended |
| `.github/workflows/release.yaml` | tags only | none | no |
| `apps/web/.github/workflows/*` | vendored at `5247f14c`, nested, inert | BumpyClock adds 5 workflows with `runs-on: boat` | do not copy |
| `.mise/config.toml` | `go 1.27.3`, `node 26`; **no bun** | `build-web` relies on an external bun | recommended: add the bun pin |
| `.gitignore` | `node_modules/`, `apps/web/dist/` | keep; merge the useful BumpyClock entries, drop `AGENTS.md` and `CLAUDE.md` so guidance stays tracked | yes |
| root `package.json:7` | `"build": "bun --cwd apps/web run build"` | none | no |
| `README.md:45-70` | panel section, mise task list | text still true; add the new panel origin and the theme note | yes |
| `AGENTS.md`, `UPSTREAM_BASELINE.md`, `DECISIONS.md` | pin and lineage text | add the BumpyClock pin as a vendored source and the v8-only rule for the new tree | yes |
| `internal/managementasset/updater.go:1-6` | embedded panel, no runtime download | none | no |
| `.containerignore` | `apps/web/dist/`, `node_modules` | none | no |

### Task 5 — Import plan proposal

| Decision the coordinator must make | Options with evidence |
| --- | --- |
| Replace the tree, or transplant BumpyClock UI onto the current API layer | Replace: 88k vs 84k lines, but then v0 to v8 API work (`services/api`, 25 files) is required. Transplant: the current v8 layer expects v8 shapes (`transformers.ts`, `auth-index`, `next-cursor`) that BumpyClock components do not read. Both are large; replace is the smaller total diff. |
| Keep the routing page | BumpyClock has none. Keep it by porting `features/routing` + `services/api/routing.ts` + `types/routing.ts` onto the new tree, or drop it and keep only the `soonest-reset` config option. The fork's own backend serves `/routing/*` routes (`server_management_v8.go:29-36`). |
| Keep the v8 provider policy, auth-file policy, and Claude reset-grant editors | Absent in BumpyClock. Keep by porting those modules, or accept the loss. |
| Locales | Keep 4 (BumpyClock) or port ko/tr/vi from `apps/web` (7 files, ~2.4k lines each). |
| Where the import lands | Each fork commit must build. The import breaks the panel against the fork backend until step 3 is done, so it cannot land on `main` as one commit. |

Keep from the fork: `apps/web/panel.go`, `apps/web/LICENSE` (MIT, Router-For.ME, same as BumpyClock's), rewritten `apps/web/AGENTS.md`, `apps/web/README.md`/`README_CN.md` (rewritten), the OpenCode Go icons, and the v8-only API modules needed by the fork features.
Do not bring: BumpyClock `.github/`, `plans/`, `CHANGELOG.md`, `.cpamc-fork-upstream.env`, `assets/apimart-*.png`, the sponsor section of its `README.md`, its `AGENTS.md` (v0 text), and its `.gitignore` lines for `AGENTS.md` and `CLAUDE.md`.

### Proposed step list (one issue each)

1. Fix the panel gate: update `scripts/check-panel.sh:37` to the current heavy brand title and record the existing failing fork web test (`tests/authFileCooldowns.test.ts:228`). Low risk, independent.
2. Import step A: replace the `apps/web` tree from the pin with BumpyClock files, keep the fork-only files above, and set the `<title>`, `main.tsx` `document.title`, and `loginForm` brand test in the same commit. Gate: `bun install --frozen-lockfile`, `bun run build`, one `dist/index.html`, `stage-panel`, `check-panel`. Panel not yet compatible with the backend: work on a branch, do not merge to `main`.
3. Import step B: switch the API layer to v8 — `utils/constants.ts` prefix, `/credentials`, `/config`, `/observability/logs`, `/requests/api-call`, `/server/latest-version`, `/plugins/store`, and every module in `services/api` (63 call sites). Gate: tests.
4. Import step C: config page over the v8 config tree; bring `configPatch.ts`, `configValue.ts`, and the missing sections (`SectionDiscovery`, `SectionOAuthBehavior`) or drop them by decision.
5. Import step D: auth files over `/credentials` (status, fields, refresh, models, download, batch upload and delete); decide on the policy editors.
6. Import step E: logs and plugins over `/observability/*` and `/plugins/*`; keep plugin capability gating.
7. Import step F: provider workbench over v8 key groups; remove the sponsor brands and the sponsor campaign; port the `4d1b8e47` group fix and its test.
8. Import step G: OpenCode Go provider and quota tab, including `opencode-go-api-key`, `registry.ts`, `observations`, icons, and auth-file filter.
9. Import step H: `soonest-reset` in parser, union, `SectionNetwork`, and search index; routing board by decision.
10. Import step I: brand and i18n pass (LPA/LPAMC text, 4 or 7 locales), and the release-check test.
11. Import step J: repository guidance and pin docs (`AGENTS.md`, `README.md`, `UPSTREAM_BASELINE.md`, `DECISIONS.md`), remove BumpyClock CI and plan files, and pin bun in `.mise` and `Containerfile`.
12. Add a root CI job that runs `bun install --frozen-lockfile` and `bun run verify` in `apps/web` (optional, avoids a release-time surprise).

### Risks

| Risk | Evidence |
| --- | --- |
| Feature loss if steps 4 to 9 are skipped: no routing board, no provider error rules and runtime policy editors, no credential policy fields, no Claude reset grants. | `Only in official/src/...` list in Task 1b |
| Silent data mismatch during the v0 to v8 port: BumpyClock reads `next-cursor` and `files` from `/request-error-logs`; the fork's `logs.ts` reads v8 shapes. | `diff -u official/src/services/api/logs.ts bumpy/src/services/api/logs.ts` |
| Design-token guard rejects the ported fork SCSS that uses raw values. | `tests/designTokenGuard.test.ts`, `src/styles/variables.scss` header |
| The token plugin needs `bun` on `PATH` at build time. | `scripts/design-tokens/vite.ts:9` `execFileSync('bun', …)` |
| BumpyClock quota providers for Copilot and Qoder call routes the fork backend does not serve. | `src/services/api/copilot.ts`, no `copilot` route in `internal/api` |
| `check-panel` and the brand test fail unless the title change is in the same commit as the import. | `scripts/check-panel.sh:37`, `apps/web/tests/loginForm.test.ts` |
| Losing 3 locales is a user-visible regression. | 4 vs 7 locale files |

### Claims I could not confirm

| Claim | Where I looked |
| --- | --- |
| GitHub does not run `apps/web/.github/workflows`. | Repository layout only. Not tested against GitHub. |
| Every BumpyClock v0 call site would work against the fork's v0 fallback routes. | I compared endpoint sets and sampled v0 routes; I did not call each route. |
| BumpyClock UI renders the fork's v8 payloads without change. | Not tested: no backend was started, and no browser check was made. |
| The build fails when `bun` is absent from `PATH` (token plugin). | Not tested: `bun` was always on `PATH`. |
| Cold-cache install time. | Install ran with a warm bun cache (0.26 s). |
| Whether the failing fork test is new. | Ran the current checkout only; did not bisect. |
| BumpyClock screenshots in `docs/screenshots` match the current code. | Not opened. |
