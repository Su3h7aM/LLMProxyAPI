# LLMProxyAPI

LLMProxyAPI is a proxy server with a built-in management dashboard. One binary serves the proxy API and the web panel.

Based on [router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI).

## What this project is

- A Go proxy server that fronts CLI agent subscriptions (Claude, Codex,
  Gemini, and others) behind OpenAI, Responses, Gemini, and Claude compatible
  endpoints.
- The dashboard concept comes from CPA-Manager-Plus. Routing, rotation, redaction, and usage ideas come from magpie. Operations notes come from the `t3dotgg/cliproxy-setup` gist.
- Planned first changes, from the gist: soonest-reset-first routing, session
  affinity checks, Codex websocket support, 1h cache TTL guidance, and a
  quota panel.

## Credits

All credit for the base work belongs to the upstream projects.
This fork keeps their MIT license notices.

- CPA-Manager-Plus by Seakee (MIT) —
  https://github.com/seakee/CPA-Manager-Plus
- magpie by yetone (MIT) —
  https://github.com/yetone/magpie
- Hub setup guide by t3dotgg —
  https://gist.github.com/t3dotgg/59a521d2ee8d70aaf90e280c175072af

## Upstream baseline

Pinned upstream commits live in `UPSTREAM_BASELINE.md`.
The `.refs/` directory used for review is not committed.
To update from upstream, diff the pinned hash against the current upstream
commit and port the range you need.

## Layout

- `README.md` — this file.
- `UPSTREAM_BASELINE.md` — pinned upstream commits.
- `AGENTS.md` — agent workflow rules.

## Tools and tasks

Clone the fork with `git clone https://github.com/Su3h7aM/LLMProxyAPI.git`. The default image in `compose.yaml` is `su3h7am/llm-proxy-api:latest`.

Tools (Go, Node) and tasks come from mise. The config lives in
`.mise/config.toml`. Task scripts live in `scripts/` and also run
directly without mise. Common tasks:

```
mise run build-server   # build the proxy binary
mise run build-web      # build the web panel bundle
mise run stage-panel    # stage the bundle as management.html
mise run check-panel    # boot the server and check the panel serves
mise tasks ls          # list all tasks
```

## Panel

The panel in `apps/web` is the
[BumpyClock Management Center](https://github.com/BumpyClock/Cli-Proxy-API-Management-Center)
at pin `ce5f6179`, adapted to this fork. It is part of the proxy binary:
one project, one process, one port. The binary serves it at
`/management.html` from the same origin. Nothing is downloaded at
runtime.

The panel calls `/v8/management` only. It has no API base URL setting and
no remote connect flow. The management key login stays: the browser
stores the key and sends it as `Authorization: Bearer` on every
management call.

### Build the panel

With mise, run `mise run build-web` and `mise run stage-panel`. The chain
`build-web`, `stage-panel`, `build-server` runs in order, because each
task depends on the previous. `stage-panel` copies
`apps/web/dist/index.html` to `apps/web/dist/management.html`, which
`apps/web/panel.go` embeds.

With bun alone:

```
cd apps/web
bun install --frozen-lockfile
bun run verify
```

Bun 1.4.4 is required. `bun run verify` runs the tests, lint, type-check,
and the production build. The build needs `bun` on `PATH`, because the
design token plugin calls it.

For UI work, point the server at a live bundle:

```
MANAGEMENT_STATIC_PATH=$PWD/apps/web/dist ./llmproxy -config config.yaml
```

The external management API (`/v0`, `/v8`) keeps working for other
clients.
