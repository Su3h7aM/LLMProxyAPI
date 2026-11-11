# LLM Proxy API Management Center

Web panel for LLM Proxy API. It is built from this directory and
embedded into the proxy binary: the root build copies
`dist/index.html` to `dist/management.html`, which `panel.go`
embeds and the server serves at `/management.html`.

It is based on the BumpyClock Management Center (pin `ce5f6179`),
with sponsor content and CLIProxyAPIPlus-only providers removed
and the brand changed to LLM Proxy API. The API layer still calls
`/v0/management`; the move to `/v8/management` happens in later
issues.

## Build

Needs bun 1.4.4.

```sh
bun install --frozen-lockfile
bun run verify
```

`bun run verify` runs tests, lint, and the production build.
