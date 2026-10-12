import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

vi.mock("mermaid", () => ({
  default: { initialize: vi.fn(), render: vi.fn().mockRejectedValue(new Error("bad syntax")) },
}));

import { Mermaid } from "./Mermaid";

test("shows the raw text in a pre when rendering fails", async () => {
  render(<Mermaid text="graph TD; not valid ((" />);
  const pre = await screen.findByText("graph TD; not valid ((");
  expect(pre.tagName).toBe("PRE");
  expect(screen.getByText(/failed to render/i)).toBeInTheDocument();
});
