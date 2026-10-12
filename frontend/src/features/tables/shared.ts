import { useQuery } from "@tanstack/react-query";
import { EngagementService, type Engagement, type Host } from "@/lib/api";
import { keys } from "@/lib/query";

/** Same query key as the rest of the app, so all tables share one fetch. */
export function useEngagementQuery() {
  return useQuery({ queryKey: keys.engagement(), queryFn: () => EngagementService.Current() });
}

export function hostsOf(e: Engagement): Host[] {
  return e.Hosts ?? [];
}

/** Resolve a host ID to its IP for display; blank IDs render as a dash. */
export function hostIp(hosts: Host[], id: string): string {
  if (!id) return "-";
  return hosts.find((h) => h.id === id)?.ip ?? "unknown host";
}

export const NONE = "none";
