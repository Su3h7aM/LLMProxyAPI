# Upstream baseline

This file pins the upstream commits this fork is based on.
`.refs/` is not committed, so this file is the record for future updates.
To update, compare each pinned hash with the current upstream commit and
port the range you need.

Pinned on 2026-11-06 (UTC).

| Upstream | Remote | Commit | Date | Subject |
| --- | --- | --- | --- | --- |
| CLIProxyAPI | git@github.com:router-for-me/CLIProxyAPI.git | `e5e6ad1e1b91fc20ab9964e6fb22c0c269b9d882` | 2026-11-06 | Merge pull request #7295 from router-for-me/fix/codex-hosted-image-request-scope |
| CPA-Manager-Plus | git@github.com:seakee/CPA-Manager-Plus.git | `a67ed8834a914ab24c36d2490fa6bb6d4166724c` | 2026-10-29 | Merge pull request #1105 from seakee/dev |
| magpie | git@github.com:yetone/magpie.git | `8cbaeecae976d2a8561293c1abc5daf6aa89d77f` | 2026-11-06 | providers: errors=pass sends a vendor's error to the agent as it came, asked once |

Notes:

- CLIProxyAPI is the main source. Its license is MIT.
- CPA-Manager-Plus is the dashboard source. Its license is MIT.
- magpie is the ideas source. Its license is MIT, with extra notices in its
  `THIRD_PARTY_NOTICES` file.
- The `t3dotgg/cliproxy-setup` gist is an operations guide, not a code base.
