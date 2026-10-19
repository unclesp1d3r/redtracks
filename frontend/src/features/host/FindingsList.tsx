import { Badge } from "@/components/ui/badge";
import { EvidenceLinkKind } from "@/lib/api";
import type { Finding } from "@/lib/api";
import { ProofStrip } from "@/features/evidence/ProofStrip";
import { plainLink } from "@/features/evidence/proof";
import { useTargetRow } from "@/features/evidence/usePaste";
import { cn } from "@/lib/utils";

function FindingRow({ ip, finding: f }: { ip: string; finding: Finding }) {
  const link = plainLink(EvidenceLinkKind.EvidenceLinkFinding, f.id);
  const name = `finding ${f.title}`;
  const rowProps = useTargetRow(ip, link, name);
  return (
    <li>
      <div {...rowProps} className={cn(rowProps.className, "space-y-1 rounded-md border p-2 text-sm")}>
        <div className="flex items-center gap-2">
          <Badge variant="outline">{f.severity}</Badge>
          <span className="font-medium">{f.title}</span>
          <span className="ml-auto text-xs text-muted-foreground">{f.status}</span>
        </div>
        {f.evidence && <p className="whitespace-pre-wrap text-xs text-muted-foreground">{f.evidence}</p>}
        <ProofStrip ip={ip} link={link} name={name} />
      </div>
    </li>
  );
}

/** Findings already filtered to the selected service by the caller. */
export function FindingsList({ ip, findings }: { ip: string; findings: Finding[] }) {
  if (findings.length === 0) return <p className="text-sm text-muted-foreground">No findings for this service.</p>;
  return (
    <ul className="space-y-2">
      {findings.map((f) => (
        <FindingRow key={f.id} ip={ip} finding={f} />
      ))}
    </ul>
  );
}
