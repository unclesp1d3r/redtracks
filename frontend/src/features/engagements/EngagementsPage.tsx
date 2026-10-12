import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, TriangleAlertIcon } from "lucide-react";
import { useNavigate } from "react-router";
import { QueryStates } from "@/components/QueryStates";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { EngagementService, errorMessage } from "@/lib/api";
import { keys } from "@/lib/query";
import { CreateEngagementDialog } from "./CreateEngagementDialog";

export function EngagementsPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [creating, setCreating] = useState(false);
  const engagements = useQuery({
    queryKey: keys.engagements(),
    queryFn: () => EngagementService.List(),
  });
  const root = useQuery({
    queryKey: keys.root(),
    queryFn: () => EngagementService.Root(),
    staleTime: Infinity,
  });

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
      <QueryStates
        query={engagements}
        isEmpty={(r) => (r.Engagements ?? []).length === 0 && (r.Broken ?? []).length === 0}
        empty={empty}
      >
        {(resp) => (
          <>
            {(resp.Engagements ?? []).length === 0 ? (
              empty
            ) : (
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
                  {(resp.Engagements ?? []).map((e) => (
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
            {(resp.Broken ?? []).length > 0 && (
              <section aria-labelledby="broken-heading" className="mt-6 space-y-2">
                <h2 id="broken-heading" className="text-sm font-semibold">
                  Could not read
                </h2>
                <ul className="space-y-1">
                  {(resp.Broken ?? []).map((b) => (
                    <li key={b.Dir} className="flex items-start gap-2 text-sm text-destructive">
                      <TriangleAlertIcon className="mt-0.5 size-4 shrink-0" aria-hidden />
                      <span>
                        <span className="font-mono">{b.Dir}</span>: {b.Error}
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            )}
          </>
        )}
      </QueryStates>
      {openEngagement.isError && (
        <p role="alert" className="flex items-center gap-2 text-sm text-destructive">
          <TriangleAlertIcon className="size-4" aria-hidden />
          Could not open engagement: {errorMessage(openEngagement.error)}
        </p>
      )}
      {root.data && <p className="font-mono text-xs text-muted-foreground">Folder: {root.data}</p>}
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
