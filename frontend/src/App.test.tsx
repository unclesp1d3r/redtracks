import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

vi.mock("@/lib/api", async (orig) => ({
  ...(await orig<typeof import("@/lib/api")>()),
  EngagementService: {
    List: vi.fn().mockResolvedValue({
      Engagements: [{ Dir: "/r/pg", Name: "pg-test", Type: "osai", Hosts: 3, CreatedAt: "2026-10-12T00:00:00Z" }],
      Broken: [{ Dir: "bad-one", Error: "engagement.json: unexpected end of JSON input" }],
    }),
    Root: vi.fn().mockResolvedValue("/home/me/RedTracks"),
  },
}));

import App from "./App";

test("opens on the engagements list", async () => {
  window.location.hash = "#/";
  render(<App />);
  expect(await screen.findByText("pg-test")).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Engagements" })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Could not read" })).toBeInTheDocument();
  expect(screen.getByText("bad-one")).toBeInTheDocument();
  expect(await screen.findByText(/\/home\/me\/RedTracks/)).toBeInTheDocument();
});
