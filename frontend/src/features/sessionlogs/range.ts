/** Most lines one clip may carry; mirrors the backend limit. */
export const MAX_CLIP_LINES = 5000;

export interface LineRange {
  /** 0-based, inclusive */
  from: number;
  /** 0-based, inclusive */
  to: number;
}

export type RangeResult = { range: LineRange; error?: undefined } | { range?: undefined; error: string };

/** Parses the 1-based "from-to" (or single "n") text into a 0-based inclusive range within the log. */
export function parseRange(text: string, lines: number): RangeResult {
  const m = /^\s*(\d+)\s*(?:-\s*(\d+))?\s*$/.exec(text);
  if (!m) return { error: "Enter a range like 10-49" };
  const from = Number(m[1]);
  const to = m[2] === undefined ? from : Number(m[2]);
  if (from < 1 || to < from || to > lines) return { error: `Range must stay within lines 1 to ${lines}` };
  if (to - from + 1 > MAX_CLIP_LINES) {
    return { error: `${to - from + 1} lines is over the limit of ${MAX_CLIP_LINES.toLocaleString("en-US")} lines per clip` };
  }
  return { range: { from: from - 1, to: to - 1 } };
}

export const lineCount = (r: LineRange) => r.to - r.from + 1;
