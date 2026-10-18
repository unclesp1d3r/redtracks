import { useState } from "react";
import { TriangleAlertIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { errorMessage } from "@/lib/api";
import type { InboxFile } from "@/lib/api";
import { TargetPicker, type PickedTarget } from "@/features/evidence/TargetPicker";
import { useAssign, useInboxQuery } from "./useInbox";

interface Props {
  file: InboxFile;
  token: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AssignDialog({ file, token, open, onOpenChange }: Props) {
  const [target, setTarget] = useState<PickedTarget>({ ip: "", links: [] });
  const [caption, setCaption] = useState("");
  const assign = useAssign();
  const inbox = useInboxQuery();

  const submit = () =>
    assign.mutate(
      { token, name: file.Name, ip: target.ip, caption, links: target.links },
      { onSuccess: () => onOpenChange(false) },
    );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Assign {file.Name}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 text-sm">
          <TargetPicker onChange={setTarget} />
          <div className="flex items-center gap-2">
            <Input
              aria-label="Caption"
              placeholder={file.Name}
              value={caption}
              onChange={(e) => setCaption(e.target.value)}
            />
            <Button size="sm" variant="outline" onClick={() => setCaption(file.Name)}>
              Use file name
            </Button>
          </div>
          {assign.isError && (
            <div role="alert" className="flex flex-wrap items-center gap-2 text-destructive">
              <TriangleAlertIcon className="size-4" aria-hidden />
              <span>{errorMessage(assign.error)}</span>
              <Button
                size="xs"
                variant="outline"
                onClick={() => {
                  assign.reset();
                  void inbox.refetch();
                }}
              >
                Reload
              </Button>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={target.ip === "" || assign.isPending} onClick={submit}>
            Assign to host
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
