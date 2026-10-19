import { useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { XIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { EvidenceLinkKind, HostService, hostsOf, list, serviceLabel } from "@/lib/api";
import type { EvidenceLink, Host } from "@/lib/api";
import { keys, useEngagementQuery } from "@/lib/query";
import { formatTarget } from "./dropTarget";
import { plainLink, slotLink } from "./proof";

export interface PickedTarget {
  ip: string;
  links: EvidenceLink[];
}

interface Props {
  /** Fixes the host (the host select is disabled); omit to let the operator choose. */
  ip?: string;
  onChange: (target: PickedTarget) => void;
}

interface Candidate {
  link: EvidenceLink;
  label: string;
}

/** Owed slots come from the checklist definitions already matched to the host and each service. */
function useSlotCandidates(host: Host | undefined): Candidate[] {
  const ip = host?.ip ?? "";
  const services = host?.services ?? [];
  const serviceIds = ["", ...services.map((s) => s.id)];
  const results = useQueries({
    queries: serviceIds.map((sid) => ({
      queryKey: keys.checklists(ip, sid),
      queryFn: () => list(HostService.Checklists(ip, sid)),
      enabled: ip !== "",
    })),
  });
  return results.flatMap((r, i) => {
    const sid = serviceIds[i];
    const service = services.find((s) => s.id === sid);
    return (r.data ?? []).flatMap((v) =>
      (v.Definition.slots ?? []).map((s) => ({
        link: slotLink(s.id, sid),
        label: service ? `${s.title} (${serviceLabel(service)})` : s.title,
      })),
    );
  });
}

const KIND_WORD: Record<EvidenceLinkKind, string> = {
  [EvidenceLinkKind.$zero]: "",
  [EvidenceLinkKind.EvidenceLinkSlot]: "slot",
  [EvidenceLinkKind.EvidenceLinkFinding]: "finding",
  [EvidenceLinkKind.EvidenceLinkFlag]: "flag",
  [EvidenceLinkKind.EvidenceLinkAccessPath]: "access path",
};

interface PickerProps {
  label: string;
  candidates: Candidate[];
  onPick: (link: EvidenceLink) => void;
}

function KindPicker({ label, candidates, onPick }: PickerProps) {
  return (
    <Select
      value=""
      onValueChange={(v) => {
        const hit = candidates.find((c) => formatTarget(c.link) === v);
        if (hit) onPick(hit.link);
      }}
    >
      <SelectTrigger size="sm" aria-label={label} disabled={candidates.length === 0} className="w-44">
        <SelectValue placeholder={label} />
      </SelectTrigger>
      <SelectContent>
        {candidates.map((c) => (
          <SelectItem key={formatTarget(c.link)} value={formatTarget(c.link)}>
            {c.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/** Host plus any number of slot, finding, flag and access-path links; no link at all means host only. */
export function TargetPicker({ ip: fixedIp, onChange }: Props) {
  const engagement = useEngagementQuery().data;
  const [chosenIp, setChosenIp] = useState(fixedIp ?? "");
  const [links, setLinks] = useState<EvidenceLink[]>([]);
  const hosts = engagement ? hostsOf(engagement) : [];
  const host = hosts.find((h) => h.ip === chosenIp);
  const slots = useSlotCandidates(host);

  const candidates: Record<string, Candidate[]> = {
    Slot: slots,
    Finding: (engagement?.Findings ?? [])
      .filter((f) => f.hostId === host?.id)
      .map((f) => ({ link: plainLink(EvidenceLinkKind.EvidenceLinkFinding, f.id), label: f.title })),
    Flag: (engagement?.Meta.flagTypes ?? []).map((f) => ({
      link: plainLink(EvidenceLinkKind.EvidenceLinkFlag, f.id),
      label: f.label,
    })),
    "Access path": (host?.accessPaths ?? []).map((p) => ({
      link: plainLink(EvidenceLinkKind.EvidenceLinkAccessPath, p.id),
      label: p.method,
    })),
  };
  const labelOf = (l: EvidenceLink) =>
    Object.values(candidates).flat().find((c) => formatTarget(c.link) === formatTarget(l))?.label ?? l.targetId;

  function update(nextIp: string, nextLinks: EvidenceLink[]) {
    setChosenIp(nextIp);
    setLinks(nextLinks);
    onChange({ ip: nextIp, links: nextLinks });
  }
  const add = (l: EvidenceLink) => {
    if (!links.some((x) => formatTarget(x) === formatTarget(l))) update(chosenIp, [...links, l]);
  };

  return (
    <div className="space-y-2">
      <Select value={chosenIp} onValueChange={(v) => update(v ?? "", [])} disabled={fixedIp !== undefined}>
        <SelectTrigger size="sm" aria-label="Host" className="w-48">
          <SelectValue placeholder="Choose host" />
        </SelectTrigger>
        <SelectContent>
          {hosts.map((h) => (
            <SelectItem key={h.ip} value={h.ip}>
              {h.ip}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <div className="flex flex-wrap gap-2">
        {Object.entries(candidates).map(([label, list]) => (
          <KindPicker key={label} label={label} candidates={list} onPick={add} />
        ))}
      </div>
      <ul aria-label="Selected links" className="flex flex-wrap gap-1.5">
        {links.length === 0 && <li className="text-xs text-muted-foreground">Host only</li>}
        {links.map((l) => (
          <li key={formatTarget(l)} className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs">
            <span className="text-muted-foreground">{KIND_WORD[l.kind]}</span>
            <span>{labelOf(l)}</span>
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label={`Remove ${KIND_WORD[l.kind]} ${labelOf(l)}`}
              onClick={() => update(chosenIp, links.filter((x) => formatTarget(x) !== formatTarget(l)))}
            >
              <XIcon aria-hidden />
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
}
