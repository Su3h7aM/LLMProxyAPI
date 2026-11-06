# DECISIONS

Format: date, context, decision, reason. Newest first.

- 2026-11-06: Panel is part of the fork. The official CLI Proxy API
  Management Center is vendored into `apps/web` (pin `953fb24`) and the
  server binary embeds `dist/management.html`. No console bundle is
  downloaded at runtime. Reason: developer order; single unified
  project, first intentional difference from upstream.
- 2026-11-06: Never bring Plus code in without asking first. Standing
  developer rule.
- 2026-11-06: Upstream-to-fork data compatibility is mandatory; fork to
  upstream is not. Config, auth files, and kv.db from upstream must load
  and keep working. Any future store change ships a migration that
  preserves existing data. Current state: formats unchanged, new fields
  purely additive.
- 2026-11-06: Fork versions use CalVer YY.MM.DD with a letter for same-day
  fixes (26.11.06, 26.11.06a). Same tag-driven release flow as upstream;
  `scripts/version.sh` prints the next version. Service image runs as the
  `llmproxy` user, never root.
- 2026-11-06: OpenCode Go subscription is first-class in the backend
  (Sketch 1). Same key-list and quota/fetch section as Codex/Anthropic.
  Reference: magpie `readOpenCodeGo` plus the Go usage endpoint docs.
