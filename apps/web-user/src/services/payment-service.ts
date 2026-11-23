import { PublicApiError, requestPublicJson, type PaymentInstructions, type PaymentStatus } from "@/lib/api-contract";
import { mapMvpPaymentInstructions, mapMvpPaymentStatus } from "@/lib/payment/mvp";

function pathFor(orderId: string, suffix: "instructions" | "status", mockScenario?: string) {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return `/api/payments/orders/${encodeURIComponent(orderId)}/${suffix}${params.size ? `?${params}` : ""}`;
}

const pendingGrants = new Map<string, Promise<void>>();
async function ensureGuestAccess(orderId: string) {
  let pending = pendingGrants.get(orderId);
  if (!pending) {
    pending = requestPublicJson<void>(`/api/payments/orders/${encodeURIComponent(orderId)}/guest-access`, { method: "POST", credentials: "include", body: {} }).then(() => undefined).finally(() => pendingGrants.delete(orderId));
    pendingGrants.set(orderId, pending);
  }
  await pending;
}
async function paymentRead<T>(orderId: string, suffix: "instructions" | "status", scenario?: string) {
  const read = () => requestPublicJson<T>(pathFor(orderId, suffix, scenario), { credentials: "include" });
  try { return await read(); }
  catch(cause) {
    if (!(cause instanceof PublicApiError) || cause.status !== 401 || cause.code !== "GUEST_PAYMENT_ACCESS_REQUIRED") throw cause;
    await ensureGuestAccess(orderId);
    return read();
  }
}

export const paymentService = {
  async getInstructions(orderId: string, mockScenario?: string) {
    const response = await paymentRead<PaymentInstructions>(orderId, "instructions", mockScenario);
    return mapMvpPaymentInstructions(response.data);
  },

  async getStatus(orderId: string, mockScenario?: string) {
    const response = await paymentRead<PaymentStatus>(orderId, "status", mockScenario);
    return mapMvpPaymentStatus(response.data);
  }
};
