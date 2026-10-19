import { useEffect, useRef, useState } from "react";
import { EyeIcon, FileTextIcon, FolderOpenIcon, ImageIcon, LinkIcon, CopyIcon, XIcon } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { EvidenceKind, EvidenceService, errorMessage } from "@/lib/api";
import type { EvidenceItemView } from "@/lib/api";
import { captionFocus, usePendingCaption } from "@/lib/capture";
import { EvidenceViewer, MissingNotice } from "./EvidenceViewer";
import { TargetPicker, type PickedTarget } from "./TargetPicker";
import { linkOf } from "./proof";
import { useCapabilities, useEvidenceActions } from "./useEvidence";

interface Props {
  ip: string;
  item: EvidenceItemView;
}

function CaptionField({ ip, item }: Props) {
  const { setCaption } = useEvidenceActions(ip);
  // ui/Input does not forward refs on React 18, so the field is found through its wrapper.
  const wrap = useRef<HTMLDivElement>(null);
  const pending = usePendingCaption();
  const [value, setValue] = useState(item.caption);
  const [syncedCaption, setSyncedCaption] = useState(item.caption);
  // Follow an external caption change unless the user has typed something here.
  if (syncedCaption !== item.caption) {
    setSyncedCaption(item.caption);
    if (value === syncedCaption) setValue(item.caption);
  }
  useEffect(() => {
    if (pending?.id !== item.id) return;
    wrap.current?.querySelector("input")?.focus();
    captionFocus.clear();
  }, [pending?.id, item.id]);

  const commit = () => value !== item.caption && setCaption.mutate({ itemId: item.id, caption: value });
  return (
    <div ref={wrap}>
    <Input
      aria-label={`Caption for ${item.file}`}
      placeholder="Caption"
      className="h-7 max-w-md"
      value={value}
      onChange={(e) => setValue(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => e.key === "Enter" && commit()}
    />
    </div>
  );
}

function FileActions({ ip, item }: Props) {
  const { data: caps } = useCapabilities();
  const { drop } = useEvidenceActions(ip);
  const [isConfirming, setIsConfirming] = useState(false);

  async function copyPath() {
    try {
      const path = await EvidenceService.CopyPath(ip, item.id);
      if (!navigator.clipboard) {
        toast.message(`Clipboard unavailable. Path: ${path}`);
        return;
      }
      await navigator.clipboard.writeText(path);
      toast.success("Path copied");
    } catch (e) {
      toast.error(errorMessage(e));
    }
  }

  return (
    <>
      {caps?.Reveal ? (
        <Button size="xs" variant="ghost" aria-label={`Reveal ${item.file}`} onClick={() => void EvidenceService.Reveal(ip, item.id).catch((e: unknown) => toast.error(errorMessage(e)))}>
          <FolderOpenIcon aria-hidden /> Reveal
        </Button>
      ) : (
        <Button size="xs" variant="ghost" aria-label={`Copy path of ${item.file}`} onClick={() => void copyPath()}>
          <CopyIcon aria-hidden /> Copy path
        </Button>
      )}
      {isConfirming ? (
        <>
          <Button size="xs" variant="destructive" aria-label={`Confirm drop ${item.file}`} onClick={() => drop.mutate(item.id, { onSettled: () => setIsConfirming(false) })}>
            Confirm drop
          </Button>
          <Button size="xs" variant="ghost" onClick={() => setIsConfirming(false)}>
            Cancel
          </Button>
        </>
      ) : (
        <Button size="xs" variant="ghost" aria-label={`Drop ${item.file}`} onClick={() => setIsConfirming(true)}>
          Drop
        </Button>
      )}
    </>
  );
}

function AddLinkDialog({ ip, item, open, onOpenChange }: Props & { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { link } = useEvidenceActions(ip);
  const [picked, setPicked] = useState<PickedTarget>({ ip, links: [] });
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Add link to {item.file}</DialogTitle>
        </DialogHeader>
        <TargetPicker ip={ip} onChange={setPicked} />
        <DialogFooter>
          <Button
            disabled={picked.links.length === 0}
            onClick={() => {
              picked.links.forEach((l) => link.mutate({ itemId: item.id, link: l }));
              onOpenChange(false);
            }}
          >
            Add links
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function LinkChips({ ip, item }: Props) {
  const { unlink } = useEvidenceActions(ip);
  return (
    <ul aria-label="Links" className="flex flex-wrap gap-1.5">
      {(item.links ?? []).map((l) => (
        <li key={`${l.kind}:${l.targetId}:${l.serviceId ?? ""}`} className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs">
          <span className="text-muted-foreground">{l.kind}</span>
          <span>{l.label}</span>
          {l.removed && <Badge variant="destructive">removed</Badge>}
          <Button size="icon-xs" variant="ghost" aria-label={`Unlink ${l.label}`} onClick={() => unlink.mutate({ itemId: item.id, link: linkOf(l) })}>
            <XIcon aria-hidden />
          </Button>
        </li>
      ))}
    </ul>
  );
}

/** One evidence item: kind, time, caption, links and file actions. Shared by the host tab and the browse views. */
export function EvidenceItemRow({ ip, item }: Props) {
  const [isViewing, setIsViewing] = useState(false);
  const [isLinking, setIsLinking] = useState(false);
  const KindIcon = item.kind === EvidenceKind.EvidenceKindText ? FileTextIcon : ImageIcon;
  return (
    <li data-testid={`evidence-${item.id}`} className="space-y-1.5 rounded-md border p-2 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="outline">
          <KindIcon aria-hidden />
          {item.kind}
        </Badge>
        <time dateTime={item.capturedAt} className="text-xs text-muted-foreground">
          {new Date(item.capturedAt).toLocaleString()}
        </time>
        <span className="font-mono text-xs text-muted-foreground">{item.file}</span>
        {item.missing && <MissingNotice />}
        <span className="ml-auto flex flex-wrap items-center gap-1">
          <Button size="xs" variant="ghost" aria-label={`View ${item.file}`} onClick={() => setIsViewing(true)}>
            <EyeIcon aria-hidden /> View
          </Button>
          <Button size="xs" variant="ghost" aria-label={`Add link to ${item.file}`} onClick={() => setIsLinking(true)}>
            <LinkIcon aria-hidden /> Add link
          </Button>
          <FileActions ip={ip} item={item} />
        </span>
      </div>
      <CaptionField ip={ip} item={item} />
      <LinkChips ip={ip} item={item} />
      {isViewing && <EvidenceViewer ip={ip} item={item} open onOpenChange={setIsViewing} />}
      {isLinking && <AddLinkDialog ip={ip} item={item} open onOpenChange={setIsLinking} />}
    </li>
  );
}
