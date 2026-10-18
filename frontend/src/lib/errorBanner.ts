import { useSyncExternalStore } from "react";

/** Payload of the backend's engagement:error event. Path is relative to the engagement folder. */
export interface ErrorEvent {
  Kind: "engagement" | "checklists" | "inbox";
  Path: string;
  Message: string;
}

const entryKey = (e: ErrorEvent) => `${e.Kind}:${e.Path}`;

let entries: readonly ErrorEvent[] = [];
const listeners = new Set<() => void>();

function emit(next: readonly ErrorEvent[]) {
  if (next.length === entries.length && next.every((e, i) => e === entries[i])) return;
  entries = next;
  listeners.forEach((l) => l());
}

/** Unresolved engagement:error entries, one per Kind+Path, shown by AppShell. */
export const errorBanner = {
  get: () => entries,
  add: (ev: ErrorEvent) => emit([...entries.filter((e) => entryKey(e) !== entryKey(ev)), ev]),
  /** Clears engagement entries for the changed paths. Entries with no path stay until the engagement is reopened. */
  clearChanged: (paths: readonly string[]) =>
    emit(entries.filter((e) => e.Kind !== "engagement" || e.Path === "" || !paths.includes(e.Path))),
  clearKind: (kind: ErrorEvent["Kind"]) => emit(entries.filter((e) => e.Kind !== kind)),
  subscribe: (l: () => void) => {
    listeners.add(l);
    return () => {
      listeners.delete(l);
    };
  },
};

export function useErrorBanner(): readonly ErrorEvent[] {
  return useSyncExternalStore(errorBanner.subscribe, errorBanner.get);
}
