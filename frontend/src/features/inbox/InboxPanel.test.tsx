import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  await import("./inboxMocks");
  return { ...actual, ...apiMocks };
});

import { EvidenceKind } from "@/lib/api";
import { queryClient } from "@/lib/query";
import { resetApiMocks } from "@/test/evidenceMocks";
import { renderApp } from "@/test/render";
import { inboxApi, inboxFile, listing } from "./inboxMocks";
import { InboxCountBadge, InboxPanel } from "./InboxPanel";

describe("InboxPanel", () => {
  beforeEach(() => {
    resetApiMocks();
    Object.values(inboxApi).forEach((f) => f.mockReset());
    inboxApi.Root.mockResolvedValue("/eng");
    inboxApi.Inbox.mockResolvedValue(listing());
    inboxApi.DiscardInboxFile.mockResolvedValue(undefined);
  });
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("shows loading, then the file row", async () => {
    renderApp(<InboxPanel />);
    expect(screen.getByTestId("query-pending")).toBeInTheDocument();
    expect(await screen.findByRole("listitem", { name: "shot.png" })).toBeInTheDocument();
  });

  it("shows the empty copy with the folder path, also with no engagement token", async () => {
    inboxApi.Inbox.mockResolvedValue({ Token: "", Files: null });
    renderApp(<InboxPanel />);
    expect(await screen.findByText(/Nothing waiting\. Drop screenshots into \/eng\/inbox/)).toBeInTheDocument();
  });

  it("shows the error with a retry", async () => {
    inboxApi.Inbox.mockRejectedValue(new Error("inbox unreadable"));
    renderApp(<InboxPanel />);
    expect(await screen.findByText("inbox unreadable")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("renders image, text and unsupported kinds differently", async () => {
    inboxApi.Inbox.mockResolvedValue(
      listing([
        inboxFile("shot.png"),
        inboxFile("notes.txt", { Kind: EvidenceKind.EvidenceKindText, Preview: "uid=0(root)" }),
        inboxFile("blob.bin", { Kind: EvidenceKind.$zero }),
      ]),
    );
    renderApp(<InboxPanel />);
    const img = within(await screen.findByRole("listitem", { name: "shot.png" })).getByRole("img");
    expect(img).toHaveAttribute("src", "/inbox/shot.png");
    expect(img).toHaveAttribute("loading", "lazy");
    expect(within(screen.getByRole("listitem", { name: "notes.txt" })).getByText("uid=0(root)")).toBeInTheDocument();
    expect(within(screen.getByRole("listitem", { name: "blob.bin" })).getByText("not supported")).toBeInTheDocument();
  });

  it("encodes odd file names in the image URL", async () => {
    inboxApi.Inbox.mockResolvedValue(listing([inboxFile("a b#1.png")]));
    renderApp(<InboxPanel />);
    const img = within(await screen.findByRole("listitem", { name: "a b#1.png" })).getByRole("img");
    expect(img).toHaveAttribute("src", "/inbox/a%20b%231.png");
  });

  it("discard needs a second confirm click before calling the service", async () => {
    renderApp(<InboxPanel />);
    fireEvent.click(await screen.findByRole("button", { name: "Discard shot.png" }));
    expect(inboxApi.DiscardInboxFile).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Confirm discard shot.png" }));
    await waitFor(() => expect(inboxApi.DiscardInboxFile).toHaveBeenCalledWith("shot.png"));
  });

  it("cancel backs out of discard", async () => {
    renderApp(<InboxPanel />);
    fireEvent.click(await screen.findByRole("button", { name: "Discard shot.png" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("button", { name: "Discard shot.png" })).toBeInTheDocument();
    expect(inboxApi.DiscardInboxFile).not.toHaveBeenCalled();
  });

  it("the count badge shows the ready-file count and renders nothing at zero", async () => {
    inboxApi.Inbox.mockResolvedValue(listing([inboxFile("a.png"), inboxFile("b.png")]));
    const { unmount } = renderApp(<InboxCountBadge />);
    expect(await screen.findByLabelText("2 files in inbox")).toHaveTextContent("2");
    unmount();
    queryClient.clear();
    inboxApi.Inbox.mockResolvedValue(listing([]));
    const { container } = renderApp(<InboxCountBadge />);
    await waitFor(() => expect(inboxApi.Inbox).toHaveBeenCalledTimes(2));
    expect(container).toBeEmptyDOMElement();
  });
});
