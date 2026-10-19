import { useEffect, useState } from "react";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { TriangleAlertIcon } from "lucide-react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { QueryStates } from "@/components/QueryStates";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { ImportBadge, ImportService, hostsOf, type PreviewResponse } from "@/lib/api";
import { keys, useEngagementQuery } from "@/lib/query";
import { DropZone } from "./DropZone";
import { PreviewTable } from "./PreviewTable";
import { isResolved, ResolveHosts, toOverrides, type Choices } from "./ResolveHosts";

export function Preview({ response, onDone }: { response: PreviewResponse; onDone: () => void }) {
  const navigate = useNavigate();
  const { Token: token, Preview: preview } = response;
  const unresolved = preview.Unresolved ?? [];
  const dropped = preview.Dropped ?? [];
  const [choices, setChoices] = useState<Choices>({});
  const [skip, setSkip] = useState(false);
  const engagement = useEngagementQuery();

  const pending = unresolved.filter((u) => !isResolved(choices[u.Key]));
  const overrides = toOverrides(choices);
  const hasOverrides = Object.keys(overrides).length > 0;
  // An override changes which rows the import writes, so the table is recomputed
  // with it before Confirm. The unresolved list keeps coming from the original
  // preview so a choice can still be changed.
  const repreview = useQuery({
    queryKey: ["import", token, "repreview", overrides],
    queryFn: () => ImportService.Repreview(token, overrides),
    enabled: hasOverrides,
    placeholderData: keepPreviousData,
  });
  const rows = (hasOverrides && repreview.data ? repreview.data.Rows : preview.Rows) ?? [];
  const previewing = hasOverrides && (repreview.isFetching || !repreview.isSuccess);
  const apply = useMutation({
    mutationFn: () => ImportService.Apply(token, overrides),
    onSuccess: (applied) => {
      const count = (b: string) => (applied.Rows ?? []).filter((r) => r.Badge === b).length;
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
      {(dropped.length > 0 || preview.Skipped > 0) && (
        <p role="status" className="flex items-start gap-1 text-sm text-amber-600 dark:text-amber-400">
          <TriangleAlertIcon className="mt-0.5 size-4 shrink-0" aria-hidden />
          <span>
            {dropped.length > 0 && `Dropped ${dropped.length} invalid IP${dropped.length === 1 ? "" : "s"}: ${dropped.join(", ")}. `}
            {preview.Skipped > 0 && `Skipped ${preview.Skipped} unreadable row${preview.Skipped === 1 ? "" : "s"}.`}
          </span>
        </p>
      )}
      {repreview.isError && (
        <p role="alert" className="text-sm text-destructive">
          Could not preview the resolved hosts: {String(repreview.error)}
        </p>
      )}
      {rows.length > 0 ? (
        <PreviewTable rows={rows} />
      ) : (
        unresolved.length === 0 && (
          <p className="text-sm text-muted-foreground">Nothing in this file would change the engagement.</p>
        )
      )}
      {unresolved.length > 0 && (
        <section className="space-y-2" aria-label="Unresolved hosts">
          <h2 className="text-sm font-semibold">Unresolved hosts ({unresolved.length})</h2>
          <QueryStates query={engagement} isEmpty={() => false} empty={null}>
            {(eng) => (
              <ResolveHosts
                unresolved={unresolved}
                hostIps={hostsOf(eng).map((h) => h.ip)}
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
        <Button disabled={busy || previewing || (pending.length > 0 && !skip)} onClick={() => apply.mutate()}>
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
  const qc = useQueryClient();
  // A native drop's preview arrives over evidence:dropped and is parked in the query cache.
  const dropped = useQuery({
    queryKey: keys.droppedPreview(),
    queryFn: () => Promise.resolve<PreviewResponse | null>(null),
    enabled: false,
    staleTime: Infinity,
  }).data;
  useEffect(() => {
    if (!dropped) return;
    setResponse(dropped);
    qc.setQueryData(keys.droppedPreview(), null);
  }, [dropped, qc]);
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
