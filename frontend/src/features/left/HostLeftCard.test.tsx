import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import type { HostSummary } from "@/lib/api";
import { HostLeftCard } from "./HostLeftCard";

const base = {
  HostID: "h1",
  IP: "10.0.0.5",
  Hostname: "nibbles",
  Status: "untouched",
  MissingFlags: null,
  MissingSlots: null,
  UntouchedServices: null,
  OpenItems: 0,
  UntriedCreds: null,
  NoAccessPath: false,
  FlagPoints: 0,
  TotalPoints: 0,
  ItemsDone: 0,
  ItemsTotal: 0,
} as HostSummary;

function renderCard(summary: HostSummary) {
  render(
    <MemoryRouter>
      <HostLeftCard summary={summary} flagLabels={new Map([["root", "root.txt"]])} creds={new Map()} />
    </MemoryRouter>,
  );
}

afterEach(cleanup);

describe("HostLeftCard", () => {
  it("renders a missing flag label and an untouched service", () => {
    renderCard({ ...base, MissingFlags: ["root"], UntouchedServices: ["80/tcp http"] });
    expect(screen.getByText("root.txt")).toBeTruthy();
    expect(screen.getByText("80/tcp http")).toBeTruthy();
    expect(screen.queryByText("Nothing left")).toBeNull();
  });

  it("renders Nothing left when nothing remains", () => {
    renderCard(base);
    expect(screen.getByText("Nothing left")).toBeTruthy();
  });

  it("lists a host-level slot once and a service-scoped slot with its service label", () => {
    const slot = (id: string, title: string, hint: string, ServiceID = "", ServiceLabel = "") => ({
      DefinitionID: "d", SlotID: id, Title: title, Hint: hint, ServiceID, ServiceLabel,
    });
    renderCard({
      ...base,
      MissingSlots: [
        slot("whoami", "whoami output", "run whoami /all"),
        slot("shot", "Login page", "screenshot it", "s1", "8080/tcp http"),
        slot("shot", "Login page", "screenshot it", "s2", "443/tcp https"),
      ],
    });
    expect(screen.getAllByText("whoami output")).toHaveLength(1);
    expect(screen.getByText("run whoami /all")).toBeTruthy();
    expect(screen.getAllByText("Login page")).toHaveLength(2);
    expect(screen.getByText("8080/tcp http")).toBeTruthy();
    expect(screen.getByText("443/tcp https")).toBeTruthy();
    expect(screen.queryByText("Nothing left")).toBeNull();
  });
});
