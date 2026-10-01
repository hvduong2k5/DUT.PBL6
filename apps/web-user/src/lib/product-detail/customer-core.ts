import type { ProductDetail } from "./types";

export interface CustomerCoreProductDetail {
  product_id: string;
  name: string;
  slug: string;
  description: string;
  images: string[];
  ocop_star: number | null;
  ocop_certificate_no: string | null;
  story: string | null;
  variants: Array<{
    variant_id: string;
    sku_code: string;
    name: string;
    weight_gram: number;
    price: { currency_code: string; units: number; nanos: number };
    original_price: { currency_code: string; units: number; nanos: number };
    stock_available: number;
  }>;
}

export function mapCustomerCoreProductDetail(product: CustomerCoreProductDetail): ProductDetail {
  return {
    id: product.product_id,
    slug: product.slug,
    name: product.name,
    shortDescription: product.description,
    longDescription: product.story || product.description,
    images: product.images.map((url, index) => ({ id: `${product.product_id}-${index + 1}`, url, alt: `${product.name} — ảnh ${index + 1}` })),
    ocopCertification: product.ocop_star ? {
      stars: product.ocop_star,
      label: `OCOP ${product.ocop_star} sao${product.ocop_certificate_no ? ` · ${product.ocop_certificate_no}` : ""}`
    } : null,
    skus: product.variants.map((variant) => ({
      skuId: variant.sku_code,
      label: variant.name,
      weightGrams: variant.weight_gram,
      flavor: null,
      packageType: variant.name,
      priceVnd: variant.price.units,
      isAvailable: variant.stock_available > 0,
      unavailableReason: variant.stock_available > 0 ? null : "Tạm hết hàng",
      imageUrl: product.images[0] ?? null
    }))
  };
}
