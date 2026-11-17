// MOCK-API: simulates GET /api/recommendations, GET|PUT /api/recommendations/consent and
// POST /api/recommendations/events with the same response shapes as the real contract
// (docs/03_api_specs/analytics-recommendation.candidate.openapi.yaml).
//
// How it is used: `recommendation-service.ts` calls these only when the real API is not enabled,
// fails, or returns no items. To remove: delete this folder and every block marked `MOCK-API`
// in `src/services/recommendation-service.ts`.

import type { RecommendationConsent, RecommendationContext, RecommendationItem, Recommendations } from "@/lib/recommendation/types";
import { MOCK_PRODUCTS, type MockProduct } from "./seed";

const POLICY_VERSION = "personalization-v1";
const CONSENT_KEY = "mock_rec_consent";
const LIMIT = 6;

function toItem(product: MockProduct, rank: number, algorithmVersion: string, reasonCode: string): RecommendationItem {
  return {
    productId: product.productId,
    skuCode: product.skuCode,
    slug: product.slug,
    name: product.name,
    thumbnailUrl: product.thumbnailUrl,
    priceVnd: product.priceVnd,
    rank,
    algorithmVersion,
    reasonCode
  };
}

function readConsent(): RecommendationConsent {
  const fallback: RecommendationConsent = { enabled: false, consentVersion: 0, policyVersion: POLICY_VERSION, grantedAt: null, revokedAt: null };
  try {
    const raw = globalThis.sessionStorage?.getItem(CONSENT_KEY);
    return raw ? { ...fallback, ...(JSON.parse(raw) as Partial<RecommendationConsent>) } : fallback;
  } catch {
    return fallback;
  }
}

export const mockRecommendationApi = {
  list(context: RecommendationContext): Recommendations {
    const exclude = new Set<string>([...(context.skuCodes ?? []), ...(context.skuCode ? [context.skuCode] : [])]);
    const anchor = MOCK_PRODUCTS.find((product) => product.productId === context.productId);
    let pool = MOCK_PRODUCTS.filter((product) => product.productId !== context.productId && !exclude.has(product.skuCode));
    let strategy: Recommendations["strategy"] = "FALLBACK";
    let algorithmVersion = "fallback_catalog_v1";
    let reasonCode = "MODEL_NOT_READY";

    if (context.placement === "product_detail") {
      // Same category first, like related_content_v1.
      pool = [...pool].sort((a, b) => Number(b.category === anchor?.category) - Number(a.category === anchor?.category));
      strategy = "RELATED";
      algorithmVersion = "related_content_v1";
      reasonCode = "SIMILAR_ATTRIBUTES";
    } else if (context.placement === "cart") {
      strategy = "BOUGHT_TOGETHER";
      algorithmVersion = "basket_pairs_v1";
      reasonCode = "FREQUENTLY_BOUGHT_TOGETHER";
    } else if (readConsent().enabled) {
      strategy = "PERSONALIZED";
      algorithmVersion = "personalized_v1";
      reasonCode = "CATEGORY_AFFINITY";
    }

    const items = pool.slice(0, LIMIT).map((product, index) => toItem(product, index + 1, algorithmVersion, reasonCode));
    return {
      recommendationId: crypto.randomUUID(),
      placement: context.placement,
      strategy: items.length ? strategy : "EMPTY",
      reasonCodes: strategy === "FALLBACK" ? [reasonCode] : [],
      items
    };
  },

  consent(): RecommendationConsent {
    return readConsent();
  },

  setConsent(enabled: boolean): RecommendationConsent {
    const current = readConsent();
    if (current.enabled === enabled) return current;
    const now = new Date().toISOString();
    const next: RecommendationConsent = {
      enabled,
      consentVersion: current.consentVersion + 1,
      policyVersion: POLICY_VERSION,
      grantedAt: enabled ? now : current.grantedAt,
      revokedAt: enabled ? null : now
    };
    try { globalThis.sessionStorage?.setItem(CONSENT_KEY, JSON.stringify(next)); } catch { /* storage unavailable */ }
    return next;
  },

  event(): { status: "ACCEPTED" } {
    return { status: "ACCEPTED" };
  }
};
