---
status: in-progress
branch: issue/0042-triage-fork-usage-capabilities
type: triage (read-only, no code changes)
agent: deepseek-v4.1-flash
---

# Triage: what the fork already has for usage data

## Goal

The fork needs a built-in usage DB and statistics dashboard (see issue 0041 for the reference product). This issue records the in-process hooks, storage, and panel code that the fork already has, and the gaps. It is read-only. The coordinator writes the design.

## References (read-only)

- Go code in this repo: `internal/redisqueue`, `internal/kvstore`, `internal/store`, `internal/pluginhost`, `sdk/cliproxy`, `sdk/api/handlers` (usage reporting), `internal/api/server_management_v8.go`, `internal/api/handlers/management/`, `internal/config`.
- Panel: `apps/web/src` (pages, services, stores). The Usage page and the dashboard usage section are hidden because the backend has no aggregate data. Find them with `grep -ri usage apps/web/src`.
- Upstream: `.refs/CLIProxyAPI` (how upstream publishes usage records; compare with the fork).
- `DECISIONS.md`, `issues/0018-*.md` to `issues/0040-*.md` (grep for "usage").

## Tasks

1. **Usage records.** Where the proxy creates a usage record per request. Give the Go type (file and line), every field (name, type, meaning, can it be empty), and when it is emitted (success, error, stream end, cancel). Is there token data for cache read, cache write, reasoning? Is the credential (auth id, provider, account label), API key (hashed?), model (requested and routed), latency, status, and request id in the record?
2. **Consumers.** Every consumer of these records today (redisqueue, plugins, logs, quota code). Is there an in-process subscriber API that a built-in DB can use? What is the cost of a slow subscriber (blocking, drop, buffer)?
3. **Storage.** What persistent stores exist (bbolt in `kvstore`, files in auth-dir). Which Go SQL or embedded DB modules are already in `go.mod` or the module cache? Is CGO used in the build? Is the build static for the container (check `Containerfile`)? List options for an embedded DB that needs no CGO, with module name and version, from `go.mod`/`go.sum` evidence only (do not add dependencies).
4. **Panel.** What the panel has now for usage: pages, components, services, types, locale keys, hidden flags, tests. Which backend routes does it call or expect? List the dead code that will be reused or removed.
5. **Management API.** Existing `/v8/management/observability/...` routes: method, path, handler, response. Which ones carry usage data today (`usage` per API key, request log, etc.)?
6. **Pricing and models.** Does the fork have model metadata (context size, prices) in `internal/registry`, `internal/modelconfig`, or the model definitions? List fields and sources.
7. **Container.** Data directory and volume layout (`Containerfile`, `compose*.yaml`, config): where a new DB file would live and what must persist.

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence (file and line, command and output) and a proposal. Mark each claim you could not confirm and say where you looked.
- Use `/usr/bin/go`. Do NOT run `mise trust`. Builds only in `/tmp`.
- English only. Use ASD-STE100 Simplified Technical English, be concise. Use tables.

## Workflow

1. Create branch `issue/0042-triage-fork-usage-capabilities` in your checkout (from `main`). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end set `status: done`, fill `## Result`, commit, and report to the coordinator. Do not merge to `main`; the coordinator merges.

## Log

- 2026-11-11: Set `status: in-progress`. Read `AGENTS.md`, this issue,
  `DECISIONS.md`, issues 0027 and 0032 (panel usage port/hide), and the
  Go usage path. Read-only; only this file changes.

## Follow-ups

## Result
