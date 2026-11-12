---
status: done
branch: issue/0041-triage-cpamp-usage-inventory
type: triage (read-only, no code changes)
agent: muse-spark-1.3-contributor
---

# Triage: CPA-Manager-Plus usage and observability inventory

## Goal

The fork needs a built-in usage DB and statistics dashboard as robust as CPA-Manager-Plus (CPAMP), so users need no external container. This issue records what CPAMP does. It is read-only. A second issue (0042) records what the fork already has. The coordinator then writes the design.

## References (read-only)

- CPAMP source: `.refs/CPA-Manager-Plus` (pin a67ed883). Backend in `apps/manager-server` (Go), UI in `apps/web`, docs in `apps/docs` and `docs/`.
- Panel design reference: `.refs/BumpyClock-Cli-Proxy-API-Management-Center` (only for how its pages look; not the work of this issue).
- `AGENTS.md`, `DECISIONS.md` (2026-11-11 entries), `apps/web/AGENTS.md`.

## Tasks

1. **Ingest.** How CPAMP gets usage data from CPA (queue, polling, webhook, log parsing). Give the exact protocol, the record fields (name, type, meaning), batching, retry, de-duplication, and how it recovers after a restart or gap.
2. **Storage.** DB engine, schema (tables, columns, indexes, keys), migrations, retention and pruning, rollups or aggregates, size on disk per 1M requests if the docs say so, and what is redacted before storage.
3. **Query API.** Every HTTP endpoint the dashboard uses (method, path, query parameters, response shape). Group them: dashboard, monitoring, request history, usage analytics, cost, credentials, quota, health, automation, settings.
4. **Cost model.** How tokens become cost: price table source, per-model fields (input, output, cache read, cache write, reasoning), currency, unknown-model handling, user overrides.
5. **Dashboard features.** Every user-visible page, tab, chart, table, filter, time range, export, and setting. One row each: name, data it shows, source endpoint, and a priority guess (core, useful, optional).
6. **Automation.** Everything that acts, not only reads (health inspection, quota refresh, auto-disable, alerts, webhooks). For each: trigger, action, config.
7. **Auth and deployment.** How the admin key works, what runs as a separate process, ports, volumes, env vars. Mark parts that exist only because CPAMP is a separate service.
8. **Limits.** Known issues, scale limits, and performance notes from `docs/performance`, release notes, and open code TODOs.

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give evidence (file and line) and a proposal. Mark each claim you could not confirm and say where you looked.
- Result is one document in `## Result` with tables, not prose. Put long tables in `issues/0041-cpamp-inventory.md` and link it.
- English only. Use ASD-STE100 Simplified Technical English, be concise.

## Workflow

1. Create branch `issue/0041-triage-cpamp-usage-inventory` in your checkout (from `main`). Set `status: in-progress` here and commit.
2. Keep `## Log` current. Commit often.
3. At the end set `status: done`, fill `## Result`, commit, and report to the coordinator. Do not merge to `main`; the coordinator merges.

## Log

- 2026-11-12: Work on assigned worktree branch `t3/triage-cpamp-usage-inventory` (no new branch; coordinator owns merge). Reference pin confirmed `a67ed883`. Read collector, event, queue, resp, cpa client, migrate, config, all HTTP controllers, pricing, automation, auth, web routes, perf doc. Write `issues/0041-cpamp-inventory.md` next.
- 2026-11-12: Inventory done. Wrote `issues/0041-cpamp-inventory.md` (8 sections, all evidence with file and line). No code changed.

## Follow-ups

## Result

Full inventory: `issues/0041-cpamp-inventory.md`. Read-only triage. No code changed.

| # | Area | Key fact for design (issue 0042) |
|---|---|---|
| 1 | Ingest | Drain of CPA Redis queue over RESP (`LPOP`/`RPOP`), HTTP (`GET usage-queue`), or subscribe. Default batch 100, poll 500 ms. Dedup by `event_hash`. No local cursor; gap safety depends on CPA retention. |
| 2 | Storage | One SQLite file (modernc driver, WAL). Raw `usage_events` + identity ledger + ~10 rollup families with checkpoint resume. No time-based delete; archive workflow prunes. ~1.66 kB per event synthetic; index +1.25%. |
| 3 | Query API | 30+ endpoints in 12 groups (dashboard, monitoring x4, usage x8 incl. import and archives, prices x5, aliases, quota, cooldowns, candidates, inspection, setup, config, updates). All listed with method, path, params. |
| 4 | Cost | USD per 1M. 5 price fields + context tiers + service tiers + legacy long-context and tier multipliers. Unknown model = 0 cost (except gpt-5.6 built-ins). Sources: models.dev, LiteLLM, OpenRouter. |
| 5 | UI | 11 usage routes; core = dashboard, usage analytics (19 include flags), monitoring center (3 tabs), model prices. |
| 6 | Automation | Quota auto-disable + recover, auth-issue queue (+auto-disable), Codex inspection, quota writer, rollups, retention (off by default), update check. No webhooks found. |
| 7 | Deploy | One binary, port 18317, `/data` volume. Separate-only parts: CPA link, CPA proxy, extra admin key, setup flow. Price and release sources stay external. |
| 8 | Limits | 120-day P95 ~25 s; caps 50k rows, 200 accounts, 400 windows; poll (no live tail); retention off by default; unpriced models cost 0. |

Proposals for coordinator: reuse `event_hash` dedup and ledger idea; reuse rollup checkpoint pattern; reuse price book shape with configured flags; keep archive-before-delete. Open: confirm CPA queue durability in CPA source (see unconfirmed table in inventory).
