import { CheckIcon, TriangleAlertIcon } from "lucide-react";
import { Link } from "react-router";
import { Badge } from "@/components/ui/badge";
import { TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { hostPath, type FlagType, type Host, type HostSummary } from "@/lib/api";

const STATUS_STYLE: Record<string, string> = {
  root: "bg-primary text-primary-foreground",
  done: "bg-primary text-primary-foreground",
  user: "bg-accent/20 text-accent",
  foothold: "bg-accent/20 text-accent",
};

interface Props {
  summary: HostSummary;
  host: Host | undefined;
  flagTypes: FlagType[];
}

export function HostRow({ summary, host, flagTypes }: Props) {
  const captured = new Set((host?.flags ?? []).filter((f) => f.value !== "").map((f) => f.typeId));
  const pathCount = host?.accessPaths?.length ?? 0;
  const href = hostPath(summary.IP);
  return (
    <TableRow>
      <TableCell>
        <Link to={href} className="font-mono text-sm font-medium hover:underline">
          {summary.IP}
        </Link>
        {summary.Hostname && <span className="ml-2 text-muted-foreground">{summary.Hostname}</span>}
      </TableCell>
      <TableCell>
        <Badge variant="secondary" className={cn(STATUS_STYLE[summary.Status])}>
          {summary.Status}
        </Badge>
      </TableCell>
      <TableCell>
        <div className="flex gap-1">
          {flagTypes.map((ft) => {
            const isCaptured = captured.has(ft.id);
            return (
              <Badge key={ft.id} variant={isCaptured ? "secondary" : "outline"}>
                {isCaptured && <CheckIcon aria-hidden />}
                {ft.label}
              </Badge>
            );
          })}
        </div>
      </TableCell>
      <TableCell className="font-mono text-xs tabular-nums">
        {summary.ItemsDone}/{summary.ItemsTotal}
      </TableCell>
      <TableCell>
        {summary.NoAccessPath ? (
          <span className="inline-flex items-center gap-1 text-xs text-accent">
            <TriangleAlertIcon className="size-3.5" aria-hidden />
            no access path
          </span>
        ) : (
          <span className="font-mono text-xs tabular-nums">{pathCount}</span>
        )}
      </TableCell>
    </TableRow>
  );
}
