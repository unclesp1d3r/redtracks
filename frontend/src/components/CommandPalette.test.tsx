import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CommandPalette, SCREENS } from "./CommandPalette";

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => navigate,
}));
vi.mock("@/lib/query", () => ({ useEngagementQuery: () => ({ data: undefined }) }));

// cmdk measures its list with ResizeObserver, which jsdom lacks.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal("ResizeObserver", ResizeObserverStub);
Element.prototype.scrollIntoView = () => {};

afterEach(() => {
  cleanup();
  navigate.mockClear();
});

describe("CommandPalette", () => {
  it("lists the Evidence, Inbox and Session logs screens and navigates on select", () => {
    render(<CommandPalette onAddHost={() => {}} />);
    fireEvent.keyDown(window, { key: "k", metaKey: true });

    for (const label of ["Evidence", "Inbox", "Session logs"]) {
      expect(SCREENS.some((s) => s.label === label)).toBe(true);
      expect(screen.getByText(`Go to ${label}`)).toBeInTheDocument();
    }
    fireEvent.click(screen.getByText("Go to Evidence"));
    expect(navigate).toHaveBeenCalledWith("/evidence");
  });

  it("offers no evidence search hits", () => {
    render(<CommandPalette onAddHost={() => {}} />);
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(screen.queryByText(/evidence search/i)).toBeNull();
    expect(screen.queryByRole("group", { name: "Evidence" })).toBeNull();
  });
});
