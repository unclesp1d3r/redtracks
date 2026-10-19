import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { Table, TableBody } from "@/components/ui/table";
import type { HostSummary } from "@/lib/api";
import { HostRow } from "./HostRow";

afterEach(cleanup);

const slot = (id: string) => ({ DefinitionID: "d", SlotID: id, Title: id, Hint: "", ServiceID: "", ServiceLabel: "" });
const summary = (ip: string, over: Partial<HostSummary>) =>
  ({ HostID: ip, IP: ip, Hostname: "", Status: "untouched", MissingFlags: null, MissingSlots: null, OpenItems: 0,
    NoAccessPath: false, ItemsDone: 0, ItemsTotal: 0, ...over }) as HostSummary;

describe("HostRow owed proof", () => {
  it("AE4: keeps the given order and counts flags plus slots, or says proof complete", () => {
    const rows = [
      summary("10.0.0.9", { MissingFlags: ["root"], MissingSlots: [slot("a")] }),
      summary("10.0.0.1", { MissingFlags: ["user"] }),
      summary("10.0.0.5", {}),
    ];
    render(
      <MemoryRouter>
        <Table>
          <TableBody>
            {rows.map((s) => (
              <HostRow key={s.HostID} summary={s} host={undefined} flagTypes={[]} />
            ))}
          </TableBody>
        </Table>
      </MemoryRouter>,
    );
    expect(screen.getAllByRole("link").map((a) => a.textContent)).toEqual(["10.0.0.9", "10.0.0.1", "10.0.0.5"]);
    expect(screen.getAllByTestId("owed-proof").map((e) => e.textContent)).toEqual([
      "2 proofs owed",
      "1 proof owed",
      "proof complete",
    ]);
  });
});
