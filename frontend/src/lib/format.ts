export function progressPercent(complete: number, total: number): number {
  if (!Number.isFinite(complete) || !Number.isFinite(total) || total <= 0) return 0;
  return Math.round(Math.min(100, Math.max(0, (complete / total) * 100)));
}

export function formatBytes(bytes: number): string {
  const value = Number.isFinite(bytes) ? Math.max(0, bytes) : 0;
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let scaled = value;
  let unitIndex = 0;
  while (scaled >= 1024 && unitIndex < units.length - 1) {
    scaled /= 1024;
    unitIndex += 1;
  }
  const digits = unitIndex === 0 || scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2;
  return `${Number(scaled.toFixed(digits))} ${units[unitIndex]}`;
}

export function formatSpeed(bytesPerSecond: number): string {
  return `${formatBytes(bytesPerSecond)}/s`;
}

export function formatNumber(value: number, locale?: string): string {
  return new Intl.NumberFormat(locale).format(value);
}

export function formatCompactNumber(value: number, locale?: string): string {
  if (Math.abs(value) < 1000) return formatNumber(value, locale);
  const divisor = Math.abs(value) >= 500_000 ? 1_000_000 : 1_000;
  const suffix = divisor === 1_000_000 ? "M" : "K";
  return `${new Intl.NumberFormat(locale, { maximumSignificantDigits: 3 }).format(value / divisor)}${suffix}`;
}

export function errorMessage(error: unknown): string {
  if (!error) return "";
  if (typeof error === "string") return error;
  if (error instanceof Error) return error.message || String(error);
  return String(error);
}
