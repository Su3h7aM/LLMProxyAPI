---
status: todo
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

## Follow-ups

## Result
