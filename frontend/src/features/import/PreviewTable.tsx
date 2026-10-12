import { Badge } from "@/components/ui/badge";
import { ImportBadge, type Row } from "@/lib/api";
import { cn } from "@/lib/utils";

// Colour is paired with the badge text, never the only signal.
const BADGE_STYLE: Record<string, string> = {
  [ImportBadge.BadgeNew]: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  [ImportBadge.BadgeFills]: "bg-sky-500/15 text-sky-700 dark:text-sky-300",
  [ImportBadge.BadgeDuplicate]: "bg-muted text-muted-foreground",
};

function groupByHost(rows: Row[]): [string, Row[]][] {
  const groups = new Map<string, Row[]>();
  for (const row of rows) groups.set(row.HostIP, [...(groups.get(row.HostIP) ?? []), row]);
  return [...groups.entries()];
}

export function PreviewTable({ rows }: { rows: Row[] }) {
  return (
    <div className="space-y-4">
      {groupByHost(rows).map(([ip, group]) => (
        <section key={ip} aria-label={`Host ${ip}`}>
          <h3 className="mb-1 font-mono text-sm font-semibold">{ip}</h3>
          <ul className="divide-y rounded-md border">
            {group.map((row, i) => (
              <li key={`${row.Kind}-${row.Label}-${i}`} className="flex items-center gap-3 px-3 py-1.5 text-sm">
                <Badge variant="outline" className={cn("w-20 border-transparent", BADGE_STYLE[row.Badge])}>
                  {row.Badge}
                </Badge>
                <span className="w-16 text-xs text-muted-foreground">{row.Kind}</span>
                <span className="font-mono">{row.Label}</span>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
