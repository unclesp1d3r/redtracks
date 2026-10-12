import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { HostService, errorMessage } from "@/lib/api";

const IPV4 = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;
const IPV6 = /^[0-9a-f:.]+$/i;

export function isValidIp(s: string): boolean {
  return IPV4.test(s) || (s.includes(":") && IPV6.test(s));
}

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddHostDialog({ open, onOpenChange }: Props) {
  const [ip, setIp] = useState("");
  const [hostname, setHostname] = useState("");
  const [error, setError] = useState<string | null>(null);

  const add = useMutation({
    meta: { silent: true },
    mutationFn: () => HostService.Add(ip.trim(), hostname.trim()),
    onSuccess: () => {
      setIp("");
      setHostname("");
      setError(null);
      onOpenChange(false);
    },
    onError: (e) => setError(errorMessage(e)),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!isValidIp(ip.trim())) {
      setError("Enter a valid IPv4 or IPv6 address.");
      return;
    }
    setError(null);
    add.mutate();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={submit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Add host</DialogTitle>
            <DialogDescription>Hosts are keyed by IP address.</DialogDescription>
          </DialogHeader>
          <label className="block space-y-1 text-sm">
            <span>IP address</span>
            <Input value={ip} onChange={(e) => setIp(e.target.value)} placeholder="10.10.10.5" autoFocus />
          </label>
          <label className="block space-y-1 text-sm">
            <span>Hostname (optional)</span>
            <Input value={hostname} onChange={(e) => setHostname(e.target.value)} placeholder="nibbles" />
          </label>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={add.isPending}>
              Add host
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
