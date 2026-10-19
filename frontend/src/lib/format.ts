const SIZE_UNITS = ["B", "KB", "MB", "GB"];

/** Formats a byte count with one decimal above bytes, up to GB. */
export function humanSize(bytes: number): string {
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < SIZE_UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  return unit === 0 ? `${value} B` : `${value.toFixed(1)} ${SIZE_UNITS[unit]}`;
}
