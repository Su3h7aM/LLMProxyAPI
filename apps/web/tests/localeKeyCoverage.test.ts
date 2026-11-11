import { describe, expect, test } from 'bun:test';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import en from '@/i18n/locales/en.json';
import zhCN from '@/i18n/locales/zh-CN.json';
import zhTW from '@/i18n/locales/zh-TW.json';
import ru from '@/i18n/locales/ru.json';

/**
 * Locale key coverage guard.
 *
 * The panel must never show a raw translation key. This test reads every
 * static `t('namespace.key')` call in `src` and checks that each locale
 * defines that key. Dynamic keys (`t(`prefix.${name}`)`) are out of reach of
 * a static scan and are skipped. Plural keys count as defined when the
 * locale has any of the i18next plural suffixes.
 */

const LOCALES = { en, 'zh-CN': zhCN, 'zh-TW': zhTW, ru } as const;
type LocaleMessages = { [key: string]: unknown };

/**
 * Calls that pass `defaultValue` render a fallback instead of the raw key.
 * They only hide the gap when the fallback text is intentional. List such a
 * key here; every entry needs a reason. The list is empty on purpose: all
 * keys that use `defaultValue` today exist in every locale.
 */
const DEFAULT_VALUE_ALLOWED = new Set<string>([]);

const SRC = fileURLToPath(new URL('../src', import.meta.url));

const STATIC_KEY = /^[A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)+$/;
/** Static call: t('key') or i18n.t('key'), with an optional second argument. */
const CALL = /(?<![\w$.])(?:i18n\.)?t\(\s*(['"`])([^'"`]*)\1([\s\S]{0,240}?)\)/g;

function sourceFiles(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) sourceFiles(full, out);
    else if (full.endsWith('.ts') || full.endsWith('.tsx')) out.push(full);
  }
  return out;
}

interface StaticCall {
  key: string;
  file: string;
  hasDefaultValue: boolean;
}

function collectStaticCalls(files: readonly string[], root: string): StaticCall[] {
  const calls: StaticCall[] = [];
  for (const file of files) {
    const source = readFileSync(file, 'utf8');
    for (const match of source.matchAll(CALL)) {
      const key = match[2];
      if (!STATIC_KEY.test(key)) continue;
      calls.push({
        key,
        file: file.slice(root.length + 1),
        hasDefaultValue: match[3].includes('defaultValue'),
      });
    }
  }
  return calls;
}

function messageAt(messages: LocaleMessages, key: string): unknown {
  let current: unknown = messages;
  for (const part of key.split('.')) {
    if (current === null || typeof current !== 'object' || !(part in current)) return undefined;
    current = (current as Record<string, unknown>)[part];
  }
  return current;
}

const PLURAL_SUFFIXES = ['_zero', '_one', '_two', '_few', '_many', '_other'] as const;

function isDefined(messages: LocaleMessages, key: string): boolean {
  const direct = messageAt(messages, key);
  if (typeof direct === 'string' && direct.trim().length > 0) return true;
  return PLURAL_SUFFIXES.some((suffix) => {
    const value = messageAt(messages, `${key}${suffix}`);
    return typeof value === 'string' && value.trim().length > 0;
  });
}

export function coverageIssues(
  locales: Record<string, LocaleMessages>,
  calls: readonly StaticCall[],
  allowDefaultValue: ReadonlySet<string> = DEFAULT_VALUE_ALLOWED
): string[] {
  const issues: string[] = [];
  const seen = new Set<string>();
  for (const call of calls) {
    if (call.hasDefaultValue && allowDefaultValue.has(call.key)) continue;
    for (const [locale, messages] of Object.entries(locales)) {
      const issue = `${locale}: ${call.key} (${call.file})`;
      if (!isDefined(messages, call.key) && !seen.has(issue)) {
        seen.add(issue);
        issues.push(issue);
      }
    }
  }
  return issues;
}

describe('locale key coverage', () => {
  test('every static t() key exists in all four locales', () => {
    const files = sourceFiles(SRC);
    expect(files.length).toBeGreaterThan(100);
    const calls = collectStaticCalls(files, SRC);
    expect(calls.length).toBeGreaterThan(500);
    expect(coverageIssues(LOCALES, calls)).toEqual([]);
  });

  test('the scan finds known static calls and skips dynamic keys', () => {
    const files = sourceFiles(SRC);
    const calls = collectStaticCalls(files, SRC);
    const keys = new Set(calls.map((call) => call.key));
    // Pinned examples: the keys this guard was added for and a plural call.
    expect(keys.has('common.clear')).toBe(true);
    expect(keys.has('providersPage.discovery.syncHint')).toBe(true);
    // Dynamic keys, template keys and non-i18n calls do not enter the scan.
    expect([...keys].some((key) => key.includes('${'))).toBe(false);
    expect([...keys].every((key) => STATIC_KEY.test(key))).toBe(true);
  });

  test('a missing key is reported once per locale and file', () => {
    const calls: StaticCall[] = [
      { key: 'ghost.namespace.key', file: 'src/Ghost.tsx', hasDefaultValue: false },
    ];
    expect(coverageIssues({ en: { ghost: { namespace: { other: 'x' } } }, ru: {} }, calls)).toEqual(
      ['en: ghost.namespace.key (src/Ghost.tsx)', 'ru: ghost.namespace.key (src/Ghost.tsx)']
    );
  });

  test('plural forms and empty values follow the guard rules', () => {
    const plural: StaticCall[] = [
      { key: 'item.count', file: 'src/List.tsx', hasDefaultValue: false },
    ];
    const locales = {
      en: { item: { count: 'no' } },
      ru: {
        item: {
          count_one: '{{count}} штука',
          count_few: '{{count}} штуки',
          count_many: '{{count}} штук',
          count_other: '{{count}} штуки',
        },
      },
    };
    expect(
      coverageIssues(
        {
          ...locales,
          en: { item: { count_one: '{{count}} item', count_other: '{{count}} items' } },
        },
        plural
      )
    ).toEqual([]);
    expect(coverageIssues({ en: { item: { count: ' ' } } }, plural)).toEqual([
      'en: item.count (src/List.tsx)',
    ]);
  });

  test('defaultValue calls are skipped only when allow-listed', () => {
    const calls: StaticCall[] = [
      { key: 'ghost.fallback', file: 'src/Ghost.tsx', hasDefaultValue: true },
    ];
    expect(coverageIssues({ en: {} }, calls)).toEqual(['en: ghost.fallback (src/Ghost.tsx)']);
    expect(coverageIssues({ en: {} }, calls, new Set(['ghost.fallback']))).toEqual([]);
  });
});
