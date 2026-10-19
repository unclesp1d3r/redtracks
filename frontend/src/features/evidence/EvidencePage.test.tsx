import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const extra = vi.hoisted(() => ({ ListAll: vi.fn(), Search: vi.fn() }));
vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  return { ...actual, ...apiMocks, EvidenceService: { ...apiMocks.EvidenceService, ...extra } };
});

import { EvidenceLinkKind } from "@/lib/api";
import { queryClient } from "@/lib/query";
import { apiMocks, resetApiMocks } from "@/test/evidenceMocks";
import { engagement, item, link, view } from "@/test/evidenceFixtures";
import { renderApp } from "@/test/render";
import { EvidencePage } from "./EvidencePage";

const all = [
  {
    IP: "10.0.0.1",
    View: view([
      item("a", [link(EvidenceLinkKind.EvidenceLinkSlot, "s1")], { caption: "alpha" }),
      item("b", [link(EvidenceLinkKind.EvidenceLinkFlag, "f", "f", { removed: true })], { caption: "bravo" }),
    ]),
  },
  {
    IP: "10.0.0.2",
    View: view([item("c", [], { caption: "charlie", capturedAt: "2026-11-01T12:00:00Z" })]),
  },
];
const rowIds = () => screen.queryAllByTestId(/^evidence-/).map((r) => r.getAttribute("data-testid"));

describe("EvidencePage", () => {
  beforeEach(() => {
    resetApiMocks();
    extra.ListAll.mockReset().mockResolvedValue(all);
    extra.Search.mockReset().mockResolvedValue([]);
    apiMocks.EngagementService.Current.mockResolvedValue(engagement());
  });
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("shows loading, then rows grouped by host with a count", async () => {
    renderApp(<EvidencePage />);
    expect(screen.getByTestId("query-pending")).toBeInTheDocument();
    await screen.findByTestId("evidence-a");
    expect(rowIds()).toEqual(["evidence-a", "evidence-b", "evidence-c"]);
    expect(screen.getByRole("heading", { name: "10.0.0.2" }).querySelector("a")).toHaveAttribute("href", "/host/10.0.0.2");
    expect(screen.getByText("3 items")).toBeInTheDocument();
  });

  it("shows the empty state", async () => {
    extra.ListAll.mockResolvedValue([{ IP: "10.0.0.1", View: view() }]);
    renderApp(<EvidencePage />);
    expect(await screen.findByText(/No evidence yet\. Capture from a host page/)).toBeInTheDocument();
  });

  it("shows the error state", async () => {
    extra.ListAll.mockRejectedValue(new Error("index unreadable"));
    renderApp(<EvidencePage />);
    expect(await screen.findByText("index unreadable")).toBeInTheDocument();
  });

  it("nothing yet lists only unlinked items, removed links included", async () => {
    renderApp(<EvidencePage />);
    await screen.findByTestId("evidence-a");
    fireEvent.change(screen.getByLabelText("Target kind"), { target: { value: "nothing" } });
    expect(rowIds()).toEqual(["evidence-b", "evidence-c"]);
    expect(screen.getByText("2 of 3 items")).toBeInTheDocument();
  });

  it("filters by host and by capture date range, and shows no-matches", async () => {
    renderApp(<EvidencePage />);
    await screen.findByTestId("evidence-a");
    fireEvent.change(screen.getByLabelText("Host"), { target: { value: "10.0.0.2" } });
    expect(rowIds()).toEqual(["evidence-c"]);
    fireEvent.change(screen.getByLabelText("Host"), { target: { value: "" } });
    fireEvent.change(screen.getByLabelText("Captured from"), { target: { value: "2026-10-15" } });
    expect(rowIds()).toEqual(["evidence-c"]);
    fireEvent.change(screen.getByLabelText("Captured to"), { target: { value: "2026-10-20" } });
    expect(rowIds()).toEqual([]);
    expect(screen.getByText(/No evidence matches/)).toBeInTheDocument();
  });

  it("keeps every item listed while the search is pending", async () => {
    extra.Search.mockReturnValue(new Promise(() => {}));
    renderApp(<EvidencePage />);
    await screen.findByTestId("evidence-a");
    fireEvent.change(screen.getByLabelText("Search evidence"), { target: { value: "needle" } });
    expect(await screen.findByText("Searching…")).toBeInTheDocument();
    expect(rowIds()).toEqual(["evidence-a", "evidence-b", "evidence-c"]);
    expect(screen.queryByText(/No evidence matches/)).toBeNull();
  });

  it("keeps every item listed and shows an alert when the search fails", async () => {
    extra.Search.mockRejectedValue(new Error("index offline"));
    renderApp(<EvidencePage />);
    await screen.findByTestId("evidence-a");
    fireEvent.change(screen.getByLabelText("Search evidence"), { target: { value: "needle" } });
    expect(await screen.findByRole("alert")).toHaveTextContent("index offline");
    expect(rowIds()).toEqual(["evidence-a", "evidence-b", "evidence-c"]);
    expect(screen.queryByText(/No evidence matches/)).toBeNull();
  });

  it("AE8: a search lists only the matching item with its snippet", async () => {
    extra.Search.mockResolvedValue([{ ip: "10.0.0.1", itemId: "b", snippet: "...needle here..." }]);
    renderApp(<EvidencePage />);
    await screen.findByTestId("evidence-a");
    fireEvent.change(screen.getByLabelText("Search evidence"), { target: { value: "needle" } });
    await waitFor(() => expect(rowIds()).toEqual(["evidence-b"]));
    expect(extra.Search).toHaveBeenCalledWith("needle");
    expect(within(screen.getByTestId("evidence-b").parentElement as HTMLElement).getByText("...needle here...")).toBeInTheDocument();
  });
});
