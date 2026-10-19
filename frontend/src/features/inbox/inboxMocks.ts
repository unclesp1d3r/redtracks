import { vi } from "vitest";
import { EvidenceKind } from "../../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
import { apiMocks } from "@/test/evidenceMocks";

// Enums come from the bindings, not @/lib/api (mocked in these tests).
export const inboxApi = {
  Inbox: vi.fn(),
  Assign: vi.fn(),
  DiscardInboxFile: vi.fn(),
  Root: vi.fn(),
};
Object.assign(apiMocks.EvidenceService, { Inbox: inboxApi.Inbox, Assign: inboxApi.Assign, DiscardInboxFile: inboxApi.DiscardInboxFile });
Object.assign(apiMocks.EngagementService, { Root: inboxApi.Root });

export const TOKEN = "tok-1";

export function inboxFile(name: string, extra: Partial<{ Size: number; Kind: EvidenceKind; Preview: string }> = {}) {
  return { Name: name, Size: 2048, ModTime: "2026-10-01T12:00:00Z", Kind: EvidenceKind.EvidenceKindImage, Preview: "", ...extra };
}

export const listing = (files = [inboxFile("shot.png")], token = TOKEN) => ({ Token: token, Files: files });
