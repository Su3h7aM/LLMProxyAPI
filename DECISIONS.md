# DECISIONS

Format: date, context, decision, reason. Newest first.

- 2026-11-11: Panel migration plan (after triage 0015, 0016, 0017).
  The BumpyClock panel is v0-based and older than the vendored official
  panel in coverage (no routing board, no v8 policy editors, no Claude
  reset grants, 4 locales instead of 7). Decisions: (1) replace the
  `apps/web` tree with the BumpyClock tree, then port its API layer to v8
  only, using the pre-import panel (tag `panel-pre-bumpyclock`) as the
  reference for v8 request and response shapes; (2) work on branch
  `integration/panel-bumpyclock`, merge to `main` only when the panel
  reaches parity (builds, serves, works on v8, fork features present),
  because the import alone would drop fork features; (3) parity rule: the
  features of the current panel (routing board, provider and auth-file
  policy editors, Claude reset grants, OpenCode Go, soonest-reset) are
  rebuilt in the BumpyClock design, none is dropped; (4) locales: keep the
  4 of BumpyClock for now; ko, tr, vi are a later, low-priority step; (5)
  remove sponsor content and the Copilot and Qoder pages (no backend
  route); (6) remove version negotiation and legacy probes; keep the
  revision guard and 401 handling; (7) shortcut endpoints (`views/`
  prefix, `observed_at`) come after parity, as additive endpoints only;
  contract generation stays deferred. Reason: developer order.
- 2026-11-11: Panel design rule. The BumpyClock interface, style, and
  design system are not changed. Fork-only features are added to it so that
  they look native: they use its components, tokens, layout, and patterns.
  Fork-only features: first-class OpenCode Go (keys, quota), the reset
  system (quota reset windows), and soonest-reset routing (use the account
  whose window resets first). No support for old backends or old panel
  versions anywhere in the panel. Reason: developer order.
- 2026-11-11: The panel and the proxy are one unit (one container, one
  repository, one version). Meaning for the panel: v8 API only, no v0
  compatibility; no code for a separate or unknown server (backend probe,
  version negotiation, endpoint-missing fallbacks, custom API address,
  remote connect). Prefer direct paths: where the panel needs several
  calls or heavy client work only because the API is generic, add a
  panel-oriented endpoint. Limit: released `/v0` and `/v8` contracts never
  change or shrink; only additive endpoints. The management key login
  stays. Reason: developer order.
- 2026-11-11: The dashboard (`apps/web`) moves to the BumpyClock
  Management Center (pin `ce5f6179`): its UI and its frontend stack replace
  the vendored official panel. The panel stays part of the binary. It is
  same-origin: no API base URL setting and no remote server connect flow;
  the management key login stays (security). Fork panel features stay on
  top (OpenCode Go and quota tab, soonest-reset in routing, no sponsor
  content, fork release check). After the move, BumpyClock is only a
  reference for later improvements. Sponsor content and pages for
  CLIProxyAPIPlus-only providers are not copied unless the fork backend
  serves them. Reason: developer order.
- 2026-11-11: Port order swap: `00c6ae82` is ported before `cc95661c`.
  Upstream `cc95661c` does not build alone (it uses a cache helper that
  `00c6ae82` adds). Each fork commit must build.
- 2026-11-11: Fork focus is the API on a server or in a container (VPS,
  container image); the developer connects harnesses to the API only.
  Features for a program on the user's own machine (CLI login helpers,
  harness auto-configuration, OS-specific behavior) are low priority: do
  not spend effort on them, do not remove them. Port them when they come
  with a normal upstream sync, because staying close to upstream makes
  later syncs easier. Example: `bebd92c4` (`-login-proxy`) is ported;
  macOS wake handling (`8c6d8b18`, `03c394ee`) and CLI re-login
  (`3891ebaa`) stay in place. Reason: developer order.
- 2026-11-11: Gate for ports: `go build ./... && go vet ./... && go test
  ./...` all pass, and `gofmt -l` lists nothing. `go vet` and `gofmt` were
  made clean in issue 0013 (before that, one old vet finding existed in
  `internal/pluginhost/host_test.go`). No new finding is allowed.
- 2026-11-11: Upstream sync, round 1 (baseline e5e6ad1e to 8bba7a77).
  Port bug fixes and valid improvements only; one fork commit per upstream
  commit with an `Upstream:` trailer. Skip: `ce38c75` (reverted upstream by
  `2c5b01f`), sponsor and README commits (`8bba7a77`, `d1ea84f4`,
  `418b824c`, `59a2b353`). Reason: developer order; fork features stay.
- 2026-11-11: Weighted-routing rewrite chain is deferred: `40c16182`
  (rebind-on `never`), `d2045150`, `e479ab12`, `e84d4836`, `9d7d012b`.
  These are features, not fixes (~8,500 lines) and they rewrite the selector
  that the fork extends (soonest-reset, affinity persister, strikes).
  Revisit as one dedicated issue after the other ports land.
- 2026-11-11: Out-of-range dependencies are ported first: `f36b195e`
  before `cc95661`; `4b3f2c7e` before `ece3f52`. Both are small, tested
  fixes. Reason: the dependent commits do not apply without them.
- 2026-11-11: The upstream clone lives in `.refs/CLIProxyAPI`, named after
  the upstream repository (`router-for-me/CLIProxyAPI`), not after the fork.
- 2026-11-11: Go gate for ports: `/usr/bin/go build ./... && go vet ./... &&
  go test ./...` (needs a stub `apps/web/dist/management.html`). Plain `go`
  is a mise shim that needs `mise trust`; agents must not trust it.
- 2026-11-06: Panel is part of the fork. The official CLI Proxy API
  Management Center is vendored into `apps/web` (pin `953fb24`) and the
  server binary embeds `dist/management.html`. No console bundle is
  downloaded at runtime. Reason: developer order; single unified
  project, first intentional difference from upstream.
- 2026-11-06: Never bring Plus code in without asking first. Standing
  developer rule.
- 2026-11-06: Upstream-to-fork data compatibility is mandatory; fork to
  upstream is not. Config, auth files, and kv.db from upstream must load
  and keep working. Any future store change ships a migration that
  preserves existing data. Current state: formats unchanged, new fields
  purely additive.
- 2026-11-06: Fork versions use CalVer YY.MM.DD with a letter for same-day
  fixes (26.11.06, 26.11.06a). Same tag-driven release flow as upstream;
  `scripts/version.sh` prints the next version. Service image runs as the
  `llmproxy` user, never root.
- 2026-11-06: OpenCode Go subscription is first-class in the backend
  (Sketch 1). Same key-list and quota/fetch section as Codex/Anthropic.
  Reference: magpie `readOpenCodeGo` plus the Go usage endpoint docs.
