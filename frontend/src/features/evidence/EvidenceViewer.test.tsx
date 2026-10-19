import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EvidenceKind } from "@/lib/api";
import { IP, item } from "@/test/evidenceFixtures";
import { evidenceUrl } from "./evidenceUrl";
import { EvidenceViewer } from "./EvidenceViewer";

describe("EvidenceViewer", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("builds an escaped relative URL", () => {
    expect(evidenceUrl("fe80::1", "a b.png")).toBe("/evidence/fe80%3A%3A1/a%20b.png");
  });

  it("shows an image lazily with alt text from the caption", () => {
    render(<EvidenceViewer ip={IP} item={item("a", [], { caption: "login page" })} open onOpenChange={() => undefined} />);
    const img = screen.getByRole("img", { name: "login page" });
    expect(img).toHaveAttribute("src", `/evidence/${IP}/a.png`);
    expect(img).toHaveAttribute("loading", "lazy");
  });

  it("falls back to the kind and capture time for alt text", () => {
    render(<EvidenceViewer ip={IP} item={item("a")} open onOpenChange={() => undefined} />);
    expect(screen.getByRole("img").getAttribute("alt")).toMatch(/^image captured /);
  });

  it("renders text evidence in a pre, verbatim", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, text: async () => "$(id) <img onerror=x>" }));
    render(
      <EvidenceViewer ip={IP} item={item("t", [], { kind: EvidenceKind.EvidenceKindText, file: "t.txt" })} open onOpenChange={() => undefined} />,
    );
    await waitFor(() => expect(document.querySelector("pre")).toHaveTextContent("$(id) <img onerror=x>"));
    expect(document.querySelector("img")).toBeNull();
  });

  it("shows the missing state instead of the file", () => {
    render(<EvidenceViewer ip={IP} item={item("a", [], { missing: true })} open onOpenChange={() => undefined} />);
    expect(screen.getByText("file missing")).toBeInTheDocument();
    expect(screen.queryByRole("img")).toBeNull();
  });
});
