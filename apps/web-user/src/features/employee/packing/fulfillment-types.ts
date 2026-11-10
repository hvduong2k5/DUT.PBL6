import type { IdentityProfile } from "@/lib/auth/identity";

// Mirrors the integrated Fulfillment ADMIN mapping; the service checks scope again.
export function canUseFulfillment(profile: IdentityProfile): boolean {
  return profile.principalType === "EMPLOYEE" && profile.provisioningStatus === "ACTIVE"
    && profile.roles.some((role) => role.roleCode === "ADMIN"
      && (role.scopeType === "GLOBAL" || (role.scopeType === "WAREHOUSE" && Boolean(role.scopeId))));
}

export interface FulfillmentSummary {
  task_id: string; order_id: string; workflow_state: string; version: number;
  warehouse_id: string; assignee_id: string | null; packing_due_at: string | null;
  admitted_at: string; channel: string; line_count: number; total_quantity: number;
  priority_reason: string; active_block_count: number;
}
export interface Parcel { weight_grams: number; length_cm: number; width_cm: number; height_cm: number; fragile: boolean }
export interface FulfillmentTask extends FulfillmentSummary {
  evidence_mode: string;
  items: Array<{ order_line_id: string; sku_code: string; product_name: string; packaging_text: string; quantity: number }>;
  allocations: Array<{ allocation_id: string; order_line_id: string; batch_code: string; allocated_quantity: number; counted_quantity: number; allocation_version: number; batch_valid_until: string; currently_eligible: boolean }>;
  checklist: Array<{ item_code: string; definition: string; required: boolean; result: string; confirmed_by: string | null; confirmed_at: string | null; note: string | null }>;
  active_blocks: Array<{ block_id: string; reason_code: string; source: string }>;
  package: { package_id: string; state: string; seal_code: string | null; parcel: Parcel | null; primary_evidence_id: string | null };
  integration: { order_source: string; shipping: string };
  queue_as_of: string;
}
export interface FulfillmentPage { items: FulfillmentSummary[]; next_cursor: string | null; queue_as_of: string }
export interface FulfillmentHistoryPage { items: Array<{ history_id: string; task_version: number; action: string; actor_id: string; reason: string | null; occurred_at: string }>; next_cursor: string | null }
export interface PackingMedia { file_id: string; task_id?: string; package_id?: string; status: string; verified_sha256?: string | null }
export interface PendingPackingCommand {
  taskId: string; action: string; method: "POST" | "PUT"; body: Record<string, unknown>; version: number; key: string;
}
export class FulfillmentError extends Error {
  constructor(public status: number, public code: string, message: string) { super(message); }
}
