import { EvidenceLinkKind } from "@/lib/api";
import type { EvidenceItemView, EvidenceLink, LinkView } from "@/lib/api";

/** A slot link carries its service id ("" for a host-level slot); other kinds carry none. */
export function slotLink(slotId: string, serviceId: string): EvidenceLink {
  return { kind: EvidenceLinkKind.EvidenceLinkSlot, targetId: slotId, serviceId };
}

export function plainLink(kind: EvidenceLinkKind, targetId: string): EvidenceLink {
  return { kind, targetId };
}

export function linkOf(l: LinkView): EvidenceLink {
  return { kind: l.kind, targetId: l.targetId, serviceId: l.serviceId };
}

/** A host-level slot and a service slot with the same id are different targets. */
function isSameTarget(l: LinkView, target: EvidenceLink): boolean {
  if (l.kind !== target.kind || l.targetId !== target.targetId) return false;
  return target.kind !== EvidenceLinkKind.EvidenceLinkSlot || (l.serviceId ?? "") === (target.serviceId ?? "");
}

export function itemsLinkedTo(items: readonly EvidenceItemView[], target: EvidenceLink): EvidenceItemView[] {
  return items.filter((i) => (i.links ?? []).some((l) => isSameTarget(l, target)));
}

/** Proven means a present file is linked to a target that still exists. */
export function isProven(items: readonly EvidenceItemView[], target: EvidenceLink): boolean {
  return items.some((i) => !i.missing && (i.links ?? []).some((l) => !l.removed && isSameTarget(l, target)));
}

export function itemName(i: { caption: string; file: string }): string {
  return i.caption || i.file;
}
