import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  return { ...actual, ...apiMocks };
});

import { EvidenceKind, EvidenceLinkKind } from "@/lib/api";
import { queryClient } from "@/lib/query";
import { apiMocks, resetApiMocks } from "@/test/evidenceMocks";
import { IP, engagement, item, link, slotViews, view } from "@/test/evidenceFixtures";
import { renderApp } from "@/test/render";
import { PasteScope } from "./PasteScope";
import { EvidenceTab } from "./EvidenceTab";
import { ChecklistPane } from "@/features/host/ServicePanel";

const svc = apiMocks.EvidenceService;
const { Slot, Flag, Finding } = {
  Slot: EvidenceLinkKind.EvidenceLinkSlot,
  Flag: EvidenceLinkKind.EvidenceLinkFlag,
  Finding: EvidenceLinkKind.EvidenceLinkFinding,
};

const setup = () =>
  renderApp(
    <PasteScope ip={IP}>
      <EvidenceTab ip={IP} />
    </PasteScope>,
  );

describe("EvidenceTab", () => {
  beforeEach(() => {
    resetApiMocks();
    apiMocks.EngagementService.Current.mockResolvedValue(engagement());
  });
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("shows a loading state, then an empty state with the add control", async () => {
    setup();
    expect(screen.getByTestId("query-pending")).toBeInTheDocument();
    expect(await screen.findByText(/No evidence yet/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add evidence" })).toBeInTheDocument();
  });

  it("shows the error with a retry when the list fails", async () => {
    svc.List.mockRejectedValue(new Error("evidence index unreadable"));
    setup();
    expect(await screen.findByText("evidence index unreadable")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("lists items in the order the backend returns them (capture order)", async () => {
    svc.List.mockResolvedValue(view([item("a", [], { caption: "first" }), item("b", [], { caption: "second" })]));
    setup();
    const rows = await screen.findAllByTestId(/^evidence-/);
    expect(rows.map((r) => r.getAttribute("data-testid"))).toEqual(["evidence-a", "evidence-b"]);
  });

  it("AE1: an item linked to a slot and a flag renders once in the tab", async () => {
    svc.List.mockResolvedValue(
      view([item("a", [link(Slot, "proof.txt", "proof.txt slot"), link(Flag, "proof.txt", "proof.txt")])]),
    );
    setup();
    const rows = await screen.findAllByTestId(/^evidence-/);
    expect(rows).toHaveLength(1);
    expect(within(rows[0]).getByText("proof.txt slot")).toBeInTheDocument();
    expect(within(rows[0]).getByText("proof.txt")).toBeInTheDocument();
  });

  it("AE14: a link to a deleted target reads removed and its unlink calls Unlink", async () => {
    svc.List.mockResolvedValue(view([item("a", [link(Finding, "gone", "Old finding", { removed: true })])]));
    setup();
    const row = await screen.findByTestId("evidence-a");
    expect(within(row).getByText("removed")).toBeInTheDocument();
    fireEvent.click(within(row).getByRole("button", { name: "Unlink Old finding" }));
    await waitFor(() =>
      expect(svc.Unlink).toHaveBeenCalledWith(IP, "a", { kind: "finding", targetId: "gone", serviceId: undefined }),
    );
  });

  it("AE2: a missing file reads 'file missing' and Drop needs the inline confirm", async () => {
    svc.List.mockResolvedValue(view([item("a", [], { missing: true })]));
    setup();
    const row = await screen.findByTestId("evidence-a");
    expect(within(row).getByText("file missing")).toBeInTheDocument();
    fireEvent.click(within(row).getByRole("button", { name: "Drop a.png" }));
    expect(svc.Drop).not.toHaveBeenCalled();
    fireEvent.click(within(row).getByRole("button", { name: "Confirm drop a.png" }));
    await waitFor(() => expect(svc.Drop).toHaveBeenCalledWith(IP, "a"));
  });

  it("saves a caption edit on blur", async () => {
    svc.List.mockResolvedValue(view([item("a")]));
    setup();
    const input = await screen.findByLabelText("Caption for a.png");
    fireEvent.change(input, { target: { value: "login page" } });
    fireEvent.blur(input);
    await waitFor(() => expect(svc.SetCaption).toHaveBeenCalledWith(IP, "a", "login page"));
  });

  it("renders a not-indexed file with an adopt button that calls Adopt", async () => {
    svc.List.mockResolvedValue(view([], ["hand-copied.png"]));
    svc.Adopt.mockResolvedValue(item("n"));
    setup();
    fireEvent.click(await screen.findByRole("button", { name: "Adopt hand-copied.png" }));
    await waitFor(() => expect(svc.Adopt).toHaveBeenCalledWith(IP, "hand-copied.png"));
  });

  it("offers copy path when reveal is unavailable, and reveal when it is", async () => {
    svc.List.mockResolvedValue(view([item("a")]));
    const first = setup();
    expect(await screen.findByRole("button", { name: "Copy path of a.png" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reveal a.png" })).toBeNull();
    first.unmount();
    queryClient.clear();
    svc.Capabilities.mockResolvedValue({ NativeDrop: false, NativeDialog: false, Reveal: true });
    setup();
    fireEvent.click(await screen.findByRole("button", { name: "Reveal a.png" }));
    await waitFor(() => expect(svc.Reveal).toHaveBeenCalledWith(IP, "a"));
  });

  it("the add button is keyboard reachable and opens the hidden file input", async () => {
    setup();
    const button = await screen.findByRole("button", { name: "Add evidence" });
    button.focus();
    expect(button).toHaveFocus();
    const input = screen.getByLabelText("Evidence file");
    const click = vi.spyOn(input, "click");
    fireEvent.click(button);
    expect(click).toHaveBeenCalled();
  });

  it("adopting a file then linking it to a slot flips the slot row from missing to filled", async () => {
    apiMocks.HostService.Checklists.mockResolvedValue(slotViews([{ id: "proof.txt", title: "proof.txt", hint: "cat it" }]));
    let current = view([], ["hand-copied.png"]);
    svc.List.mockImplementation(async () => current);
    svc.Adopt.mockImplementation(async () => {
      current = view([item("n")]);
      return item("n");
    });
    svc.Link.mockImplementation(async (_ip: string, id: string, l: { kind: EvidenceLinkKind; targetId: string }) => {
      current = view([item(id, [link(l.kind, l.targetId)])]);
    });
    renderApp(
      <PasteScope ip={IP}>
        <ChecklistPane ip={IP} serviceId="" domain="" />
        <EvidenceTab ip={IP} />
      </PasteScope>,
    );
    const slot = await screen.findByRole("group", { name: "Paste proof for slot proof.txt" });
    expect(await within(slot).findByText("missing")).toBeInTheDocument();

    fireEvent.click(await screen.findByRole("button", { name: "Adopt hand-copied.png" }));
    fireEvent.click(await screen.findByRole("button", { name: "Add link to n.png" }));
    const slotPicker = await screen.findByLabelText("Slot");
    await waitFor(() => expect(slotPicker).toBeEnabled());
    fireEvent.click(slotPicker);
    fireEvent.click(await screen.findByRole("option", { name: "proof.txt" }));
    fireEvent.click(screen.getByRole("button", { name: "Add links" }));

    await waitFor(() => expect(svc.Link).toHaveBeenCalledWith(IP, "n", { kind: "slot", targetId: "proof.txt", serviceId: "" }));
    expect(await within(slot).findByText("filled")).toBeInTheDocument();
  });

  it("a text item shows its kind icon label", async () => {
    svc.List.mockResolvedValue(view([item("t", [], { kind: EvidenceKind.EvidenceKindText, file: "t.txt" })]));
    setup();
    const row = await screen.findByTestId("evidence-t");
    expect(within(row).getByText("text")).toBeInTheDocument();
  });
});
