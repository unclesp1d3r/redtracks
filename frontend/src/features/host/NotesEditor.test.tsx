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
    let finishWrite: () => void = () => undefined;
    writeNotes.mockReturnValue(new Promise<void>((resolve) => (finishWrite = resolve)));
    fireEvent.change(area, { target: { value: "first" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(writeNotes).toHaveBeenCalledWith("10.0.0.1", "first", "h1"));

    fireEvent.change(area, { target: { value: "first plus more" } });
    readNotes.mockResolvedValue({ Text: "first", Hash: "h2" });
    await act(async () => finishWrite());
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
});
