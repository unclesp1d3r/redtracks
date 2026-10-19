import { Events } from "@wailsio/runtime";
import { MutationCache, QueryCache, QueryClient, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { ChecklistService, EngagementService, errorMessage, list } from "./api";
import type { DropResult, PreviewResponse } from "./api";
import { EVIDENCE_PREFIX, reportDropResult } from "./capture";
import { errorBanner, type ErrorEvent } from "./errorBanner";

declare module "@tanstack/react-query" {
  interface Register {
    mutationMeta: { silent?: boolean };
  }
}

export const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    // The backend only emits engagement:changed for edits made outside the app,
    // so the app's own writes refresh the same queries here.
    onSuccess: () => invalidateEngagementQueries(),
    onError: (error, _vars, _ctx, mutation) => {
      if (mutation.meta?.silent) return;
      toast.error(errorMessage(error), { duration: Infinity });
    },
  }),
  // Failed queries render through QueryStates; log only.
  queryCache: new QueryCache({
    onError: (error) => console.error("query failed:", error),
  }),
  defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
});

export const keys = {
  engagements: () => ["engagements"] as const,
  engagement: () => ["engagement"] as const,
  summaries: () => ["summaries"] as const,
  graph: () => ["graph"] as const,
  checklists: (ip: string, serviceId: string) => ["checklists", ip, serviceId] as const,
  notes: (ip: string) => ["notes", ip] as const,
  definitions: () => ["definitions"] as const,
  checklistStatus: () => ["checklistStatus"] as const,
  root: () => ["root"] as const,
  evidence: (ip: string) => [EVIDENCE_PREFIX, ip] as const,
  evidenceAll: () => ["evidenceAll"] as const,
  inbox: () => ["inbox"] as const,
  sessionLogs: () => ["sessionLogs"] as const,
  capabilities: () => ["capabilities"] as const,
  /** An import preview delivered by a native drop, waiting for the import page to take it. */
  droppedPreview: () => ["droppedPreview"] as const,
};

/** Same query key as the rest of the app, so every screen shares one fetch. */
export function useEngagementQuery() {
  return useQuery({ queryKey: keys.engagement(), queryFn: () => EngagementService.Current() });
}

export function useSummariesQuery() {
  return useQuery({ queryKey: keys.summaries(), queryFn: () => list(EngagementService.Summaries()) });
}

/** Notes are deliberately not invalidated: the notes editor owns its own refresh. */
const INVALIDATED_ON_CHANGE = [
  keys.engagement(),
  keys.summaries(),
  keys.graph(),
  ["checklists"],
  keys.definitions(),
  [EVIDENCE_PREFIX],
  keys.evidenceAll(),
] as const;

function invalidateEngagementQueries() {
  INVALIDATED_ON_CHANGE.forEach((queryKey) => void queryClient.invalidateQueries({ queryKey }));
}

/** Loads checklist definition status; a clean load also clears any checklists error entries. */
export async function fetchChecklistStatus() {
  const status = await ChecklistService.Status();
  if (!status.Error && !status.InstallError) errorBanner.clearKind("checklists");
  return status;
}

export function useChecklistStatusQuery() {
  return useQuery({ queryKey: keys.checklistStatus(), queryFn: fetchChecklistStatus });
}

function refreshChecklistStatus() {
  void queryClient.fetchQuery({ queryKey: keys.checklistStatus(), queryFn: fetchChecklistStatus, staleTime: 0 }).catch((e) => console.error("checklist status failed:", e));
}

export function subscribeToEngagementEvents(): () => void {
  const offChanged = Events.On("engagement:changed", (e: { data: unknown }) => {
    // The payload is the list of changed paths; "checklists" means the definitions reloaded.
    const paths = Array.isArray(e.data) ? e.data.filter((p): p is string => typeof p === "string") : [];
    errorBanner.clearChanged(paths);
    if (paths.includes("checklists")) refreshChecklistStatus();
    if (paths.includes("inbox")) {
      errorBanner.clearKind("inbox");
      void queryClient.invalidateQueries({ queryKey: keys.inbox() });
      // "inbox" alone means only the inbox folder changed; the engagement files did not.
      if (paths.every((p) => p === "inbox")) return;
    }
    invalidateEngagementQueries();
  });
  const offError = Events.On("engagement:error", (e: { data: ErrorEvent }) => {
    errorBanner.add(e.data);
    if (e.data.Kind === "checklists") refreshChecklistStatus();
    if (e.data.Kind === "inbox") void queryClient.invalidateQueries({ queryKey: keys.inbox() });
  });
  const offDropped = Events.On("evidence:dropped", (e: { data: DropResult }) => handleDropped(e.data));
  return () => {
    offChanged();
    offError();
    offDropped();
  };
}

/** Toasts a native drop and parks an import preview for ImportPage to pick up. */
export function handleDropped(result: DropResult) {
  reportDropResult(queryClient, result);
  if (result.Preview) queryClient.setQueryData<PreviewResponse>(keys.droppedPreview(), result.Preview);
}
