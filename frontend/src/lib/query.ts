import { Events } from "@wailsio/runtime";
import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { errorMessage } from "./api";
import { errorBanner } from "./errorBanner";

declare module "@tanstack/react-query" {
  interface Register {
    mutationMeta: { silent?: boolean };
  }
}

export const queryClient = new QueryClient({
  mutationCache: new MutationCache({
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
};

/** Notes are deliberately not invalidated: the notes editor owns its own refresh. */
const INVALIDATED_ON_CHANGE = [
  keys.engagement(),
  keys.summaries(),
  keys.graph(),
  ["checklists"],
  keys.definitions(),
] as const;

export function subscribeToEngagementEvents(): () => void {
  const offChanged = Events.On("engagement:changed", () => {
    errorBanner.clear();
    INVALIDATED_ON_CHANGE.forEach((queryKey) =>
      void queryClient.invalidateQueries({ queryKey }),
    );
  });
  const offError = Events.On("engagement:error", (e: { data: unknown }) => {
    errorBanner.set(typeof e.data === "string" ? e.data : errorMessage(e.data));
  });
  return () => {
    offChanged();
    offError();
  };
}
