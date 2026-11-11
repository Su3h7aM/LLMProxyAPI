---
status: in-progress
branch: issue/0017-triage-panel-decoupling-and-shortcuts
type: triage (read-only, no code changes)
agent: muse-spark-1.3-contributor
---

# Triage panel migration C: decoupling code and shortcut opportunities

## Context

The fork dashboard (`apps/web`) will be replaced by the BumpyClock Management Center. Read `DECISIONS.md` (entries of 2026-11-11 about the dashboard and about "one unit") and `issues/0015-triage-panel-api-and-content.md` (merged result is on the 0015 branch; read it with `git show issue/0015-triage-panel-api-and-content:issues/0015-triage-panel-api-and-content.md` if it is not on `main` yet).

The panel and the proxy ship together: one container, one repository, one version. The panel can drop everything that exists only because it may talk to an unknown or separate server. Where the panel needs many calls or heavy client work only because the API is generic, a direct endpoint on the proxy is better. Limit: released `/v0` and `/v8` contracts never change or shrink; new endpoints must be additive.

References (read-only, never edit or commit there):

- BumpyClock panel: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center`
- Fork panel today: `apps/web` in your checkout
- Fork backend: your checkout (`internal/api`, `internal/api/handlers/management`, `sdk/cliproxy`)

## Tasks

1. **Decoupling code to delete.** In the BumpyClock source and in the fork panel (`apps/web`), list code that exists only because the panel is a separate client: backend probes, version negotiation and `x-cpa-*` / `x-server-version` header handling, capability checks that try an endpoint and fall back on 404, legacy backend detection (`legacyBackendProbe`), API base detection and normalization, custom/remote connect UI and storage scoped by base, v0 paths and v0-vs-v8 branching, CORS or proxy related workarounds, "minimum version" texts and i18n strings. Give file:line, what it does, and the safe deletion (what must stay, for example the management key login, 401 handling, ETag or revision guards that protect concurrent writes).
2. **Shortcut candidates.** Find places where the panel does client-side stitching that a single new endpoint could do: several round trips for one screen, merging or reshaping data in the browser, per-credential fan-out requests (for example quota or model fetch through `POST /api-call` to reach a provider), polling loops, repeated config reads. For each: current calls (file:line), what the browser computes, a proposed additive endpoint (method, path under `/v8/management`, request and response sketch), expected gain, and risk. Check the backend code to say if the data is already available inside the server (cite the Go function). Rank by value.
3. **Contract sharing.** Check if the Go management handlers and the TypeScript types can share one source of truth cheaply (for example a generated TS types file from Go structs or from an OpenAPI-like list). Say what exists today (cite files), what the cost is, and if it is worth doing now or later. Do not implement.
4. **Embedded delivery.** List what the server does today to serve the panel (`/management.html`, static path override, auto-update or download of the panel, `remote-management` options such as `disable-auto-update-panel`, `MANAGEMENT_STATIC_PATH`) and what of this is dead weight for a panel that is embedded in the binary (for example panel update and download logic). Give file:line and a safe removal list, marked as "released config key" if removing it would break an existing config file (config compat is mandatory: such keys must keep loading and be ignored).

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence and a proposal. The coordinator decides.
- Back each claim with a file path and line, a command, or its output. Mark each claim you could not confirm and say where you looked.
- Environment: use `/usr/bin/go`, not the `go` shim (do NOT run `mise trust`). bun is at `/home/su3h7am/.local/share/mise/installs/bun/latest/bin/bun`. Do builds only in a scratch copy under `/tmp`.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0017-triage-panel-decoupling-and-shortcuts` in your checkout (rename your branch if T3 gave another name). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often (Conventional Commits).
3. When done: set `status: done`, fill `## Result` (tables per task, ranked shortcut list, risks, claims you could not confirm), commit. Do NOT merge to main. Do not create other issue files.
4. Report to the coordinator: short summary, branch, commit range, key findings, questions, surprises.

## Log

- 2026-11-11: Set `status: in-progress`. Read `DECISIONS.md`, issue 0015 result. Created branch.
- 2026-11-11: Ran parallel evidence pulls (decoupling grep, shortcut grep, contract sharing, embedded delivery). Verifying claims with file:line.

## Follow-ups

## Result
