import { useEffect, useState } from "react";
import { TriangleAlertIcon } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { EvidenceKind, errorMessage } from "@/lib/api";
import type { EvidenceItemView } from "@/lib/api";
import { evidenceUrl } from "./evidenceUrl";
import { itemName } from "./proof";

interface Props {
  ip: string;
  item: EvidenceItemView;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MissingNotice() {
  return (
    <p className="flex items-center gap-1 text-sm text-amber-600 dark:text-amber-400">
      <TriangleAlertIcon className="size-4" aria-hidden />
      <span>file missing</span>
    </p>
  );
}

type TextState = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; text: string };

/** Fetches the text file from the evidence route; rendered by the caller as plain text only. */
function useTextContent(url: string, enabled: boolean): TextState {
  const [state, setState] = useState<TextState>({ status: "loading" });
  useEffect(() => {
    if (!enabled) return;
    let isCurrent = true;
    fetch(url)
      .then((r) => (r.ok ? r.text() : Promise.reject(new Error(`could not load file (HTTP ${r.status})`))))
      .then((text) => isCurrent && setState({ status: "ready", text }))
      .catch((e: unknown) => isCurrent && setState({ status: "error", message: errorMessage(e) }));
    return () => {
      isCurrent = false;
    };
  }, [url, enabled]);
  return state;
}

function TextBody({ url }: { url: string }) {
  const state = useTextContent(url, true);
  if (state.status === "loading") return <p className="text-sm text-muted-foreground">Loading...</p>;
  if (state.status === "error") {
    return (
      <p role="alert" className="font-mono text-xs text-destructive">
        {state.message}
      </p>
    );
  }
  return <pre className="max-h-[70vh] overflow-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap">{state.text}</pre>;
}

export function EvidenceViewer({ ip, item, open, onOpenChange }: Props) {
  const url = evidenceUrl(ip, item.file);
  const captured = new Date(item.capturedAt).toLocaleString();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{itemName(item)}</DialogTitle>
        </DialogHeader>
        {item.missing ? (
          <MissingNotice />
        ) : item.kind === EvidenceKind.EvidenceKindText ? (
          <TextBody url={url} />
        ) : (
          <img
            src={url}
            loading="lazy"
            alt={item.caption || `${item.kind} captured ${captured}`}
            className="max-h-[70vh] w-auto max-w-full object-contain"
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
