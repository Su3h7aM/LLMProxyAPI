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

The panel lives in `apps/web` and ships inside the proxy binary: one
project, one binary. Build order is `build-web`, `stage-panel`,
`build-server` (each depends on the previous). The binary serves the
embedded panel at `/management.html`. Nothing is downloaded at runtime.
Panel development can still point at a live bundle:

```
MANAGEMENT_STATIC_PATH=$PWD/apps/web/dist ./llmproxy -config config.yaml
```

The external management API (`/v0`, `/v8`) keeps working for other
clients.
