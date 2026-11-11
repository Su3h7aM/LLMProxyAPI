---
status: in-progress
branch: issue/0014-port-cooldown-registry-usage-classification
type: port
agent: muse-spark-1.3-contributor
---

# Port upstream: shared model cooldowns, registry recovery, usage error classification, 429 headers

## Goal

Port the upstream commits below into the fork, one fork commit per upstream
commit, in the listed order. Upstream is `router-for-me/CLIProxyAPI`, cloned at
`/home/su3h7am/Apps/LLMProxyAPI/.refs/CLIProxyAPI` (read-only: never edit,
commit, or change it; read with `git -C ... show/diff/log`). Triage reports are
in `issues/0001-*.md` and `issues/0002-*.md`.

## Commits (apply in this order)

```
098af7d5 fix(auth): share and migrate model cooldowns across equivalent model spellings
18883a57 fix(usage,auth): propagate request-scoped error rule classification to usage fail.code
cb70eacf fix(claude): emit anthropic-ratelimit-unified headers on local cooldown 429s
69ed202a fix(registry): restore model listings once bounded cooldown deadlines expire
ecebc713 fix(auth): keep request retry rounds after scheduler plugin reject
```

Triage notes: Triage: these touch the auth conductor and selector files that the fork extends (`SoonestResetSelector`, `AffinityPersister`, strikes in `SessionAffinitySelector`, `CandidateFilter`). Hunks were judged disjoint, but check by running the fork routing tests (`sdk/cliproxy/auth`, soonest-reset, session-affinity). `098af7d` migrates and shares cooldowns across equivalent model spellings: check the persisted cooldown JSON: any changed field must be additive (`omitempty`) so an old record still loads; report what you checked. `18883a57` had context from `216f2947` and `bfac8586` (both on `main` now) and line context from upstream `40c16182` (rebind-on `never`, NOT ported, deferred by decision): resolve by hand, small. `ecebc713`: one line touches `normalizeSessionAffinityNeverTerminalError`, which exists only with `40c16182`. Port the commit WITHOUT that part and add to the commit body: `Port note: omitted the 40c16182-dependent change (rebind-on never mode is deferred).` Do not add the `never` mode. Keep the order.

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
- You do not decide. If a conflict is semantic (not import order), if a test fails and you cannot fix it by a small adaptation inside the upstream commit's intent, or if anything is unclear: stop, write the question in `## Log`, commit, and report to the coordinator. Do not skip a commit or weaken a test to get green.
- Do not edit `DECISIONS.md` or other issue files. List follow-ups under `## Follow-ups`.

## Workflow

1. In your checkout create branch `issue/0014-port-cooldown-registry-usage-classification` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current (what you did, what you found, open questions).
3. When all commits are in and the gate passes: `git rebase main` (resolve conflicts; re-run the gate), set `status: done`, fill `## Result` (commits ported with new fork hashes, gate output summary, deviations from upstream, known gaps), commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

## Follow-ups

## Result
