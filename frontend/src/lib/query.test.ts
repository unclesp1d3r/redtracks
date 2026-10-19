import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const handlers = new Map<string, (e: { data: unknown }) => void>();
vi.mock("@wailsio/runtime", () => ({
  Events: {
    On: (name: string, cb: (e: { data: unknown }) => void) => {
      handlers.set(name, cb);
      return () => handlers.delete(name);
    },
  },
}));
vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { success: vi.fn(), warning: vi.fn(), error: vi.fn() }) }));
vi.mock("./api", async () => {
  const actual = await vi.importActual<typeof import("./api")>("./api");
  return {
    ...actual,
    ChecklistService: { Status: vi.fn().mockResolvedValue({ Error: "", InstallError: "" }) },
    EngagementService: { Current: vi.fn(), Summaries: vi.fn() },
  };
});

import { toast } from "sonner";
import { errorBanner } from "./errorBanner";
import { keys, queryClient, subscribeToEngagementEvents } from "./query";

const fire = (name: string, data: unknown) => handlers.get(name)?.({ data });

describe("engagement events", () => {
  let off: () => void;
  beforeEach(() => {
    off = subscribeToEngagementEvents();
  });
  afterEach(() => {
    off();
    queryClient.clear();
    vi.clearAllMocks();
  });

  it("an inbox-only change invalidates only the inbox key", () => {
    const spy = vi.spyOn(queryClient, "invalidateQueries");
    fire("engagement:changed", ["inbox"]);
    expect(spy.mock.calls.map((c) => c[0]?.queryKey)).toEqual([keys.inbox()]);
  });

  it("an inbox change mixed with other paths also refreshes the engagement queries", () => {
    const spy = vi.spyOn(queryClient, "invalidateQueries");
    fire("engagement:changed", ["inbox", "hosts/10.0.0.1/host.json"]);
    const invalidated = spy.mock.calls.map((c) => c[0]?.queryKey);
    expect(invalidated).toContainEqual(keys.inbox());
    expect(invalidated).toContainEqual(keys.engagement());
  });

  it("an inbox error clears when the inbox changes and refreshes the inbox query", () => {
    errorBanner.add({ Kind: "inbox", Path: "", Message: "watch failed" });
    const spy = vi.spyOn(queryClient, "invalidateQueries");
    fire("engagement:error", { Kind: "inbox", Path: "", Message: "watch failed" });
    expect(spy.mock.calls.map((c) => c[0]?.queryKey)).toContainEqual(keys.inbox());
    fire("engagement:changed", ["inbox"]);
    expect(errorBanner.get().filter((e) => e.Kind === "inbox")).toEqual([]);
  });

  it("evidence:dropped toasts each entry and parks an import preview for the import page", () => {
    const preview = { Token: "t", Preview: { Tool: "nmap" } };
    fire("evidence:dropped", {
      Added: [
        { Path: "/a.png", Item: { id: "e1", file: "a.png", kind: "image", caption: "" }, Duplicate: false },
        { Path: "/b.png", Item: { id: "e2", file: "b.png", kind: "image", caption: "" }, Duplicate: true },
      ],
      Refused: [{ Path: "/c.svg", Reason: "svg is not accepted evidence" }],
      Preview: preview,
    });
    expect(toast.success).toHaveBeenCalledTimes(1);
    expect(toast.warning).toHaveBeenCalledTimes(1);
    expect(toast.error).toHaveBeenCalledTimes(1);
    expect(vi.mocked(toast.error).mock.calls[0][0]).toContain("svg is not accepted evidence");
    expect(queryClient.getQueryData(keys.droppedPreview())).toEqual(preview);
  });
});

describe("evidence invalidation on mutation success", () => {
  it("marks the host evidence key and the all-hosts key stale", async () => {
    queryClient.setQueryData(keys.evidence("10.0.0.1"), { items: [], notIndexed: [] });
    queryClient.setQueryData(keys.evidenceAll(), []);
    await queryClient.getMutationCache().build(queryClient, { mutationFn: async () => 1 }).execute(undefined);
    expect(queryClient.getQueryState(keys.evidence("10.0.0.1"))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(keys.evidenceAll())?.isInvalidated).toBe(true);
  });
});
