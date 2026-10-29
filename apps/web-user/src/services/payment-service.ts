import { requestPublicJson, type PaymentInstructions, type PaymentStatus } from "@/lib/api-contract";
import { mapMvpPaymentInstructions, mapMvpPaymentStatus } from "@/lib/payment/mvp";

function pathFor(orderId: string, suffix: "instructions" | "status", mockScenario?: string) {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return `/api/payments/orders/${encodeURIComponent(orderId)}/${suffix}${params.size ? `?${params}` : ""}`;
}

export const paymentService = {
  async getInstructions(orderId: string, mockScenario?: string) {
    const response = await requestPublicJson<PaymentInstructions>(pathFor(orderId, "instructions", mockScenario), {
      credentials: "include"
    });
    return mapMvpPaymentInstructions(response.data);
  },

  async getStatus(orderId: string, mockScenario?: string) {
    const response = await requestPublicJson<PaymentStatus>(pathFor(orderId, "status", mockScenario), {
      credentials: "include"
    });
    return mapMvpPaymentStatus(response.data);
  }
};
