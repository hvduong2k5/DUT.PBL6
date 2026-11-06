export type ProductSaleStatus = "ON_SALE" | "PAUSED" | "DISCONTINUED";
export type SkuSaleStatus = "ON_SALE" | "PAUSED" | "DISCONTINUED";

export interface AdminProductSummary {
  productId: string;
  name: string;
  categoryId: string;
  categoryName: string;
  saleStatus: ProductSaleStatus;
  saleStatusLabel: string;
  skuCount: number;
  onSaleSkuCount: number;
  basePriceFromVnd: number | null;
  foodInformationComplete: boolean;
  updatedAt: string;
  revision: number;
}

export interface AdminProductList {
  items: AdminProductSummary[];
  statusCounts: Array<{ status: "ALL" | ProductSaleStatus; label: string; count: number }>;
  categories: Array<{ categoryId: string; categoryName: string }>;
  calculatedAt: string;
}

export interface AdminProductSku {
  skuId: string;
  skuCode: string;
  label: string;
  weightGrams: number | null;
  flavor: string | null;
  packageType: string;
  basePriceVnd: number;
  currency: "VND";
  saleStatus: SkuSaleStatus;
  saleStatusLabel: string;
  updatedAt: string;
  revision: number;
}

export interface AdminProductDetail extends AdminProductSummary {
  shortDescription: string;
  longDescription: string;
  coverImageUrl: string | null;
  coverImageAlt: string;
  foodInformation: null | {
    ingredients: string;
    allergenStatement: string | null;
    storageInstructions: string;
    manufacturingDatePolicy: string;
    shelfLifeDescription: string;
  };
  skus: AdminProductSku[];
}

interface CatalogErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminCatalogApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: CatalogErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu danh mục sản phẩm.");
    this.name = "AdminCatalogApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
  }
}
