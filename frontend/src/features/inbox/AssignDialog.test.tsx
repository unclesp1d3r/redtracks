import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  await import("./inboxMocks");
  return { ...actual, ...apiMocks };
});

import { EvidenceKind } from "@/lib/api";
import { queryClient } from "@/lib/query";
import { apiMocks, resetApiMocks } from "@/test/evidenceMocks";
import { IP, engagement, item, slotViews } from "@/test/evidenceFixtures";
import { renderApp } from "@/test/render";
import { AssignDialog } from "./AssignDialog";
import { TOKEN, inboxApi, inboxFile, listing } from "./inboxMocks";

const file = inboxFile("shot.png");

async function pickHost() {
  fireEvent.click(await screen.findByLabelText("Host"));
  fireEvent.click(await screen.findByRole("option", { name: IP }));
}

describe("AssignDialog", () => {
  beforeEach(() => {
    resetApiMocks();
    Object.values(inboxApi).forEach((f) => f.mockReset());
    apiMocks.EngagementService.Current.mockResolvedValue(engagement());
    apiMocks.HostService.Checklists.mockResolvedValue(slotViews([{ id: "proof.txt", title: "proof.txt" }]));
    inboxApi.Inbox.mockResolvedValue(listing());
    inboxApi.Assign.mockResolvedValue({ ...item("new"), kind: EvidenceKind.EvidenceKindImage });
  });
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  const setup = (onOpenChange = vi.fn()) => {
    renderApp(<AssignDialog file={file} token={TOKEN} open onOpenChange={onOpenChange} />);
    return onOpenChange;
  };

  it("confirm is disabled until a host is chosen", async () => {
    setup();
    expect(await screen.findByRole("dialog", { name: "Assign shot.png" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Assign to host" })).toBeDisabled();
    await pickHost();
    expect(screen.getByRole("button", { name: "Assign to host" })).toBeEnabled();
  });

  it("AE3: assigns with host, slot link, caption and the listing token, then refetches the inbox", async () => {
    const onOpenChange = setup();
    await pickHost();
    const slotPicker = await screen.findByLabelText("Slot");
    await waitFor(() => expect(slotPicker).toBeEnabled());
    fireEvent.click(slotPicker);
    fireEvent.click(await screen.findByRole("option", { name: "proof.txt" }));
    fireEvent.click(screen.getByRole("button", { name: "Use file name" }));
    expect(screen.getByLabelText("Caption")).toHaveValue("shot.png");
    const before = inboxApi.Inbox.mock.calls.length;
    fireEvent.click(screen.getByRole("button", { name: "Assign to host" }));

    await waitFor(() =>
      expect(inboxApi.Assign).toHaveBeenCalledWith(TOKEN, "shot.png", IP, "shot.png", [
        { kind: "slot", targetId: "proof.txt", serviceId: "" },
      ]),
    );
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    await waitFor(() => expect(inboxApi.Inbox.mock.calls.length).toBeGreaterThan(before));
  });

  it("AE12: a stale token error shows inline, keeps the dialog open and reloads on request", async () => {
    inboxApi.Assign.mockRejectedValue(new Error("the engagement changed since this inbox listing; reload the inbox"));
    const onOpenChange = setup();
    await pickHost();
    fireEvent.click(screen.getByRole("button", { name: "Assign to host" }));
    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("the engagement changed since this inbox listing; reload the inbox");
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    const before = inboxApi.Inbox.mock.calls.length;
    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    await waitFor(() => expect(inboxApi.Inbox.mock.calls.length).toBeGreaterThan(before));
  });
});
