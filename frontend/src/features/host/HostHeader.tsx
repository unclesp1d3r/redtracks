import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { HostService, HostStatus } from "@/lib/api";
import type { EngagementMeta, Host } from "@/lib/api";
import { FlagsBar } from "./FlagsBar";

const STATUSES = [
  HostStatus.HostStatusUntouched,
  HostStatus.HostStatusEnumerating,
  HostStatus.HostStatusFoothold,
  HostStatus.HostStatusUser,
  HostStatus.HostStatusRoot,
  HostStatus.HostStatusDone,
];

export function HostHeader({ host, meta }: { host: Host; meta: EngagementMeta }) {
  const [hostname, setHostname] = useState(host.hostname);
  const [os, setOs] = useState(host.os);
  useEffect(() => setHostname(host.hostname), [host.hostname]);
  useEffect(() => setOs(host.os), [host.os]);

  const setStatus = useMutation({ mutationFn: (st: HostStatus) => HostService.SetStatus(host.ip, st) });
  const update = useMutation({ mutationFn: (h: Host) => HostService.Update(h) });

  function commit() {
    if (hostname === host.hostname && os === host.os) return;
    update.mutate({ ...host, hostname, os });
  }

  return (
    <header className="sticky top-0 z-10 space-y-2 border-b bg-background pb-2">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="font-mono text-xl font-semibold">{host.ip}</h1>
        <Input aria-label="Hostname" placeholder="hostname" className="w-48" value={hostname} onChange={(e) => setHostname(e.target.value)} onBlur={commit} />
        <Input aria-label="OS" placeholder="OS" className="w-40" value={os} onChange={(e) => setOs(e.target.value)} onBlur={commit} />
        <Select value={host.status} onValueChange={(v) => v && setStatus.mutate(v as HostStatus)}>
          <SelectTrigger aria-label="Status">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {STATUSES.map((s) => (
              <SelectItem key={s} value={s}>
                {s}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <FlagsBar host={host} flagTypes={meta.flagTypes ?? []} />
    </header>
  );
}
