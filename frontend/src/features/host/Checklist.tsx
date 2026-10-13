import { useState, type KeyboardEvent } from "react";
import { ChevronDownIcon, ChevronRightIcon, CopyIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Textarea } from "@/components/ui/textarea";
import { ChecklistService, ItemStatus, errorMessage } from "@/lib/api";
import type { ChecklistView, ItemView } from "@/lib/api";
import { cn } from "@/lib/utils";

type OnSet = (itemId: string, status: ItemStatus, note: string) => void;

interface Props {
  views: ChecklistView[];
  onSet: OnSet;
  targetIp: string;
  domain: string;
}

const IGNORED_FOCUS = "input,textarea,button,select,[role=menuitem]";

export function Checklist({ views, onSet, targetIp, domain }: Props) {
  return (
    <div className="space-y-4">
      {views.map((v) => (
        <section key={v.Definition.id} aria-label={v.Definition.title}>
          <h3 className="mb-1 text-sm font-semibold">{v.Definition.title}</h3>
          <ul className="divide-y rounded-md border">
            {(v.Items ?? []).map((item) => (
              <ChecklistRow key={item.id} item={item} onSet={onSet} targetIp={targetIp} domain={domain} />
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

interface RowProps {
  item: ItemView;
  onSet: OnSet;
  targetIp: string;
  domain: string;
}

function ChecklistRow({ item, onSet, targetIp, domain }: RowProps) {
  const status = item.State.status || ItemStatus.ItemTodo;
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState(item.State.note);
  // Follow an external note edit unless the user has typed something here.
  const [syncedNote, setSyncedNote] = useState(item.State.note);
  if (syncedNote !== item.State.note) {
    setSyncedNote(item.State.note);
    if (note === syncedNote) setNote(item.State.note);
  }
  const isStruck = status === ItemStatus.ItemNA;

  function toggle(e: KeyboardEvent<HTMLLIElement>) {
    if (e.key !== " ") return;
    if ((e.target as HTMLElement).closest(IGNORED_FOCUS)) return;
    e.preventDefault();
    onSet(item.id, status === ItemStatus.ItemDone ? ItemStatus.ItemTodo : ItemStatus.ItemDone, note);
  }

  function setSpecial(target: ItemStatus) {
    onSet(item.id, status === target ? ItemStatus.ItemTodo : target, note);
  }

  const commands = item.commands ?? [];
  return (
    <li tabIndex={0} onKeyDown={toggle} data-testid={`row-${item.id}`} className="p-2 outline-none focus-visible:ring-2 focus-visible:ring-ring">
      <div className="flex items-center gap-2">
        <Checkbox
          aria-label={`Done: ${item.title}`}
          checked={status === ItemStatus.ItemDone}
          onCheckedChange={(c) => onSet(item.id, c ? ItemStatus.ItemDone : ItemStatus.ItemTodo, note)}
        />
        <span className={cn("flex-1 text-sm", isStruck && "line-through text-muted-foreground")}>{item.title}</span>
        {(status === ItemStatus.ItemSkipped || status === ItemStatus.ItemNA) && (
          <Badge variant="outline">{status === ItemStatus.ItemNA ? "n/a" : "skipped"}</Badge>
        )}
        <Button size="xs" variant="ghost" aria-pressed={status === ItemStatus.ItemSkipped} onClick={() => setSpecial(ItemStatus.ItemSkipped)}>
          Skip
        </Button>
        <Button size="xs" variant="ghost" aria-pressed={isStruck} onClick={() => setSpecial(ItemStatus.ItemNA)}>
          N/A
        </Button>
        <Button
          size="icon-xs"
          variant="ghost"
          aria-expanded={open}
          aria-label={open ? `Collapse ${item.title}` : `Expand ${item.title}`}
          onClick={() => setOpen(!open)}
        >
          {open ? <ChevronDownIcon /> : <ChevronRightIcon />}
        </Button>
      </div>
      {open && (
        <div className="mt-2 space-y-2 pl-6">
          {item.hint && <p className="text-sm text-muted-foreground">{item.hint}</p>}
          {commands.map((cmd) => (
            <CommandLine key={cmd} cmd={cmd} targetIp={targetIp} domain={domain} />
          ))}
          <Textarea
            aria-label={`Note for ${item.title}`}
            placeholder="Note"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            onBlur={() => note !== item.State.note && onSet(item.id, status, note)}
          />
        </div>
      )}
    </li>
  );
}

function CommandLine({ cmd, targetIp, domain }: { cmd: string; targetIp: string; domain: string }) {
  const [error, setError] = useState<string | null>(null);
  async function copy() {
    try {
      const resolved = await ChecklistService.Command(cmd, targetIp, domain);
      await navigator.clipboard.writeText(resolved);
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    }
  }
  return (
    <div>
      <div className="flex items-start gap-2">
        <pre className="flex-1 overflow-auto rounded-md bg-muted p-1.5 font-mono text-xs">{cmd}</pre>
        <Button size="icon-xs" variant="outline" aria-label={`Copy command ${cmd}`} onClick={() => void copy()}>
          <CopyIcon />
        </Button>
      </div>
      {error && (
        <p role="alert" className="mt-1 text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
