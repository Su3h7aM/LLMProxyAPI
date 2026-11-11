---
status: todo
branch: issue/0015-triage-panel-api-and-content
type: triage (read-only, no code changes)
agent: muse-spark-1.3-contributor
---

# Triage panel migration A: backend API compatibility and content audit

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

Find out what in the BumpyClock panel works against the fork backend, and what does not.

## Tasks

1. **Endpoint map.** List every HTTP call of the BumpyClock panel (all of `src/services/api/*`, plus any other `axios`/`fetch` use in `src`): method, path, file:line. For each, check the fork backend routes (`internal/api/server.go` and `internal/api/handlers/management/`, versions `/v0/management` and `/v8/management`): SERVED / MISSING / DIFFERENT (request or response shape differs; say how). Compare also with the official panel that the fork vendors (`apps/web/src/services/api`): which endpoints are new in BumpyClock, which were dropped.
2. **Dependent features.** For each MISSING or DIFFERENT endpoint list the pages, components, and stores that depend on it (for example `copilot.ts`, `kimiQuota.ts`, `devinQuota.ts`, `metaQuota.ts`, `antigravitySubscription.ts`, `plugins.ts`, `authFileCooldowns.ts`, `apiKeyUsage.ts`). Say if the feature looks CLIProxyAPIPlus-only (not in upstream CLIProxyAPI) or is in upstream and the fork backend already serves it.
3. **Sponsor and promo content** in the BumpyClock repo: files, assets, i18n keys, components, README. Give a removal list.
4. **Connection model.** Describe how the BumpyClock panel finds and connects to the backend: login page, API base detection, storage keys, management key handling, remote/other server connect flow, auto-login, version checks (`version.ts`, latest release check, minimum backend version). Then do the same for the fork panel (`apps/web`, `src/stores/useAuthStore.ts`, `src/utils/connection`). The target is same-origin: no API base URL setting and no remote server connect flow; management key login stays. List the BumpyClock files that must change for that, and what is the smallest change.
5. **Plus lineage.** The panel comes from a fork made for CLIProxyAPIPlus (`.cpamc-fork-upstream.env`). List code that clearly exists only for CLIProxyAPIPlus (provider names, endpoints, strings) so that the coordinator can decide.

## Result format

Tables per task (task 1 as one table: method, path, file:line, status, note). End with: proposed list of BumpyClock features to keep, hide, or remove (with the reason), risks, and a list of claims you could not confirm.


## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence and a proposal. The coordinator decides.
- Back each claim with a file path and line, a command, or its output. Mark each claim you could not confirm and say where you looked.
- Environment: use `/usr/bin/go`, not the `go` shim (mise config is untrusted; do NOT run `mise trust`). Use bun at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin/bun`. Do builds and installs only in a scratch copy under `/tmp` (for example `cp -r` the panel there). Do not leave `node_modules` or `dist` in your checkout.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0015-triage-panel-api-and-content` in your checkout (rename your branch if T3 gave another name). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often (Conventional Commits).
3. When done: set `status: done`, fill `## Result`, commit. Do NOT merge to main. Do not create other issue files.
4. Report to the coordinator: short summary, branch, commit range, key findings, questions, surprises.

## Log

## Follow-ups

## Result
