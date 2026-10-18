import { act, cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  return { ...actual, ...apiMocks };
});

import { EvidenceLinkKind } from "@/lib/api";
import type { Finding, PreviewResponse } from "@/lib/api";
import { FindingsList } from "@/features/host/FindingsList";
import { DropZone } from "@/features/import/DropZone";
import { ImportPage } from "@/features/import/ImportPage";
import { handleDropped, queryClient } from "@/lib/query";
import { dropEvent, pngFile } from "@/test/clipboard";
import { apiMocks, resetApiMocks } from "@/test/evidenceMocks";
import { IP, engagement } from "@/test/evidenceFixtures";
import { renderApp } from "@/test/render";
import { dropAttrs, formatTarget } from "./dropTarget";
import { PasteScope } from "./PasteScope";

const finding = engagement().Findings[0] as unknown as Finding;
const NATIVE = { NativeDrop: true, NativeDialog: false, Reveal: false };

describe("formatTarget", () => {
  it("encodes kind:targetId[:serviceId] the way Dropper.parseTarget reads it", () => {
    expect(formatTarget({ kind: EvidenceLinkKind.EvidenceLinkFinding, targetId: "f1" })).toBe("finding:f1");
    expect(formatTarget({ kind: EvidenceLinkKind.EvidenceLinkSlot, targetId: "proof.txt", serviceId: "s1" })).toBe("slot:proof.txt:s1");
    expect(formatTarget({ kind: EvidenceLinkKind.EvidenceLinkSlot, targetId: "proof.txt", serviceId: "" })).toBe("slot:proof.txt");
  });

  it("omits the target attribute for a host-only drop", () => {
    expect(dropAttrs(IP, undefined)).toEqual({ "data-file-drop-target": "evidence", "data-evidence-host": IP });
  });
});

describe("drop targets", () => {
  beforeEach(resetApiMocks);
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("a finding row carries the native attributes when NativeDrop is true", async () => {
    apiMocks.EvidenceService.Capabilities.mockResolvedValue(NATIVE);
    renderApp(
      <PasteScope ip={IP}>
        <FindingsList ip={IP} findings={[finding]} />
      </PasteScope>,
    );
    const row = screen.getByRole("group", { name: "Paste proof for finding SQLi" });
    await waitFor(() => expect(row).toHaveAttribute("data-file-drop-target"));
    expect(row).toHaveAttribute("data-evidence-host", IP);
    expect(row).toHaveAttribute("data-evidence-target", "finding:f1");
  });

  it("without NativeDrop the row handles an HTML5 drop and links the finding", async () => {
    renderApp(
      <PasteScope ip={IP}>
        <FindingsList ip={IP} findings={[finding]} />
      </PasteScope>,
    );
    const row = screen.getByRole("group", { name: "Paste proof for finding SQLi" });
    await waitFor(() => expect(apiMocks.EvidenceService.Capabilities).toHaveBeenCalled());
    expect(row).not.toHaveAttribute("data-file-drop-target");
    act(() => {
      row.dispatchEvent(dropEvent([pngFile("drop.png")]));
    });
    await waitFor(() => expect(apiMocks.EvidenceService.AddFromData).toHaveBeenCalled());
    const call = apiMocks.EvidenceService.AddFromData.mock.calls[0];
    expect(call[1]).toBe("drop.png");
    expect(call[5]).toEqual([{ kind: "finding", targetId: "f1" }]);
  });
});

describe("import DropZone", () => {
  beforeEach(resetApiMocks);
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("carries the native import attributes when NativeDrop is true", async () => {
    apiMocks.EvidenceService.Capabilities.mockResolvedValue(NATIVE);
    renderApp(<DropZone onPreview={() => undefined} />);
    const zone = screen.getByTestId("drop-zone");
    await waitFor(() => expect(zone).toHaveAttribute("data-file-drop-target"));
    expect(zone).toHaveAttribute("data-evidence-kind", "import");
  });

  it("without NativeDrop an HTML5 drop previews the file", async () => {
    const response = { Token: "t", Preview: { Tool: "nmap", Rows: [], Unresolved: [], Dropped: [], Skipped: 0 } } as unknown as PreviewResponse;
    apiMocks.ImportService.PreviewData.mockResolvedValue(response);
    const onPreview = vi.fn();
    renderApp(<DropZone onPreview={onPreview} />);
    const zone = screen.getByTestId("drop-zone");
    await waitFor(() => expect(apiMocks.EvidenceService.Capabilities).toHaveBeenCalled());
    expect(zone).not.toHaveAttribute("data-file-drop-target");
    fireEvent(zone, dropEvent([new File(["x"], "scan.xml")]));
    await waitFor(() => expect(onPreview).toHaveBeenCalledWith(response));
  });

  it("routes a preview received over evidence:dropped into the import page", async () => {
    apiMocks.EngagementService.Current.mockResolvedValue(engagement());
    renderApp(<ImportPage />);
    expect(screen.getByTestId("drop-zone")).toBeInTheDocument();
    act(() =>
      handleDropped({
        Added: null,
        Refused: null,
        Preview: { Token: "t", Preview: { Tool: "gobuster", Rows: [], Unresolved: [], Dropped: [], Skipped: 0 } } as unknown as PreviewResponse,
      }),
    );
    expect(await screen.findByText("gobuster")).toBeInTheDocument();
    expect(screen.queryByTestId("drop-zone")).toBeNull();
  });
});
