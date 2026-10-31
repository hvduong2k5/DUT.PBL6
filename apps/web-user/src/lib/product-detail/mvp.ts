import { parseVndMoney } from "@/lib/api-contract";
import type { CatalogProductDetail } from "@/lib/api-contract";
import type { ProductAvailability } from "@/lib/catalog/types";
import type { ProductDetail } from "./types";

function availability(stock: number | null): ProductAvailability {
  if (stock === null) return "UNKNOWN";
  return stock > 0 ? "AVAILABLE" : "UNAVAILABLE";
}

export function mapCatalogProductDetail(product: CatalogProductDetail): ProductDetail {
  return {
    id: product.product_id,
    slug: product.slug,
    name: product.name,
    shortDescription: product.description,
    longDescription: product.story || product.description,
    images: product.images.map((url, index) => ({
      id: `${product.product_id}-${index + 1}`,
      url,
      alt: `${product.name} — ảnh ${index + 1}`
    })),
    ocopCertification: product.ocop_star > 0 ? {
      stars: product.ocop_star,
      label: `OCOP ${product.ocop_star} sao${product.ocop_certificate_no ? ` · ${product.ocop_certificate_no}` : ""}`
    } : null,
    skus: product.variants.map((variant) => {
      const state = availability(variant.stock_available);
      return {
        skuId: variant.sku_code,
        label: variant.name,
        weightGrams: variant.weight_gram,
        flavor: null,
        packageType: variant.name,
        priceVnd: parseVndMoney(variant.price),
        availability: state,
        unavailableReason: state === "UNKNOWN" ? "Tồn kho đang được đồng bộ" : state === "UNAVAILABLE" ? "Tạm hết hàng" : null,
        imageUrl: product.images[0] ?? null
      };
    })
  };
}
