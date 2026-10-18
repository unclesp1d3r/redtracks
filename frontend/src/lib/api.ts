// Single import point for the Wails bindings so features never reach into ../bindings directly.
export {
  ChecklistService,
  DiagnosticsService,
  EngagementService,
  EvidenceService,
  HostService,
  ImportService,
  SessionLogService,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/app";
export type {
  ChecklistView,
  AddResult,
  Capabilities,
  DefinitionsStatus,
  EvidenceView,
  HostEvidence,
  InboxListing,
  ListResponse,
  Notes,
  PreviewResponse,
  SearchHit,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/app";
export {
  EngagementType,
  EvidenceKind,
  EvidenceLinkKind,
  HostStatus,
  ItemStatus,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
import type {
  EvidenceItem,
  Engagement,
  Host,
  Service,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/model";
import type { PreviewResponse } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/app";
export type {
  AccessPath,
  ChecklistState,
  Credential,
  Creds,
  Engagement,
  EngagementMeta,
  EvidenceItem,
  EvidenceLink,
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
export type { HostSummary, SlotRef } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/progress";
// checklist's ItemView keeps the plain name, so the evidence one is aliased.
export type {
  InboxFile,
  ItemView as EvidenceItemView,
  LinkView,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/evidence";
export type {
  Match as SessionLogMatch,
  Source as SessionLogSource,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/sessionlogs";
export type { Summary } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/store";
export type {
  Definition,
  Item,
  ItemView,
  Match,
} from "../../bindings/github.com/unclesp1d3r/redtracks/internal/checklist";
export { Badge as ImportBadge, RowKind } from "../../bindings/github.com/unclesp1d3r/redtracks/internal/importers";
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
  return e instanceof Error ? e.message : String(e);
}

export function hostsOf(e: Engagement): Host[] {
  return e.Hosts ?? [];
}

export function hostPath(ip: string): string {
  return `/host/${encodeURIComponent(ip)}`;
}

export function serviceLabel(s: Service): string {
  return `${s.port}/${s.protocol} ${s.type}`.trim();
}

/**
 * Payload of the evidence:dropped event (internal/app.DropResult). It is an event payload, not a
 * bound return type, so the generator does not emit it; field names stay PascalCase like the Go struct.
 */
export interface DropResult {
  Added: { Path: string; Item: EvidenceItem; Duplicate: boolean }[] | null;
  Refused: { Path: string; Reason: string }[] | null;
  Preview?: PreviewResponse | null;
}
