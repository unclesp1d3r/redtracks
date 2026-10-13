import { useState } from "react";
import { PlusIcon } from "lucide-react";
import { Link } from "react-router";
import { QueryStates } from "@/components/QueryStates";
import { Button, buttonVariants } from "@/components/ui/button";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { hostsOf } from "@/lib/api";
import { useEngagementQuery, useSummariesQuery } from "@/lib/query";
import { ChecklistStatus } from "../host/ChecklistStatus";
import { AddHostDialog } from "./AddHostDialog";
import { AttackGraph } from "./AttackGraph";
import { HostRow } from "./HostRow";

export function BoardPage() {
  const [adding, setAdding] = useState(false);
  const summaries = useSummariesQuery();
  const engagement = useEngagementQuery();

  const empty = (
    <div className="flex flex-col items-start gap-3 py-10">
      <p className="text-sm text-muted-foreground">No hosts yet. Add one or import scan output.</p>
      <div className="flex gap-2">
        <Button onClick={() => setAdding(true)}>
          <PlusIcon /> Add host
        </Button>
        <Link to="/import" className={buttonVariants({ variant: "outline" })}>
          Import
        </Link>
      </div>
    </div>
  );

  const flagTypes = engagement.data?.Meta.flagTypes ?? [];
  const hostsByIp = new Map((engagement.data ? hostsOf(engagement.data) : []).map((h) => [h.ip, h]));

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Board</h1>
        <Button size="sm" onClick={() => setAdding(true)}>
          <PlusIcon /> Add host
        </Button>
      </div>
      <ChecklistStatus />
      <AttackGraph />
      <QueryStates query={summaries} isEmpty={(s) => s.length === 0} empty={empty}>
        {(rows) => (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Host</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Flags</TableHead>
                <TableHead>Checklist</TableHead>
                <TableHead>Access</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((s) => (
                <HostRow key={s.HostID} summary={s} host={hostsByIp.get(s.IP)} flagTypes={flagTypes} />
              ))}
            </TableBody>
          </Table>
        )}
      </QueryStates>
      <AddHostDialog open={adding} onOpenChange={setAdding} />
    </div>
  );
}
