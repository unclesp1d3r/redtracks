import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return { ...actual, ChecklistService: { Command: vi.fn() } };
});

import { ItemStatus } from "@/lib/api";
import type { ChecklistView, ItemView } from "@/lib/api";
import { Checklist } from "./Checklist";

function views(status: ItemStatus, noteOfB = ""): ChecklistView[] {
  const item = (id: string, s: ItemStatus, note = ""): ItemView => ({
    id, title: `Item ${id}`, hint: "", commands: [],
    State: { status: s, note, updatedAt: "" },
  });
  return [
    {
      Definition: { id: "d", title: "Def", match: { serviceTypes: [], hostOS: [] }, items: [] },
      Items: [item("a", ItemStatus.ItemTodo), item("b", status, noteOfB), item("c", ItemStatus.ItemTodo)],
    },
  ];
}

describe("Checklist", () => {
  afterEach(cleanup);
  it("space toggles done then todo on the focused row", () => {
    const onSet = vi.fn();
    const { rerender } = render(<Checklist views={views(ItemStatus.ItemTodo)} onSet={onSet} targetIp="10.0.0.1" domain="" />);
    fireEvent.keyDown(screen.getByTestId("row-b"), { key: " " });
    expect(onSet).toHaveBeenLastCalledWith("b", "done", "");
    rerender(<Checklist views={views(ItemStatus.ItemDone)} onSet={onSet} targetIp="10.0.0.1" domain="" />);
    fireEvent.keyDown(screen.getByTestId("row-b"), { key: " " });
    expect(onSet).toHaveBeenLastCalledWith("b", "todo", "");
    expect(onSet).toHaveBeenCalledTimes(2);
  });

  it("ignores space inside the note textarea", () => {
    const onSet = vi.fn();
    render(<Checklist views={views(ItemStatus.ItemTodo)} onSet={onSet} targetIp="10.0.0.1" domain="" />);
    fireEvent.click(screen.getByLabelText("Expand Item b"));
    const note = screen.getByLabelText("Note for Item b");
    note.focus();
    fireEvent.keyDown(note, { key: " " });
    expect(onSet).not.toHaveBeenCalled();
  });

  it("follows an external note change but keeps text the user is typing", () => {
    const props = { onSet: vi.fn(), targetIp: "10.0.0.1", domain: "" };
    const { rerender } = render(<Checklist views={views(ItemStatus.ItemTodo, "old")} {...props} />);
    fireEvent.click(screen.getByLabelText("Expand Item b"));
    const note = screen.getByLabelText("Note for Item b") as HTMLTextAreaElement;
    expect(note.value).toBe("old");

    rerender(<Checklist views={views(ItemStatus.ItemTodo, "edited on disk")} {...props} />);
    expect(note.value).toBe("edited on disk");

    fireEvent.change(note, { target: { value: "typing" } });
    rerender(<Checklist views={views(ItemStatus.ItemTodo, "edited again")} {...props} />);
    expect(note.value).toBe("typing");
  });
});
