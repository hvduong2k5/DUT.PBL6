export interface Evidence { id: string; reference?: string; kind: string; description: string }
export interface Certificate {
  id: string; type: "OCOP" | "ATTP" | "OTHER"; number: string; issuer: string;
  scope: "PRODUCT" | "SKU" | "BATCH"; scope_id: string; issued_on: string;
  valid_until: string | null; no_expiry_stated: boolean; review_due_at: string | null;
  stars: number | null; verification_method: "MANUAL"; verified_at: string; revoked: boolean; evidence_ids: string[];
}
export interface TraceContent {
  product_name: string; producer: string; workshop_location: string; artisan_name: string | null;
  process: string; ingredients: Array<{ name: string; origin: string }>;
  certificates: Certificate[]; evidence: Evidence[]; crafting_video_url: string | null;
}
export interface TraceRevision {
  number: number; status: "DRAFT" | "IN_REVIEW" | "APPROVED" | "REJECTED" | "PUBLISHED" | "ARCHIVED";
  content: TraceContent; contributors: string[]; created_by: string;
}
export interface TraceBinding { product_id: string; sku_code: string; source_namespace: string; batch_id: string }
export interface TraceProfile extends TraceBinding {
  id: string; warehouse_id?: string; qr_code: string; qr_url?: string; status: "ACTIVE" | "SUSPENDED" | "ARCHIVED";
  version: number; public_revision: number | null; revisions: TraceRevision[]; ever_published: boolean;
}
export interface TraceBatchContext extends TraceBinding {
  warehouse_id: string; batch_code: string; production_date: string; expiry_date: string;
  restricted: boolean; depleted: boolean; catalog_version: string; batch_version: string;
}
export interface TraceProfilePage { items: TraceProfile[]; next_cursor: string | null; total: number }
export class TraceApiError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) { super(message); }
}
export function emptyTraceContent(): TraceContent {
  return { product_name: "", producer: "", workshop_location: "", artisan_name: null, process: "", ingredients: [], certificates: [], evidence: [], crafting_video_url: null };
}
