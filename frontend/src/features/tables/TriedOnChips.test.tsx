import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Credential, Host } from "@/lib/api";
import { TriedOnChips } from "./TriedOnChips";

const hosts = [
  { id: "h1", ip: "10.0.0.1" },
  { id: "h2", ip: "10.0.0.2" },
] as Host[];

describe("TriedOnChips", () => {
  it("renders works and tried labels with host IPs", () => {
    const cred = { id: "c", worksOn: ["h1"], triedOn: ["h1", "h2"] } as Credential;
    render(<TriedOnChips credential={cred} hosts={hosts} />);
    expect(screen.getByText(/works 10\.0\.0\.1/)).toBeTruthy();
    expect(screen.getByText(/tried 10\.0\.0\.2/)).toBeTruthy();
    expect(screen.queryByText(/tried 10\.0\.0\.1/)).toBeNull();
  });
});
