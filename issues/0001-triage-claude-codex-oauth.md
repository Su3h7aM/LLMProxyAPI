---
status: in-progress
branch: issue/0001-triage-claude-codex-oauth
type: triage (read-only, no code changes)
---

# Triage upstream: Claude, Codex, OAuth, SIWC

## Goal

Review upstream commits and decide, per commit, if the fork should take it.
Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/LLMProxyAPI` (read-only: never edit or
commit there). Baseline pinned in `UPSTREAM_BASELINE.md`:
`e5e6ad1e1b91fc20ab9964e6fb22c0c269b9d882`. Range: `e5e6ad1e..HEAD`.

Area: commits that touch claude, codex, oauth, or siwc files

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
68cd75724e4de4e0450475cc99c7491166cf3392 fix(codex): prevent replay after uncertain refresh outcomes
0559fa3cfd5722cf1755a7dc3a90573ede3835f6 feat(management): expose oauth_provider in credential list
bfac85864cd41a8ad253364fda4381cf5eeef50a fix(codex): normalize small output budgets for all source formats
2443e8830c401e6054370c109388bb3b59daf21c fix(codex): normalize nested custom tools across Chat Completions round trips
80ee739f33c403cef25c5489e2734fab343723e9 fix(codex): preserve SIWC rotations and honor renewal windows
3891ebaab5f1e8b27507ce25dbf379166739d51e fix(codex): reuse the issued SIWC client during CLI reauthorization
8c6d8b1801eb466b52dcc8950786a55aeab60e09 fix(codex): defer SIWC renewal during macOS maintenance wakes
baec297fec2943271058ba13a60d0d55f8955092 fix(auth): retain partition failures alongside SIWC wake cancellation
03c394ee3ffeadcf1366210d27ce2d71dca50f73 test(codex): make SIWC power-state checks deterministic
a46be7ae3bc13225abbb9e217e9ae1674d54c0e4 fix(codex): retain SIWC renewal windows in refresh receipts
374056f28bfaa6b7582a82cba9f8ca84868f642e fix(codex): preserve prompt cache prefix when loading deferred tools via ToolSearch
482efbf5a777771523c1237ed7aaee965451c51d feat(auth): enhance OAuth provider handling with explicit hints and metadata preservation
ea0b471291a992c5d80358917dabc6cb9f0f25f6 fix(claude): align OAuth authorization state and scopes
277628193f54d99b2eeca57ecd7f9ff9b349a996 fix(claude): keep mid-session responses system items in place to preserve prompt cache
5f38d5572c432c67ec260be55f73b0ec72ac0764 fix(auth): keep native Codex available during SIWC recovery
69637a662a05aebc64805760be960284b2459cb2 fix(openai): skip whitespace-only reasoning when opening Claude thinking blocks
ce38c75c624df1ceb0777f3e7c21f892883569ed fix(claude): drain streamed response body after message_stop to reuse upstream connections
e6931d51e328343088d5902e5cbd06edbe4d8633 fix(translator): preserve standalone Claude output_config.effort on OpenAI requests
09b3a57981652fa80ffe2a349a88e18e45fea56e fix(codex): harden SIWC refresh deferral and expires_in parsing
05e668cf2d3c58428c199e254147c92c19b45939 fix(responses): sanitize orphan tool outputs lacking call_id
2c5b01f4394819ee375f37c3bf9cbe81312f4656 fix(claude): disable default upstream connection reuse and revert post-stream draining
098af7d54309c488d9e1a7c5a1c807302df8f131 fix(auth): share and migrate model cooldowns across equivalent model spellings
542a525fcf2a20ec920e538d94d78471b9dda2f5 fix(claude): recognize native client user-agent extensions
593043301416879f5476939045db063003928771 fix(claude): retain request fields with stabilized client profiles
cc95661ce438a528fa7283bf93f7dbf684d1e3fe feat(gemini): support asymmetric thoughtSignature replay KV across Gemini executors
00c6ae82caa21cce5f2133897603de168ff7ddfc perf(cache): eliminate double base64 encoding for reasoning replay KV arrays
216f2947fc19756581ebacb38cc231e68dee5e46 feat(home): report reused dispatch turns in usage
beb4335954f2bc85a710d1cb4d0210f4c234b813 fix(claude): materialize top-level cache_control in cloaked requests
23e31fe288181e2d97617a32e204618141a6ea50 fix(executor): stabilize Codex refresh and websocket tests under CPU contention
714f052618c8a8c017efe81a97aa51311d96ddcd fix(codex): allow hosted image_generation injection on standard Responses without reasoning
b4e7dbd2c984ec4a74d829ee0425430a20146c77 fix(claude): bump Claude Code CLI fingerprint baseline to 2.1.321
6db7d4b839d86c5b24e17bbe73bcd4db18ef1b1a fix(claude): extend preserved thinking and per-turn timing gates to Claude Fable 5.2
0e0cab0de65a033183f1b7bbe622992e4a57cd1f fix(openai): inject placeholder reasoning_content for compat Claude tool calls
f0c9c2d01898127b5f33b2baebb85d9d9feca018 fix(claude): embed tool schemas with rejected property names in descriptions
61f1ad5e3e2239e9a3296a415e8fd89d13b2a996 fix(codex): support local compaction for compatible Responses endpoints
6876a2eb416d10235f4aac7458bcd04c751e26f7 fix(claude): treat cli-bg x-app header as native claude code client
cb70eacfa533d44efb1eec5007098321e396fc3c fix(claude): emit anthropic-ratelimit-unified headers on local cooldown 429s
ece3f52a4146f2bdf49e28d2d65572668a3f897e fix(logging): preserve bounded upstream responses in error logs
9a7e1e6b412747729dedecefd7e23bd2e3ee1d3d feat(executor): add sensitive-words-replace and harden Antigravity obfuscation
65de937cf19b88436ee8519caa42f0fcb66e2307 feat(claude): consume Home credential_options cloak settings
fa11a573b252b30256bef4de952a95feee3c8c9d fix(claude): make server-side fallback opt-in and surface serving models
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
