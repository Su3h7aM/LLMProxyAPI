# Usage prices: operator notes

## Where prices come from

The price book syncs with `https://models.dev/api.json` (USD per 1M tokens).
The first sync runs 30 s after start, then every `interval-hours`
(default 24). A failed sync keeps the old prices and records the error
(visible at `GET /observability/usage/prices/status`). The binary embeds a
trimmed snapshot, so the first start works offline: it loads as
`source=embedded` when the price table is empty.

Config (`observability.usage-db.price-sync`): `enabled` (default true),
`url` (default the models.dev URL), `interval-hours` (default 24).

## Frozen history

Each price is a version valid from `valid_from_ms` until `valid_to_ms`
(open end for the current price). Cost is computed at insert with the
version valid at the event time and stored as integer micro-USD. A price
change never rewrites old cost. There is no automatic recompute of events
that have a cost.

Rates store as integers: `micro-USD per 1M = round(USD per 1M * 1e6)`,
half up. Each event rounds once.

## Unpriced events

An event with no matching price is stored with NULL cost
(`cost_source=none`) and counts as unpriced in the hourly rollup. When a
price for its model appears later, the store fills it in the background
with the version valid at the event time (or the nearest version for
events older than the first version), marked `cost_source=price-book-late`.
`POST /observability/usage/prices/price-unpriced` does the same on demand.
Neither path touches an event that already has a cost.

## Overrides

`PUT /observability/usage/prices` appends a manual version that wins over
synced prices for the time it covers. `effective_from_ms` sets the start
(default now). A manual version never changes stored cost by itself; use
price-unpriced or the correction below for old events. A sync never
replaces a manual version (it counts `skipped_overrides`).
`DELETE /observability/usage/prices/manual` closes the manual version; the
last synced rates apply again from now.

## Corrections

`POST /observability/usage/prices/correct` recomputes cost for a date
range (max 366 days) with the time-valid versions, fixes the hourly
rollups in the same transaction, and writes an audit row to
`cost_corrections`. It needs `confirm: true` and a `reason`.
