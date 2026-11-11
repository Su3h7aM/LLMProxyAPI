---
status: in-progress
branch: issue/0002-triage-core-and-rest
type: triage (read-only, no code changes)
---

# Triage upstream: auth routing, translators, plugins, rest

## Goal

Review upstream commits and decide, per commit, if the fork should take it.
Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/LLMProxyAPI` (read-only: never edit or
commit there). Baseline pinned in `UPSTREAM_BASELINE.md`:
`e5e6ad1e1b91fc20ab9964e6fb22c0c269b9d882`. Range: `e5e6ad1e..HEAD`.

Area: all other commits (auth scheduler and routing, translators, pluginhost, watcher, config, management API, cache, registry, logging, docs)

## Rules

- Do not change code. Your only edits: this issue file (`## Log`, `## Result`).
- You do not decide. Give a verdict and evidence. The coordinator decides.
- Take only fixes, bug fixes, and valid improvements. Fork features stay.
- Fork differences to protect (see `DECISIONS.md`):
  - Go module is `github.com/Su3h7aM/LLMProxyAPI` (upstream: `github.com/router-for-me/CLIProxyAPI/v8`).
  - Panel is vendored in `apps/web` and embedded in the binary.
  - OpenCode Go provider and quota support, routing panel features.
  - Sponsor and promotional content is removed. Skip such changes.
  - Upstream config, auth files, and kv.db must keep loading (compat is mandatory).
  - Plus (CPA-Manager-Plus) code: never bring in.
- To see how the fork differs from the baseline, compare the fork file with the
  baseline file: `git -C .refs/LLMProxyAPI show e5e6ad1e:<path>` vs the fork path
  (apply the module-path rename before you compare).

## Commits to review (oldest first)

```
418b824ca6652847c2e2abc663292f4452c77655 docs(readme): mention cpa-status in Hermes entry
59a2b3531378e2cf56bd7e9337c31c257b22d13d docs(readme): add SWE-2 Flash to Devin models
c5251b57b3fa5dcbc689f514f4aa166daacb4481 fix(translator): preserve reasoning and text across multi-part gemini and antigravity chunks
f4d3f0618d158c3168e113340fd06af15200d67a fix(openai): replace responses websocket transcript when pending tool calls are unanswered
b5b724d06df01748055259512d163605fd2f482d fix(pluginhost): reuse content-addressed plugin shadow copies across restarts
91e0dee827abf476f9f86246f68c500a112f7155 fix(openai): prevent SSE keep-alive injection mid-line during image streaming
f69db99d5ebb9f0a68ea2e6741257968eb27536d feat(plugin-store): support opt-in release cache refresh on store listing
40c16182b82689c4b7f6038d5992861fec1e1e38 feat(auth): support session-affinity-rebind-on never mode
0074897481fee1dcd595c1caa1d57e6a379b6be6 feat(pluginhost): expose token modality details in usage.handle
14e577174a37b339c6b6862b4a08543f2e10a15e fix(pluginhost): support credential-scoped rate limit errors from plugin executors
f36b195e7e337c91d3b2872e00282fa10ac7354b perf(antigravity): skip reasoning replay KV for symmetric gemini/interactions protocols
0561334f06c5c2bf4f1ba85566a551d39bbff138 fix(home): preserve fork kind in home dispatch
d20451504f473d303a877a8f4edef195f988da4e feat(auth): compensate weighted routing for sticky sessions
e479ab12538cb027ea558bbfa070fef1ca9d187c perf(auth): compact weighted round-robin pools
bebd92c4c878a19bce9df842c42b90adedc37ff7 feat(cmd): add -login-proxy flag for CLI OAuth logins
eb6e25abe6dbac17f85cb70aebafa56c46e203d0 fix(devin): count cache-write tokens in total input
e84d4836b7b67c0e010b73778bc49dabaadaa664 fix(auth): account pinned traffic in bare weighted routing
4b3f2c7ec7cd77b3b53b39583edb3a30b4a0d3f7 fix(logging): skip request error dumps for unrouted 404 responses
628e54abf3bf5b18e287aeaf86a45aa8e7396e7e fix(watcher): update active auth directory and fsnotify watch on runtime auth-dir changes
19a0a8541fe173773d2c6eb9f68e678762145634 feat(models): add claude-fable-5-2 and gemini-omni-1.2-flash to model registry
845345845b44743ba44828f5f5152048df1a2f88 docs(config): document X-Session-ID guidance for full-history compaction under session affinity
13fdcc5e09be9f557d49d5a760403ed1cea153fc fix(thinking): recognize claude thinking.display updates in summary pipeline
c9dad8470368301d29ac25f85b19724abb2b0378 fix(home): only renew liveness on heartbeat pong
18883a57b4b1b040bff93a41be0e7f7ad261e501 fix(usage,auth): propagate request-scoped error rule classification to usage fail.code
46d48eb5af062dda4e2d79ceeefedfde3f6917ba fix(openai): start responses websocket keepalive after upgrade
d576416a0dd423b585a5c2a627dff16f4b3c7a8d fix(compaction): strip text format constraints and configuration_update from summary requests
d1ea84f4b338f553d2818091ec9425a4ff742c78 docs(readme): remove AICodeMirror sponsorship banner
effbac168acf1bc14ebdbd1c97de8264ac449f0f feat(models): add Gemini 4 Carbon model entry to models.json
949b43b330f919fddeeb45e27b90cc9413736ef7 fix(xai): classify non-stream terminal SSE errors and empty or incomplete streams
69ed202a1ac3b36147b696d48bc0ba37fe0b4582 fix(registry): restore model listings once bounded cooldown deadlines expire
f77a3c104bc00474f6b366bf1582440deb42fbea fix(meta): pass through explicit client prompt_cache_retention
ecebc71345b7c651a0afc6d551ac9265691edef1 fix(auth): keep request retry rounds after scheduler plugin reject
9d7d012b35f082ddc037c2b90fd3219e32d940ab fix(auth): preserve large weighted pool fairness and force cooldowns
8bba7a77c6bcbfd993516d26606bfecaa4ff49d0 docs(README): add AICodeMirror sponsor banner to EN, CN, and JA readmes
```

## Method

For each commit: read the diff (`git show <hash>`), check if the touched files
differ in the fork, check dependencies on other upstream commits (inside or
outside your list), and check if tests exist. Do a dry-run apply in your
workspace if useful (`git apply --check` or `git cherry-pick` on a throwaway
branch after the module-path rewrite), then discard it. Do not leave code changes.

## Result format

One table in `## Result`, one row per commit:

| Hash | Subject | Verdict | Fork conflicts | Depends on | Size | Evidence / notes |

Verdict is one of: PORT, SKIP (reason), NEEDS-DECISION (question for coordinator).
After the table: proposed port groups (ordered, each group small enough for one
issue), risks, and a list of claims you could not confirm (say where you looked).

## Log

- 2026-11-11: Branch renamed from `t3/triage-core-rest` to
  `issue/0002-triage-core-and-rest`. Status set to in-progress.
- Collected `git show --stat` for all 34 commits in the list (the list has 34
  commits, not 40).
- Compared every touched file with the fork file and the baseline file
  (`git show e5e6ad1e:<path>`, module path rewritten). Result: most fork
  differences are import-order only. Semantic fork differences are in
  `config.example.yaml`, `internal/config/config_types.go`,
  `sdk/cliproxy/auth/selector.go`, `sdk/cliproxy/auth/conductor_selection.go`,
  `sdk/cliproxy/service_config.go`, `cmd/server/main.go`, `README.md`.
- Fork-specific code found in the auth area: `SoonestResetSelector`
  (`sdk/cliproxy/auth/selector_soonest_reset.go`), `AffinityPersister`,
  transient-strike logic in `SessionAffinitySelector`, `CandidateFilter`
  handling, rebind-on default `quota`, and `native-oauth-prompt-cache-1h`.
- Dry-run method: scratch clone `/tmp/up-dry` of the upstream refs clone
  (read-only source), cherry-pick the group there, take `git diff
  e5e6ad1e..HEAD`, rewrite the module path, apply to a scratch clone of the
  fork `/tmp/triage-dry`. No code change in this checkout.
- Groups that apply cleanly (build OK, package tests pass):
  translator/openai (c5251b57, f4d3f061, 91e0dee8, 46d48eb5);
  pluginhost/store (b5b724d0, f69db99d, 00748974, 14e57717);
  executors/thinking (f36b195e, eb6e25ab, 949b43b3, f77a3c10, d576416a,
  13fdcc5e); logging/watcher/home/cmd (4b3f2c7e, 628e54ab, c9dad847,
  0561334f, bebd92c4); models/docs (19a0a854, effbac16, 84534584);
  registry (69ed202a). Only `*_test.go` import-order conflicts, fixed by
  `patch -F3`; no semantic conflict.
- Auth routing commits cannot be replayed alone. `d2045150` needs
  `40c16182`, `098af7d5`, `5f38d557`; `9d7d012b` needs `d2045150`,
  `e479ab12`, `e84d4836`, `cb70eacf`, `098af7d5`, `80ee739f`, `09b3a579`,
  `18883a57`, `14e57717`; `18883a57` needs `216f2947` and `bfac8586`
  (issue 0001); `ecebc713` needs `18883a57` and `40c16182`.
- Open question 1: is the weighted-routing rewrite chain wanted? It is
  ~7,100 added lines and rewrites the selector path that the fork already
  extends (soonest-reset, affinity persister, strikes).
- Open question 2: is the rebind-on `never` mode wanted? It is a new config
  value, not a fix, and conflicts with the strike block in `selector.go`.

## Follow-ups

- If the weighted chain is accepted, port it in one issue after the issue
  0001 commits that it needs, and re-run the fork's routing tests.
- The panel (`apps/web`) has no UI for the plugin-store `refresh` query
  parameter; add a follow-up if the backend change lands.
- Re-check `docs/management-api-v8.md` after any weighted-routing port; the
  fork copy is still identical to the baseline.

## Result

| Hash | Subject | Verdict | Fork conflicts | Depends on | Size | Evidence / notes |
| --- | --- | --- | --- | --- | --- | --- |
| `418b824c` | docs(readme): mention cpa-status in Hermes entry | SKIP (not applicable) | `README.md` differs by 385 lines; no `README_CN.md`/`README_JA.md` in fork | none | 3 files, +3/-3 | The fork `README.md` is a 71-line rewrite with no Hermes or provider section. The changed paragraph does not exist. |
| `59a2b353` | docs(readme): add SWE-2 Flash to Devin models | SKIP (not applicable) | same as above | none | 3 files, +3/-3 | The fork README has no Devin row. |
| `c5251b57` | fix(translator): preserve reasoning and text across multi-part gemini and antigravity chunks | PORT | none (files identical to baseline) | none | 4 files, +194/-7 | Applies clean; `go test ./internal/translator/...` passes. Two regression tests included. |
| `f4d3f061` | fix(openai): replace responses websocket transcript when pending tool calls are unanswered | PORT | test import order only | none | 2 files, +57/-5 | Main file identical. Regression test included; `sdk/api/handlers` tests pass. |
| `b5b724d0` | fix(pluginhost): reuse content-addressed plugin shadow copies across restarts | PORT | none | none | 9 files, +1228/-42 | Applies clean; pluginhost tests pass. Adds Unix/Windows shadow files. |
| `91e0dee8` | fix(openai): prevent SSE keep-alive injection mid-line during image streaming | PORT | test import order only | none | 5 files, +527/-95 | Applies with `patch -F3`; handlers tests pass. |
| `f69db99d` | feat(plugin-store): support opt-in release cache refresh on store listing | PORT | test import order only | none | 5 files, +244/-28 | Additive `?refresh=true` query. Management handler tests pass. Panel has no control for it. |
| `40c16182` | feat(auth): support session-affinity-rebind-on never mode | NEEDS-DECISION | `config.example.yaml`, `internal/config/config_types.go`, `sdk/cliproxy/auth/selector.go`, `sdk/cliproxy/service_config.go` (semantic; strike block) | `bfac8586`, `5f38d557` (both issue 0001) | 9 files, +977/-28 | New config value `never`, not a fix. Adds `rebindOnNever` to the same struct that the fork extends with strikes. Does not change the default (`any` upstream, `quota` in fork). Question: port this mode? |
| `00748974` | feat(pluginhost): expose token modality details in usage.handle | PORT | none | none | 5 files, +330/-0 | Additive SDK fields. `coreusage.CloneInt64Ptr` and the modality fields already exist in the fork (`sdk/cliproxy/usage/detail.go`). Tests pass. |
| `14e57717` | fix(pluginhost): support credential-scoped rate limit errors from plugin executors | PORT | `conductor_cooldown.go` import order only | `68cd7572` (issue 0001) context only | 7 files, +556/-6 | Applies clean; pluginhost and auth tests pass. |
| `f36b195e` | perf(antigravity): skip reasoning replay KV for symmetric gemini/interactions protocols | PORT | none | none | 2 files, +90/-0 | Applies clean; executor tests pass. |
| `0561334f` | fix(home): preserve fork kind in home dispatch | PORT | none | `40c16182` line context only (applies clean without it) | 2 files, +67/-2 | Applies clean; home/auth tests pass. |
| `d2045150` | feat(auth): compensate weighted routing for sticky sessions | NEEDS-DECISION | `config.example.yaml`, `conductor_selection.go`, `selector.go` (semantic) | `40c16182`, `098af7d5`, `5f38d557`, `f69db99d` (docs only) | 15 files, +2213/-74 | Rewrites weighted selection into `selector_weighted.go` with reservations and changes `SessionAffinitySelector.Pick`. Upstream replay fails without `40c16182`. Question: port the weighted pool rewrite? |
| `e479ab12` | perf(auth): compact weighted round-robin pools | NEEDS-DECISION (with `d2045150`) | new `selector_weighted.go` | `d2045150` | 11 files, +400/-178 | Only meaningful after `d2045150`; the file does not exist in the fork. |
| `bebd92c4` | feat(cmd): add -login-proxy flag for CLI OAuth logins | PORT | `cmd/server/main.go` differs (XDG, no managementasset) but applies clean | none | 12 files, +181/-0 | `sdk/proxyutil` exists at baseline. Additive flag. Applies clean; `sdk/auth` tests pass. |
| `eb6e25ab` | fix(devin): count cache-write tokens in total input | PORT | test import order only | none | 2 files, +27/-8 | `DevinUsage.CacheWriteTokens` exists at baseline. Executor tests pass. |
| `e84d4836` | fix(auth): account pinned traffic in bare weighted routing | NEEDS-DECISION (with `d2045150`) | `config.example.yaml`, `conductor_selection.go`, `selector.go` | `d2045150`, `e479ab12`, `098af7d5` | 10 files, +483/-54 | Fixes a gap in `d2045150`; needs the new scheduler pool code. |
| `4b3f2c7e` | fix(logging): skip request error dumps for unrouted 404 responses | PORT | none | none | 6 files, +434/-29 | Applies clean; logging tests pass. |
| `628e54ab` | fix(watcher): update active auth directory and fsnotify watch on runtime auth-dir changes | PORT | `internal/watcher/watcher.go` import order only | none | 7 files, +615/-50 | Applies with `patch -F3`; watcher tests pass. |
| `19a0a854` | feat(models): add claude-fable-5-2 and gemini-omni-1.2-flash to model registry | PORT | none | none | 1 file, +119/-1 | Data-only `models.json`. Applies clean; registry/config tests pass. |
| `84534584` | docs(config): document X-Session-ID guidance for full-history compaction under session affinity | PORT | `config.example.yaml` comment context | `40c16182` comment context only | 3 files, +44/-6 | Comment-only change plus `config_v8_test.go` (identical to baseline). Applies with `patch -F3`; config tests pass. |
| `13fdcc5e` | fix(thinking): recognize claude thinking.display updates in summary pipeline | PORT | none | none | 3 files, +42/-1 | Applies clean; thinking/executor tests pass. |
| `c9dad847` | fix(home): only renew liveness on heartbeat pong | PORT | none | none | 4 files, +162/-10 | Applies clean; home tests pass. |
| `18883a57` | fix(usage,auth): propagate request-scoped error rule classification to usage fail.code | PORT | `usage_helpers.go` import order; `conductor_execution.go` and `conductor_home_execution.go` line drift | `216f2947`, `bfac8586` (issue 0001) context; `40c16182` line context | 10 files, +317/-79 | Self-contained symbols (`ResolveRequestScopedErrorRules`, `FailureForceCooldownCode`). Cannot be replayed on the baseline alone, so no group build/test run. Manual resolution is small. |
| `46d48eb5` | fix(openai): start responses websocket keepalive after upgrade | PORT | none | `23e31fe2` (issue 0001) test context | 3 files, +427/-241 | Applies clean; handlers tests pass. |
| `d576416a` | fix(compaction): strip text format constraints and configuration_update from summary requests | PORT | none | `61f1ad5e` (issue 0001) context only | 2 files, +18/-2 | Applies clean; executor tests pass. |
| `d1ea84f4` | docs(readme): remove AICodeMirror sponsorship banner | SKIP (sponsor rule; already absent) | `README.md` differs; no CN/JA READMEs | none | 4 files, +0/-12 | Fork README has no banner. `assets/aicodemirror.png` still exists; removal is optional cleanup, not required. |
| `effbac16` | feat(models): add Gemini 4 Carbon model entry to models.json | PORT | none | `19a0a854` (same file) | 1 file, +101/-0 | Applies after `19a0a854`; registry tests pass. |
| `949b43b3` | fix(xai): classify non-stream terminal SSE errors and empty or incomplete streams | PORT | none | none | 5 files, +908/-18 | Applies clean; executor tests pass. Adds `xaiTerminalFailureErr` and `HasPatchActivity`. |
| `69ed202a` | fix(registry): restore model listings once bounded cooldown deadlines expire | PORT | none | `098af7d5` (issue 0001) line context only | 4 files, +713/-47 | Applies clean; registry and auth tests pass. |
| `f77a3c10` | fix(meta): pass through explicit client prompt_cache_retention | PORT | `config.example.yaml` comment context | `ece3f52a` (issue 0001) context only | 3 files, +250/-7 | Applies clean; executor tests pass. |
| `ecebc713` | fix(auth): keep request retry rounds after scheduler plugin reject | PORT (small adaptation) | `conductor_execution.go`, `conductor_selection.go` | `18883a57`; one line touches `40c16182` code | 6 files, +642/-11 | New `schedulerRejectError` markers are self-contained. If `40c16182` is skipped, drop the one-line change in `normalizeSessionAffinityNeverTerminalError`. |
| `9d7d012b` | fix(auth): preserve large weighted pool fairness and force cooldowns | NEEDS-DECISION (with `d2045150`) | `conductor_cooldown.go`, `conductor_lifecycle.go`, `cooldown_state.go`, `selector.go`, `service_config.go` | `d2045150`, `e479ab12`, `e84d4836`, `14e57717`, `18883a57`, `cb70eacf`, `098af7d5`, `80ee739f`, `09b3a579` | 13 files, +4394/-139 | Largest commit. Also adds force-cooldown persistence: new optional `omitempty` JSON fields in `CooldownStateRecord` (additive, compatible). Not separable from the weighted rewrite in practice. |
| `8bba7a77` | docs(README): add AICodeMirror sponsor banner to EN, CN, and JA readmes | SKIP (sponsor content) | `README.md` differs; no CN/JA READMEs | none | 4 files, +12/-0 | Rule: skip sponsor and promotional content. |

Counts: PORT 25, SKIP 4, NEEDS-DECISION 5 (34 commits total).

### Proposed port groups (ordered)

1. **Translator and OpenAI streaming** — `c5251b57`, `f4d3f061`,
   `91e0dee8`, `46d48eb5`. ~2,000 patch lines. Verified: apply, build,
   `internal/translator` and `sdk/api/handlers` tests pass. One test import
   reorder.
2. **Plugin host and store** — `b5b724d0`, `f69db99d`, `00748974`,
   `14e57717`. ~3,100 lines. Verified: apply, build, pluginhost and
   management tests pass. One test import reorder.
3. **Executors and thinking** — `f36b195e`, `eb6e25ab`, `949b43b3`,
   `f77a3c10`, `d576416a`, `13fdcc5e`. ~1,800 lines. Verified: apply,
   build, executor and thinking tests pass.
4. **Logging, watcher, home, CLI** — `4b3f2c7e`, `628e54ab`, `c9dad847`,
   `0561334f`, `bebd92c4`. ~1,500 lines. Verified: apply with `patch -F3`,
   build, logging/watcher/home/auth tests pass.
5. **Model registry and config docs** — `19a0a854`, `effbac16`,
   `84534584`. Data and comments only. Verified: apply, build,
   config/registry tests pass.
6. **Registry and usage classification** — `69ed202a`, `18883a57`,
   `ecebc713`. Verified only for `69ed202a` (apply, build, tests).
   `18883a57` and `ecebc713` need manual conflict resolution and depend on
   issue-0001 commits (`216f2947`, `bfac8586`) plus `40c16182` for one line.
   Do this group after the auth decision.
7. **Auth routing (decision first)** — `40c16182`, then `d2045150`,
   `e479ab12`, `e84d4836`, `9d7d012b`. ~8,500 lines. Must be one issue and
   must wait for the issue-0001 commits that it needs. High conflict risk
   with fork routing code.

### Risks

- **Weighted-routing rewrite (group 7).** It rewrites the selector that the
  fork already extends with `SoonestResetSelector`, `AffinityPersister`,
  transient strikes, and `CandidateFilter`. Conflicts are semantic, not
  cosmetic. It also changes selection behavior for all routing strategies,
  not only `weighted-round-robin`.
- **Hidden coupling to issue 0001.** `18883a57`, `ecebc713`, and the whole
  weighted chain need claude/codex/auth commits from issue 0001. Porting
  them first creates a dependency on that work.
- **Cooldown persistence (`9d7d012b`).** Adds optional JSON fields to
  `CooldownStateRecord`. The change is additive, but it was not tested with
  an existing `kv.db`. Run a compatibility check if it lands.
- **Test import-order conflicts.** The fork orders imports differently from
  upstream. Every `*_test.go` conflict in this triage is only that; a
  `gofmt`/import sort after port fixes it.
- **Panel drift.** `f69db99d` adds a backend query parameter with no panel
  control. `d2045150`/`e84d4836` add routing behavior that the fork's
  routing panel does not describe.
- **Model data.** `19a0a854` and `effbac16` add model IDs that only upstream
  can serve. They are registry metadata only and do not create routes.

### Claims I could not confirm

- I did not run the full `go test ./...` suite. I ran only the package tests
  listed per group, on patched scratch clones.
- I did not verify `18883a57`, `ecebc713`, `40c16182`, or the weighted chain
  by build or test. They cannot be replayed on the baseline alone; I checked
  dependencies with `git log --oneline e5e6ad1e..<hash>^ -- <file>` and with
  failed cherry-picks.
- I did not verify the runtime behavior of `40c16182` (`never` mode),
  `d2045150`, `e479ab12`, `e84d4836`, or `9d7d012b`.
- I did not test `kv.db` compatibility for the new `CooldownStateRecord`
  fields. I only read the struct tags (`omitempty`, additive).
- I did not check whether the fork's `soonest-reset` selector and the
  upstream weighted pool can coexist without behavior changes.
- For the README commits I checked only `README.md` and the asset. I did not
  search `docs/` for the same sponsor or provider text.
- I did not run a browser or UI check. No UI change was made.
