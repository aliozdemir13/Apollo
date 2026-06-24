// Small formatting helpers shared across views.

const DOW = ["SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"];
const MON = [
  "JAN", "FEB", "MAR", "APR", "MAY", "JUN",
  "JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
];

const pad2 = (n: number): string => (n < 10 ? "0" + n : "" + n);

/** "01:00" — 24-hour clock. */
export function clockTime(d: Date): string {
  return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

export function seconds(d: Date): string {
  return pad2(d.getSeconds());
}

/** "SAT. 18 MAY" */
export function clockDate(d: Date): string {
  return `${DOW[d.getDay()]}. ${d.getDate()} ${MON[d.getMonth()]}`;
}

/** Relative time like "3m", "2h", "5d" from an ISO timestamp. */
export function ago(iso: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (isNaN(then)) return "";
  const secs = Math.max(0, (Date.now() - then) / 1000);
  if (secs < 60) return `${Math.floor(secs)}s`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m`;
  if (secs < 86400) return `${Math.floor(secs / 3600)}h`;
  return `${Math.floor(secs / 86400)}d`;
}
