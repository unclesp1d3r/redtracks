import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { SessionLogService, list } from "@/lib/api";
import type { SessionLogSource } from "@/lib/api";
import { reportAdded } from "@/lib/capture";
import { keys } from "@/lib/query";

/** The server caps a search at this many matches; getting this many back means there may be more. */
export const SEARCH_LIMIT = 500;

export function useSessionLogsQuery() {
  return useQuery({ queryKey: keys.sessionLogs(), queryFn: () => list(SessionLogService.List()) });
}

/** Imports the given path, or asks the native dialog for one when none is given. A cancelled dialog resolves null. */
export function useImportLog(onImported: (source: SessionLogSource) => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (path?: string) => {
      const chosen = path ?? (await SessionLogService.PickLogFile());
      return chosen === "" ? null : SessionLogService.ImportPath(chosen);
    },
    onSuccess: (source) => {
      if (!source) return;
      void qc.invalidateQueries({ queryKey: keys.sessionLogs() });
      onImported(source);
    },
  });
}

export function usePage(id: string, from: number, count: number) {
  return useQuery({
    queryKey: ["sessionLog", id, from, count],
    queryFn: () => list(SessionLogService.Page(id, from, count)),
    placeholderData: keepPreviousData,
  });
}

/** Runs only once a non-empty query has been submitted. */
export function useLogSearch(id: string, query: string) {
  return useQuery({
    queryKey: ["sessionLogSearch", id, query],
    queryFn: () => list(SessionLogService.Search(id, query, -1, SEARCH_LIMIT)),
    enabled: query !== "",
  });
}

export interface ClipArgs {
  ip: string;
  from: number;
  to: number;
}

export function useClip(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ ip, from, to }: ClipArgs) => SessionLogService.Clip(id, ip, from, to),
    onSuccess: (added, { ip }) => {
      reportAdded(qc, added, false);
      void qc.invalidateQueries({ queryKey: keys.evidence(ip) });
      void qc.invalidateQueries({ queryKey: keys.evidenceAll() });
    },
  });
}
