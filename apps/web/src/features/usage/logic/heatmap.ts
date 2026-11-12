/**
 * Pure helpers of the heatmap: cell intensities and accessible labels.
 */

export interface HeatmapCell {
  day: number;
  hour: number;
  value: number;
  /** 0 to 1 against the matrix peak; 0 when the peak is 0. */
  intensity: number;
}

/** One cell per matrix entry, row-major (7 rows, Monday first). */
export function buildHeatmapCells(matrix: number[][]): HeatmapCell[] {
  const peak = matrix.flat().reduce((max, value) => Math.max(max, value), 0);
  return matrix.flatMap((row, day) =>
    row.map((value, hour) => ({
      day,
      hour,
      value,
      intensity: peak > 0 ? Math.max(0, Math.min(1, value / peak)) : 0,
    }))
  );
}

/** Accessible name of one cell: weekday, hour, and the formatted value. */
export function heatmapCellLabel(
  weekday: string,
  hour: number,
  formattedValue: string
): string {
  return `${weekday}, ${hour}:00, ${formattedValue}`;
}
