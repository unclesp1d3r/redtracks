import { useState } from "react";
import { Link } from "react-router";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { TargetPicker } from "@/features/evidence/TargetPicker";
import { hostPath } from "@/lib/api";
import type { SessionLogSource } from "@/lib/api";
import { MAX_CLIP_LINES, lineCount, type LineRange } from "./range";
import { useClip } from "./useSessionLog";

interface Props extends LineRange {
  log: SessionLogSource;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ClipDialog({ log, from, to, open, onOpenChange }: Props) {
  const [ip, setIp] = useState("");
  const clip = useClip(log.ID);
  const count = lineCount({ from, to });
  const overCap = count > MAX_CLIP_LINES;
  const saved = clip.isSuccess ? clip.variables.ip : null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Clip to evidence</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          {log.Name} lines {from + 1}-{to + 1} ({count} lines)
        </p>
        {overCap && (
          <p role="alert" className="text-sm text-destructive">
            A clip holds at most {MAX_CLIP_LINES.toLocaleString("en-US")} lines.
          </p>
        )}
        {saved === null ? (
          <TargetPicker onChange={(t) => setIp(t.ip)} />
        ) : (
          <p role="status" className="text-sm">
            Saved to {saved}.{" "}
            <Link className="underline" to={hostPath(saved)} onClick={() => onOpenChange(false)}>
              Open host {saved} (Evidence tab)
            </Link>
          </p>
        )}
        {clip.isError && (
          <p role="alert" className="text-sm text-destructive">
            {clip.error.message}
          </p>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {saved === null ? "Cancel" : "Close"}
          </Button>
          {saved === null && (
            <Button disabled={ip === "" || overCap || clip.isPending} onClick={() => clip.mutate({ ip, from, to })}>
              {clip.isPending ? "Saving..." : "Save clip"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
