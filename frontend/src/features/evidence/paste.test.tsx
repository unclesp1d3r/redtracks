import { act, cleanup, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  const { apiMocks } = await import("@/test/evidenceMocks");
  return { ...actual, ...apiMocks };
});

import type { Finding, Host } from "@/lib/api";
import { queryClient } from "@/lib/query";
import { MAX_IMPORT_BYTES, textToBase64 } from "@/lib/encode";
import { FindingsList } from "@/features/host/FindingsList";
import { FlagsBar } from "@/features/host/FlagsBar";
import { pasteEvent, pngFile } from "@/test/clipboard";
import { apiMocks, resetApiMocks } from "@/test/evidenceMocks";
import { IP, engagement } from "@/test/evidenceFixtures";
import { renderApp } from "@/test/render";
import { PasteIndicator, PasteScope } from "./PasteScope";

const add = apiMocks.EvidenceService.AddFromData;
const finding = engagement().Findings[0] as unknown as Finding;
const host = engagement().Hosts[0] as unknown as Host;

function setup() {
  return renderApp(
    <PasteScope ip={IP}>
      <PasteIndicator />
      <FlagsBar host={host} flagTypes={[{ id: "proof.txt", label: "proof.txt", points: 1 }]} />
      <FindingsList ip={IP} findings={[finding]} />
      <textarea aria-label="Notes" />
    </PasteScope>,
  );
}

const paste = (target: Element, init: Parameters<typeof pasteEvent>[0]) => {
  const event = pasteEvent(init);
  act(() => {
    target.dispatchEvent(event);
  });
  return event;
};

describe("paste capture", () => {
  beforeEach(resetApiMocks);
  afterEach(() => {
    cleanup();
    queryClient.clear();
  });

  it("AE7: a PNG pasted with a finding row focused is linked to that finding only", async () => {
    setup();
    const row = screen.getByRole("group", { name: "Paste proof for finding SQLi" });
    act(() => row.focus());
    expect(screen.getByRole("status")).toHaveTextContent("Paste target: finding SQLi");
    paste(row, { files: [pngFile("shot.png")] });
    await waitFor(() => expect(add).toHaveBeenCalledTimes(1));
    const [ip, name, data, modified, caption, links] = add.mock.calls[0];
    expect([ip, name, caption]).toEqual([IP, "shot.png", ""]);
    expect(typeof data).toBe("string");
    expect(modified).toBe(1_700_000_000_000);
    expect(links).toEqual([{ kind: "finding", targetId: "f1" }]);
  });

  it("captures an image exposed only through clipboardData.items", async () => {
    setup();
    paste(document.body, { itemFiles: [pngFile("image.png")] });
    await waitFor(() => expect(add).toHaveBeenCalledTimes(1));
    expect(add.mock.calls[0][1]).toBe("image.png");
  });

  it("creates a text item from pasted text", async () => {
    setup();
    paste(document.body, { text: "uid=0(root)" });
    await waitFor(() => expect(add).toHaveBeenCalledTimes(1));
    const [, name, data, , , links] = add.mock.calls[0];
    expect(name).toBe("paste.txt");
    expect(data).toBe(textToBase64("uid=0(root)"));
    expect(links).toEqual([]);
  });

  it("refuses a file over the import limit with the import zone's message", async () => {
    const { toast } = await import("sonner");
    const spy = vi.spyOn(toast, "error");
    setup();
    paste(document.body, { files: [pngFile("huge.png", MAX_IMPORT_BYTES + 1)] });
    await waitFor(() => expect(spy).toHaveBeenCalled());
    expect(spy.mock.calls[0][0]).toBe("huge.png is larger than 20 MB.");
    expect(add).not.toHaveBeenCalled();
  });

  it("attaches to a focused flag row, and names it in the indicator", async () => {
    setup();
    const row = screen.getByRole("group", { name: "Paste proof for flag proof.txt" });
    act(() => row.focus());
    expect(screen.getByRole("status")).toHaveTextContent("Paste target: flag proof.txt");
    paste(row, { files: [pngFile()] });
    await waitFor(() => expect(add).toHaveBeenCalled());
    expect(add.mock.calls[0][5]).toEqual([{ kind: "flag", targetId: "proof.txt" }]);
  });

  it("attaches to the host only when nothing is focused", async () => {
    setup();
    expect(screen.getByRole("status")).toHaveTextContent("Paste target: host");
    paste(document.body, { files: [pngFile()] });
    await waitFor(() => expect(add).toHaveBeenCalled());
    expect(add.mock.calls[0][5]).toEqual([]);
  });

  it("returns to the host once focus leaves the row", () => {
    setup();
    const row = screen.getByRole("group", { name: "Paste proof for finding SQLi" });
    act(() => row.focus());
    act(() => row.blur());
    expect(screen.getByRole("status")).toHaveTextContent("Paste target: host");
  });

  it("leaves a paste into a text field alone", () => {
    setup();
    const event = paste(screen.getByLabelText("Notes"), { text: "just notes" });
    expect(event.defaultPrevented).toBe(false);
    expect(add).not.toHaveBeenCalled();
  });
});
