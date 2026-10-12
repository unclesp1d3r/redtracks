import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon } from "lucide-react";
import { useNavigate } from "react-router";
import { QueryStates } from "@/components/QueryStates";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { EngagementService, list } from "@/lib/api";
import { keys } from "@/lib/query";
import { CreateEngagementDialog } from "./CreateEngagementDialog";

export function EngagementsPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [creating, setCreating] = useState(false);
  const engagements = useQuery({ queryKey: keys.engagements(), queryFn: () => list(EngagementService.List()) });

  const openEngagement = useMutation({
    mutationFn: (dir: string) => EngagementService.Open(dir),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: keys.engagement() });
      navigate("/board");
    },
  });

  const empty = (
    <div className="flex flex-col items-start gap-3 py-10">
      <p className="text-sm text-muted-foreground">No engagements yet.</p>
      <Button onClick={() => setCreating(true)}>
        <PlusIcon /> New engagement
      </Button>
    </div>
  );

  return (
    <main className="mx-auto max-w-3xl space-y-6 p-8">
      <header className="flex items-end justify-between">
        <div>
          <p className="text-xs font-semibold tracking-widest text-primary uppercase">RedTracks</p>
          <h1 className="text-3xl font-semibold tracking-tight">Engagements</h1>
        </div>
        <Button onClick={() => setCreating(true)}>
          <PlusIcon /> New engagement
        </Button>
      </header>
      <QueryStates query={engagements} isEmpty={(e) => e.length === 0} empty={empty}>
        {(rows) => (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="text-right">Hosts</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((e) => (
                <TableRow key={e.Dir} className="cursor-pointer" onClick={() => openEngagement.mutate(e.Dir)}>
                  <TableCell>
                    <button
                      type="button"
                      className="font-medium hover:underline"
                      onClick={(ev) => {
                        ev.stopPropagation();
                        openEngagement.mutate(e.Dir);
                      }}
                    >
                      {e.Name}
                    </button>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{e.Type}</Badge>
                  </TableCell>
                  <TableCell className="text-right font-mono tabular-nums">{e.Hosts}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {new Date(e.CreatedAt).toLocaleDateString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </QueryStates>
      <CreateEngagementDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={() => {
          setCreating(false);
          void qc.invalidateQueries({ queryKey: keys.engagements() });
          void qc.invalidateQueries({ queryKey: keys.engagement() });
          navigate("/board");
        }}
      />
    </main>
  );
}
