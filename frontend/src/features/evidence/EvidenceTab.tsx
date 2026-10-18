import { QueryStates } from "@/components/QueryStates";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { AttachFile } from "./AttachFile";
import { EvidenceItemRow } from "./EvidenceItemRow";
import { DROP_ACTIVE_CLASSES, useDropTarget } from "./dropTarget";
import { useEvidence, useEvidenceActions } from "./useEvidence";

/** A host's evidence in capture order, plus files found in its folder that have no entry yet. */
export function EvidenceTab({ ip }: { ip: string }) {
  const query = useEvidence(ip);
  const { adopt } = useEvidenceActions(ip);
  const drop = useDropTarget(ip);
  return (
    <section aria-label="Evidence" {...drop.props} className={cn("space-y-2 rounded-md p-1", DROP_ACTIVE_CLASSES)}>
      <div className="flex items-center gap-2">
        <h2 className="text-sm font-semibold">Evidence</h2>
        <AttachFile ip={ip} label="Add evidence" inputLabel="Evidence file" isTextVisible />
      </div>
      <QueryStates
        query={query}
        isEmpty={(v) => (v.items ?? []).length === 0 && (v.notIndexed ?? []).length === 0}
        empty={
          <p className="text-sm text-muted-foreground">
            No evidence yet. Drop or paste a screenshot or text here, or use Add evidence.
          </p>
        }
      >
        {(v) => (
          <>
            <ul className="space-y-2">
              {(v.items ?? []).map((i) => (
                <EvidenceItemRow key={i.id} ip={ip} item={i} />
              ))}
            </ul>
            {(v.notIndexed ?? []).length > 0 && (
              <section aria-label="Not indexed" className="space-y-1">
                <h3 className="text-sm font-semibold">Not indexed</h3>
                <ul className="space-y-1">
                  {(v.notIndexed ?? []).map((name) => (
                    <li key={name} className="flex items-center gap-2 text-sm">
                      <span className="font-mono text-xs">{name}</span>
                      <span className="text-xs text-muted-foreground">in the folder, not indexed</span>
                      <Button size="xs" variant="outline" aria-label={`Adopt ${name}`} onClick={() => adopt.mutate(name)}>
                        Adopt
                      </Button>
                    </li>
                  ))}
                </ul>
              </section>
            )}
          </>
        )}
      </QueryStates>
    </section>
  );
}
