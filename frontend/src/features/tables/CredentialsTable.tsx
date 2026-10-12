import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { CopyIcon, EyeIcon, EyeOffIcon } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { EngagementService, errorMessage, type Credential, type Host } from "@/lib/api";
import { RowTable } from "./RowTable";
import { NONE, hostIp, hostOptions } from "./shared";
import { TriedOnChips } from "./TriedOnChips";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const REVEAL_MS = 10_000;
const MASK = "••••••••";

function Secret({ credential }: { credential: Credential }) {
  const [revealed, setRevealed] = useState(false);
  useEffect(() => {
    if (!revealed) return;
    const t = setTimeout(() => setRevealed(false), REVEAL_MS);
    return () => clearTimeout(t);
  }, [revealed]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(credential.secret);
      toast.success("Secret copied");
    } catch (e) {
      toast.error(`Copy failed: ${errorMessage(e)}`);
    }
  }
  return (
    <span className="inline-flex items-center gap-1">
      <code>{revealed ? credential.secret : MASK}</code>
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label={`${revealed ? "Hide" : "Reveal"} secret for ${credential.username}`}
        aria-pressed={revealed}
        onClick={() => setRevealed((r) => !r)}
      >
        {revealed ? <EyeOffIcon /> : <EyeIcon />}
      </Button>
      <Button variant="ghost" size="icon-sm" aria-label={`Copy secret for ${credential.username}`} onClick={() => void copy()}>
        <CopyIcon />
      </Button>
    </span>
  );
}

function MarkTried({ credential, hosts }: { credential: Credential; hosts: Host[] }) {
  const [ip, setIp] = useState("");
  const [works, setWorks] = useState(false);
  const mark = useMutation({ mutationFn: () => EngagementService.MarkTried(credential.id, ip, works) });
  const options = hostOptions(hosts, "ip");
  return (
    <div className="flex items-center gap-2">
      <Select items={options} value={ip || NONE} onValueChange={(v) => setIp(v === NONE || v === null ? "" : v)}>
        <SelectTrigger size="sm" aria-label={`Host tried for ${credential.username}`}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <label className="inline-flex items-center gap-1 text-xs">
        <Checkbox checked={works} onCheckedChange={(c) => setWorks(c === true)} aria-label={`Works for ${credential.username}`} />
        works
      </label>
      <Button size="sm" variant="outline" disabled={!ip || mark.isPending} onClick={() => mark.mutate()}>
        Mark
      </Button>
    </div>
  );
}

export function CredentialsTable() {
  return (
    <RowTable<Credential>
      kind="credential"
      noun="credential"
      select={(e) => e.Creds.credentials ?? []}
      headers={["Username", "Domain", "Kind", "Secret", "Found on", "Tried", "Mark tried"]}
      cells={(c, hosts) => [
        c.username,
        c.domain || "-",
        c.kind,
        <Secret key="s" credential={c} />,
        hostIp(hosts, c.foundOnHostId),
        <TriedOnChips key="t" credential={c} hosts={hosts} />,
        <MarkTried key="m" credential={c} hosts={hosts} />,
      ]}
      name={(c) => c.username}
    />
  );
}
