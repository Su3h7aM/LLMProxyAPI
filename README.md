# LLMProxyAPI

LLMProxyAPI is a fork of CLIProxyAPI with the management dashboard built in.
One binary serves the proxy API and the web panel.
It also adopts selected ideas from magpie.

Status: early scaffold. No proxy code lives here yet.

## What this project is

- A Go proxy server that fronts CLI agent subscriptions (Claude, Codex,
  Gemini, and others) behind OpenAI, Responses, Gemini, and Claude compatible
  endpoints.
- The base code comes from CLIProxyAPI. The dashboard concept comes from
  CPA-Manager-Plus. Routing, rotation, redaction, and usage ideas come from
  magpie. Operations notes come from the `t3dotgg/cliproxy-setup` gist.
- Planned first changes, from the gist: soonest-reset-first routing, session
  affinity checks, Codex websocket support, 1h cache TTL guidance, and a
  quota panel.

## Credits

All credit for the base work belongs to the upstream projects.
This fork keeps their MIT license notices.

- CLIProxyAPI by Router-For.ME (MIT) —
  https://github.com/router-for-me/CLIProxyAPI
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
