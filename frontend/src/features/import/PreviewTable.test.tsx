import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ImportBadge, RowKind, type Row } from "@/lib/api";
import { PreviewTable } from "./PreviewTable";

const rows: Row[] = [
  { Kind: RowKind.RowKindHost, HostIP: "10.0.0.1", Badge: ImportBadge.BadgeNew, Label: "10.0.0.1" },
  { Kind: RowKind.RowKindService, HostIP: "10.0.0.1", Badge: ImportBadge.BadgeFills, Label: "80/tcp http" },
  { Kind: RowKind.RowKindHost, HostIP: "10.0.0.2", Badge: ImportBadge.BadgeDuplicate, Label: "10.0.0.2" },
];

describe("PreviewTable", () => {
  it("renders one header per host and a text badge per row", () => {
    render(<PreviewTable rows={rows} />);
    expect(screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent)).toEqual([
      "10.0.0.1",
      "10.0.0.2",
    ]);
    for (const text of ["new", "fills", "duplicate"]) expect(screen.getByText(text)).toBeInTheDocument();
  });
});
