# LLM Proxy API Management Center

Web panel for LLM Proxy API. It is built from this directory and
embedded into the proxy binary: the root build copies
`dist/index.html` to `dist/management.html`, which `panel.go`
embeds and the server serves at `/management.html`.

The panel comes from the BumpyClock Management Center (pin `ce5f6179`),
with sponsor content and CLIProxyAPIPlus-only providers removed and the
brand changed to LLM Proxy API.

The panel and the proxy are one unit. The panel calls
`/v8/management` only. It has no API base URL setting, no remote
connect flow, and no version negotiation or fallback to an older
backend. The management key login stays.

## Build

Needs bun 1.4.4.

```sh
bun install --frozen-lockfile
bun run verify
```

`bun run verify` runs the tests, lint, type-check, and the production
build. Use the same command before you push. The build needs `bun` on
`PATH`, because the design token plugin calls it.

## Run

The proxy serves the built panel. To check it end to end, run
`bun run build`, then start the server with
`MANAGEMENT_STATIC_PATH=$PWD/dist` (see the root `README.md`).
`bun run dev` starts the Vite dev server, but it does not proxy the API.

## Add a page

1. Create `src/features/<area>/<Name>Page.tsx` with an optional
   `<Name>Page.module.scss`. Use the design system: its components,
   tokens, and layout. Do not write raw colors, sizes, or fonts; use
   `src/design-system` and run `bun run tokens:check`.
2. Add the route to `src/router/MainRoutes.tsx`.
3. Add the item to `src/components/layout/MainLayout.tsx` and the label
   keys to the locales in `src/i18n/locales`.
4. Add API calls in `src/services/api/` against `/v8/management`.
   Check the route and the response shape in the Go handlers under
   `internal/api/handlers/management/` before you write the client call.
5. Add a test in `tests/` for the logic the page owns.
