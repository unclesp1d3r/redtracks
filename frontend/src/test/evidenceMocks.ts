import { vi } from "vitest";
import { EvidenceKind } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
import { item, view } from "./evidenceFixtures";

/** Shared service mocks; tests mock "@/lib/api" with these (same module instance via dynamic import). */
export const apiMocks = {
  EvidenceService: {
    List: vi.fn(),
    AddFromData: vi.fn(),
    Link: vi.fn(),
    Unlink: vi.fn(),
    SetCaption: vi.fn(),
    Drop: vi.fn(),
    Adopt: vi.fn(),
    Capabilities: vi.fn(),
    Reveal: vi.fn(),
    CopyPath: vi.fn(),
  },
  HostService: { Checklists: vi.fn(), SetFlag: vi.fn(), RemoveAccessPath: vi.fn(), AddAccessPath: vi.fn() },
  EngagementService: { Current: vi.fn() },
  ImportService: { PreviewData: vi.fn(), Apply: vi.fn(), Discard: vi.fn(), Repreview: vi.fn() },
};

/** Resets every mock to a quiet default: no native integrations, empty evidence, no checklists. */
export function resetApiMocks() {
  Object.values(apiMocks).forEach((svc) => Object.values(svc).forEach((fn) => fn.mockReset()));
  const e = apiMocks.EvidenceService;
  e.Capabilities.mockResolvedValue({ NativeDrop: false, NativeDialog: false, Reveal: false });
  e.List.mockResolvedValue(view());
  e.AddFromData.mockResolvedValue({ Item: { ...item("new"), kind: EvidenceKind.EvidenceKindImage }, Duplicate: false });
  [e.Link, e.Unlink, e.SetCaption, e.Drop, e.Reveal].forEach((fn) => fn.mockResolvedValue(undefined));
  e.CopyPath.mockResolvedValue("/eng/hosts/10.0.0.1/evidence/x.png");
  apiMocks.HostService.Checklists.mockResolvedValue([]);
}
