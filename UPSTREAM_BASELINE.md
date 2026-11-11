# Upstream baseline

This file pins the upstream commits this fork is based on.
`.refs/` is not committed, so this file is the record for future updates.
To update, compare each pinned hash with the current upstream commit and
port the range you need.

Pinned on 2026-11-06 (UTC).

| Upstream | Remote | Commit | Date | Subject |
| --- | --- | --- | --- | --- |
| CLIProxyAPI | git@github.com:router-for-me/CLIProxyAPI.git | `8bba7a77c6bcbfd993516d26606bfecaa4ff49d0` | 2026-11-11 | docs(README): add AICodeMirror sponsor banner to EN, CN, and JA readmes |
| CPA-Manager-Plus | git@github.com:seakee/CPA-Manager-Plus.git | `a67ed8834a914ab24c36d2490fa6bb6d4166724c` | 2026-10-29 | Merge pull request #1105 from seakee/dev |
| magpie | git@github.com:yetone/magpie.git | `8cbaeecae976d2a8561293c1abc5daf6aa89d77f` | 2026-11-06 | providers: errors=pass sends a vendor's error to the agent as it came, asked once |
| Cli-Proxy-API-Management-Center | https://github.com/router-for-me/Cli-Proxy-API-Management-Center | `953fb2444350c6c36052dda46cac0abce6b23ea6` | 2026-11-06 | feat(aiCodeMirror): add AIcodemirror provider with support for multiple protocols and localization |
| BumpyClock-Cli-Proxy-API-Management-Center | https://github.com/BumpyClock/Cli-Proxy-API-Management-Center | `ce5f617961fb15b47ad178c512958abe1d3f2b92` | 2026-11-07 | fix(logs): retain path filters and support long-line wrapping (#17) |

Notes:

- CLIProxyAPI is the main source. Its license is MIT.
- BumpyClock-Cli-Proxy-API-Management-Center is the panel source. The
  panel in `apps/web` is vendored from it at pin `ce5f6179`. Its license
  is MIT (Router-For.ME notice). Lineage: official Management Center,
  then `kaitranntt` fork (dashboard for CLIProxyAPIPlus, base `v1.24.2`),
  then BumpyClock. The clone is
  `.refs/BumpyClock-Cli-Proxy-API-Management-Center`. The fork changed
  these on top of the pin:
  - the API layer speaks `/v8/management` only (no v0, no version
    negotiation, no legacy probes);
  - same-origin login: the panel ships inside the binary, so it has no
    API base URL setting and no remote connect flow; the management key
    login stays;
  - sponsor content and pages for CLIProxyAPIPlus-only providers
    (Copilot, Qoder) are removed;
  - fork features are rebuilt in the BumpyClock design: OpenCode Go
    keys and quota, the routing page with soonest-reset, SIWC, and
    Claude reset grants.
- Cli-Proxy-API-Management-Center (official) at pin `953fb24` was the
  panel source before the move to BumpyClock. It is no longer the
  source; `apps/web` no longer contains its tree. Its license is MIT.
- CPA-Manager-Plus is reference-only. Its license is MIT.
- magpie is the ideas source. Its license is MIT, with extra notices in its
  `THIRD_PARTY_NOTICES` file.
- The `t3dotgg/cliproxy-setup` gist is an operations guide, not a code base.

## Sync status: CLIProxyAPI

Reviewed up to `8bba7a77` (2026-11-11). The previous pin was `e5e6ad1e`.
Of the 75 non-merge commits in that range, 65 are ported. Each port commit
carries an `Upstream: router-for-me/CLIProxyAPI@<hash>` trailer; list them
with `git log --grep 'Upstream:'`.

Not ported (next sync starts from `8bba7a77`; handle these first):

| Commit | Reason |
| --- | --- |
| `40c16182` | Deferred. Rebind-on `never` mode. Port with the weighted chain. The omitted line of `ecebc713` goes with it. |
| `d2045150`, `e479ab12`, `e84d4836`, `9d7d012b` | Deferred. Weighted-routing rewrite (~8,500 lines); conflicts with fork selector code. |
| `ce38c75c` | Skipped. Reverted upstream by `2c5b01f4`. |
| `418b824c`, `59a2b353`, `d1ea84f4`, `8bba7a77` | Skipped. README and sponsor content. |

See `DECISIONS.md` for the reasons.
