import { EngagementService, type Host } from "@/lib/api";

export type RowKind = "credential" | "user" | "share" | "loot" | "finding";
export type Option = { value: string; label: string };

/**
 * Wails binding methods are plain functions, so they can be detached from
 * EngagementService and dispatched by row kind.
 */
export const ROW_API = {
  credential: {
    add: EngagementService.AddCredential,
    update: EngagementService.UpdateCredential,
    remove: EngagementService.DeleteCredential,
  },
  user: {
    add: EngagementService.AddUser,
    update: EngagementService.UpdateUser,
    remove: EngagementService.DeleteUser,
  },
  share: {
    add: EngagementService.AddShare,
    update: EngagementService.UpdateShare,
    remove: EngagementService.DeleteShare,
  },
  loot: {
    add: EngagementService.AddLoot,
    update: EngagementService.UpdateLoot,
    remove: EngagementService.DeleteLoot,
  },
  finding: {
    add: EngagementService.AddFinding,
    update: EngagementService.UpdateFinding,
    remove: EngagementService.DeleteFinding,
  },
} satisfies Record<
  RowKind,
  {
    add: (r: never) => Promise<void>;
    update: (r: never) => Promise<void>;
    remove: (id: string) => Promise<void>;
  }
>;

export const NONE = "none";

export const FINDING_STATUSES = ["noted", "confirmed", "exploited", "false-positive"] as const;

export function hostOptions(hosts: Host[], key: "id" | "ip" = "id"): Option[] {
  return [{ value: NONE, label: "(none)" }, ...hosts.map((h) => ({ value: h[key], label: h.ip }))];
}

/** Resolve a host ID to its IP for display; blank IDs render as a dash. */
export function hostIp(hosts: Host[], id: string): string {
  if (!id) return "-";
  return hosts.find((h) => h.id === id)?.ip ?? "unknown host";
}
