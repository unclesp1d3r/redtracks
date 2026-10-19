import { useState } from "react";
import { CircleCheckIcon, CircleDashedIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { EvidenceItemView, EvidenceLink } from "@/lib/api";
import { AttachFile } from "./AttachFile";
import { EvidenceViewer } from "./EvidenceViewer";
import { isProven, itemName, itemsLinkedTo } from "./proof";
import { useEvidence, useEvidenceActions } from "./useEvidence";

export interface ProofWords {
  yes: string;
  no: string;
}

interface Props {
  ip: string;
  link: EvidenceLink;
  /** Names the target in control labels, e.g. "flag proof.txt". */
  name: string;
  /** Words for the proof state; absent for targets (findings) that have no proven/unproven state. */
  words?: ProofWords;
}

function ProofState({ isProven: proven, words }: { isProven: boolean; words: ProofWords }) {
  const Icon = proven ? CircleCheckIcon : CircleDashedIcon;
  return (
    <span className="inline-flex items-center gap-1 text-xs font-medium">
      <Icon className="size-3.5" aria-hidden />
      <span>{proven ? words.yes : words.no}</span>
    </span>
  );
}

function LinkExisting({ ip, link, name, items }: { ip: string; link: EvidenceLink; name: string; items: EvidenceItemView[] }) {
  const { link: linkItem } = useEvidenceActions(ip);
  const linked = new Set(itemsLinkedTo(items, link).map((i) => i.id));
  const candidates = items.filter((i) => !linked.has(i.id));
  return (
    <Select value="" onValueChange={(id) => id && linkItem.mutate({ itemId: id, link })}>
      <SelectTrigger size="sm" aria-label={`Link existing item to ${name}`} disabled={candidates.length === 0} className="h-6 w-40 text-xs">
        <SelectValue placeholder="Link existing item" />
      </SelectTrigger>
      <SelectContent>
        {candidates.map((i) => (
          <SelectItem key={i.id} value={i.id}>
            {itemName(i)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/** Proof state, linked-item chips, and the link and attach controls for one target row. */
export function ProofStrip({ ip, link, name, words }: Props) {
  const query = useEvidence(ip);
  const [viewing, setViewing] = useState<EvidenceItemView | null>(null);
  const items = query.data?.items ?? [];
  const linked = itemsLinkedTo(items, link);
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {words && (query.isSuccess ? <ProofState isProven={isProven(items, link)} words={words} /> : <span className="text-xs">checking</span>)}
      {linked.map((i) => (
        <Button key={i.id} size="xs" variant="secondary" onClick={() => setViewing(i)}>
          {itemName(i)}
          {i.missing && <span className="text-amber-600 dark:text-amber-400">(file missing)</span>}
        </Button>
      ))}
      <LinkExisting ip={ip} link={link} name={name} items={items} />
      <AttachFile ip={ip} link={link} label={`Attach file to ${name}`} inputLabel={`Evidence file for ${name}`} />
      {viewing && <EvidenceViewer ip={ip} item={viewing} open onOpenChange={(o) => !o && setViewing(null)} />}
    </div>
  );
}
