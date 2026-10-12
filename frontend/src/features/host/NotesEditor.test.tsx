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
vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return { ...actual, EngagementService: { ReadNotes: (ip: string) => readNotes(ip), WriteNotes: vi.fn() } };
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
});
