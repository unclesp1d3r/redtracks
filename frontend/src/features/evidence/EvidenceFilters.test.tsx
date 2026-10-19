import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { item, link } from "@/test/evidenceFixtures";
import { EvidenceLinkKind } from "../../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
import { EMPTY_FILTERS, EvidenceFilters, matchesFilters } from "./EvidenceFilters";

afterEach(cleanup);

describe("matchesFilters", () => {
  const linked = item("a", [link(EvidenceLinkKind.EvidenceLinkSlot, "s1")]);
  const onlyRemoved = item("b", [link(EvidenceLinkKind.EvidenceLinkFlag, "f", "f", { removed: true })]);

  it("treats an item with only removed links as nothing yet", () => {
    const f = { ...EMPTY_FILTERS, kind: "nothing" as const };
    expect(matchesFilters("1.1.1.1", onlyRemoved, f)).toBe(true);
    expect(matchesFilters("1.1.1.1", linked, f)).toBe(false);
  });

  it("filters by link kind, host and capture date range", () => {
    expect(matchesFilters("1.1.1.1", linked, { ...EMPTY_FILTERS, kind: EvidenceLinkKind.EvidenceLinkSlot })).toBe(true);
    expect(matchesFilters("1.1.1.1", linked, { ...EMPTY_FILTERS, kind: EvidenceLinkKind.EvidenceLinkFlag })).toBe(false);
    expect(matchesFilters("1.1.1.1", linked, { ...EMPTY_FILTERS, host: "2.2.2.2" })).toBe(false);
    expect(matchesFilters("1.1.1.1", linked, { ...EMPTY_FILTERS, from: "2026-10-02" })).toBe(false);
    expect(matchesFilters("1.1.1.1", linked, { ...EMPTY_FILTERS, to: "2026-09-30" })).toBe(false);
    expect(matchesFilters("1.1.1.1", linked, { ...EMPTY_FILTERS, from: "2026-10-01", to: "2026-10-01" })).toBe(true);
  });
});

describe("EvidenceFilters", () => {
  it("exposes labelled controls and reports changes", () => {
    const onChange = vi.fn();
    render(<EvidenceFilters hosts={["10.0.0.1", "10.0.0.2"]} value={EMPTY_FILTERS} onChange={onChange} />);
    fireEvent.change(screen.getByLabelText("Host"), { target: { value: "10.0.0.2" } });
    expect(onChange).toHaveBeenCalledWith({ ...EMPTY_FILTERS, host: "10.0.0.2" });
    fireEvent.change(screen.getByLabelText("Target kind"), { target: { value: "nothing" } });
    expect(onChange).toHaveBeenCalledWith({ ...EMPTY_FILTERS, kind: "nothing" });
    expect(screen.getByLabelText("Captured from")).toBeInTheDocument();
    expect(screen.getByLabelText("Captured to")).toBeInTheDocument();
  });
});
