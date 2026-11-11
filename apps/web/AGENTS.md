# Panel agent notes

Source: BumpyClock Cli Proxy API Management Center at pin
`ce5f617961fb15b47ad178c512958abe1d3f2b92`. It is a reference only.
Do not copy sponsor content or CLIProxyAPIPlus-only providers
(Copilot, Qoder, sponsor brands) unless the fork backend serves them.

Rules:

- One unit: the panel ships inside the proxy binary (same origin, no
  remote connect, no custom API base). Management key login stays.
- v8 only: the panel targets `/v8/management`. No v0 compatibility
  code, no version negotiation, no legacy probes, no fallback to an
  older backend. Check the route and the response shape in the Go
  handlers under `internal/api/handlers/management/` before you write a
  client call.
- Design tokens: never write raw sizes, colors, or fonts in styles.
  Use the tokens in `src/design-system`. After a token change run
  `bun run tokens`; `bun run tokens:check` must pass.
- Brand: product name is `LLM Proxy API`, short names `LPAMC`/`LPA`.
  Do not rename technical identifiers (`x-cpa-*` headers, storage
  keys, CSS class names, file names).

Build: `bun install --frozen-lockfile`, then `bun run verify`
(test, lint, type-check, build). The build writes one file,
`dist/index.html`; the root build copies it to
`dist/management.html`, which `panel.go` embeds.

Add a page:

1. `src/features/<area>/<Name>Page.tsx` plus an optional `.module.scss`.
   Use the design system components, tokens, and layout.
2. Route in `src/router/MainRoutes.tsx`.
3. Sidebar item in `src/components/layout/MainLayout.tsx`, label keys in
   `src/i18n/locales`.
4. API calls in `src/services/api/` against `/v8/management`.
5. Test in `tests/`.

`bun run type-check` and the tests of your area run after each commit.
