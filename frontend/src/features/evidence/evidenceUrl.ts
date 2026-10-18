/** Relative URL the evidence asset route serves (same origin in the desktop and server builds). */
export function evidenceUrl(ip: string, file: string): string {
  return `/evidence/${encodeURIComponent(ip)}/${encodeURIComponent(file)}`;
}
