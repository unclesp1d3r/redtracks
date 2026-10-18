import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { success: vi.fn(), warning: vi.fn(), error: vi.fn() }) }));

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  const { vi: v } = await import("vitest");
  return {
    ...actual,
    ...apiMocks,
    SessionLogService: {
      PickLogFile: v.fn(),
      ImportPath: v.fn(),
      List: v.fn(),
      Page: v.fn(),
      Search: v.fn(),
      Clip: v.fn(),
    },
  };
});

import { toast } from "sonner";
import { SessionLogService } from "@/lib/api";
import type { SessionLogSource } from "@/lib/api";
import { queryClient } from "@/lib/query";
import { IP, engagement, item } from "@/test/evidenceFixtures";
import { apiMocks, resetApiMocks } from "@/test/evidenceMocks";
import { renderApp } from "@/test/render";
import { ClipDialog } from "./ClipDialog";
import { SessionLogsPage } from "./SessionLogsPage";
import { SEARCH_LIMIT } from "./useSessionLog";

const svc = vi.mocked(SessionLogService);

const source = (over: Partial<SessionLogSource> = {}): SessionLogSource => ({
  ID: "log1",
  Name: "log1.txt",
  OriginalName: "log1.txt",
  Size: 2048,
  ModTime: "2026-01-01T00:00:00Z",
  ImportedAt: "2026-01-02T00:00:00Z",
  Lines: 300,
  ...over,
});

const pageOf = (from: number, count: number) => Array.from({ length: count }, (_, i) => `text ${from + i}`);

beforeEach(() => {
  resetApiMocks();
  apiMocks.EngagementService.Current.mockResolvedValue(engagement());
  Object.values(svc).forEach((f) => (f as ReturnType<typeof vi.fn>).mockReset());
  svc.List.mockResolvedValue([source()]);
  svc.Page.mockImplementation((_id, from, count) => Promise.resolve(pageOf(from, count)) as never);
  svc.Search.mockResolvedValue([]);
  svc.Clip.mockResolvedValue(item("clip") as never);
});
afterEach(() => {
  cleanup();
  queryClient.clear();
});

async function openLog() {
  renderApp(<SessionLogsPage />);
  fireEvent.click(await screen.findByRole("button", { name: "Open log1.txt" }));
  return screen.findByRole("region", { name: "Log log1.txt" });
}

describe("SessionLogsPage list", () => {
  it("shows loading, then the empty state", async () => {
    svc.List.mockResolvedValue([]);
    renderApp(<SessionLogsPage />);
    expect(screen.getByTestId("query-pending")).toBeInTheDocument();
    expect(await screen.findByText(/No session logs yet/)).toBeInTheDocument();
  });

  it("shows the error with retry", async () => {
    svc.List.mockRejectedValue(new Error("logs dir unreadable"));
    renderApp(<SessionLogsPage />);
    expect(await screen.findByText("logs dir unreadable")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("lists name, size and line count", async () => {
    svc.List.mockResolvedValue([source({ OriginalName: "orig.log" })]);
    renderApp(<SessionLogsPage />);
    const row = await screen.findByRole("button", { name: "Open log1.txt" });
    expect(row).toHaveTextContent("orig.log");
    expect(row).toHaveTextContent("2.0 KB");
    expect(row).toHaveTextContent("300 lines");
  });
});

describe("import control", () => {
  it("without a native dialog submits the typed path", async () => {
    svc.List.mockResolvedValue([]);
    svc.ImportPath.mockResolvedValue(source());
    renderApp(<SessionLogsPage />);
    const input = await screen.findByLabelText("Path to a session log");
    expect(screen.queryByRole("button", { name: "Choose a log file" })).toBeNull();
    fireEvent.change(input, { target: { value: "/tmp/s.log" } });
    svc.List.mockResolvedValue([source()]);
    fireEvent.click(screen.getByRole("button", { name: "Import" }));
    await waitFor(() => expect(svc.ImportPath).toHaveBeenCalledWith("/tmp/s.log"));
    // AE16: the new log appears selected.
    expect(await screen.findByRole("region", { name: "Log log1.txt" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open log1.txt" })).toHaveAttribute("aria-current", "true");
  });

  it("shows a busy state and an inline error", async () => {
    svc.List.mockResolvedValue([]);
    svc.ImportPath.mockRejectedValue(new Error("not a regular file"));
    renderApp(<SessionLogsPage />);
    fireEvent.change(await screen.findByLabelText("Path to a session log"), { target: { value: "/x" } });
    fireEvent.click(screen.getByRole("button", { name: "Import" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("not a regular file");
    // The error is inline; the global mutation toast would show it a second time, until dismissed.
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("with a native dialog calls PickLogFile then ImportPath, and a cancel does nothing", async () => {
    apiMocks.EvidenceService.Capabilities.mockResolvedValue({ NativeDrop: true, NativeDialog: true, Reveal: true });
    renderApp(<SessionLogsPage />);
    const choose = await screen.findByRole("button", { name: "Choose a log file" });
    expect(screen.queryByLabelText("Path to a session log")).toBeNull();
    svc.PickLogFile.mockResolvedValueOnce("");
    fireEvent.click(choose);
    await waitFor(() => expect(svc.PickLogFile).toHaveBeenCalledTimes(1));
    expect(svc.ImportPath).not.toHaveBeenCalled();
    svc.PickLogFile.mockResolvedValueOnce("/home/me/t.log");
    svc.ImportPath.mockResolvedValue(source());
    fireEvent.click(choose);
    await waitFor(() => expect(svc.ImportPath).toHaveBeenCalledWith("/home/me/t.log"));
  });
});

describe("LogViewer", () => {
  it("clips lines 10-49 (display) as 9..48 to the chosen host", async () => {
    const region = await openLog();
    await within(region).findByText("text 9");
    fireEvent.change(within(region).getByLabelText("Clip range"), { target: { value: "10-49" } });
    expect(within(region).getByText("40 lines selected")).toBeInTheDocument();
    fireEvent.click(within(region).getByRole("button", { name: "Clip to evidence" }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByRole("combobox", { name: "Host" }));
    fireEvent.click(await screen.findByRole("option", { name: IP }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Save clip" }));
    await waitFor(() => expect(svc.Clip).toHaveBeenCalledWith("log1", IP, 9, 48));
  });

  it("click then shift-click selects a range", async () => {
    const region = await openLog();
    await within(region).findByText("text 4");
    fireEvent.click(within(region).getByRole("button", { name: "Select line 5" }));
    fireEvent.click(within(region).getByRole("button", { name: "Select line 8" }), { shiftKey: true });
    expect(within(region).getByLabelText("Clip range")).toHaveValue("5-8");
    expect(within(region).getByText("4 lines selected")).toBeInTheDocument();
  });

  it("disables clip over 5,000 lines and says why", async () => {
    svc.List.mockResolvedValue([source({ Lines: 10000 })]);
    const region = await openLog();
    fireEvent.change(within(region).getByLabelText("Clip range"), { target: { value: "1-5001" } });
    expect(within(region).getByRole("status")).toHaveTextContent(/over the limit of 5,000 lines/);
    expect(within(region).getByRole("button", { name: "Clip to evidence" })).toBeDisabled();
  });

  it("search starts at the first line (afterLine -1)", async () => {
    svc.Search.mockResolvedValue([{ Line: 0, Snippet: "first line" }]);
    const region = await openLog();
    fireEvent.change(within(region).getByLabelText("Search in log"), { target: { value: "first" } });
    fireEvent.submit(within(region).getByLabelText("Search in log"));
    expect(await within(region).findByRole("button", { name: "Line 1" })).toBeInTheDocument();
    expect(svc.Search).toHaveBeenCalledWith("log1", "first", -1, SEARCH_LIMIT);
  });

  it("can clip a second range after a successful clip", async () => {
    const region = await openLog();
    await within(region).findByText("text 9");
    fireEvent.change(within(region).getByLabelText("Clip range"), { target: { value: "10-49" } });
    fireEvent.click(within(region).getByRole("button", { name: "Clip to evidence" }));
    let dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByRole("combobox", { name: "Host" }));
    fireEvent.click(await screen.findByRole("option", { name: IP }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Save clip" }));
    await within(dialog).findByText(/Saved to/);
    // The dialog also has an icon-only close button named "Close"; take the footer one.
    const footerClose = within(dialog).getAllByRole("button", { name: "Close" }).find((b) => b.getAttribute("data-slot") === "button");
    fireEvent.click(footerClose!);
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

    fireEvent.change(within(region).getByLabelText("Clip range"), { target: { value: "60-70" } });
    fireEvent.click(within(region).getByRole("button", { name: "Clip to evidence" }));
    dialog = await screen.findByRole("dialog");
    expect(within(dialog).queryByText(/Saved to/)).toBeNull();
    expect(within(dialog).getByRole("button", { name: "Save clip" })).toBeInTheDocument();
  });

  it("search results jump the window to the line", async () => {
    svc.List.mockResolvedValue([source({ Lines: 5000 })]);
    svc.Search.mockResolvedValue([{ Line: 3999, Snippet: "uid=0(root)" }]);
    const region = await openLog();
    fireEvent.change(within(region).getByLabelText("Search in log"), { target: { value: "uid" } });
    fireEvent.submit(within(region).getByLabelText("Search in log"));
    const hit = await within(region).findByRole("button", { name: "Line 4000" });
    expect(hit).toHaveTextContent("line 4000: uid=0(root)");
    svc.Page.mockClear();
    fireEvent.click(hit);
    await waitFor(() => {
      const [, from, count] = svc.Page.mock.calls[svc.Page.mock.calls.length - 1];
      expect(from).toBeLessThanOrEqual(3999);
      expect(from + count).toBeGreaterThan(3999);
    });
    expect(await within(region).findByText("text 3999")).toBeInTheDocument();
  });

  it("shows no-matches and the truncation note", async () => {
    const region = await openLog();
    fireEvent.change(within(region).getByLabelText("Search in log"), { target: { value: "zzz" } });
    fireEvent.submit(within(region).getByLabelText("Search in log"));
    expect(await within(region).findByText("No matches")).toBeInTheDocument();
    svc.Search.mockResolvedValue(Array.from({ length: 500 }, (_, i) => ({ Line: i, Snippet: "s" })));
    fireEvent.change(within(region).getByLabelText("Search in log"), { target: { value: "s" } });
    fireEvent.submit(within(region).getByLabelText("Search in log"));
    expect(await within(region).findByText(/More matches not shown/)).toBeInTheDocument();
  });

  it("requests the tail window when jumping near the end of a long log", async () => {
    svc.List.mockResolvedValue([source({ Lines: 100000 })]);
    const region = await openLog();
    await within(region).findByText("text 0");
    svc.Page.mockClear();
    fireEvent.change(within(region).getByLabelText("Jump to line"), { target: { value: "99991" } });
    fireEvent.submit(within(region).getByLabelText("Jump to line"));
    await waitFor(() => {
      const [, from, count] = svc.Page.mock.calls[svc.Page.mock.calls.length - 1];
      expect(from).toBeLessThanOrEqual(99990);
      expect(from + count).toBeGreaterThanOrEqual(100000);
    });
  });

  it("Next and Previous move the window", async () => {
    const region = await openLog();
    await within(region).findByText("text 0");
    fireEvent.click(within(region).getByRole("button", { name: "Next" }));
    await within(region).findByText("text 200");
    fireEvent.click(within(region).getByRole("button", { name: "Previous" }));
    await within(region).findByText("text 0");
  });
});

describe("ClipDialog", () => {
  it("disables Save clip for a 5,001-line range and shows the reason", async () => {
    renderApp(<ClipDialog log={source({ Lines: 10000 })} from={0} to={5000} open onOpenChange={() => undefined} />);
    expect(await screen.findByRole("alert")).toHaveTextContent("at most 5,000 lines");
    expect(screen.getByRole("button", { name: "Save clip" })).toBeDisabled();
  });
});
