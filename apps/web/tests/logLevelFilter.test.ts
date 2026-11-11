import { describe, expect, test } from 'bun:test';
import { parseLogLine } from '../src/pages/hooks/logParsing';
import {
  buildLogLevelCounts,
  filterParsedLogLines,
  type StructuredLogFilterSets,
} from '../src/pages/hooks/useLogFilters';
import { LOG_LEVELS, type LogLevel } from '../src/pages/hooks/logTypes';
import en from '../src/i18n/locales/en.json';
import zhCN from '../src/i18n/locales/zh-CN.json';
import zhTW from '../src/i18n/locales/zh-TW.json';
import ru from '../src/i18n/locales/ru.json';

const line = (level: string, message = 'request served') =>
  parseLogLine(`[2026-11-07 12:00:00] [abcd1234] [${level}] [server.go:1] ${message}`);

const noFilters = (): StructuredLogFilterSets => ({
  methods: new Set(),
  statuses: new Set(),
  levels: new Set(),
  paths: new Set(),
});

describe('log level filter', () => {
  test('levels match the set the Go log lines carry', () => {
    expect([...LOG_LEVELS]).toEqual(['trace', 'debug', 'info', 'warn', 'error', 'fatal']);
  });

  test('counts lines by parsed level and skips lines without one', () => {
    const parsed = [line('info'), line('warn'), line('ERROR'), parseLogLine('plain text line')];
    expect(buildLogLevelCounts(parsed)).toEqual({ info: 1, warn: 1, error: 1 });
  });

  test('an active level keeps only matching lines', () => {
    const parsed = [line('info'), line('warn'), line('error')];
    expect(filterParsedLogLines(parsed, { ...noFilters(), levels: new Set(['warn']) })).toEqual([
      parsed[1],
    ]);
    const both = filterParsedLogLines(parsed, {
      ...noFilters(),
      levels: new Set<LogLevel>(['info', 'error']),
    });
    expect(both).toEqual([parsed[0], parsed[2]]);
  });

  test('a line without a level never matches an active level filter', () => {
    const parsed = [parseLogLine('plain text line'), line('info')];
    expect(filterParsedLogLines(parsed, { ...noFilters(), levels: new Set(['info']) })).toEqual([
      parsed[1],
    ]);
  });

  test('combines level with method, status and path filters', () => {
    const access = (status: number, level: string) =>
      parseLogLine(
        `[2026-11-07 12:00:00] [abcd1234] [${level}] [gin_logger.go:1] ${status} | 1ms | ::1 | POST "/v1/messages"`
      );
    const parsed = [access(200, 'info'), access(500, 'error')];
    expect(
      filterParsedLogLines(parsed, {
        ...noFilters(),
        statuses: new Set(['5xx']),
        levels: new Set<LogLevel>(['error']),
      })
    ).toEqual([parsed[1]]);
    expect(
      filterParsedLogLines(parsed, {
        ...noFilters(),
        statuses: new Set(['5xx']),
        levels: new Set<LogLevel>(['info']),
      })
    ).toEqual([]);
    expect(parsed).toHaveLength(2);
  });
});

describe('log level filter translations', () => {
  for (const [locale, document] of Object.entries({ en, zhCN, zhTW, ru })) {
    test(`${locale} names the level filter group`, () => {
      expect(document.logs.filter_level.trim().length).toBeGreaterThan(0);
    });
  }
});
