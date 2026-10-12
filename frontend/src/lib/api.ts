// Single import point for the Wails bindings so features never reach into ../bindings directly.
export {
  ChecklistService,
  EngagementService,
  HostService,
  ImportService,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/app";
export type {
  ChecklistView,
  Notes,
  PreviewResponse,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/app";
export {
  EngagementType,
  HostStatus,
  ItemStatus,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
export type {
  AccessPath,
  ChecklistState,
  Credential,
  Creds,
  Engagement,
  EngagementMeta,
  Finding,
  Flag,
  FlagType,
  Host,
  ItemState,
  Loot,
  Service,
  Share,
  User,
  WebPath,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
export type { HostSummary } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/progress";
export type { Summary } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/store";
export type {
  Definition,
  Item,
  ItemView,
  Match,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/checklist";
export { Badge as ImportBadge } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/importers";
export type {
  Preview,
  Row,
  Unresolved,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/importers";

/** Go nil slices arrive as null; normalise to an empty array. */
export async function list<T>(p: Promise<T[] | null>): Promise<T[]> {
  return (await p) ?? [];
}

export function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  return String(e);
}
