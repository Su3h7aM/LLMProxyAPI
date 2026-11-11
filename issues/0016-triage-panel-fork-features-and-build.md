---
status: todo
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

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence and a proposal. The coordinator decides.
- Back each claim with a file path and line, a command, or its output. Mark each claim you could not confirm and say where you looked.
- Environment: use `/usr/bin/go`, not the `go` shim (mise config is untrusted; do NOT run `mise trust`). Use bun at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin/bun`. Do builds and installs only in a scratch copy under `/tmp` (for example `cp -r` the panel there). Do not leave `node_modules` or `dist` in your checkout.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0016-triage-panel-fork-features-and-build` in your checkout (rename your branch if T3 gave another name). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often (Conventional Commits).
3. When done: set `status: done`, fill `## Result`, commit. Do NOT merge to main. Do not create other issue files.
4. Report to the coordinator: short summary, branch, commit range, key findings, questions, surprises.

## Log

## Follow-ups

## Result
