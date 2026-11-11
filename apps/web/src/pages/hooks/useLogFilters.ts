import { useCallback, useMemo } from 'react';
import { useLocalStorage } from '@/hooks/useLocalStorage';
import { type HttpMethod, type LogLevel, type ParsedLogLine, type StatusGroup } from './logTypes';
import { resolveStatusGroup } from './logTypes';

const PATH_FILTER_LIMIT = 12;

interface UseLogFiltersOptions {
  parsedLines: ParsedLogLine[];
}

interface UseLogFiltersReturn {
  methodFilters: HttpMethod[];
  statusFilters: StatusGroup[];
  levelFilters: LogLevel[];
  pathFilters: string[];
  methodFilterSet: Set<HttpMethod>;
  statusFilterSet: Set<StatusGroup>;
  levelFilterSet: Set<LogLevel>;
  pathFilterSet: Set<string>;
  hasStructuredFilters: boolean;
  methodCounts: Partial<Record<HttpMethod, number>>;
  statusCounts: Partial<Record<StatusGroup, number>>;
  levelCounts: Partial<Record<LogLevel, number>>;
  pathOptions: Array<{ path: string; count: number }>;
  toggleMethodFilter: (method: HttpMethod) => void;
  toggleStatusFilter: (group: StatusGroup) => void;
  toggleLevelFilter: (level: LogLevel) => void;
  togglePathFilter: (path: string) => void;
  clearStructuredFilters: () => void;
}

export function useLogFilters(options: UseLogFiltersOptions): UseLogFiltersReturn {
  const { parsedLines } = options;

  const [methodFilters, setMethodFilters] = useLocalStorage<HttpMethod[]>(
    'logsPage.methodFilters',
    []
  );
  const [statusFilters, setStatusFilters] = useLocalStorage<StatusGroup[]>(
    'logsPage.statusFilters',
    []
  );
  const [levelFilters, setLevelFilters] = useLocalStorage<LogLevel[]>('logsPage.levelFilters', []);
  const [storedPathFilters, setPathFilters] = useLocalStorage<string[]>('logsPage.pathFilters', []);
  const pathFilters = useMemo(
    () => normalizeLogPathFilters(storedPathFilters),
    [storedPathFilters]
  );

  const methodFilterSet = useMemo(() => new Set(methodFilters), [methodFilters]);
  const statusFilterSet = useMemo(() => new Set(statusFilters), [statusFilters]);
  const levelFilterSet = useMemo(() => new Set(levelFilters), [levelFilters]);
  const pathFilterSet = useMemo(() => new Set(pathFilters), [pathFilters]);
  const hasStructuredFilters =
    methodFilters.length > 0 ||
    statusFilters.length > 0 ||
    levelFilters.length > 0 ||
    pathFilters.length > 0;

  const methodCounts = useMemo(() => {
    const counts: Partial<Record<HttpMethod, number>> = {};
    parsedLines.forEach((line) => {
      if (!line.method) return;
      counts[line.method] = (counts[line.method] ?? 0) + 1;
    });
    return counts;
  }, [parsedLines]);

  const statusCounts = useMemo(() => {
    const counts: Partial<Record<StatusGroup, number>> = {};
    parsedLines.forEach((line) => {
      const statusGroup = resolveStatusGroup(line.statusCode);
      if (!statusGroup) return;
      counts[statusGroup] = (counts[statusGroup] ?? 0) + 1;
    });
    return counts;
  }, [parsedLines]);

  const levelCounts = useMemo(() => buildLogLevelCounts(parsedLines), [parsedLines]);

  const pathOptions = useMemo(
    () => buildLogPathOptions(parsedLines, pathFilters),
    [parsedLines, pathFilters]
  );

  const toggleMethodFilter = (method: HttpMethod) => {
    setMethodFilters((prev) =>
      prev.includes(method) ? prev.filter((item) => item !== method) : [...prev, method]
    );
  };

  const toggleStatusFilter = (group: StatusGroup) => {
    setStatusFilters((prev) =>
      prev.includes(group) ? prev.filter((item) => item !== group) : [...prev, group]
    );
  };

  const toggleLevelFilter = (level: LogLevel) => {
    setLevelFilters((prev) =>
      prev.includes(level) ? prev.filter((item) => item !== level) : [...prev, level]
    );
  };

  const togglePathFilter = (path: string) => {
    setPathFilters((stored) => {
      const prev = normalizeLogPathFilters(stored);
      return prev.includes(path) ? prev.filter((item) => item !== path) : [...prev, path];
    });
  };

  const clearStructuredFilters = useCallback(() => {
    setMethodFilters([]);
    setStatusFilters([]);
    setLevelFilters([]);
    setPathFilters([]);
  }, [setMethodFilters, setStatusFilters, setLevelFilters, setPathFilters]);

  return {
    methodFilters,
    statusFilters,
    levelFilters,
    pathFilters,
    methodFilterSet,
    statusFilterSet,
    levelFilterSet,
    pathFilterSet,
    hasStructuredFilters,
    methodCounts,
    statusCounts,
    levelCounts,
    pathOptions,
    toggleMethodFilter,
    toggleStatusFilter,
    toggleLevelFilter,
    togglePathFilter,
    clearStructuredFilters,
  };
}

// Earlier versions stored formatter quotes around selected paths.
export function normalizeLogPathFilters(paths: string[]): string[] {
  return [
    ...new Set(
      paths.map((path) => (path.startsWith('"') && path.endsWith('"') ? path.slice(1, -1) : path))
    ),
  ];
}

/** Lines without a parsed level never match an active level filter. */
export function buildLogLevelCounts(parsedLines: ParsedLogLine[]) {
  const counts: Partial<Record<LogLevel, number>> = {};
  parsedLines.forEach((line) => {
    if (!line.level) return;
    counts[line.level] = (counts[line.level] ?? 0) + 1;
  });
  return counts;
}

export interface StructuredLogFilterSets {
  methods: Set<HttpMethod>;
  statuses: Set<StatusGroup>;
  levels: Set<LogLevel>;
  paths: Set<string>;
}

/** One place decides structured matching, so the page and the tests agree. */
export function filterParsedLogLines(
  parsedLines: ParsedLogLine[],
  filters: StructuredLogFilterSets
): ParsedLogLine[] {
  return parsedLines.filter((line) => {
    if (filters.methods.size > 0 && (!line.method || !filters.methods.has(line.method))) {
      return false;
    }
    const statusGroup = resolveStatusGroup(line.statusCode);
    if (filters.statuses.size > 0 && (!statusGroup || !filters.statuses.has(statusGroup))) {
      return false;
    }
    if (filters.levels.size > 0 && (!line.level || !filters.levels.has(line.level))) {
      return false;
    }
    if (filters.paths.size > 0 && (!line.path || !filters.paths.has(line.path))) {
      return false;
    }
    return true;
  });
}

/** Selected paths stay removable after they leave the current log window. */
export function buildLogPathOptions(parsedLines: ParsedLogLine[], selectedPaths: string[]) {
  const counts = new Map<string, number>();
  parsedLines.forEach((line) => {
    if (line.path) counts.set(line.path, (counts.get(line.path) ?? 0) + 1);
  });
  const popularPaths = Array.from(counts.entries())
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .slice(0, PATH_FILTER_LIMIT)
    .map(([path]) => path);
  return Array.from(new Set([...popularPaths, ...selectedPaths])).map((path) => ({
    path,
    count: counts.get(path) ?? 0,
  }));
}
