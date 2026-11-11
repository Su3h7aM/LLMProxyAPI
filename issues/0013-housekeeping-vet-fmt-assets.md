---
status: in-progress
branch: issue/0013-housekeeping-vet-fmt-assets
type: chore
agent: deepseek-v4.1-flash
---

# Housekeeping: vet finding, gofmt, unused partner banners

## Goal

Three small cleanups on `main`. No upstream port here.

1. `go vet ./...` reports `internal/pluginhost/host_test.go:1637:8: assignment copies lock value to _: sync.Once contains sync.noCopy`. Fix it with the smallest change that keeps the test meaning (for example take a pointer or compare a field instead of copying the value). Do not weaken the test. After the fix `go vet ./...` must be clean.
2. Two files are not gofmt-clean: `internal/runtime/executor/claude_thinking_replay_test.go` and `internal/runtime/executor/codex_stream_bootstrap_buffering_test.go`. Run `gofmt -w` on these two files only. Check that `gofmt -l $(git ls-files '*.go')` lists nothing else; if it lists other files, do not touch them, log the list.
3. Partner/sponsor banner images in `assets/` are unused. At triage time none of these top-level files is referenced anywhere outside `assets/`: `apikey.png axisnow.png bestproxy.png chenju.png cubence.png fennoai.png infistar.png kedaya.jpg lingtrue.png modelflare.png openlux.png packycode-cn.png packycode-en.png packycode.png patewayai.png poixeai.png qiniucloud.png rapidproxy.png runapi.png swiftproxy.png xcode.gif`. For each file run `grep -rIl --exclude-dir=.refs --exclude-dir=node_modules --exclude-dir=.git <name> .` (ignore hits inside `assets/`). Delete only files with zero hits. Do NOT touch `assets/logo/`. Report any file that has a hit.

One commit per item (Conventional Commits: `fix(test): ...`, `style: ...`, `chore: remove unused partner banners`).

## Environment

- Use `/usr/bin/go` (1.27.3). Plain `go` is a mise shim and fails here (untrusted mise config). Do NOT run `mise trust` or change anything outside this checkout.
- `go build ./...` needs `apps/web/dist/management.html` (git ignores it). Create a stub: `mkdir -p apps/web/dist && echo '<html></html>' > apps/web/dist/management.html`. Never commit it.
- Code, docs, comments, commit messages: English only. Use ASD-STE100 Simplified Technical English in all your text, be concise.
- Gate at the end: `go build ./... && go vet ./... && go test ./...` all pass (vet must be fully clean now).

## Rules

- Change only what this issue lists.
- You do not decide. If something is unclear, stop, log the question in `## Log`, commit, and report.
- Do not edit `DECISIONS.md` or other issue files.

## Workflow

1. In your checkout create branch `issue/0013-housekeeping-vet-fmt-assets` (rename your branch if T3 gave another name). Set `status: in-progress` in this file and commit.
2. Work in small steps, commit often. Keep `## Log` current.
3. When done and the gate passes: `git rebase main`, set `status: done`, fill `## Result`, commit.
4. Do NOT merge to main. Report to the coordinator: short summary, branch, commit range, follow-ups, surprises. Mark each claim you could not confirm.

## Log

## Follow-ups

## Result
