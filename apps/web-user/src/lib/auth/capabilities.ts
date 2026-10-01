export const CUSTOMER_CAPABILITIES = [
  "CATALOG_VIEW",
  "CART_MANAGE",
  "CHECKOUT_CREATE",
  "GUEST_ORDER_TRACK",
  "SUPPORT_CREATE",
  "PROFILE_MANAGE",
  "ADDRESS_MANAGE",
  "ORDER_HISTORY_VIEW",
  "LOYALTY_VIEW",
  "REVIEW_CREATE",
  "B2B_COMPANY_VIEW",
  "B2B_QUOTE_CREATE",
  "B2B_QUOTE_VIEW",
  "B2B_QUOTE_ACCEPT"
] as const;

export type CustomerCapability = (typeof CUSTOMER_CAPABILITIES)[number];
export type CustomerActor = "GUEST" | "REGISTERED" | "B2B" | "MARKETPLACE" | "OFFLINE";

export interface CustomerCapabilityProjection {
  actor: CustomerActor;
  capabilities: CustomerCapability[];
  version: number;
}

const ACTORS = new Set<CustomerActor>(["GUEST", "REGISTERED", "B2B", "MARKETPLACE", "OFFLINE"]);
const CAPABILITIES = new Set<CustomerCapability>(CUSTOMER_CAPABILITIES);

export function parseCustomerCapabilityProjection(value: unknown): CustomerCapabilityProjection | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  const source = value as Record<string, unknown>;
  if (!ACTORS.has(source.actor as CustomerActor) || !Array.isArray(source.capabilities) || !Number.isInteger(source.version)) return undefined;
  const capabilities = source.capabilities.filter((item): item is CustomerCapability => typeof item === "string" && CAPABILITIES.has(item as CustomerCapability));
  if (capabilities.length !== source.capabilities.length) return undefined;
  return { actor: source.actor as CustomerActor, capabilities: [...new Set(capabilities)], version: source.version as number };
}
