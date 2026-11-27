/**
 * Device-local memory of orders placed as a guest on this browser.
 * The storefront has no guest order list upstream, so after checkout the
 * only durable pointer to a guest order is kept here; the order detail
 * itself is still read from the server under the guest session.
 */
export interface GuestDeviceOrder {
  orderId: string;
  orderCode: string;
  totalVnd?: number;
  placedAt: string;
}

const STORAGE_KEY = "oma_guest_orders_v1";
const MAX_ENTRIES = 20;

function isGuestDeviceOrder(value: unknown): value is GuestDeviceOrder {
  if (!value || typeof value !== "object") return false;
  const entry = value as Record<string, unknown>;
  return typeof entry.orderId === "string" && entry.orderId.length > 0
    && typeof entry.orderCode === "string" && entry.orderCode.length > 0
    && typeof entry.placedAt === "string";
}

export function listGuestOrders(): GuestDeviceOrder[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isGuestDeviceOrder);
  } catch {
    return [];
  }
}

export function recordGuestOrder(entry: GuestDeviceOrder): void {
  if (typeof window === "undefined") return;
  try {
    const rest = listGuestOrders().filter((order) => order.orderId !== entry.orderId);
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify([entry, ...rest].slice(0, MAX_ENTRIES)));
  } catch {
    // Storage full or unavailable: losing the convenience pointer is non-fatal.
  }
}
