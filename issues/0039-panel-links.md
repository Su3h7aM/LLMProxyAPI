---
status: done
branch: issue/0039-panel-links
type: panel
agent: deepseek-v4.1-flash
base: integration/panel-bumpyclock
---

# Panel: fix repository links

## Goal

Fix a wrong repository link on the System page and check all other external links.

Read first: `DECISIONS.md` (entries of 2026-11-11: dashboard, "one unit", design rule, migration plan) and the triage results `issues/0015-*.md` (endpoint map v0 to v8, content), `issues/0016-*.md` (fork feature inventory, stack), `issues/0017-*.md` (what to delete, shortcuts, delivery).

## References (read-only)

- The panel as it is now: branch `integration/panel-bumpyclock` (your base). It is the BumpyClock panel (pin `ce5f6179`) after issue 0018. It calls the v0 API.
- The pre-import fork panel: tag `panel-pre-bumpyclock` (read files with `git show panel-pre-bumpyclock:apps/web/<path>`). It already speaks the v8 API. Use it as the reference for v8 request paths, request bodies, response shapes, transformers, and for the fork-only features. Its UI styling is NOT the target: copy behavior and data handling, not look.
- BumpyClock source: `/home/su3h7am/Apps/LLMProxyAPI/.refs/BumpyClock-Cli-Proxy-API-Management-Center` (pin `ce5f6179`, never edit it).
- Fork backend: Go code in this repo. v8 routes are in `internal/api/server_management_v8.go` and `internal/api/handlers/management/`. Check the real route and the real response shape in Go before you write a client call.

## Design rule

The BumpyClock interface, style, and design system do not change. Where a feature needs new UI (a part that BumpyClock has no page for), build it from the BumpyClock design system: its components, tokens, layout, and patterns, in the same way its existing pages do. Find a similar BumpyClock page first and copy its structure. No raw colors, sizes, or spacing: the token guard test rejects them (`bun run tokens`, `tests/designTokenGuard.test.ts`). Do not restyle existing BumpyClock UI.

## Branching

Your work goes on top of the integration branch: `git checkout -B issue/0039-panel-links integration/panel-bumpyclock`. The coordinator merges it back into `integration/panel-bumpyclock` (not `main`). The panel is in a half-ported state on that branch: some modules still call v0 paths that do not exist in v8. That is expected; port only your own area.

## Steps (small commits, Conventional Commits, English)

1. **Main repository link.** `src/pages/SystemPage.tsx` (around line 427) links to `https://github.com/router-for-me/LLMProxyAPI`, a repository that does not exist. Make it point to this fork: `https://github.com/Su3h7aM/LLMProxyAPI`. Check the locale key `system_info.link_main_repo` text and that the link title fits.
2. **Other external links.** List every `https://` URL in `apps/web/src` (static strings and locale files) that names a GitHub repository or a documentation site. For each, say in `## Log` whether it is right for this fork. Fix links that point to a repository that does not exist or to a project that is not the source of this fork (the fork repository is `Su3h7aM/LLMProxyAPI`; the upstream proxy is `router-for-me/CLIProxyAPI`; links to upstream documentation can stay). Do not change links inside license or attribution text.
3. **Tests.** Update tests that pin the old URL; add one test that fails if the System page links to `router-for-me/LLMProxyAPI`.
4. **Gate.** `bun run verify` green.
Do NOT do in this issue: other changes.

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

- 2026-11-11: Read `DECISIONS.md`, issues 0015 to 0017, `apps/web/src`, and
  `git show panel-pre-bumpyclock:apps/web/src/pages/SystemPage.tsx`.
- 2026-11-11: Step 1 done. `src/pages/SystemPage.tsx` main repository link
  now points to `https://github.com/Su3h7aM/LLMProxyAPI`. Locale keys
  `system_info.link_main_repo` ("Main Repository", "LLM Proxy API core
  program source code") and `system_info.link_webui_repo` ("WebUI
  Repository") fit the fork text; no locale change needed. Note: commit
  `5e5b131c` (already on the base) moved the WebUI card to the fork, so
  both repository cards now point to the same URL.
- 2026-11-11: `tests/round2SystemPage.test.ts` now checks each quick-link
  card on its own and fails if the page links
  `router-for-me/LLMProxyAPI`.
- 2026-11-11: Step 2 audit. Every `https://` URL in `apps/web/src`
  (static strings and the four locale files). Only these name a GitHub
  repository or a documentation site:

  | URL | Where | For this fork |
  | --- | --- | --- |
  | `github.com/router-for-me/LLMProxyAPI` | `pages/SystemPage.tsx` main repo card | Wrong: the repository does not exist. Fixed to the fork. |
  | `github.com/Su3h7aM/LLMProxyAPI` | `pages/SystemPage.tsx` WebUI repo card | Right: this is the fork. Set by `5e5b131c`. |
  | `help.router-for.me` | `pages/SystemPage.tsx` documentation card | Right: upstream documentation for the proxy the fork carries. Kept (issue allows it). |
  | `github.com/lucide-icons/lucide` | `components/ui/icons.tsx` comment | Right and out of scope: attribution of the icon set. Kept. |
  | `github.com/router-for-me/Cli-Proxy-API-Management-Center` | `features/config/components/sections/SectionConnectivity.tsx` placeholder | Open question, see below. |
  | `github.com/router-for-me/` | `features/plugins/pluginResources.ts` `OFFICIAL_PLUGIN_REPO_PREFIX` | Open question, see below. |
  | `api.github.com`, `github.com/<slug>/releases`, `api.github.com/repos/owner/repo/releases/` | `features/plugins/*` | Right: generic GitHub API host and per-repository release URLs built from user input. Not repository links to a fixed project. |
  | `example.com/cliproxy-plugins/registry.json` (4 locales) | plugin registry placeholder | Right: an example value, not a link. |

  Open questions (not changed, this issue does not decide them):
  1. The `Panel Repository` input placeholder names the reference
     panel repository. The `remote-management.panel-github-repository`
     key is a released config key and its Go default is the same URL
     (`internal/config/config_defaults.go:4`). Issue 0032 left this
     open. It is not a clickable link, so it is not a broken link. A
     change must move the placeholder and the Go default together, and
     this issue says do not change Go code.
  2. `OFFICIAL_PLUGIN_REPO_PREFIX` decides which plugin repositories
     count as first-party. It is a trust rule, not a navigation link.
     Who is trusted after the fork is a security decision.
  3. Both repository cards now point to `https://github.com/Su3h7aM/LLMProxyAPI`,
     because the fork holds the proxy and the panel (`5e5b131c`). Two
     cards with one target may be redundant. Removing or retargeting a
     card is a UI change that this issue does not list.

## Follow-ups

- Decide the `remote-management.panel-github-repository` placeholder in
  `SectionConnectivity.tsx` together with the Go default
  (`internal/config/config_defaults.go:4`). Still the reference panel
  repository. Carried over from issue 0032.
- Decide who is first-party for plugins: `OFFICIAL_PLUGIN_REPO_PREFIX`
  in `pluginResources.ts` still names `github.com/router-for-me/`. It is
  a trust rule, so it needs a security decision.
- The main repository card and the WebUI repository card point to the
  same URL. Decide if the System page needs both cards.

## Result

Landed on `issue/0039-panel-links` (base `integration/panel-bumpyclock`
at `10999bd2`):

- `14555bd4` fix(panel): point the main repository link at the fork
  - `src/pages/SystemPage.tsx`: the main repository card links
    `https://github.com/Su3h7aM/LLMProxyAPI` instead of the
    non-existent `router-for-me/LLMProxyAPI`.
  - `tests/round2SystemPage.test.ts`: the URL test now reads each
    quick-link card on its own and fails if the page links
    `router-for-me/LLMProxyAPI` or the reference panel repository.
- `3fca91ea` docs(0039): audit the external links of the panel
  - full list and verdict for each external link in `apps/web/src` and
    the locale files, in `## Log`.

Gate after the rebase onto `integration/panel-bumpyclock`: `bun install
--frozen-lockfile`, `bun run verify` green (2092 tests pass, eslint
clean, type-check clean, `vite build` writes `dist/index.html`).

Known gaps:

- No locale text changed. `Main Repository` and `WebUI Repository`
  still describe the cards correctly for the fork.
- The audit items in `## Log` under "Open questions" are not changed.

Follow-ups: see `## Follow-ups`.
