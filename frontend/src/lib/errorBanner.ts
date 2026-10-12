import { useSyncExternalStore } from "react";

let message: string | null = null;
const listeners = new Set<() => void>();

function emit(next: string | null) {
  message = next;
  listeners.forEach((l) => l());
}

/** Latest engagement:error text, shown by AppShell until the next engagement:changed. */
export const errorBanner = {
  get: () => message,
  set: (m: string) => emit(m),
  clear: () => {
    if (message !== null) emit(null);
  },
  subscribe: (l: () => void) => {
    listeners.add(l);
    return () => {
      listeners.delete(l);
    };
  },
};

export function useErrorBanner(): string | null {
  return useSyncExternalStore(errorBanner.subscribe, errorBanner.get);
}
