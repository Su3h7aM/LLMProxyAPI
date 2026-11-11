---
status: todo
branch: issue/0035-triage-panel-parity
type: triage (read-only, no code changes)
agent: deepseek-v4.1-flash
---

# Triage panel parity: pre-import panel vs integration branch

## Goal

Find what the old fork panel could do that the new panel (branch `integration/panel-bumpyclock`) cannot, and find calls without a backend route. This is read-only. Other agents still work on OpenCode Go, logic fixes, cleanup, the routing indicator, and CI: list their areas as "known, in progress" and do not count them as gaps.

## References (read-only)

- Old panel: tag `panel-pre-bumpyclock` (`git show panel-pre-bumpyclock:apps/web/<path>`, `git ls-tree -r panel-pre-bumpyclock --name-only apps/web`).
- New panel: branch `integration/panel-bumpyclock` (your checkout has `main`; read with `git show integration/panel-bumpyclock:apps/web/<path>` or create a scratch worktree under `/tmp`).
- Backend: Go code in this repo (`internal/api/server_management_v8.go`, `internal/api/handlers/management/`).
- Read `DECISIONS.md` (entries of 2026-11-11) and `issues/0015-*.md` to `issues/0029-*.md` (what was done and what is known).
- Known and decided (not gaps): sponsor content removed; Copilot and Qoder pages removed; Usage page and dashboard usage section to be hidden (no backend data); locales ko, tr, vi later; v0 and legacy support removed.

## Tasks

1. **Feature list.** For the old panel list every user-visible page, tab, action, setting, editor, dialog, and filter (use the router, nav items, page components, and locale keys). For the new panel do the same. Build a table: feature, old location, new location or MISSING, status (OK, MISSING, CHANGED with note, KNOWN-IN-PROGRESS with the issue number).
2. **Config coverage.** Compare the settings that the visual config page can edit in both panels (sections and fields). Compare with the fields of the Go config types (`internal/config/config_types.go`, `config_v8.go`): list Go settings that neither panel edits, and panel fields with no Go field.
3. **Route check.** Extract every API path that the new panel calls (static and template strings in `apps/web/src`; use a script in `/tmp`, do not commit it) and match it with the route table in `internal/api/server_management_v8.go` (and handlers for sub-paths). Table: path, method, caller file:line, route found (yes or no). Also list v8 routes that the old panel called and the new panel does not.
4. **Locale coverage.** For the 4 locales of the new panel: keys used in code (`t('...')`) but missing in a locale; keys with empty or copied English text in the zh and ru files for fork-only features.
5. **Quality checks.** Run in a scratch copy under `/tmp`: `bun install --frozen-lockfile`, `bun run verify`, `bun run tokens:check`; report results and the bundle size.

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence (file and line, command and output) and a proposal. Mark each claim you could not confirm and say where you looked.
- Use bun from `/home/su3h7am/.local/share/mise/installs/bun/latest/bin` and `/usr/bin/go`. Do NOT run `mise trust`. Do builds only in `/tmp`.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0035-triage-panel-parity` in your checkout (from `main`). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. When done: `status: done`, fill `## Result` (tables, proposed fix list ranked by user impact, claims you could not confirm), commit. Do NOT merge. Report: short summary, branch, commit range, key findings.

## Log

## Follow-ups

## Result
