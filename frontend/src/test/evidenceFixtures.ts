// Enums come from the bindings, not @/lib/api: the api module is mocked with these mocks, so importing it here would deadlock.
import { EvidenceKind, EvidenceLinkKind } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
import type { EvidenceItemView, EvidenceLink, EvidenceView, LinkView } from "@/lib/api";

export const IP = "10.0.0.1";

export function link(kind: EvidenceLinkKind, targetId: string, label = targetId, extra: Partial<LinkView> = {}): LinkView {
  return { kind, targetId, removed: false, label, ...extra };
}

export function toEvidenceLink(l: LinkView): EvidenceLink {
  return { kind: l.kind, targetId: l.targetId, serviceId: l.serviceId };
}

export function item(id: string, links: LinkView[] = [], extra: Partial<EvidenceItemView> = {}): EvidenceItemView {
  return {
    id,
    file: `${id}.png`,
    kind: EvidenceKind.EvidenceKindImage,
    capturedAt: "2026-10-01T12:00:00Z",
    caption: "",
    missing: false,
    links,
    ...extra,
  };
}

export function view(items: EvidenceItemView[] = [], notIndexed: string[] = []): EvidenceView {
  return { items, notIndexed };
}

/** A minimal engagement with one host: a 80/tcp service, one finding, one access path. */
export function engagement() {
  return {
    Meta: { name: "e", domain: "", flagTypes: [{ id: "proof.txt", label: "proof.txt", points: 1 }] },
    Creds: { credentials: [] },
    Findings: [{ id: "f1", hostId: "h1", serviceId: "s1", title: "SQLi", severity: "high", source: "", evidence: "", status: "open" }],
    Hosts: [
      {
        id: "h1",
        ip: IP,
        hostname: "",
        os: "linux",
        status: "untouched",
        services: [{ id: "s1", port: 80, protocol: "tcp", type: "http", webPaths: [] }],
        flags: [],
        accessPaths: [{ id: "ap1", method: "chisel", viaHostId: "", credentialId: "", command: "", notes: "" }],
      },
    ],
  };
}

export function slotViews(slots: { id: string; title: string; hint?: string }[]) {
  return [
    {
      Definition: {
        id: "d1",
        title: "Host checklist",
        match: { serviceTypes: [], hostOS: ["*"] },
        items: [],
        slots: slots.map((s) => ({ hint: "", ...s })),
      },
      Items: [],
    },
  ];
}
