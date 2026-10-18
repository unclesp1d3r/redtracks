import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { success: vi.fn(), warning: vi.fn(), error: vi.fn() }) }));

import { toast } from "sonner";
import { EvidenceKind } from "@/lib/api";
import { captionFocus, reportAdded } from "./capture";

const added = { id: "e2", file: "b.png", kind: EvidenceKind.EvidenceKindImage, capturedAt: "", caption: "", sha256: "abc", links: [] };

describe("reportAdded", () => {
  beforeEach(() => vi.clearAllMocks());

  it("offers an Add caption action that requests focus on the new item", () => {
    reportAdded(new QueryClient(), added, false);
    const opts = vi.mocked(toast.success).mock.calls[0][1] as { action: { label: string; onClick: () => void } };
    expect(opts.action.label).toBe("Add caption");
    opts.action.onClick();
    expect(captionFocus.get()).toBe("e2");
    captionFocus.clear();
  });

  it("names the identical existing item in the duplicate warning", () => {
    const qc = new QueryClient();
    qc.setQueryData(["evidence", "10.0.0.1"], {
      items: [{ ...added, id: "e1", file: "a.png", caption: "login page" }],
      notIndexed: [],
    });
    reportAdded(qc, added, true);
    expect(vi.mocked(toast.warning).mock.calls[0][0]).toContain("login page");
  });
});
