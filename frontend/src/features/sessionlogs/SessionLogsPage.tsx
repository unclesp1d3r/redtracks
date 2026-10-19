import { useState } from "react";
import { FileTextIcon } from "lucide-react";
import { QueryStates } from "@/components/QueryStates";
import type { SessionLogSource } from "@/lib/api";
import { cn } from "@/lib/utils";
import { ImportControl } from "./ImportControl";
import { LogViewer } from "./LogViewer";
import { humanSize } from "@/lib/format";
import { useSessionLogsQuery } from "./useSessionLog";

function LogRow({ log, isSelected, onOpen }: { log: SessionLogSource; isSelected: boolean; onOpen: () => void }) {
  return (
    <button
      type="button"
      aria-label={`Open ${log.Name}`}
      aria-current={isSelected}
      onClick={onOpen}
      className={cn(
        "w-full rounded-md border px-3 py-2 text-left text-sm hover:bg-muted",
        isSelected && "border-primary bg-muted",
      )}
    >
      <span className="flex items-center gap-1 font-medium">
        <FileTextIcon className="size-4 shrink-0" aria-hidden />
        <span className="truncate">{log.OriginalName && log.OriginalName !== log.Name ? log.OriginalName : log.Name}</span>
        {isSelected && <span className="text-xs text-muted-foreground">(open)</span>}
      </span>
      <span className="block text-xs text-muted-foreground">
        {humanSize(log.Size)} · {log.Lines} lines · imported {new Date(log.ImportedAt).toLocaleString()}
      </span>
    </button>
  );
}

export function SessionLogsPage() {
  const logs = useSessionLogsQuery();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const selected = logs.data?.find((l) => l.ID === selectedId);

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Session logs</h1>
      <ImportControl onImported={(s) => setSelectedId(s.ID)} />
      <QueryStates
        query={logs}
        isEmpty={(d) => d.length === 0}
        empty={
          <p className="text-sm text-muted-foreground">
            No session logs yet. Import one recorded with script or tmux, then clip the lines you need to a host as evidence.
          </p>
        }
      >
        {(data) => (
          <div className="grid gap-4 md:grid-cols-[18rem_1fr]">
            <ul aria-label="Imported logs" className="space-y-2">
              {data.map((l) => (
                <li key={l.ID}>
                  <LogRow log={l} isSelected={l.ID === selectedId} onOpen={() => setSelectedId(l.ID)} />
                </li>
              ))}
            </ul>
            {selected && <LogViewer key={selected.ID} log={selected} />}
          </div>
        )}
      </QueryStates>
    </div>
  );
}
