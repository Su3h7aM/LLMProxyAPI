import { isMap, isSeq } from 'yaml';
import type { parseDocument } from 'yaml';
import { DEFAULT_VISUAL_VALUES } from '@/types/visualConfig';
import type {
  VisualConfigValues,
  VisualConfigValidationErrors,
} from '@/types/visualConfig';
import { readConfigBoolean } from './visualConfigBoolean';

// Source: backend config_v8.go/config_types.go. Shared upstream behavior lives
// under upstream.*/client.codex; only OAuth-only provider paths keep oauth.providers.*.
export const ADDITION_FIELDS = [
  {
    key: 'routingSessionAffinitySubagents',
    path: 'routing.session-affinity-subagents'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'saveCooldownStatus',
    path: 'routing.cooldown.save-cooldown-status'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'transientErrorCooldownSeconds',
    path: 'routing.cooldown.transient-error-cooldown-seconds'.split('.'),
    kind: 'integer',
  },
  {
    key: 'videoResultAuthCacheTTL',
    path: 'multimedia.video-result-auth-cache-ttl'.split('.'),
    kind: 'string',
  },
  {
    key: 'claudeHeaderTimezone',
    path: 'upstream.claude.header-defaults.timezone'.split('.'),
    kind: 'string',
  },
  {
    key: 'claudeModelLevelCooling',
    path: 'upstream.claude.model-level-cooling'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'claudeDisableCloakMode',
    path: 'upstream.claude.disable-claude-cloak-mode'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'claudeCodeDisableCloakingModelList',
    path: 'upstream.claude.disable-cloaking-model-list'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexDisableCloaking',
    path: 'upstream.codex.disable-codex-cloaking'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexModelLevelCooling',
    path: 'upstream.codex.model-level-cooling'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexStreamBootstrapBuffering',
    path: 'upstream.codex.stream-bootstrap-buffering'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexStreamBootstrapTimeout',
    path: 'upstream.codex.stream-bootstrap-timeout'.split('.'),
    kind: 'string',
  },
  {
    key: 'codexOptimizeMultiAgentV2',
    path: 'client.codex.optimize-multi-agent-v2'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexOrphanDelegationCompatibility',
    path: 'upstream.codex.orphan-delegation-compatibility'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexResponseSteering',
    path: 'upstream.codex.response-steering'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'antigravityConnectionPoolEnabled',
    path: 'oauth.providers.antigravity.connection-pool.enabled'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'antigravityConnectionPoolIdleTimeout',
    path: 'oauth.providers.antigravity.connection-pool.idle-conn-timeout'.split('.'),
    kind: 'string',
  },
  {
    key: 'antigravityConnectionPoolMaxIdleConnsPerHost',
    path: 'oauth.providers.antigravity.connection-pool.max-idle-conns-per-host'.split('.'),
    kind: 'integer',
  },
  {
    key: 'xaiInjectXSearch',
    path: 'upstream.xai.inject-x-search'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexLiveMediaRelayEnabled',
    path: 'oauth.providers.codex.live-media-relay.enabled'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexLiveMediaRelayMaxSessions',
    path: 'oauth.providers.codex.live-media-relay.max-sessions'.split('.'),
    kind: 'integer',
  },
  {
    key: 'codexLiveMediaRelayDisablePrivateRemoteIPs',
    path: 'oauth.providers.codex.live-media-relay.disable-private-remote-ips'.split('.'),
    kind: 'boolean',
  },
  {
    key: 'codexLiveMediaRelayPublicIP',
    path: 'oauth.providers.codex.live-media-relay.public-ip'.split('.'),
    kind: 'string',
  },
  {
    key: 'codexLiveMediaRelayUDPPortMin',
    path: 'oauth.providers.codex.live-media-relay.udp-port-min'.split('.'),
    kind: 'integer',
  },
  {
    key: 'codexLiveMediaRelayUDPPortMax',
    path: 'oauth.providers.codex.live-media-relay.udp-port-max'.split('.'),
    kind: 'integer',
  },
] as const;
export const ICE_KEY = 'codexLiveMediaRelayICEServers';
export const ICE_PATH = ['oauth', 'providers', 'codex', 'live-media-relay', 'ice-servers'];
type Doc = ReturnType<typeof parseDocument>;
export function readVisualAdditions(doc: Doc) {
  const values = {} as Pick<
    VisualConfigValues,
    (typeof ADDITION_FIELDS)[number]['key'] | typeof ICE_KEY
  >;
  for (const { key, path, kind } of ADDITION_FIELDS) {
    const raw = doc.getIn(path);
    Object.assign(values, {
      [key]:
        kind === 'boolean' ? readConfigBoolean(raw, DEFAULT_VISUAL_VALUES[key]) : String(raw ?? ''),
    });
  }
  const raw = doc.getIn(ICE_PATH, true);
  values[ICE_KEY] = isSeq(raw)
    ? raw.items.map((node, index) => {
        const row = isMap(node) ? node.toJSON() : {};
        return {
          id: `ice-${index}`,
          urlsText: Array.isArray(row.urls) ? row.urls.join('\n') : '',
          username: String(row.username ?? ''),
          credential: String(row.credential ?? ''),
        };
      })
    : [];
  return values;
}
export const iceURLs = (text: string) =>
  text
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean);

/** Go time.ParseDuration grammar (not JS dates); bounds are signed int64 nanoseconds. */
export function goDurationSeconds(input: string): number | undefined {
  if (/^[+-]?0$/.test(input)) return 0;
  if (!/^[+-]?(?:(?:\d+(?:\.\d*)?|\.\d+)(?:ns|us|µs|μs|ms|s|m|h))+$/.test(input)) return undefined;
  const units: Record<string, bigint> = {
    ns: 1n,
    us: 1000n,
    µs: 1000n,
    μs: 1000n,
    ms: 1000000n,
    s: 1000000000n,
    m: 60000000000n,
    h: 3600000000000n,
  };
  let nanos = 0n;
  for (const match of input.matchAll(/(\d+(?:\.\d*)?|\.\d+)(ns|us|µs|μs|ms|s|m|h)/g)) {
    const [whole, fraction = ''] = match[1].split('.');
    const unit = units[match[2]];
    nanos += BigInt(whole || '0') * unit;
    if (fraction) nanos += (BigInt(fraction) * unit) / 10n ** BigInt(fraction.length);
  }
  const negative = input.startsWith('-');
  if (nanos > (negative ? 9223372036854775808n : 9223372036854775807n)) return undefined;
  return Number(negative ? -nanos : nanos) / 1e9;
}
function validIP(raw: string): boolean {
  if (!raw.includes(':'))
    return (
      /^(?:0|[1-9]\d{0,2})(?:\.(?:0|[1-9]\d{0,2})){3}$/.test(raw) &&
      raw.split('.').every((part) => Number(part) <= 255)
    );
  if (!/^[\da-fA-F:.]+$/.test(raw)) return false;
  if (raw.includes('.') && !validIP(raw.slice(raw.lastIndexOf(':') + 1))) return false;
  try {
    return new URL(`http://[${raw}]/`).hostname.startsWith('[');
  } catch {
    return false;
  }
}
export function validateVisualAdditions(
  values: VisualConfigValues,
  dirtyFields?: ReadonlySet<string>
): VisualConfigValidationErrors {
  const errors: VisualConfigValidationErrors = {
    transientErrorCooldownSeconds: undefined,
    videoResultAuthCacheTTL: undefined,
    claudeHeaderTimezone: undefined,
    codexStreamBootstrapTimeout: undefined,
    antigravityConnectionPoolIdleTimeout: undefined,
    antigravityConnectionPoolMaxIdleConnsPerHost: undefined,
    codexLiveMediaRelayMaxSessions: undefined,
    codexLiveMediaRelayPublicIP: undefined,
    codexLiveMediaRelayUDPPortMin: undefined,
    codexLiveMediaRelayUDPPortMax: undefined,
    codexLiveMediaRelayICEServers: undefined,
  };
  for (const { key, kind } of ADDITION_FIELDS) {
    const value = String(values[key]).trim();
    if (!value || kind !== 'integer') continue;
    const signed =
      key === 'transientErrorCooldownSeconds' ||
      key === 'antigravityConnectionPoolMaxIdleConnsPerHost' ||
      (key === 'codexLiveMediaRelayMaxSessions' && !values.codexLiveMediaRelayEnabled);
    if (
      !/^-?\d+$/.test(value) ||
      !Number.isSafeInteger(Number(value)) ||
      (!signed && Number(value) < 0)
    ) {
      Object.assign(errors, { [key]: signed ? 'integer' : 'non_negative_integer' });
    }
  }
  for (const key of [
    'videoResultAuthCacheTTL',
    'antigravityConnectionPoolIdleTimeout',
    'codexStreamBootstrapTimeout',
  ] as const) {
    // The backend accepts these strings with runtime fallbacks. Existing values must not
    // block unrelated edits, but newly edited durations still receive strict validation.
    if (dirtyFields && !dirtyFields.has(key)) continue;
    const value = values[key].trim();
    if (!value) continue;
    const codex = key === 'codexStreamBootstrapTimeout';
    if (codex && /^(none|unlimited|disabled|off|never)$/i.test(value)) continue;
    const duration =
      codex && /^\+?\d+$/.test(value) && Number(value) <= 9223372036
        ? Number(value)
        : goDurationSeconds(value);
    if (duration === undefined) errors[key] = 'invalid_duration';
    else if (codex && duration < 0) errors[key] = 'invalid_duration';
    else if (key === 'videoResultAuthCacheTTL' && duration <= 0) errors[key] = 'positive_duration';
    // Pool timeout <= 0 switches to short-lived connections; > 210s is clamped by the backend.
  }
  const timezone = values.claudeHeaderTimezone.trim();
  // Go time.LoadLocation also accepts Local (the backend host's local timezone).
  if (timezone && timezone !== 'Local') {
    try {
      if (/^[+-]/.test(timezone)) throw new Error('Not an IANA timezone');
      new Intl.DateTimeFormat('en', { timeZone: timezone });
    } catch {
      errors.claudeHeaderTimezone = 'invalid_timezone';
    }
  }
  // uint16 decoding applies even to a disabled relay; operational constraints do not.
  for (const key of ['codexLiveMediaRelayUDPPortMin', 'codexLiveMediaRelayUDPPortMax'] as const) {
    const value = values[key].trim();
    if (value && (!/^\d+$/.test(value) || Number(value) > 65535))
      errors[key] = 'integer_range_0_65535';
  }
  if (!values.codexLiveMediaRelayEnabled) return errors;
  const ip = values.codexLiveMediaRelayPublicIP.trim();
  if (ip && !validIP(ip)) errors.codexLiveMediaRelayPublicIP = 'invalid_ip';
  const min = Number(values.codexLiveMediaRelayUDPPortMin.trim());
  const max = Number(values.codexLiveMediaRelayUDPPortMax.trim());
  if (!errors.codexLiveMediaRelayUDPPortMin && !errors.codexLiveMediaRelayUDPPortMax) {
    if ((min === 0) !== (max === 0) || min > max) {
      errors.codexLiveMediaRelayUDPPortMin = errors.codexLiveMediaRelayUDPPortMax = 'udp_port_pair';
    } else if (
      min &&
      !errors.codexLiveMediaRelayMaxSessions &&
      max - min + 1 < (Number(values.codexLiveMediaRelayMaxSessions) || 32) * 2
    ) {
      errors.codexLiveMediaRelayUDPPortMin = errors.codexLiveMediaRelayUDPPortMax =
        'udp_port_capacity';
    }
  }
  if (
    values[ICE_KEY].some((row) => {
      const urls = iceURLs(row.urlsText);
      return (
        !urls.length ||
        urls.some((url) => {
          if (!/^(stun|stuns|turn|turns):/i.test(url) || /\s|%(?![\da-f]{2})/i.test(url))
            return true;
          try {
            new URL(url);
            return false;
          } catch {
            return true;
          }
        })
      );
    })
  )
    errors[ICE_KEY] = 'invalid_ice_servers';
  return errors;
}
