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
  isAvailable: boolean;
  unavailableReason?: string | null;
  imageUrl?: string | null;
}

export interface ProductDetail {
  id: string;
  slug: string;
  name: string;
  shortDescription: string;
  longDescription: string;
  category: ProductCategory;
  productType: string;
  images: ProductImage[];
  ocopCertification?: OcopCertification | null;
  foodInformation: FoodInformation;
  skus: ProductSku[];
}

interface ProductDetailErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  errors?: Array<{ field: string; message: string }>;
}

export class ProductDetailApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: ProductDetailErrorBody) {
    super(body.message || "Không thể tải chi tiết sản phẩm lúc này.");
    this.name = "ProductDetailApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.errors = body.errors ?? [];
  }
}
