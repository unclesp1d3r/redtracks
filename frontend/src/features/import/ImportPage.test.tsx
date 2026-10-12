import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    EngagementService: { Current: vi.fn().mockResolvedValue({ Hosts: [{ ip: "10.0.0.1" }] }) },
    ImportService: { Apply: vi.fn(), Discard: vi.fn() },
  };
});

import type { PreviewResponse } from "@/lib/api";
import { Preview } from "./ImportPage";

const response: PreviewResponse = {
  Token: "tok",
  Preview: {
    Tool: "gobuster",
    Rows: [],
    Unresolved: [{ Key: "k1", Hostname: "", URL: "" }],
    Dropped: ["999.1.1.1"],
    Skipped: 2,
  },
};

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <Preview response={response} onDone={() => undefined} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ImportPage preview", () => {
  afterEach(cleanup);

  it("keeps Confirm disabled until the unresolved entry is resolved or skipped", async () => {
    setup();
    const confirm = screen.getByRole("button", { name: "Confirm" });
    expect(confirm).toBeDisabled();
    // The label falls back to the key when URL and hostname are empty.
    expect(await screen.findByText("k1")).toBeInTheDocument();
    expect(screen.queryByText(/Nothing in this file/)).toBeNull();

    fireEvent.click(screen.getByRole("checkbox"));
    expect(confirm).toBeEnabled();
    fireEvent.click(screen.getByRole("checkbox"));
    expect(confirm).toBeDisabled();

    fireEvent.click(await screen.findByLabelText("Host for k1"));
    fireEvent.click(await screen.findByRole("option", { name: "10.0.0.1" }));
    await waitFor(() => expect(confirm).toBeEnabled());
  });

  it("shows dropped IPs and the skipped count in the header", () => {
    setup();
    const status = screen.getByRole("status");
    expect(within(status).getByText(/999\.1\.1\.1/)).toBeInTheDocument();
    expect(within(status).getByText(/Skipped 2/)).toBeInTheDocument();
  });
});
