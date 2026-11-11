# PLAN: vendor proxy + dashboard into one binary

Goal: bring `.refs/CLIProxyAPI` and `.refs/CPA-Manager-Plus` into this repo
as one project where the panel ships native with the proxy, while the
external management API keeps working. No feature is removed in this plan.

## Steps

1. [x] Vendor LLMProxyAPI tree to repo root (skip `.git`, `.jj`, root meta
   files that collide: `.gitignore`, `AGENTS.md`, `README*`, `LICENSE`).
   Check: `go build ./...` passes in the repo root.
   Evidence: `go build ./...` EXIT:0 with Go 1.27.3, first run fetched
   modules. Root `LICENSE` keeps the upstream MIT notice.
2. Vendor CPAMP `apps/web` and `apps/manager-server` with a root
   `package.json` workspace and a `go.work` that covers both Go modules.
   Check: web build passes and both Go modules build.
3. Serve the CPAMP web bundle from the proxy binary at `/management.html`.
   Resolve the route conflict: proxy owns `/management.html` today via a
   downloaded asset; serve the vendored bundle instead.
   Check: one process serves `GET /management.html` with 200.
4. Run the manager backend in the same process (loopback transport as an
   interim step, documented as such).
   Check: panel loads from the single binary with no separate deploy.
5. Follow-up plan: replace the loopback hop with in-process handler calls
   (proxy service, auth-file mutations, plugin origin rewrite), merge
   config, ports, auth, and stores, and retire the separate deploy path.
   Check: manager-server tests pass with no TCP hop for local panel flows.

## Decisions

- Keep upstream file layouts identical so future diffs against pinned
  commits stay trivial.
- Keep the external management API (`/v0`, `/v8`) unchanged for other
  clients.
- The loopback hop in step 4 is interim, not the end state.
