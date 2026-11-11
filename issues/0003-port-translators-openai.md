---
status: done
branch: issue/0003-port-translators-openai
type: port
agent: deepseek-v4.1-flash
---

# Port upstream: translators and OpenAI streaming fixes

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
c5251b57 fix(translator): preserve reasoning and text across multi-part gemini and antigravity chunks
f4d3f061 fix(openai): replace responses websocket transcript when pending tool calls are unanswered
69637a66 fix(openai): skip whitespace-only reasoning when opening Claude thinking blocks
e6931d51 fix(translator): preserve standalone Claude output_config.effort on OpenAI requests
91e0dee8 fix(openai): prevent SSE keep-alive injection mid-line during image streaming
05e668cf fix(responses): sanitize orphan tool outputs lacking call_id
0e0cab0d fix(openai): inject placeholder reasoning_content for compat Claude tool calls
46d48eb5 fix(openai): start responses websocket keepalive after upgrade
```

Triage notes: Triage: all apply clean; only test import order differs. Verified by triage for the first four.

## How to port

1. Get the patch: `git -C /home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI show --format= <hash> > /tmp/<hash>.patch`.
2. Rewrite the module path in the patch: `github.com/router-for-me/CLIProxyAPI/v8` -> `github.com/Su3h7aM/LLMProxyAPI`. Then `git apply --3way` (or `patch -F3`). Fix only import-order context conflicts.
3. Run `gofmt -l` on touched Go files and fix. Keep the fork import order.
4. Commit: Conventional Commits, same subject as upstream, body ends with the line `Upstream: router-for-me/CLIProxyAPI@<full-hash>`. Also end with the Co-Authored-By trailer your harness gives you, if any.
5. Gate after each commit: `go build ./...` and `go test` for the touched packages. Gate at the end: `go build ./... && go vet ./... && go test ./...` (all must pass; baseline on `main` is green).

## Environment

- Use `/usr/bin/go` (1.27.3). Plain `go` is a mise shim and fails here (untrusted mise config). Do NOT run `mise trust` or change anything outside this checkout.
- `go build ./...` needs `apps/web/dist/management.html` (git ignores it). Create a stub: `mkdir -p apps/web/dist && echo '<html></html>' > apps/web/dist/management.html`. Never commit it.
- Code, docs, comments, commit messages: English only. Use ASD-STE100 Simplified Technical English in all your text, be concise.

## Rules

- Take only what the commits change. Do not refactor, rename, or "improve" anything else.
- Fork features stay. Protect: Go module `github.com/Su3h7aM/LLMProxyAPI`, vendored panel in `apps/web`, OpenCode Go provider/quota support, routing panel and `soonest-reset`, session affinity with strikes (`SessionAffinitySelector`, `AffinityPersister`), no sponsor/promo content, no Plus code. Upstream config, auth files, and `kv.db` must keep loading.
- You do not decide. If a conflict is semantic (not import order), if a test fails and you cannot fix it by a small adaptation inside the upstream commit's intent, or if anything is unclear: stop, write the question in `## Log

- Branch renamed from `t3/port-openai-translators` to `issue/0003-port-translators-openai`.
- Port method: `git show --format= <hash>` then `sed` module path to
  `github.com/Su3h7aM/LLMProxyAPI`, then `git apply --3way`. Every commit kept the
  upstream file set and the upstream insertion/deletion counts.
- Conflicts seen, all import-order or missing-blob context only:
  - `e6931d51`: `internal/thinking/claude_standalone_effort_test.go` import block.
    The fork file already had the codex test; the patch added the openai imports
    and `TestClaudeToOpenAIStandaloneEffort`. Applied correctly.
  - `05e668cf`: `sdk/api/handlers/openai/openai_responses_handlers.go:25` import
    block. Applied correctly.
  - `0e0cab0d`: `internal/runtime/executor/helps/model_capabilities.go:1` import
    block. Applied correctly.
  - `46d48eb5`: "repository lacks the necessary blob to perform 3-way merge" for
    `openai_responses_websocket_test.go`; plain application then applied cleanly.
- No semantic conflict. No upstream test weakened or skipped.
- `gofmt -l` is clean for all touched files. Two files that the port does not
  touch are unformatted on `main` too:
  `internal/runtime/executor/claude_thinking_replay_test.go` and
  `internal/runtime/executor/codex_stream_bootstrap_buffering_test.go`. Left as is.
- Gate results on the final branch:
  - `go build ./...` passes.
  - `go test ./...` passes (all packages ok).
  - `go vet ./...` exits 1 with one pre-existing finding:
    `internal/pluginhost/host_test.go:1637:8: assignment copies lock value to _: sync.Once contains sync.noCopy`.
    I checked `main` in a temporary worktree: it gives the same finding and the
    same exit 1. The port does not touch `internal/pluginhost`. So the baseline
    claim "vet is green on main" is not correct. I did not change that file
    because the issue says to take only what the commits change. Open question
    for the coordinator: is a pre-existing vet finding a gate failure, and who
    fixes it?

## Follow-ups

- `go vet ./...` fails on `main` and on this branch with the same
  `internal/pluginhost/host_test.go:1637` lock-copy finding. Needs a separate
  issue, or a coordinator decision to drop `go vet` from the gate.
- Two files in `internal/runtime/executor/` are not gofmt-clean on `main`.
  Needs a separate formatting issue if the repo wants `gofmt -l` clean.

## Result

All 8 upstream commits are ported, one fork commit per upstream commit, in the
listed order. Nothing was skipped, reworded, or merged into another commit.

| # | Upstream | Fork | Subject |
|---|----------|------|---------|
| 1 | `c5251b57b3fa5dcbc689f514f4aa166daacb4481` | `9bb12978` | fix(translator): preserve reasoning and text across multi-part gemini and antigravity chunks |
| 2 | `f4d3f0618d158c3168e113340fd06af15200d67a` | `7f1f5810` | fix(openai): replace responses websocket transcript when pending tool calls are unanswered |
| 3 | `69637a662a05aebc64805760be960284b2459cb2` | `10f3e9bb` | fix(openai): skip whitespace-only reasoning when opening Claude thinking blocks |
| 4 | `e6931d51e328343088d5902e5cbd06edbe4d8633` | `8af1cc02` | fix(translator): preserve standalone Claude output_config.effort on OpenAI requests |
| 5 | `91e0dee827abf476f9f86246f68c500a112f7155` | `72d9e901` | fix(openai): prevent SSE keep-alive injection mid-line during image streaming |
| 6 | `05e668cf2d3c58428c199e254147c92c19b45939` | `54823326` | fix(responses): sanitize orphan tool outputs lacking call_id |
| 7 | `0e0cab0de65a033183f1b7bbe622992e4a57cd1f` | `34abd6f0` | fix(openai): inject placeholder reasoning_content for compat Claude tool calls |
| 8 | `46d48eb5af062dda4e2d79ceeefedfde3f6917ba` | `87d77d1f` | fix(openai): start responses websocket keepalive after upgrade |

Each commit body ends with `Upstream: router-for-me/CLIProxyAPI@<full-hash>`.

Deviations from upstream: none in content. Only import-order context in three
files, as listed in the Log.

Gate: `go build ./...` passes, `go test ./...` passes, `go vet ./...` has one
pre-existing finding that also occurs on `main` (see Log and Follow-ups).

Known gaps: the vet finding above, and the two unformatted files on `main`.

Branch rebased onto `main` (`dde1a3f5`) with no conflict. Not merged, per the
issue.
