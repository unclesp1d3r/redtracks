import { useMutation } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { EngagementService, type Finding } from "@/lib/api";
import { RowTable } from "./RowTable";
import { FINDING_STATUSES, hostIp } from "./shared";

function StatusSelect({ finding }: { finding: Finding }) {
  const update = useMutation({ mutationFn: (status: string) => EngagementService.UpdateFinding({ ...finding, status }) });
  return (
    <Select value={finding.status} onValueChange={(v) => v && update.mutate(v)}>
      <SelectTrigger size="sm" aria-label={`Status of ${finding.title}`}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {FINDING_STATUSES.map((s) => (
          <SelectItem key={s} value={s}>
            {s}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function FindingsTable() {
  return (
    <RowTable<Finding>
      kind="finding"
      noun="finding"
      select={(e) => e.Findings ?? []}
      headers={["Host", "Title", "Severity", "Source", "Status"]}
      cells={(f, hosts) => [
        hostIp(hosts, f.hostId),
        f.title,
        <Badge key="s" variant="outline">{f.severity}</Badge>,
        f.source,
        <StatusSelect key="st" finding={f} />,
      ]}
      name={(f) => f.title}
    />
  );
}
