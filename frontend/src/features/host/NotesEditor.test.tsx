import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const handlers: Array<() => void> = [];
vi.mock("@wailsio/runtime", () => ({
  Events: {
    On: (_name: string, cb: () => void) => {
      handlers.push(cb);
      return () => undefined;
    },
  },
}));
const readNotes = vi.fn();
const writeNotes = vi.fn();
vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return { ...actual, EngagementService: { ReadNotes: (ip: string) => readNotes(ip), WriteNotes: (ip: string, text: string, hash: string) => writeNotes(ip, text, hash) } };
});

import { fireEvent } from "@testing-library/react";
import { NotesEditor } from "./NotesEditor";

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={qc}>
      <NotesEditor ip="10.0.0.1" />
    </QueryClientProvider>,
  );
}

async function fireChanged() {
  await act(async () => handlers.forEach((h) => h()));
}

describe("NotesEditor", () => {
  afterEach(cleanup);
  beforeEach(() => {
    handlers.length = 0;
    readNotes.mockReset();
    writeNotes.mockReset();
    readNotes.mockResolvedValue({ Text: "v1", Hash: "h1" });
  });

  it("keeps a dirty buffer and shows the banner when notes change on disk", async () => {
    setup();
    const area = (await screen.findByLabelText("Notes text")) as HTMLTextAreaElement;
    fireEvent.change(area, { target: { value: "mine" } });
    readNotes.mockResolvedValue({ Text: "v2", Hash: "h2" });
    await fireChanged();
    await waitFor(() => expect(screen.getByText(/changed on disk/)).toBeInTheDocument());
    expect(area.value).toBe("mine");
  });

  it("adopts new text when the buffer is clean", async () => {
    setup();
    const area = (await screen.findByLabelText("Notes text")) as HTMLTextAreaElement;
    readNotes.mockResolvedValue({ Text: "v2", Hash: "h2" });
    await fireChanged();
    await waitFor(() => expect(area.value).toBe("v2"));
    expect(screen.queryByText(/changed on disk/)).toBeNull();
  });

  it("keeps text typed during a save dirty and does not adopt it away", async () => {
    setup();
    const area = (await screen.findByLabelText("Notes text")) as HTMLTextAreaElement;
    let finishWrite: (hash: string) => void = () => undefined;
    writeNotes.mockReturnValue(new Promise<string>((resolve) => (finishWrite = resolve)));
    fireEvent.change(area, { target: { value: "first" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(writeNotes).toHaveBeenCalledWith("10.0.0.1", "first", "h1"));

    fireEvent.change(area, { target: { value: "first plus more" } });
    readNotes.mockResolvedValue({ Text: "first", Hash: "h2" });
    await act(async () => finishWrite("h2"));
    await waitFor(() => expect(screen.queryByText("Unsaved changes")).not.toBeNull());

    await fireChanged();
    expect(area.value).toBe("first plus more");
    expect(screen.queryByText("Unsaved changes")).not.toBeNull();
  });

  it("ignores an older ReadNotes response that resolves after a newer one", async () => {
    setup();
    const area = (await screen.findByLabelText("Notes text")) as HTMLTextAreaElement;
    let resolveFirst!: (n: { Text: string; Hash: string }) => void;
    let resolveSecond!: (n: { Text: string; Hash: string }) => void;
    readNotes
      .mockReturnValueOnce(new Promise((r) => (resolveFirst = r)))
      .mockReturnValueOnce(new Promise((r) => (resolveSecond = r)));
    await fireChanged();
    await fireChanged();

    await act(async () => resolveSecond({ Text: "newer", Hash: "h3" }));
    await waitFor(() => expect(area.value).toBe("newer"));
    await act(async () => resolveFirst({ Text: "older", Hash: "h2" }));
    expect(area.value).toBe("newer");
  });

  async function typeAndSave(value: string) {
    const area = (await screen.findByLabelText("Notes text")) as HTMLTextAreaElement;
    fireEvent.change(area, { target: { value } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    return area;
  }

  it("opens the conflict dialog, and Overwrite re-reads the hash before writing", async () => {
    setup();
    writeNotes.mockRejectedValueOnce(new Error("conflict: notes.md changed on disk"));
    await typeAndSave("mine");
    await screen.findByText("Notes changed on disk");

    readNotes.mockResolvedValue({ Text: "theirs", Hash: "h9" });
    writeNotes.mockResolvedValueOnce("h10");
    readNotes.mockClear();
    fireEvent.click(screen.getByRole("button", { name: "Overwrite" }));
    await waitFor(() => expect(writeNotes).toHaveBeenLastCalledWith("10.0.0.1", "mine", "h9"));
    expect(readNotes).toHaveBeenCalledWith("10.0.0.1");
    await waitFor(() => expect(screen.queryByText("Notes changed on disk")).toBeNull());
  });

  it("Reload in the conflict dialog adopts the disk text and clears dirty", async () => {
    setup();
    writeNotes.mockRejectedValueOnce(new Error("conflict: notes.md changed on disk"));
    const area = await typeAndSave("mine");
    await screen.findByText("Notes changed on disk");

    readNotes.mockResolvedValue({ Text: "theirs", Hash: "h9" });
    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    await waitFor(() => expect(area.value).toBe("theirs"));
    expect(screen.queryByText("Unsaved changes")).toBeNull();
    expect(screen.queryByText("Notes changed on disk")).toBeNull();
  });

  it("shows a failed Reload inside the conflict dialog", async () => {
    setup();
    writeNotes.mockRejectedValueOnce(new Error("conflict: notes.md changed on disk"));
    await typeAndSave("mine");
    const dialog = await screen.findByRole("dialog");

    readNotes.mockRejectedValue(new Error("disk unreadable"));
    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    await waitFor(() => expect(dialog).toHaveTextContent("disk unreadable"));
  });

  it("sends the hash WriteNotes returned on the next save", async () => {
    setup();
    writeNotes.mockResolvedValueOnce("h2").mockResolvedValueOnce("h3");
    const area = await typeAndSave("one");
    await waitFor(() => expect(writeNotes).toHaveBeenLastCalledWith("10.0.0.1", "one", "h1"));
    await waitFor(() => expect(screen.queryByText("Unsaved changes")).toBeNull());
    fireEvent.change(area, { target: { value: "two" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(writeNotes).toHaveBeenLastCalledWith("10.0.0.1", "two", "h2"));
  });
});
