import type { ProductAvailability } from "@/lib/catalog/types";

export interface ProductImage {
  id: string;
  url: string;
  alt: string;
}

export interface ProductCategory {
  slug: string;
  name: string;
}

export interface OcopCertification {
  stars: number;
  label: string;
}

export interface FoodInformation {
  ingredients: string;
  allergenStatement?: string | null;
  storageInstructions: string;
  manufacturingDatePolicy: string;
  shelfLifeDescription: string;
}

export interface ProductSku {
  skuId: string;
  label: string;
  weightGrams: number;
  flavor?: string | null;
  packageType: string;
  priceVnd: number;
  availability: ProductAvailability;
  unavailableReason?: string | null;
  imageUrl?: string | null;
}

export interface ProductDetail {
  id: string;
  slug: string;
  name: string;
  shortDescription: string;
  longDescription: string;
  category?: ProductCategory;
  productType?: string;
  images: ProductImage[];
  ocopCertification?: OcopCertification | null;
  foodInformation?: FoodInformation;
  skus: ProductSku[];
}
