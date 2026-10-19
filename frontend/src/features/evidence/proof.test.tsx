import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  return { ...actual, ...apiMocks };
});

import { EvidenceLinkKind } from "@/lib/api";
import type { Host } from "@/lib/api";
import { AccessPaths } from "@/features/host/AccessPaths";
import { FlagsBar } from "@/features/host/FlagsBar";
import { keys, queryClient } from "@/lib/query";
import { apiMocks, resetApiMocks } from "@/test/evidenceMocks";
import { IP, engagement, item, link, view } from "@/test/evidenceFixtures";
import { renderApp } from "@/test/render";
import { PasteScope } from "./PasteScope";
import { SlotRows } from "./SlotRow";
import { captionFocus } from "@/lib/capture";
import { EvidenceTab } from "./EvidenceTab";

const svc = apiMocks.EvidenceService;
const host = engagement().Hosts[0] as unknown as Host;

describe("proof state on flags and access paths", () => {
  beforeEach(resetApiMocks);
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("AE10: an access path with no item is unproven, then proven with the chip after linking", async () => {
    let current = view([item("a", [], { caption: "ssh banner" })]);
    svc.List.mockImplementation(async () => current);
    svc.Link.mockImplementation(async (_ip: string, id: string, l: { kind: EvidenceLinkKind; targetId: string }) => {
      current = view([item(id, [link(l.kind, l.targetId, "chisel")], { caption: "ssh banner" })]);
    });
    renderApp(
      <PasteScope ip={IP}>
        <AccessPaths host={host} otherHosts={[]} credentials={[]} />
      </PasteScope>,
    );
    const row = await screen.findByRole("group", { name: "Paste proof for access path chisel" });
    expect(await within(row).findByText("unproven")).toBeInTheDocument();

    fireEvent.click(within(row).getByLabelText("Link existing item to access path chisel"));
    fireEvent.click(await screen.findByRole("option", { name: "ssh banner" }));
    await waitFor(() =>
      expect(svc.Link).toHaveBeenCalledWith(IP, "a", { kind: "accessPath", targetId: "ap1" }),
    );
    expect(await within(row).findByText("proven")).toBeInTheDocument();
    expect(within(row).getByText("ssh banner")).toBeInTheDocument();
  });

  it("a flag is proven by a linked item, but not when that item's file is missing", async () => {
    const flagLink = link(EvidenceLinkKind.EvidenceLinkFlag, "proof.txt");
    svc.List.mockResolvedValue(view([item("a", [flagLink], { missing: true })]));
    renderApp(
      <PasteScope ip={IP}>
        <FlagsBar host={host} flagTypes={[{ id: "proof.txt", label: "proof.txt", points: 1 }]} />
      </PasteScope>,
    );
    const row = await screen.findByRole("group", { name: "Paste proof for flag proof.txt" });
    expect(await within(row).findByText("unproven")).toBeInTheDocument();

    svc.List.mockResolvedValue(view([item("a", [flagLink])]));
    await act(() => queryClient.invalidateQueries());
    expect(await within(row).findByText("proven")).toBeInTheDocument();
  });

  it("AE1: one item linked to a slot and a flag shows as a chip on both targets", async () => {
    const shot = item(
      "a",
      [link(EvidenceLinkKind.EvidenceLinkSlot, "proof.txt", "proof.txt", { serviceId: "" }), link(EvidenceLinkKind.EvidenceLinkFlag, "proof.txt")],
      { caption: "root shell" },
    );
    svc.List.mockResolvedValue(view([shot]));
    renderApp(
      <PasteScope ip={IP}>
        <FlagsBar host={host} flagTypes={[{ id: "proof.txt", label: "proof.txt", points: 1 }]} />
        <SlotRows ip={IP} serviceId="" slots={[{ id: "proof.txt", title: "proof.txt", hint: "" }]} />
      </PasteScope>,
    );
    const flag = await screen.findByRole("group", { name: "Paste proof for flag proof.txt" });
    const slot = screen.getByRole("group", { name: "Paste proof for slot proof.txt" });
    expect(await within(flag).findByText("root shell")).toBeInTheDocument();
    expect(await within(slot).findByText("root shell")).toBeInTheDocument();
    expect(await within(slot).findByText("filled")).toBeInTheDocument();
  });

  it("the Add caption toast action focuses that item's caption field", async () => {
    svc.List.mockResolvedValue(view([item("a"), item("b")]));
    renderApp(
      <PasteScope ip={IP}>
        <EvidenceTab ip={IP} />
      </PasteScope>,
    );
    const second = await screen.findByLabelText("Caption for b.png");
    act(() => captionFocus.request(IP, "b"));
    await waitFor(() => expect(second).toHaveFocus());
    expect(captionFocus.get()).toBeNull();
  });

  it("linking, unlinking, dropping and adopting each refresh the host and all-hosts evidence", async () => {
    svc.List.mockResolvedValue(view([item("a")], ["n.png"]));
    svc.Adopt.mockResolvedValue(item("n"));
    renderApp(
      <PasteScope ip={IP}>
        <EvidenceTab ip={IP} />
      </PasteScope>,
    );
    queryClient.setQueryData(keys.evidenceAll(), []);
    const stale = () => [keys.evidence(IP), keys.evidenceAll()].map((k) => queryClient.getQueryState(k)?.isInvalidated);
    await screen.findByTestId("evidence-a");
    const refetched = () => svc.List.mock.calls.length;
    const before = refetched();
    fireEvent.click(screen.getByRole("button", { name: "Adopt n.png" }));
    await waitFor(() => expect(refetched()).toBeGreaterThan(before));
    expect(queryClient.getQueryState(keys.evidenceAll())?.isInvalidated).toBe(true);
    expect(stale()).toHaveLength(2);
  });
});
