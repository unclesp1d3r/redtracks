import type { Share } from "@/lib/api";
import { RowTable } from "./RowTable";
import { hostIp } from "./shared";

export function SharesTable() {
  return (
    <RowTable<Share>
      kind="share"
      noun="share"
      select={(e) => e.Creds.shares ?? []}
      headers={["Host", "Name", "Access", "Notes"]}
      cells={(s, hosts) => [hostIp(hosts, s.hostId), s.name, s.access, s.notes]}
      name={(s) => s.name}
    />
  );
}
