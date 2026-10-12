import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Events } from "@wailsio/runtime";
import { QueryStates } from "@/components/QueryStates";
import { MarkdownView } from "@/components/MarkdownView";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { EngagementService, errorMessage } from "@/lib/api";
import type { Notes } from "@/lib/api";
import { keys } from "@/lib/query";

const CONFLICT_PREFIX = "conflict";

/** Loads the notes once; NotesBody then owns refresh through engagement:changed. */
export function NotesEditor({ ip }: { ip: string }) {
  const query = useQuery({
    queryKey: keys.notes(ip),
    queryFn: () => EngagementService.ReadNotes(ip),
    staleTime: Infinity,
    gcTime: 0,
  });
  return (
    <QueryStates query={query} isEmpty={() => false} empty={null}>
      {(initial) => <NotesBody key={ip} ip={ip} initial={initial} />}
    </QueryStates>
  );
}

interface Body {
  ip: string;
  initial: Notes;
}

function NotesBody({ ip, initial }: Body) {
  const [text, setText] = useState(initial.Text);
  const [baseHash, setBaseHash] = useState(initial.Hash);
  const [isDirty, setIsDirty] = useState(false);
  const [diskChange, setDiskChange] = useState<Notes | null>(null);
  const [isConfirmingReload, setIsConfirmingReload] = useState(false);
  const [isConflict, setIsConflict] = useState(false);
  const [isPreview, setIsPreview] = useState(false);
  const [isOpen, setIsOpen] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // The event handler must see current values without resubscribing on every keystroke.
  const live = useRef({ isDirty, baseHash, text });
  live.current = { isDirty, baseHash, text };

  function adopt(n: Notes) {
    setText(n.Text);
    setBaseHash(n.Hash);
    setIsDirty(false);
    setDiskChange(null);
    setIsConfirmingReload(false);
    setIsConflict(false);
  }

  // Rapid change events overlap reads; only the latest request may apply its result.
  const latestRead = useRef(0);

  useEffect(() => {
    const off = Events.On("engagement:changed", () => {
      const requestId = ++latestRead.current;
      EngagementService.ReadNotes(ip)
        .then((n) => {
          if (requestId !== latestRead.current) return;
          const { isDirty: dirty, baseHash: base } = live.current;
          if (n.Hash === base) return;
          if (dirty) setDiskChange(n);
          else adopt(n);
        })
        .catch((e: unknown) => {
          if (requestId === latestRead.current) setError(errorMessage(e));
        });
    });
    return off;
  }, [ip]);

  const save = useMutation({
    meta: { silent: true },
    mutationFn: async (overwrite: boolean) => {
      const savedText = text;
      const hash = overwrite ? (await EngagementService.ReadNotes(ip)).Hash : baseHash;
      await EngagementService.WriteNotes(ip, savedText, hash);
      // ponytail: a write landing between WriteNotes and this read is adopted as ours; the next change event still reconciles.
      return { hash: (await EngagementService.ReadNotes(ip)).Hash, savedText };
    },
    onSuccess: ({ hash, savedText }) => {
      setBaseHash(hash);
      // Keystrokes typed while the save was in flight stay dirty.
      setIsDirty(live.current.text !== savedText);
      setDiskChange(null);
      setIsConflict(false);
      setError(null);
    },
    onError: (e) => {
      const msg = errorMessage(e);
      if (msg.startsWith(CONFLICT_PREFIX)) setIsConflict(true);
      else setError(msg);
    },
  });

  async function reloadFromDisk() {
    try {
      adopt(await EngagementService.ReadNotes(ip));
    } catch (e) {
      setError(errorMessage(e));
    }
  }

  return (
    <section aria-label="Notes" className="space-y-2">
      <div className="flex items-center gap-2">
        <h2 className="text-sm font-semibold">Notes</h2>
        <Button size="xs" variant="ghost" aria-expanded={isOpen} onClick={() => setIsOpen(!isOpen)}>
          {isOpen ? "Collapse" : "Expand"}
        </Button>
      </div>
      <div hidden={!isOpen} className="space-y-2">
        {diskChange && (
          <div role="status" className="flex flex-wrap items-center gap-2 rounded-md border border-accent p-2 text-sm">
            <span>notes.md changed on disk.</span>
            {isConfirmingReload ? (
              <>
                <span>Discard your local edits?</span>
                <Button size="xs" variant="destructive" onClick={() => adopt(diskChange)}>
                  Confirm reload
                </Button>
                <Button size="xs" variant="ghost" onClick={() => setIsConfirmingReload(false)}>
                  Cancel
                </Button>
              </>
            ) : (
              <>
                <Button size="xs" variant="outline" onClick={() => setIsConfirmingReload(true)}>
                  Reload
                </Button>
                <Button size="xs" variant="ghost" onClick={() => setDiskChange(null)}>
                  Keep editing
                </Button>
              </>
            )}
          </div>
        )}
        {isPreview ? (
          <MarkdownView text={text} />
        ) : (
          <Textarea
            aria-label="Notes text"
            className="min-h-40 font-mono"
            value={text}
            onChange={(e) => {
              setText(e.target.value);
              setIsDirty(true);
            }}
          />
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={() => save.mutate(false)} disabled={!isDirty || save.isPending}>
            Save
          </Button>
          <Button size="sm" variant="outline" onClick={() => setIsPreview(!isPreview)}>
            {isPreview ? "Edit" : "Preview"}
          </Button>
          {isDirty && <span className="text-xs text-muted-foreground">Unsaved changes</span>}
        </div>
      </div>
      <Dialog open={isConflict} onOpenChange={setIsConflict}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Notes changed on disk</DialogTitle>
            <DialogDescription>notes.md was modified since you loaded it. Reload discards your edits; Overwrite replaces the file with your text.</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => void reloadFromDisk()}>
              Reload
            </Button>
            <Button variant="destructive" onClick={() => save.mutate(true)}>
              Overwrite
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
