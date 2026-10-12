import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { PlusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { HostService, errorMessage } from "@/lib/api";
import type { AccessPath, Credential, Host } from "@/lib/api";

const METHODS = ["direct", "chisel", "ssh-tunnel", "rdp", "winrm", "other"];
const NONE = "none";

interface Props {
  host: Host;
  otherHosts: Host[];
  credentials: Credential[];
}

export function AccessPaths({ host, otherHosts, credentials }: Props) {
  const [adding, setAdding] = useState(false);
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const remove = useMutation({
    mutationFn: (id: string) => HostService.RemoveAccessPath(host.ip, id),
    onSettled: () => setConfirmId(null),
  });
  const paths = host.accessPaths ?? [];
  const hostLabel = (id: string) => otherHosts.find((h) => h.id === id)?.ip ?? id;
  const credLabel = (id: string) => credentials.find((c) => c.id === id)?.username ?? id;

  return (
    <section aria-label="Access paths" className="space-y-1">
      <div className="flex items-center gap-2">
        <h2 className="text-sm font-semibold">Access paths</h2>
        <Button size="xs" variant="outline" onClick={() => setAdding(true)}>
          <PlusIcon /> Add path
        </Button>
      </div>
      {paths.length === 0 ? (
        <p className="text-sm text-muted-foreground">No access path recorded.</p>
      ) : (
        <ul className="space-y-1">
          {paths.map((p) => (
            <li key={p.id} className="flex flex-wrap items-center gap-2 text-sm">
              <span className="font-medium">{p.method}</span>
              {p.viaHostId && <span>via {hostLabel(p.viaHostId)}</span>}
              {p.credentialId && <span>as {credLabel(p.credentialId)}</span>}
              {p.command && <code className="rounded bg-muted px-1 font-mono text-xs">{p.command}</code>}
              {confirmId === p.id ? (
                <>
                  <Button size="xs" variant="destructive" onClick={() => remove.mutate(p.id)}>
                    Confirm remove
                  </Button>
                  <Button size="xs" variant="ghost" onClick={() => setConfirmId(null)}>
                    Cancel
                  </Button>
                </>
              ) : (
                <Button size="xs" variant="ghost" onClick={() => setConfirmId(p.id)}>
                  Remove
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
      <AddPathDialog ip={host.ip} open={adding} onOpenChange={setAdding} otherHosts={otherHosts} credentials={credentials} />
    </section>
  );
}

interface DialogProps {
  ip: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  otherHosts: Host[];
  credentials: Credential[];
}

function AddPathDialog({ ip, open, onOpenChange, otherHosts, credentials }: DialogProps) {
  const [method, setMethod] = useState("direct");
  const [via, setVia] = useState(NONE);
  const [cred, setCred] = useState(NONE);
  const [command, setCommand] = useState("");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState<string | null>(null);

  const add = useMutation({
    meta: { silent: true },
    mutationFn: () => {
      const p: AccessPath = {
        id: "",
        method,
        viaHostId: via === NONE ? "" : via,
        credentialId: cred === NONE ? "" : cred,
        command,
        notes,
      };
      return HostService.AddAccessPath(ip, p);
    },
    onSuccess: () => {
      setCommand("");
      setNotes("");
      setError(null);
      onOpenChange(false);
    },
    onError: (e) => setError(errorMessage(e)),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add access path</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 text-sm">
          <Field label="Method">
            <Select value={method} onValueChange={(v) => v && setMethod(v)}>
              <SelectTrigger aria-label="Method">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {METHODS.map((m) => (
                  <SelectItem key={m} value={m}>
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field label="Via host">
            <Select value={via} onValueChange={(v) => v && setVia(v)}>
              <SelectTrigger aria-label="Via host">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE}>none</SelectItem>
                {otherHosts.map((h) => (
                  <SelectItem key={h.id} value={h.id}>
                    {h.ip} {h.hostname}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field label="Credential">
            <Select value={cred} onValueChange={(v) => v && setCred(v)}>
              <SelectTrigger aria-label="Credential">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE}>none</SelectItem>
                {credentials.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.domain ? `${c.domain}\\${c.username}` : c.username}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field label="Command">
            <Input value={command} onChange={(e) => setCommand(e.target.value)} />
          </Field>
          <Field label="Notes">
            <Input value={notes} onChange={(e) => setNotes(e.target.value)} />
          </Field>
          {error && (
            <p role="alert" className="text-destructive">
              {error}
            </p>
          )}
        </div>
        <DialogFooter>
          <Button onClick={() => add.mutate()} disabled={add.isPending}>
            Add path
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1">
      <span>{label}</span>
      {children}
    </label>
  );
}
