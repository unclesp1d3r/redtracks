import { useState } from "react";
import { FileXIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { EvidenceKind } from "@/lib/api";
import type { InboxFile } from "@/lib/api";
import { AssignDialog } from "./AssignDialog";
import { useDiscard } from "./useInbox";

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

function Preview({ file }: { file: InboxFile }) {
  if (file.Kind === EvidenceKind.EvidenceKindImage) {
    return (
      <img
        src={`/inbox/${encodeURIComponent(file.Name)}`}
        alt={`Preview of ${file.Name}`}
        loading="lazy"
        className="h-24 w-32 shrink-0 rounded border object-cover"
      />
    );
  }
  if (file.Kind === EvidenceKind.EvidenceKindText) {
    return (
      <pre className="h-24 w-64 shrink-0 overflow-hidden rounded border bg-muted p-1 font-mono text-xs whitespace-pre-wrap">
        {file.Preview}
      </pre>
    );
  }
  return (
    <div className="flex h-24 w-32 shrink-0 items-center justify-center gap-1 rounded border text-xs text-muted-foreground">
      <FileXIcon className="size-4" aria-hidden /> not supported
    </div>
  );
}

export function InboxRow({ file, token }: { file: InboxFile; token: string }) {
  const [assigning, setAssigning] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const discard = useDiscard();
  const supported = file.Kind !== EvidenceKind.$zero;

  return (
    <li aria-label={file.Name} className="flex items-start gap-3 rounded-md border p-2">
      <Preview file={file} />
      <div className="min-w-0 flex-1 space-y-1 text-sm">
        <div className="truncate font-medium">{file.Name}</div>
        <div className="text-xs text-muted-foreground">
          {formatBytes(file.Size)} · {new Date(file.ModTime).toLocaleString()}
        </div>
        <div className="flex gap-2">
          <Button size="xs" disabled={!supported} onClick={() => setAssigning(true)} aria-label={`Assign ${file.Name}`}>
            Assign
          </Button>
          {confirming ? (
            <>
              <Button
                size="xs"
                variant="destructive"
                onClick={() => discard.mutate(file.Name, { onSettled: () => setConfirming(false) })}
              >
                Confirm discard {file.Name}
              </Button>
              <Button size="xs" variant="ghost" onClick={() => setConfirming(false)}>
                Cancel
              </Button>
            </>
          ) : (
            <Button size="xs" variant="ghost" aria-label={`Discard ${file.Name}`} onClick={() => setConfirming(true)}>
              Discard
            </Button>
          )}
        </div>
      </div>
      {assigning && <AssignDialog file={file} token={token} open onOpenChange={setAssigning} />}
    </li>
  );
}
