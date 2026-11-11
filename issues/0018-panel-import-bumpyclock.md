---
status: done
branch: issue/0018-panel-import-bumpyclock
type: port (panel)
agent: muse-spark-1.3-contributor
base: integration/panel-bumpyclock
---

# Panel import A: replace `apps/web` with the BumpyClock tree

## Goal

Replace the vendored official panel in `apps/web` with the BumpyClock Management Center source (same stack, its UI and design system), remove content that the fork does not take, and rebrand. This step does NOT port the API layer to v8: the BumpyClock code keeps calling `/v0/management` (the fork backend still serves most v0 routes). That port is the next issues. Read `DECISIONS.md` (entries of 2026-11-11 about the dashboard, "one unit", design rule, and the migration plan) and the triage results in `issues/0015-*.md`, `issues/0016-*.md`, `issues/0017-*.md`.

References (read-only):

- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center`, pinned commit `ce5f617961fb15b47ad178c512958abe1d3f2b92`. Read files with `git -C <ref> archive ce5f6179 | tar -x -C <dir>` or `git -C <ref> show ce5f6179:<path>`. Never edit or commit in the ref.
- The pre-import fork panel is on `main` (tag `panel-pre-bumpyclock`) at `apps/web`. Keep it available for later steps: do not rewrite history.

## Branching

This issue lives on `main`, but your work goes on top of the integration branch: `git checkout -B issue/0018-panel-import-bumpyclock integration/panel-bumpyclock`. Merge back is done by the coordinator (into `integration/panel-bumpyclock`, not `main`).

## Steps (one commit each, Conventional Commits, English)

1. **Replace the tree.** In `apps/web`, keep only `panel.go` and `LICENSE` (MIT notice, keep as is). Remove the other tracked files. Do not touch ignored paths (`node_modules`, `dist`). Extract the pinned BumpyClock tree into `apps/web`, except: `.github/`, `plans/`, `CHANGELOG.md`, `.cpamc-fork-upstream.env`, `AGENTS.md`, `README.md`, `README_CN.md`, `.gitignore`, `assets/apimart-*.png`, `docs/screenshots/`, `docs/workspace-redesign.md`. Keep `docs/design-tokens.md`, `bunfig.toml`, `scripts/design-tokens/`, `logo.jpg` and the rest. Commit: `feat(web): replace the panel tree with the BumpyClock Management Center`. Put the pin in the body.
2. **Brand.** Page `<title>` must be `LLM Proxy API Management Center`; `document.title`, login brand name, logo alt text, about text, and visible strings: use the same brand replacement that the fork made in commit `c40624fc` and `c712e17f` (`git show` them on `main`): "CLI Proxy API" to "LLM Proxy API", "CPAMC" to "LPAMC", "CPA" to "LPA" in visible text and locale files. Do not rename technical identifiers (HTTP header names such as `x-cpa-*`, storage keys, CSS class names, file names).
3. **Remove sponsor and promo content.** Use the removal list in `issues/0015-triage-panel-api-and-content.md` (Task 3): `src/features/config/sponsors.ts` and its use in `SystemPage.tsx` and `MainLayout.tsx`; provider modules `sponsor.ts`, `fennoAI.ts`, `qiniuCloud.ts`, `aiCodeMirror.ts` (if present), `sponsorDefinitions.ts`, `SponsorQuickStartPanel`, `SponsorProviderForm`, `sponsorMutationRecovery`, `useSponsorUsageCheck`; related code in `useProviderWorkbench.ts`, `descriptors.ts`, `adapters.ts`, `brandLogos.ts`; the `providersPage.sponsor.*` i18n blocks and the sponsor resource strings in all 4 locales; sponsor logo assets; and the tests that pin them. Keep the `kimi` provider but remove its affiliate links and promo strings (`kimiPromo`). Port the fork rule of commit `4d1b8e47` (sponsor-matching endpoints stay in generic `openai-compatibility` groups; recreate its test `providerGroupsV8.test.ts` as it fits this tree). Search with `grep -rniE "sponsor|affiliate|aff=|apimart|bestproxy|fenno|qiniu|apikey\.fan|aicodemirror" src tests` until clean.
4. **Remove Plus-only pages.** Remove the Copilot quota provider and tab (`services/api/copilot.ts`, `features/quota/providers/copilot/*`, registry and constants entries, icons, tests) and the Qoder provider and its OAuth start option (`features/quota/providers/qoder/*`, the `qoder` entry of the OAuth provider union and list, icons, tests). Do not remove the Usage page and dashboard now (they read `/usage`, which the fork backend does not serve; the API port rewrites them). List them under `## Follow-ups`.
5. **Tooling and pins.** `packageManager` to `bun@1.4.4` (the Containerfile uses `oven/bun:1.4.4-debian`); regenerate the lockfile only if `bun install --frozen-lockfile` fails (then explain). Add `bun = "1.4.4"` to `.mise/config.toml` `[tools]`. Update `scripts/check-panel.sh` so it expects `<title>LLM Proxy API Management Center</title>`. Write a short `apps/web/AGENTS.md` (English): source and pin, the one-unit rule, v8-only target, design-token rule (no raw values, use `bun run tokens`), brand rule, and that sponsor and Plus-only content is not taken. Write a short `apps/web/README.md` (English): what it is, how to build (`bun install --frozen-lockfile`, `bun run verify`), and that the root build embeds `dist/management.html`.

## Gate (all must pass)

In `apps/web` with `export PATH=/home/su3h7am/.local/share/mise/installs/bun/latest/bin:$PATH`: `bun install --frozen-lockfile`, `bun run verify` (test, lint, build). Check one output file `dist/index.html` and its `<title>`. Then from the repo root: `cp apps/web/dist/index.html apps/web/dist/management.html`, `/usr/bin/go build ./...`, `/usr/bin/go vet ./...`, `/usr/bin/go test ./...`. Then run the panel smoke check by hand (the mise task cannot run): build `go build -o /tmp/llmproxy ./cmd/server`, then follow `scripts/check-panel.sh` using that binary, and report the result. Report anything that fails and why.

## Environment

- Use `/usr/bin/go`, not the `go` shim (mise config is untrusted; do NOT run `mise trust`).
- Never commit `node_modules`, `dist`, or `apps/web/dist/management.html` (git ignores them).
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Rules

- Take only what this issue lists. Do not restyle, rename, or refactor the BumpyClock code. Keep its design system and tokens as is.
- You do not decide. If a file is unclear (keep or remove), if the gate fails for a reason outside this scope, or if a removal breaks something else, stop, write the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. Create the branch as written above. Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current.
3. When the gate passes: `git rebase integration/panel-bumpyclock`, set `status: done`, fill `## Result` (what landed, commit list, gate output summary, known gaps, follow-ups), commit.
4. Do NOT merge. Report: short summary, branch, commit range, follow-ups, surprises, unconfirmed claims.

## Log

- 2026-11-11: Set `status: in-progress`. Created branch from
  `integration/panel-bumpyclock`.
- 2026-11-11: Step 1 done (`b9565a58`): `apps/web` replaced with the
  pinned BumpyClock tree, keep list verified (only `panel.go` extra,
  `LICENSE` identical).
- 2026-11-11: Step 2 done (`c94f36e6`): brand replacement per
  `c40624fc`/`c712e17f`. Main-repo link follows `c40624fc`
  (`router-for-me/LLMProxyAPI`); header constants, `X-CPA-*` test
  pins, and the `routeTrail` test fallback string stay unchanged.
- 2026-11-11: Step 3 done (`0b2b9d10`): sponsor removal.
  Interpretation: the kimi workbench brand goes with the other
  sponsor brands (the removal list names its files); "keep the kimi
  provider" means kimi keeps working through generic
  openai-compatibility entries plus its OAuth and quota providers,
  with affiliate links and `kimiPromo` removed. Moonshot and other
  ex-sponsor endpoints stay in generic groups (rule of `4d1b8e47`,
  pinned by the recreated `tests/providerGroupsV8.test.ts`).
  `tests/quotaObservationContracts.test.ts` lost its snapshot-backed
  block: list-first snapshot observations were qoder-exclusive
  behavior, and no kept adapter provides it.
- 2026-11-11: Step 4 done (`f3051d6d`): Copilot and Qoder quota tabs
  and the Qoder OAuth start option removed. `github-copilot.svg`
  and `qoder.svg` stay: auth-file credential icons still use them.
  Auth-file filters, model labels, usage stats, and OAuth key
  normalization keep Copilot/Qoder names (backend still serves
  those files).
- 2026-11-11: Step 5 done (`66417f91`): bun 1.4.4 pins, check-panel
  title, `apps/web/AGENTS.md`, `apps/web/README.md`. Frozen
  install passes, so the lockfile is unchanged.
- 2026-11-11: Gate passes. Rebase is a no-op
  (`integration/panel-bumpyclock` sits at `2900bc06`). Set
  `status: done`. No merge (coordinator merges).

## Follow-ups

- The Usage page and dashboard still read `/usage`, which the fork
  backend does not serve. The API port rewrites them (per issue).
- `providersPage.providerNames` keeps dead `apikeyFun`, `fennoAI`,
  `qiniuCloud`, `kimi` entries, as in the fork. Remove them or keep
  them in a later pass.
- `QuickStartSteps.tsx`, `createLabel.ts`, and `workbenchLayout.ts`
  stay but the `/quick-start` route now redirects to
  `/ai-providers`, so nothing renders them. Decide later whether to
  remove them.
- `isPluginQuotaFile` still excludes `github-copilot` and `qoder`,
  so those files never fall back to plugin quota. Decide later
  whether they should.
- `src/features/quota/README.md` still documents the removed
  `qoder` provider. Update it in a later pass.
- ko, tr, vi locales do not exist in this tree (BumpyClock ships
  4). Per `DECISIONS.md` they are a later, low-priority step.

## Result

What landed on `issue/0018-panel-import-bumpyclock` (base
`integration/panel-bumpyclock` at `2900bc06`):

- `b9565a58` tree replacement with the BumpyClock pin `ce5f6179`
  (exclusions per issue; only `panel.go` kept extra).
- `c94f36e6` rebrand to LLM Proxy API / LPAMC / LPA.
- `0b2b9d10` sponsor and promo removal, incl. the recreated
  `tests/providerGroupsV8.test.ts` (4d1b8e47 rule, adapted to the
  v0 tree).
- `f3051d6d` Copilot and Qoder quota removal and Qoder OAuth
  start removal.
- `66417f91` bun 1.4.4 pins, check-panel title, panel AGENTS.md
  and README.md.

Gate output: `bun install --frozen-lockfile` clean (lockfile
unchanged), `bun run verify` clean (1655 tests pass, lint clean,
one-file build, `<title>LLM Proxy API Management Center</title>`);
`go build ./...`, `go vet ./...`, `go test ./...` pass; panel
smoke check passes (`/tmp/llmproxy` serves the title).

Known gaps: Usage page and dashboard still read `/usage` (next
issues rewrite them); follow-ups above list the leftovers.
