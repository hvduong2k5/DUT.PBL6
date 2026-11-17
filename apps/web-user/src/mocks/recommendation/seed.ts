// MOCK-API: temporary seed data taken from goihue_backup.sql (products + categories only).
// Delete the whole `src/mocks/recommendation` folder when the real recommendation API serves real products.

export interface MockProduct {
  productId: string;
  /** Derived from the slug (the SQL dump has no SKU column); matches the contract pattern [A-Za-z0-9_.-]. */
  skuCode: string;
  slug: string;
  name: string;
  category: string;
  priceVnd: number;
  thumbnailUrl: string;
}

export const MOCK_PRODUCTS: readonly MockProduct[] = [
  {
    productId: "8ec163dc-1893-4d6a-b981-6c096c010554",
    skuCode: "ME-XUNG-GUNG-HUE-250G",
    slug: "me-xung-gung-hue-250g",
    name: "Mè Xửng Gừng Huế 250g",
    category: "me-xung-keo-hue",
    priceVnd: 74998,
    thumbnailUrl: "/images/product-placeholder.svg"
  },
  {
    productId: "ccd10d31-954b-4618-902e-8d883d7cd0ca",
    skuCode: "ME-XUNG-DAU-PHONG-300G",
    slug: "me-xung-dau-phong-300g",
    name: "Mè Xửng Đậu Phộng 300g",
    category: "me-xung-keo-hue",
    priceVnd: 90000,
    thumbnailUrl: "/images/product-placeholder.svg"
  },
  {
    productId: "afee0ff5-937c-410b-a156-7bfad39c6efa",
    skuCode: "BANH-IN-DAU-XANH-200G",
    slug: "banh-in-dau-xanh-200g",
    name: "Bánh In Đậu Xanh 200g",
    category: "banh-cung-dinh",
    priceVnd: 60000,
    thumbnailUrl: "/images/product-placeholder.svg"
  },
  {
    productId: "9327f6df-83bf-40a3-bf2d-25a509775e49",
    skuCode: "TRA-CUNG-DINH-TUI-LOC-100G",
    slug: "tra-cung-dinh-tui-loc-100g",
    name: "Trà Cung Đình Túi Lọc 100g",
    category: "tra-cung-dinh",
    priceVnd: 110000,
    thumbnailUrl: "/images/product-placeholder.svg"
  },
  {
    productId: "f122d60d-7453-45c6-84a2-3080a85e6cd4",
    skuCode: "HOP-QUA-HUONG-HUE",
    slug: "hop-qua-huong-hue",
    name: "Hộp Quà Hương Huế",
    category: "qua-bieu",
    priceVnd: 320000,
    thumbnailUrl: "/images/product-placeholder.svg"
  }
];
