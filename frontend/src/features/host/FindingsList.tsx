import { Badge } from "@/components/ui/badge";
import type { Finding } from "@/lib/api";

/** Findings already filtered to the selected service by the caller. */
export function FindingsList({ findings }: { findings: Finding[] }) {
  if (findings.length === 0) return <p className="text-sm text-muted-foreground">No findings for this service.</p>;
  return (
    <ul className="space-y-2">
      {findings.map((f) => (
        <li key={f.id} className="rounded-md border p-2 text-sm">
          <div className="flex items-center gap-2">
            <Badge variant="outline">{f.severity}</Badge>
            <span className="font-medium">{f.title}</span>
            <span className="ml-auto text-xs text-muted-foreground">{f.status}</span>
          </div>
          {f.evidence && <p className="mt-1 whitespace-pre-wrap text-xs text-muted-foreground">{f.evidence}</p>}
        </li>
      ))}
    </ul>
  );
}
