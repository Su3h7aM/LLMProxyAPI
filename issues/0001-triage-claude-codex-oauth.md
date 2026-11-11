---
status: done
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

- 2026-11-11: Renamed checkout branch to `issue/0001-triage-claude-codex-oauth`, set
  `status: in-progress`, committed.
- 2026-11-11: Read all 42 diffs (`git -C .refs/LLMProxyAPI show <hash>`),
  full commit bodies, per-commit file stats.
- 2026-11-11: Compared each touched file fork-vs-baseline
  (`git -C .refs/LLMProxyAPI show e5e6ad1:<path>` vs fork file). Fork adds:
  module rename, agent-name rename, OpenCode Go keys, session affinity,
  soonest-reset routing, import re-sorting, web panel. No Plus code in fork
  paths touched here.
- 2026-11-11: Dry-run port on throwaway branch `tmp/0001-dryrun`: applied the
  42 rewritten patches (module path rewritten) in listed order with
  `git apply --check` then `git apply`. Result: 28 OK, 14 FAIL. Proved most
  FAILs are cascades inside port groups or trivial import-order context
  mismatches from the fork rename/re-sort. Deleted the throwaway branch.
- 2026-11-11: Root-caused each FAIL (per-file `git apply --check --verbose`,
  upstream `git log e5e6ad1..<commit> -- <file>` for out-of-scope deps).
  No sponsor/Plus/panel content in any of the 42 patches (grep scan clean).
  All 42 commits carry tests.

## Follow-ups

- None. Coordinator opens port issues per the groups below.

## Result

Legend: OK = rewritten patch applies cleanly on fork in group order.
Trivial = import-order-only context mismatch from fork rename/re-sort.

| Hash | Subject | Verdict | Fork conflicts | Depends on | Size | Evidence / notes |
| --- | --- | --- | --- | --- | --- | --- |
| 68cd757 | fix(codex): prevent replay after uncertain refresh outcomes | PORT | Trivial: 1 line in `nocopy_invariant_test.go` (fork added same-file entry with other wording) | None in scope; introduces `IsRefreshOutcomeUnknown`, journal infra | 32f +2047-34 | New refresh-journal + outcome infra; Closes #7289; 17 test files |
| 0559fa3 | feat(management): expose oauth_provider in credential list | PORT | None (OK) | None | 2f +126 | v8-only field, v0 frozen; additive |
| bfac858 | fix(codex): normalize small output budgets | PORT | None (OK) | None | 10f +272-27 | Closes #7304; min-budget 16 fix |
| 2443e88 | fix(codex): nested custom tools round trips | PORT | None (OK) | None | 5f +201-30 | Closes #7300 |
| 80ee739 | fix(codex): preserve SIWC rotations, honor renewal windows | PORT | Cascade: needs 68cd757 context in `siwc_auth.go` | 68cd757 | 8f +336-64 | Dry-run FAIL only from missing 68cd757 hunk |
| 3891eba | fix(codex): reuse issued SIWC client in reauth | PORT | Cascade: needs 80ee739 (`siwc_renewal_test.go`) | 68cd757, 80ee739 | 4f +197-2 | Plus trivial `siwc_bind.go` context |
| 8c6d8b1 | fix(codex): defer SIWC renewal on macOS wakes | PORT | None (OK) | None (new files) | 4f +110 | New darwin power files |
| baec297 | fix(auth): retain partition failures + SIWC wake cancel | PORT | Cascade: hunk needs 68cd757 `IsRefreshOutcomeUnknown` line | 68cd757 | 2f +73-1 | Fork `conductor_refresh.go` is rename-only |
| 03c394e | test(codex): deterministic SIWC power-state checks | PORT | None (OK) | Follows 8c6d8b1 | 3f +33-8 | Test-only |
| a46be7a | fix(codex): retain renewal windows in receipts | PORT | Cascade: needs files from 68cd757 + 80ee739 | 68cd757, 80ee739 | 3f +44-10 | Small |
| 374056f | fix(codex): prompt-cache prefix with ToolSearch | PORT | None (OK) | None | 2f +762-40 | Closes #7303 |
| 482efbf | feat(auth): OAuth hints + metadata preservation | PORT | Trivial: import order in `auth_files_provider_oauth.go` (`gin` moved in fork) | Follows 0559fa3 (same test file, applied OK on top) | 7f +1017-101 | Functional hunks apply; needs `host.HasMultipleAuthProviders` (present in fork), `ojson` (present) |
| ea0b471 | fix(claude): OAuth state and scopes | PORT | None (OK) | None | 7f +212-6 | Refs #7316 |
| 2776281 | fix(claude): mid-session system items placement | PORT | None (OK) | None | 10f +810-101 | Closes #7310 |
| 5f38d55 | fix(auth): native Codex available during SIWC recovery | PORT | None (OK) | 68cd757 (uses outcome-unknown codes) | 6f +494-11 | Dry-run OK; semantic link to 68cd757 codes |
| 69637a6 | fix(openai): skip whitespace-only reasoning | PORT | None (OK) | None | 2f +78-4 | Closes #7325 |
| ce38c75 | fix(claude): drain stream body to reuse connections | SKIP (reverted) | N/A | N/A | 4f +984-2 | Reverted by 2c5b01f in same range; port 2c5b01f only |
| e6931d5 | fix(translator): standalone effort on OpenAI requests | PORT | None (OK) | None | 4f +205-13 | Closes #7327 |
| 09b3a57 | fix(codex): SIWC deferral + expires_in parsing | PORT | Cascade: needs 68cd757 + 80ee739 + a46be7a hunks | 68cd757, 80ee739, a46be7a | 8f +98-11 | All failing hunks trace to skipped predecessors |
| 05e668c | fix(responses): sanitize orphan tool outputs | PORT | None (OK) | None | 5f +380-4 | Closes #7341 |
| 2c5b01f | fix(claude): disable default reuse, revert draining | PORT | None (OK) | Reverts ce38c75 (port instead of it) | 8f +313-990 | Closes #7357; deletes `detachable_context.go` |
| 098af7d | fix(auth): share/migrate cooldowns across spellings | PORT | None (OK) | None | 10f +1193-133 | Closes #7360; big but self-contained |
| 542a525 | fix(claude): native client UA extensions | PORT | None (OK) | None | 5f +401-29 | New `claude_user_agent.go` |
| 5930433 | fix(claude): retain fields with stabilized profiles | PORT | None (OK) | 542a525 (renames its files) | 5f +101-18 | Rename of 542a525 files; keep order |
| cc95661 | feat(gemini): asymmetric thoughtSignature replay KV | NEEDS-DECISION | Out-of-scope dep, not fork | f36b195e (out of scope) adds symmetric-skip logic + test this commit modifies | 10f +866-123 | Q: port f36b195e too (90 lines, tested), or adapt? Non-test hunks apply; test hunk fails without it |
| 00c6ae8 | perf(cache): no double base64 for replay KV | PORT | None (OK) | None (independent of cc95661) | 6f +112-31 | Small perf fix |
| 216f294 | feat(home): report reused dispatch turns in usage | PORT | Manual: config comment region rewritten by fork session-affinity; docs Home section rewritten; import-order context in duplex files | 61f1ad5 builds on its duplex hunk (port before 61f1ad5) | 15f +301-7 | Fork has `internal/home`; docs say "CPA" (use fork naming when porting) |
| beb4335 | fix(claude): top-level cache_control in cloaks | PORT | None (OK) | None | 4f +389-9 | Closes #7366 |
| 23e31fe | fix(executor): stabilize refresh/websocket tests | PORT | None (OK) | None | 4f +294-45 | Mostly tests + 3-line WS fix |
| 714f052 | fix(codex): hosted image_generation injection | PORT | Trivial: import order in `imagegen_test.go` (`websocket` moved in fork) | 61f1ad5 builds on its `execute.go` hunk | 10f +858-40 | Closes #7384 |
| b4e7dbd | fix(claude): CLI fingerprint baseline 2.1.321 | PORT | None (OK) | 6db7d4b builds on it | 17f +417-276 | Fingerprint-visible change |
| 6db7d4b | fix(claude): preserved thinking gates for Fable 5.2 | PORT | None (OK) | b4e7dbd | 4f +41-9 | Closes #7386 |
| 0e0cab0 | fix(openai): placeholder reasoning_content | PORT | None (OK) | None | 5f +467-10 | Closes #7398 |
| f0c9c2d | fix(claude): tool schemas with rejected names | PORT | None (OK) | None | 6f +646 | Closes #7389 |
| 61f1ad5 | fix(codex): local compaction for compat endpoints | PORT | Cascade + trivial: needs 714f052 `execute.go` hunk and 216f294 duplex hunk; import order in `compact_test.go` | 714f052, 216f294 | 16f +2189-68 | Fixes #7406; fork already has base compaction (Issue7118 capsules); `antigravity_compaction.go` hunk needs inspection at port time |
| 6876a2e | fix(claude): cli-bg x-app as native client | PORT | None (OK) | None | 3f +62-8 | Closes #7412 |
| cb70eac | fix(claude): unified ratelimit headers on 429s | PORT | None (OK) | Follows 098af7d (shared files, applied OK on top) | 10f +2042-74 | Closes #7417; touches fork soonest-reset selector areas but hunks disjoint |
| ece3f52 | fix(logging): bounded upstream responses in error logs | NEEDS-DECISION | Out-of-scope dep: `request_logging.go` hunks need 4b3f2c7e (`hasMatchedRoute`); `response_writer.go` applies with offset | 4b3f2c7e (out of scope, 434 lines + response_writer refactor) | 39f +2598-156 | Q: port 4b3f2c7e too? Largest commit in list; 8 test files |
| 9a7e1e6 | feat(executor): sensitive-words-replace, Antigravity obfuscation | PORT | Minor adjacency: `config_normalization.go:379` (fork added `SanitizeOpenCodeGoKeys` at same spot; keep both) | 65de937 and fa11a573 build on it | 26f +1281-72 | Closes #7428; changes obfuscation char U+200B to CGJ (wire-visible) |
| 65de937 | feat(claude): consume Home cloak credential_options | PORT | Cascade + trivial: `identity.go`/cloaking hunks need 9a7e1e6; import order in `claude_signing.go` | 9a7e1e6 | 7f +284-5 | Uses `SensitiveWordsReplace` from 9a7e1e6 |
| fa11a57 | fix(claude): server-side fallback opt-in | PORT | Cascade: cloaking hunk needs 9a7e1e6 struct fields | 9a7e1e6, 65de937 | 16f +1389-48 | Closes #7440; all other files apply |

Counts: PORT 39, SKIP 1 (ce38c75, reverted), NEEDS-DECISION 2
(cc95661, ece3f52).

Proposed port groups (ordered, one issue each):

1. SIWC refresh-outcome chain (in this order): 68cd757, 80ee739,
   3891eba, 8c6d8b1, baec297, 03c394e, a46be7a, 5f38d55, 09b3a57.
2. Management OAuth provider field: 0559fa3, 482efbf.
3. Codex request fixes: bfac858, 2443e88, 374056f, 05e668c.
4. Claude executor/translator fixes: ea0b471, 2776281, 69637a6,
   e6931d51, beb4335, 542a525, 5930433, b4e7dbd, 6db7d4b, 0e0cab0,
   f0c9c2d, 6876a2e, 2c5b01f, 098af7d, cb70eac.
5. Compaction/imagegen/Home usage (in this order): 714f052, 216f294,
   61f1ad5. Manual work: config comment, docs section, import contexts.
6. Cloak/sensitive-words/fallback (in this order): 9a7e1e6, 65de937,
   fa11a57.
7. Misc: 23e31fe, 00c6ae8. Blocked on coordinator: cc95661 (+f36b195e?),
   ece3f52 (+4b3f2c7e?).

Risks:

- Group 1 is ~3000 lines with tests in refresh/outcome machinery the fork
  has never carried. Behavior risk is high without running the new tests.
- 9a7e1e6 changes Antigravity obfuscation bytes and adds config keys;
  combined with fork OpenCode Go sanitize functions in the same files.
- b4e7dbd changes emulated CLI fingerprints (wire-visible).
- cb70eac and 098af7d touch the same selector/scheduler files as the fork
  soonest-reset feature. Patches apply but interaction is untested.
- 61f1ad5 extends compaction the fork already ships. Scope creep risk for
  is-compat models.
- 216f294 assumes Home/CPA accounting the fork never exercised. Rename
  CPA references to fork naming at port time.

Claims I could not confirm (no code changes allowed, no builds run):

- That each PORT patch compiles and its tests pass. Checked: `git apply
  --check` only, on `tmp/0001-dryrun` (deleted after). Where I looked:
  throwaway branch, per-file verbose checks.
- That `host.HasMultipleAuthProviders` in the fork behaves like upstream
  for 482efbf. Checked: symbol exists in fork `internal/pluginhost`, body
  not compared line by line.
- That fork `internal/home` matches upstream Home behavior needed by
  216f294/65de937. Checked: directory exists in fork, no behavior compare.
- Full semantic independence of OK commits from out-of-scope upstream
  commits. Checked: file-level (`git log e5e6ad1..<commit> -- <file>`)
  for every failing file; passing files assumed independent because their
  hunks matched the fork.
- 61f1ad5 `antigravity_compaction.go` hunk root cause beyond the one-line
  `)` context. Needs eyes at port time.
