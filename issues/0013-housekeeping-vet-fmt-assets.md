---
status: done
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

- 2026-11-11: Branch renamed from `t3/housekeeping-vet-fmt-assets` to `issue/0013-housekeeping-vet-fmt-assets`. Status set to `in-progress`.
- Item 1: the vet finding is the dead local `var reconfigureOnce sync.Once` and `_ = reconfigureOnce` in `TestHostApplyConfigWaitsForBlockedQuiesceBeforeCanceledRollbackAndReplay`. Nothing reads the variable. I removed both lines. No test assertion changed. `go vet ./internal/pluginhost/` is clean, `go test ./internal/pluginhost/` passes.
- Item 2: `gofmt -l $(git ls-files '*.go')` listed exactly the two files in the issue. After `gofmt -w` on them the list is empty. No other file needed a change.
- Gate: `go build ./...` pass, `go vet ./...` pass with no output. The first `go test ./...` run had one failure (`TestAntigravityManagerExecuteAndStream_ToppedUpCreditsRecoverDuringQuotaCooldown`, see Follow-ups); the second full run passed. `git rebase main` was a no-op because `main` did not move.
- Item 3: exact-name grep per the issue returns one hit for every listed file, and that hit is this issue file (`issues/0013-housekeeping-vet-fmt-assets.md`), which only lists the names. I treat that as the source of the list, not a reference, so all 21 files had zero real hits and I deleted them. Case-insensitive `grep -F` also returned no hit. A looser basename search (`apikey`, `fennoai`, `infistar`, `qiniucloud`, `xcode`, `packycode`, `poixeai`, `runapi`) returns only unrelated words in source, i18n, and issue 0007. No `go:embed` directive points to `assets/`. `assets/logo/` was not touched.

## Follow-ups

- `TestAntigravityManagerExecuteAndStream_ToppedUpCreditsRecoverDuringQuotaCooldown` failed once in a full parallel `go test ./...` run, then passed on rerun and 50 focused runs. It looks time-dependent (uses `time.Now()` and `time.Local`). It is outside this issue.

## Result

Landed on `issue/0013-housekeeping-vet-fmt-assets`, five commits over `c46d61a1`:

1. `fix(test): drop dead sync.Once value in plugin reconfigure case`. `go vet ./...` is now fully clean.
2. `style: gofmt claude thinking replay and codex bootstrap buffering tests`. `gofmt -l $(git ls-files '*.go')` is empty.
3. `chore: remove unused partner banners`. Deleted 21 unreferenced top-level files in `assets/`. `assets/logo/` is untouched.
4. `chore(issues): start issue 0013 housekeeping` and `docs(issues): log housekeeping progress for issue 0013`.

Gate at `HEAD`: `go build ./...` pass, `go vet ./...` pass with no output, `go test ./...` pass on the second run (first run had the flaky antigravity test above).

How to use: nothing to configure. `go vet ./...` is clean, so the CI vet step can be strict later.

Known gaps: no test asserts the vet cleanliness, so a new copy of a lock value can come back. A `go vet` step in CI is the durable guard, if the coordinator wants it.
