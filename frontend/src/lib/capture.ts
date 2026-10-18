import { useSyncExternalStore } from "react";
import type { QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { DropResult, EvidenceItem, EvidenceView } from "./api";

/** First segment of every per-host evidence query key. */
export const EVIDENCE_PREFIX = "evidence";

let pendingCaptionId: string | null = null;
const listeners = new Set<() => void>();

/** The item whose caption field should take focus next ("Add caption" toast action). */
export const captionFocus = {
  get: () => pendingCaptionId,
  request: (id: string) => {
    pendingCaptionId = id;
    listeners.forEach((l) => l());
  },
  clear: () => {
    if (pendingCaptionId === null) return;
    pendingCaptionId = null;
    listeners.forEach((l) => l());
  },
  subscribe: (l: () => void) => {
    listeners.add(l);
    return () => {
      listeners.delete(l);
    };
  },
};

export function usePendingCaption(): string | null {
  return useSyncExternalStore(captionFocus.subscribe, captionFocus.get);
}

const itemName = (i: Pick<EvidenceItem, "caption" | "file">) => i.caption || i.file;

/** Another cached item, on any host, with the same bytes. Native drops do not say which host. */
function identicalItemName(qc: QueryClient, added: EvidenceItem): string | undefined {
  if (!added.sha256) return undefined;
  for (const [, view] of qc.getQueriesData<EvidenceView>({ queryKey: [EVIDENCE_PREFIX] })) {
    const twin = (view?.items ?? []).find((i) => i.sha256 === added.sha256 && i.id !== added.id);
    if (twin) return itemName(twin);
  }
  return undefined;
}

/** Success toast (warning when the bytes already exist) with an Add caption action. */
export function reportAdded(qc: QueryClient, item: EvidenceItem, isDuplicate: boolean) {
  const action = { label: "Add caption", onClick: () => captionFocus.request(item.id) };
  if (isDuplicate) {
    const twin = identicalItemName(qc, item) ?? "an existing item";
    toast.warning(`Stored ${item.file}, but identical bytes already exist as ${twin}.`, { action });
    return;
  }
  toast.success(`Captured ${item.kind} ${item.file}`, { action });
}

export function reportRefused(name: string, reason: string) {
  toast.error(name ? `${name}: ${reason}` : reason, { duration: Infinity });
}

/** One toast per entry of a native drop. The import preview, if any, is the caller's to route. */
export function reportDropResult(qc: QueryClient, result: DropResult) {
  (result.Added ?? []).forEach((a) => reportAdded(qc, a.Item, a.Duplicate));
  (result.Refused ?? []).forEach((r) => reportRefused(r.Path, r.Reason));
}
