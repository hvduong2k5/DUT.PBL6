import { formatDecimalVersion, requestPublicJson } from "@/lib/api-contract";
import type { OrderCancellation, OrderTrackingResponse, OwnedOrder, OwnedOrdersResponse } from "@/lib/api-contract";
import { mapMvpCancellation, mapMvpOrderDetail, mapMvpOrderListResponse } from "@/lib/orders/mvp";
import type {
  CancelOrderInput,
  GuestChallenge,
  GuestVerification,
  OrderListResponse,
  OrderStatus
} from "@/lib/orders/types";

type OrderStatusFilter = OrderStatus | "ALL" | "CANCELLED";

function endpoint(path: string, mockScenario?: string): string {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return `/api/orders${path}${params.size ? `?${params}` : ""}`;
}

function matchesStatus(status: OrderStatus, filter: OrderStatusFilter): boolean {
  return filter === "ALL" || status === filter || (filter === "CANCELLED" && status.startsWith("CANCELLED_"));
}

export const orderService = {
  async list(input: { q?: string; status?: OrderStatusFilter; year?: number; mockScenario?: string }): Promise<OrderListResponse> {
    const response = await requestPublicJson<OwnedOrdersResponse>(endpoint("", input.mockScenario), { credentials: "include" });
    const mapped = mapMvpOrderListResponse(response.data);
    const query = input.q?.trim().toLocaleLowerCase("vi") ?? "";
    const status = input.status ?? "ALL";
    const items = mapped.items.filter((item) => (
      matchesStatus(item.status, status)
      && (!input.year || new Date(item.placedAt).getFullYear() === input.year)
      && (!query || item.orderNumber.toLocaleLowerCase("vi").includes(query)
        || item.previewItems.some((preview) => preview.name.toLocaleLowerCase("vi").includes(query)))
    ));
    return { ...mapped, items, resultCount: items.length };
  },

  async detail(orderId: string, mockScenario?: string) {
    const response = await requestPublicJson<OwnedOrder>(endpoint(`/${encodeURIComponent(orderId)}`, mockScenario), { credentials: "include" });
    return mapMvpOrderDetail(response.data);
  },

  async tracking(orderId: string, mockScenario?: string): Promise<OrderTrackingResponse> {
    const response = await requestPublicJson<OrderTrackingResponse>(endpoint(`/${encodeURIComponent(orderId)}/tracking`, mockScenario), { credentials: "include" });
    return response.data;
  },

  async cancel(orderId: string, input: CancelOrderInput, mockScenario?: string) {
    const reasonLabels = {
      CHANGED_MIND: "Thay đổi nhu cầu",
      WRONG_INFORMATION: "Thông tin đặt hàng chưa đúng",
      OTHER: "Lý do khác"
    } as const;
    const reason = [reasonLabels[input.reasonCode], input.note?.trim()].filter(Boolean).join(": ").slice(0, 255);
    const response = await requestPublicJson<OrderCancellation>(endpoint(`/${encodeURIComponent(orderId)}/cancel`, mockScenario), {
      method: "POST",
      credentials: "include",
      idempotencyKey: input.idempotencyKey,
      ifMatch: formatDecimalVersion(input.version),
      body: { active: true, reason }
    });
    return mapMvpCancellation(response.data);
  },

  async createGuestChallenge(input: { orderNumber: string; contact: string }, mockScenario?: string) {
    const response = await requestPublicJson<GuestChallenge>(endpoint("/guest-access/challenges", mockScenario), {
      method: "POST",
      credentials: "include",
      body: input
    });
    return response.data;
  },

  async verifyGuestChallenge(challengeId: string, otp: string, mockScenario?: string) {
    const response = await requestPublicJson<GuestVerification>(endpoint(`/guest-access/challenges/${encodeURIComponent(challengeId)}/verify`, mockScenario), {
      method: "POST",
      credentials: "include",
      body: { otp }
    });
    return response.data;
  }
};
