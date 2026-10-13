import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { EngagementService, EngagementType, errorMessage } from "@/lib/api";

const TYPES = [
  EngagementType.EngagementLab,
  EngagementType.EngagementOSCP,
  EngagementType.EngagementOSAI,
  EngagementType.EngagementCustom,
];

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (dir: string) => void;
}

export function CreateEngagementDialog({ open, onOpenChange, onCreated }: Props) {
  const [name, setName] = useState("");
  const [type, setType] = useState<EngagementType>(EngagementType.EngagementLab);
  const [domain, setDomain] = useState("");
  const [error, setError] = useState<string | null>(null);

  const create = useMutation({
    meta: { silent: true },
    mutationFn: async () => {
      const dir = await EngagementService.Create(name.trim(), domain.trim(), type);
      await EngagementService.Open(dir);
      return dir;
    },
    onSuccess: onCreated,
    onError: (e) => setError(errorMessage(e)),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    if (name.trim() === "") {
      setError("Name is required.");
      return;
    }
    setError(null);
    create.mutate();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={submit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>New engagement</DialogTitle>
          </DialogHeader>
          <label className="block space-y-1 text-sm">
            <span>Name</span>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="pg-test" autoFocus />
          </label>
          <div className="space-y-1 text-sm">
            <span id="eng-type-label">Type</span>
            <Select value={type} onValueChange={(v) => v && setType(v as EngagementType)}>
              <SelectTrigger aria-labelledby="eng-type-label" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {TYPES.map((t) => (
                  <SelectItem key={t} value={t}>
                    {t}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <label className="block space-y-1 text-sm">
            <span>Domain (optional)</span>
            <Input value={domain} onChange={(e) => setDomain(e.target.value)} placeholder="corp.local" />
          </label>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={create.isPending}>
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
