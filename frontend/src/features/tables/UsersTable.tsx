import type { User } from "@/lib/api";
import { RowTable } from "./RowTable";
import { hostIp } from "./shared";

export function UsersTable() {
  return (
    <RowTable<User>
      kind="user"
      noun="user"
      select={(e) => e.Creds.users ?? []}
      headers={["Name", "Domain", "Host", "Notes"]}
      cells={(u, hosts) => [u.name, u.domain || "-", hostIp(hosts, u.hostId), u.notes]}
      name={(u) => u.name}
    />
  );
}
