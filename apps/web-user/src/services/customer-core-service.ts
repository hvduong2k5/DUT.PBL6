import type {
  CreateReviewInput,
  LoyaltyPoints,
  ProductReview,
  ProductReviews,
  TraceabilityRecord,
  VoucherList,
  VoucherValidation
} from "@/lib/customer-core/types";
import { CustomerCoreApiError } from "@/lib/customer-core/types";

async function request<T>(path: string, init: RequestInit = {}, mockScenario?: string): Promise<T> {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  const response = await fetch(`/api/customer-core/${path}${params.size ? `?${params}` : ""}`, {
    ...init,
    cache: "no-store",
    credentials: "include",
    headers: init.body ? { "Content-Type": "application/json", ...init.headers } : init.headers
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new CustomerCoreApiError(response.status, payload);
  return payload as T;
}

export const customerCoreService = {
  getProductReviews(productId: string, mockScenario?: string) {
    return request<ProductReviews>(`reviews/products/${encodeURIComponent(productId)}`, {}, mockScenario);
  },
  createReview(input: CreateReviewInput, mockScenario?: string) {
    return request<ProductReview>("reviews", {
      method: "POST",
      body: JSON.stringify(input)
    }, mockScenario);
  },
  getTraceability(qrCode: string, mockScenario?: string) {
    return request<TraceabilityRecord>(`trace/${encodeURIComponent(qrCode)}`, {}, mockScenario);
  },
  getVouchers(mockScenario?: string) {
    return request<VoucherList>("promotions/vouchers", {}, mockScenario);
  },
  validateVoucher(input: { voucher_code: string; subtotal_amount: number }, mockScenario?: string) {
    return request<VoucherValidation>("promotions/validate", {
      method: "POST",
      body: JSON.stringify(input)
    }, mockScenario);
  },
  getLoyaltyPoints(mockScenario?: string) {
    return request<LoyaltyPoints>("loyalty/points", {}, mockScenario);
  }
};
