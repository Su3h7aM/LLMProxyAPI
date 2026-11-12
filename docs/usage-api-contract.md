# Usage query API: contract (v8, additive)

Status: contract for issues 0045 (backend) and 0046 to 0048 (panel). A change to a shape needs a coordinator decision. Price routes are in `docs/usage-prices.md` and `docs/management-api-v8.md` (issue 0044). The status route exists (issue 0043).

Base path: `/v8/management/observability/usage`. Auth: as all management routes. All responses are JSON unless noted. Errors: `{"error": "<message>"}` with 400 (bad parameter), 401, 404, 503 (usage DB disabled: `{"error":"usage db disabled"}`).

## Units

- Time: epoch milliseconds (`*_ms`). Hour buckets are UTC hours.
- Cost: integer micro-USD (`*_micro_usd`; 1,000,000 = 1 USD). The panel formats it.
- Tokens: integers. `total_tokens` is as stored by the collector.
- Durations: milliseconds.
- Rates: float 0 to 1.

## Common query parameters

| Name | Meaning | Default |
|---|---|---|
| `from_ms`, `to_ms` | Range, `from` inclusive, `to` exclusive | last 24 hours |
| `tz` | IANA zone for day and week buckets and the heatmap | `UTC` |
| `provider`, `model`, `alias`, `auth_id`, `api_key_id` | Exact match; repeat the parameter for several values (OR) | none |
| `status` | `ok` (class `2xx`) or `failed` (every class except `2xx`) | both |
| `stream` | `true` or `false` | both |
| `q` | Text search in `request_id`, `trace_id`, `model`, `alias`, `account_label`, `fail_summary` (events and failures only) | none |

Range cap: 366 days (400 above). A range with `to_ms <= from_ms` gives 400.

## Counters object (used by several routes)

```json
{
  "requests": 120,
  "failures": 3,
  "failure_rate": 0.025,
  "input_tokens": 100000,
  "output_tokens": 40000,
  "reasoning_tokens": 5000,
  "cache_read_tokens": 30000,
  "cache_write_tokens": 2000,
  "total_tokens": 177000,
  "cost_micro_usd": 1250000,
  "estimated_cost_micro_usd": 200000,
  "unpriced_requests": 4,
  "latency_avg_ms": 840,
  "latency_max_ms": 9100,
  "ttft_avg_ms": 310
}
```

`estimated_cost_micro_usd` is the part of `cost_micro_usd` that comes from late pricing (`price-book-late`). It is included in `cost_micro_usd`.

## GET /summary

Totals for the range and for the previous period of the same length.

```json
{
  "range": {"from_ms": 1794400000000, "to_ms": 1794486400000},
  "totals": { "...counters..." },
  "previous": { "...counters..." },
  "latency": {"p50_ms": 600, "p95_ms": 2100, "source": "events"},
  "first_event_ms": 1794000000000
}
```

`latency.p50_ms` and `p95_ms` are null with `source:"hourly"` when the range holds too many events (see 0045 limits).

## GET /timeseries

Parameters: common, plus `bucket` = `hour`, `day`, `week`, or `auto` (default: hour up to 3 days, day up to 120 days, else week) and optional `group_by` = `model`, `provider`, `credential`, `api_key` with `top` (default 5, max 20; the rest go into key `"other"`).

```json
{
  "bucket": "hour",
  "tz": "UTC",
  "points": [
    {"ts_ms": 1794400000000, "counters": { "...counters..." }}
  ],
  "groups": [
    {"key": "claude-sonnet-5-5", "label": "claude-sonnet-5-5",
     "points": [{"ts_ms": 1794400000000, "counters": { "...counters..." }}]}
  ]
}
```

`groups` is present only with `group_by`. Missing buckets are returned with zero counters, so charts need no gap filling.

## GET /breakdown

Parameters: common, plus `group_by` (required): `model`, `provider`, `alias`, `credential`, `api_key`, `status_class`, `stream`. Optional `sort` = `requests` (default), `tokens`, `cost`, `failures`, `latency`; `order` = `desc` (default) or `asc`; `limit` (default 20, max 200); `offset`.

```json
{
  "group_by": "model",
  "total": 14,
  "items": [
    {"key": "claude-sonnet-5-5", "label": "claude-sonnet-5-5", "share": 0.42,
     "counters": { "...counters..." }, "meta": {"provider": "claude"}}
  ]
}
```

`share` is the share of requests in the filtered range. `label`: for `credential` the credential label from the auth manager, else the latest stored `account_label`, else the id; for `api_key` the label from key labels, else `key-<first 6 of id>`; else the key. `meta` carries extra fields per group (`provider` for model; `auth_type` and `provider` for credential). `status_class` is one of `2xx`, `4xx`, `5xx`, `error`, `other`; a failed record is never `2xx` (`error` covers failed 2xx and missing codes).

## GET /events

Parameters: common (with `q`), plus `limit` (default 50, max 200), `cursor` (opaque, from the previous response), `order` = `desc` (default) or `asc`.

```json
{
  "items": [
    {
      "id": 981,
      "request_id": "0b1c...",
      "trace_id": "a1b2c3d4",
      "ts_ms": 1794400123456,
      "provider": "claude",
      "model": "claude-sonnet-5-5",
      "alias": "sonnet",
      "executor": "claude",
      "auth_id": "claude-main.json",
      "auth_type": "oauth",
      "account_label": "ab***@example.com",
      "api_key_id": "9f2c4a0e11d3b7aa",
      "api_key_label": "my-harness",
      "stream": true,
      "status_class": "2xx",
      "failed": false,
      "fail_code": "",
      "latency_ms": 1280,
      "ttft_ms": 340,
      "input_tokens": 1200,
      "output_tokens": 480,
      "reasoning_tokens": 0,
      "cache_read_tokens": 800,
      "cache_write_tokens": 0,
      "total_tokens": 2480,
      "service_tier": "",
      "reasoning_effort": "",
      "cost_micro_usd": 12500,
      "cost_source": "price-book"
    }
  ],
  "next_cursor": "opaque-or-empty",
  "total_estimate": 5230
}
```

`total_estimate` is exact when the range holds up to 100000 matching rows, else an estimate (field `total_exact: false` is added). `fail_summary` is NOT in the list.

`account_label` holds a masked email (`ab***@example.com`) only when the record source is an email; else it is empty. Free-text columns (`session_id`, `service_tier`, `reasoning_effort`, `access_provider`, `executor`, `auth_type`, `fail_code`) keep only values of letters, digits, `.`, `_`, `-`, `:` of at most 64 characters, else empty.

## GET /events/{id}

All list fields plus `fail_summary`, `session_id`, `access_provider`, `auth_index`, `price_id`, `price` (the price version used: `provider`, `model`, `valid_from_ms`, `input`, `output`, `cache_read`, `cache_write`, `source`; null if none).

`fail_summary` is scrubbed of secrets (tokens, keys, JWT shapes, secret query and JSON values become `[redacted]`) and capped at 4096 bytes.

## GET /failures

Parameters: common (with `q`), `limit` (groups, default 20, max 100), `recent` (recent failures, default 20, max 100).

```json
{
  "groups": [
    {"fail_code": "rate_limit", "status_class": "4xx", "provider": "codex",
     "auth_id": "codex-1.json", "auth_label": "ab***@example.com", "model": "gpt-6-luna",
     "count": 17, "first_ms": 1794400000000, "last_ms": 1794480000000}
  ],
  "recent": [ { "...event list item with fail_summary..." } ]
}
```

## GET /heatmap

Parameters: common, plus `metric` = `requests` (default), `tokens`, `cost`, `failures`.

```json
{"tz": "UTC", "metric": "requests", "matrix": [[0,0,...24 values...], "...7 rows, Monday first..."], "max": 120}
```

## GET /export

Parameters: common (with `q`), `format` = `jsonl` (default) or `csv`. Streams `Content-Disposition: attachment; filename=usage-<from>-<to>.<ext>`. Fields: the list item fields plus `fail_summary`. Order by `ts_ms` ascending. The stream stops when the client leaves. No cap on rows.

## GET /key-labels, PUT /key-labels

`GET` returns `{"labels":[{"api_key_id":"9f2c4a0e11d3b7aa","label":"my-harness","seen":true}]}`: all labels set, plus ids seen in events without a label (`label` empty, `seen` true). `PUT` body `{"api_key_id":"...","label":"my-harness"}` sets or clears (empty label) one label. Returns the updated item. Labels are stored in the usage DB (table `api_key_labels`). The label never contains the raw key.

## POST /maintenance

Body `{"action":"prune"|"vacuum"|"checkpoint"}`. `prune` runs retention now. `vacuum` runs `VACUUM` (blocks writers: the response says `"duration_ms"`). `checkpoint` runs `wal_checkpoint(TRUNCATE)`. Returns `{"action":"prune","deleted":1200,"duration_ms":340}` (fields per action).

## Performance targets (single node, 1 vCPU)

| Route | Data | Target |
|---|---|---|
| `summary`, `timeseries`, `breakdown` (no percentiles) | 1M events, 120-day range | under 300 ms (hourly table) |
| `summary` percentiles | 7 days, up to 1M events | under 1 s; above the limit return null with `source:"hourly"` |
| `events` page | 1M events, any indexed filter | under 150 ms |
| `heatmap`, `failures` | 1M events, 30-day range | under 500 ms |
