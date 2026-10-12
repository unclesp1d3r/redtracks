import type { Loot } from "@/lib/api";
import { RowTable } from "./RowTable";
import { hostIp } from "./shared";

export function LootTable() {
  return (
    <RowTable<Loot>
      kind="loot"
      noun="loot"
      select={(e) => e.Creds.loot ?? []}
      headers={["Host", "Label", "Path", "Notes"]}
      cells={(l, hosts) => [hostIp(hosts, l.hostId), l.label, <code key="p">{l.path}</code>, l.notes]}
      name={(l) => l.label}
    />
  );
}
