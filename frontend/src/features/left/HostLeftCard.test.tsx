import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import type { HostSummary } from "@/lib/api";
import { HostLeftCard } from "./HostLeftCard";

const base = {
  HostID: "h1",
  IP: "10.0.0.5",
  Hostname: "nibbles",
  Status: "untouched",
  MissingFlags: null,
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
});
