import { render, screen, waitFor } from "@testing-library/react";
import { expect, test, vi } from "vitest";

vi.mock("mermaid", () => ({
  default: { initialize: vi.fn(), render: vi.fn().mockRejectedValue(new Error("bad syntax")) },
}));

import mermaid from "mermaid";
import { Mermaid } from "./Mermaid";

test("shows the raw text in a pre when rendering fails", async () => {
  render(<Mermaid text="graph TD; not valid ((" />);
  const pre = await screen.findByText("graph TD; not valid ((");
  expect(pre.tagName).toBe("PRE");
  expect(screen.getByText(/failed to render/i)).toBeInTheDocument();
});

test("strips scripts and event handlers from a rendered svg", async () => {
  vi.mocked(mermaid.render).mockResolvedValueOnce({
    svg: '<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><script>alert(1)</script><rect id="ok" width="1" height="1"/></svg>',
  } as never);
  const { container } = render(<Mermaid text="graph TD; a-->b" />);
  await waitFor(() => expect(container.querySelector("#ok")).not.toBeNull());
  expect(container.querySelector("svg")).not.toBeNull();
  expect(container.querySelector("script")).toBeNull();
  expect(container.querySelector("svg")?.hasAttribute("onload")).toBe(false);
});

test("renders nothing and warns of nothing when unmounted before render resolves", async () => {
  const errorSpy = vi.spyOn(console, "error").mockImplementation(() => undefined);
  let resolveRender!: (v: { svg: string }) => void;
  vi.mocked(mermaid.render).mockReturnValueOnce(new Promise((r) => (resolveRender = r)) as never);
  const callsBefore = vi.mocked(mermaid.render).mock.calls.length;
  const { container, unmount } = render(<Mermaid text="graph TD; c-->d" />);
  await waitFor(() => expect(mermaid.render).toHaveBeenCalledTimes(callsBefore + 1));
  unmount();
  resolveRender({ svg: '<svg xmlns="http://www.w3.org/2000/svg"><rect id="late"/></svg>' });
  await new Promise((r) => setTimeout(r, 0));
  expect(container.innerHTML).toBe("");
  expect(errorSpy).not.toHaveBeenCalled();
  errorSpy.mockRestore();
});
