import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { EvidenceService } from "@/lib/api";
import type { EvidenceLink } from "@/lib/api";
import { reportAdded } from "@/lib/capture";
import { keys } from "@/lib/query";

export function useInboxQuery() {
  return useQuery({ queryKey: keys.inbox(), queryFn: () => EvidenceService.Inbox() });
}

/** Ready-file count for the nav badge; 0 while loading or on error. */
export function useInboxCount(): number {
  return useInboxQuery().data?.Files?.length ?? 0;
}

interface AssignInput {
  token: string;
  name: string;
  ip: string;
  caption: string;
  links: EvidenceLink[];
}

/** Errors are silent here: AssignDialog shows them inline (stale token) next to a Reload button. */
export function useAssign() {
  const qc = useQueryClient();
  return useMutation({
    meta: { silent: true },
    mutationFn: (v: AssignInput) => EvidenceService.Assign(v.token, v.name, v.ip, v.caption, v.links),
    onSuccess: (item, v) => {
      reportAdded(qc, item, false, v.ip);
      void qc.invalidateQueries({ queryKey: keys.inbox() });
      void qc.invalidateQueries({ queryKey: keys.evidence(v.ip) });
      void qc.invalidateQueries({ queryKey: keys.evidenceAll() });
    },
  });
}

export function useDiscard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => EvidenceService.DiscardInboxFile(name),
    onSettled: () => void qc.invalidateQueries({ queryKey: keys.inbox() }),
  });
}
