import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { SessionLogSource } from "@/lib/api";
import { useCapabilities } from "@/features/evidence/useEvidence";
import { useImportLog } from "./useSessionLog";

/** Native file picker on the desktop build, an absolute-path field on the server build. */
export function ImportControl({ onImported }: { onImported: (source: SessionLogSource) => void }) {
  const hasDialog = useCapabilities().data?.NativeDialog === true;
  const [path, setPath] = useState("");
  const imp = useImportLog(onImported);
  const busy = imp.isPending;

  function submit(e: FormEvent) {
    e.preventDefault();
    if (path.trim() !== "") imp.mutate(path.trim());
  }

  return (
    <div className="space-y-2" aria-busy={busy}>
      {hasDialog ? (
        <Button onClick={() => imp.mutate(undefined)} disabled={busy}>
          {busy ? "Importing..." : "Choose a log file"}
        </Button>
      ) : (
        <form onSubmit={submit} className="flex gap-2">
          <Input
            aria-label="Path to a session log"
            placeholder="/absolute/path/to/session.log"
            value={path}
            onChange={(e) => setPath(e.target.value)}
            disabled={busy}
          />
          <Button type="submit" disabled={busy || path.trim() === ""}>
            {busy ? "Importing..." : "Import"}
          </Button>
        </form>
      )}
      {imp.isError && (
        <p role="alert" className="text-sm text-destructive">
          {imp.error.message}
        </p>
      )}
    </div>
  );
}
