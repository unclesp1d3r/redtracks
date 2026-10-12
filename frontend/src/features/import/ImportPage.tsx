import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { QueryStates } from "@/components/QueryStates";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { EngagementService, ImportBadge, ImportService, type PreviewResponse } from "@/lib/api";
import { keys } from "@/lib/query";
import { DropZone } from "./DropZone";
import { PreviewTable } from "./PreviewTable";
import { isResolved, ResolveHosts, toOverrides, type Choices } from "./ResolveHosts";

function Preview({ response, onDone }: { response: PreviewResponse; onDone: () => void }) {
  const navigate = useNavigate();
  const { Token: token, Preview: preview } = response;
  const rows = preview.Rows ?? [];
  const unresolved = preview.Unresolved ?? [];
  const [choices, setChoices] = useState<Choices>({});
  const [skip, setSkip] = useState(false);
  const engagement = useQuery({ queryKey: keys.engagement(), queryFn: () => EngagementService.Current() });

  const pending = unresolved.filter((u) => !isResolved(choices[u.Key]));
  const apply = useMutation({
    mutationFn: () => ImportService.Apply(token, toOverrides(choices)),
    onSuccess: () => {
      const count = (b: string) => rows.filter((r) => r.Badge === b).length;
      toast.success(
        `Imported ${count(ImportBadge.BadgeNew)} new, ${count(ImportBadge.BadgeFills)} filled; ${pending.length} unresolved skipped.`,
      );
      void navigate("/board");
    },
  });
  const discard = useMutation({ mutationFn: () => ImportService.Discard(token), onSuccess: onDone });

  const busy = apply.isPending || discard.isPending;
  return (
    <div className="space-y-4">
      <p className="text-sm">
        Detected tool: <strong>{preview.Tool}</strong>
      </p>
      {rows.length === 0 ? (
        <p className="text-sm text-muted-foreground">Nothing in this file would change the engagement.</p>
      ) : (
        <PreviewTable rows={rows} />
      )}
      {unresolved.length > 0 && (
        <section className="space-y-2" aria-label="Unresolved hosts">
          <h2 className="text-sm font-semibold">Unresolved hosts ({unresolved.length})</h2>
          <QueryStates query={engagement} isEmpty={() => false} empty={null}>
            {(eng) => (
              <ResolveHosts
                unresolved={unresolved}
                hostIps={(eng.Hosts ?? []).map((h) => h.ip)}
                choices={choices}
                onChange={setChoices}
              />
            )}
          </QueryStates>
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={skip} onCheckedChange={(v) => setSkip(v === true)} />
            Skip unresolved
          </label>
        </section>
      )}
      <div className="flex gap-2">
        <Button disabled={busy || (pending.length > 0 && !skip)} onClick={() => apply.mutate()}>
          Confirm
        </Button>
        <Button variant="outline" disabled={busy} onClick={() => discard.mutate()}>
          Cancel
        </Button>
      </div>
    </div>
  );
}

export function ImportPage() {
  const [response, setResponse] = useState<PreviewResponse | null>(null);
  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Import</h1>
      {response ? (
        <Preview response={response} onDone={() => setResponse(null)} />
      ) : (
        <DropZone onPreview={setResponse} />
      )}
    </div>
  );
}
