import { requestPublicJson } from "@/lib/api-contract";
import type { RecommendationConsent, RecommendationContext, Recommendations } from "@/lib/recommendation/types";

export const recommendationsEnabled = process.env.NEXT_PUBLIC_RECOMMENDATIONS_ENABLED === "true";

function csrfHeader() {
  const token = document.cookie.split("; ").find((part) => part.startsWith("oma_rec_csrf="))?.slice("oma_rec_csrf=".length);
  return { "X-CSRF-Token": token ?? "" };
}

export const recommendationService = {
  async list(context: RecommendationContext, signal: AbortSignal): Promise<Recommendations> {
    const params = new URLSearchParams({ placement: context.placement, limit: "6" });
    if (context.productId) params.set("productId", context.productId);
    if (context.skuCode) params.set("skuCode", context.skuCode);
    for (const code of context.skuCodes ?? []) params.append("skuCodes", code);
    const result = await requestPublicJson<Recommendations>(`/api/recommendations?${params}`, { signal, credentials: "include" });
    if (!result.data || !Array.isArray(result.data.items) || typeof result.data.recommendationId !== "string"
      || result.data.items.some((item) => typeof item.skuCode !== "string" || typeof item.slug !== "string"
        || typeof item.name !== "string" || !Number.isSafeInteger(item.priceVnd) || item.priceVnd < 1)) {
      throw new Error("Invalid recommendation response");
    }
    return result.data;
  },

  async consent(): Promise<RecommendationConsent> {
    return (await requestPublicJson<RecommendationConsent>("/api/recommendations/consent", { credentials: "include" })).data;
  },

  async setConsent(enabled: boolean, current: RecommendationConsent): Promise<RecommendationConsent> {
    return (await requestPublicJson<RecommendationConsent>("/api/recommendations/consent", {
      method: "PUT", credentials: "include", headers: csrfHeader(),
      ifMatch: `"${current.consentVersion}"`, idempotencyKey: crypto.randomUUID(),
      body: { enabled, policyVersion: "personalization-v1" }
    })).data;
  },

  async event(body: object) {
    return requestPublicJson("/api/recommendations/events", {
      method: "POST", credentials: "include", headers: csrfHeader(), body
    });
  }
};
