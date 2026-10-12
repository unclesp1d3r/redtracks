import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { HostService, errorMessage } from "@/lib/api";

const PROTOCOLS = ["tcp", "udp"];
const MAX_PORT = 65535;
const TYPE_SUGGESTIONS = [
  "http", "https", "ssh", "smb", "ftp", "mysql", "mssql", "rdp",
  "llm-chat", "mcp-server", "agent", "rag-api", "vector-store", "model-registry", "inference-api",
];

interface Props {
  ip: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddServiceDialog({ ip, open, onOpenChange }: Props) {
  const [port, setPort] = useState("");
  const [protocol, setProtocol] = useState("tcp");
  const [type, setType] = useState("");
  const [product, setProduct] = useState("");
  const [version, setVersion] = useState("");
  const [error, setError] = useState<string | null>(null);

  const add = useMutation({
    meta: { silent: true },
    mutationFn: () =>
      HostService.AddService(ip, {
        id: "", port: Number(port), protocol, type: type.trim(), product, version,
        notes: "", webPaths: [], checklist: { items: {} },
      }),
    onSuccess: () => {
      setPort(""); setType(""); setProduct(""); setVersion(""); setError(null);
      onOpenChange(false);
    },
    onError: (e) => setError(errorMessage(e)),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    const n = Number(port);
    if (!Number.isInteger(n) || n < 1 || n > MAX_PORT) return setError(`Port must be 1 to ${MAX_PORT}.`);
    if (type.trim() === "") return setError("Enter a service type.");
    setError(null);
    add.mutate();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={submit} className="space-y-3 text-sm">
          <DialogHeader>
            <DialogTitle>Add service</DialogTitle>
          </DialogHeader>
          <label className="block space-y-1">
            <span>Port</span>
            <Input inputMode="numeric" value={port} onChange={(e) => setPort(e.target.value)} autoFocus />
          </label>
          <div className="space-y-1">
            <span>Protocol</span>
            <Select value={protocol} onValueChange={(v) => v && setProtocol(v)}>
              <SelectTrigger aria-label="Protocol">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PROTOCOLS.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <label className="block space-y-1">
            <span>Type</span>
            <Input list="service-type-suggestions" value={type} onChange={(e) => setType(e.target.value)} />
            <datalist id="service-type-suggestions">
              {TYPE_SUGGESTIONS.map((t) => (
                <option key={t} value={t} />
              ))}
            </datalist>
          </label>
          <label className="block space-y-1">
            <span>Product</span>
            <Input value={product} onChange={(e) => setProduct(e.target.value)} />
          </label>
          <label className="block space-y-1">
            <span>Version</span>
            <Input value={version} onChange={(e) => setVersion(e.target.value)} />
          </label>
          {error && (
            <p role="alert" className="text-destructive">
              {error}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={add.isPending}>
              Add service
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
