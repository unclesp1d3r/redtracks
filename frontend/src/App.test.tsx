import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

vi.mock("@/lib/api", async (orig) => ({
  ...(await orig<typeof import("@/lib/api")>()),
  EngagementService: {
    List: vi.fn().mockResolvedValue([
      { Dir: "/r/pg", Name: "pg-test", Type: "osai", Hosts: 3, CreatedAt: "2026-10-12T00:00:00Z" },
    ]),
  },
}));

import App from "./App";

test("opens on the engagements list", async () => {
  window.location.hash = "#/";
  render(<App />);
  expect(await screen.findByText("pg-test")).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Engagements" })).toBeInTheDocument();
});
