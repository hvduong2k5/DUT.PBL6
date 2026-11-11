export type Placement = "product_detail" | "cart" | "home";

export interface RecommendationItem {
  productId: string;
  skuCode: string;
  slug: string;
  name: string;
  thumbnailUrl: string;
  priceVnd: number;
  rank: number;
  algorithmVersion: string;
  reasonCode: string;
}

export interface Recommendations {
  recommendationId: string;
  placement: Placement;
  strategy: "RELATED" | "BOUGHT_TOGETHER" | "PERSONALIZED" | "FALLBACK" | "EMPTY";
  reasonCodes: string[];
  items: RecommendationItem[];
}

export interface RecommendationConsent {
  enabled: boolean;
  consentVersion: number;
  policyVersion: string;
  grantedAt: string | null;
  revokedAt: string | null;
}

export interface RecommendationContext {
  placement: Placement;
  productId?: string;
  skuCode?: string;
  skuCodes?: string[];
}
