# Product Detail & SKU Selection MVP — Data & Mock API Matrix

## Capability inventory

Browser chỉ gọi Next.js same-origin BFF. BFF dùng server-only upstream và không để lộ URL Mockoon/API Gateway.

| Capability ID | Method | Browser/BFF route | Mockoon route | Mục đích |
| --- | --- | --- | --- | --- |
| `DETAIL-C01` | GET | `/api/catalog/products/{slug}` | `/api/v1/catalog/products/{slug}` | Lấy public product detail và toàn bộ SKU public. |
| Local health | GET | Không expose qua BFF | `/api/v1/health` | Kiểm tra Mockoon local; không thuộc API contract. |

## Request và validation

`DETAIL-C01` không có body và không nhận business query. BFF chỉ chấp nhận:

- Method `GET`.
- Chính xác path `products/{slug}`, không cho nested path.
- `slug` lowercase kebab-case, dài 1–120: `^[a-z0-9]+(?:-[a-z0-9]+)*$`.
- `mockScenario` chỉ trong development/test, khớp `^[a-z0-9-]+$`; không forward ở production.
- Query khác bị từ chối để public API không phụ thuộc cú pháp Mockoon.

## Public response

```json
{
  "id": "PRD-001",
  "slug": "banh-ngu-sac-cung-dinh",
  "name": "Bánh Ngũ Sắc Cung Đình",
  "shortDescription": "Năm sắc bánh thanh nhã từ đậu xanh và hạt sen.",
  "longDescription": "Thức bánh mềm mịn từ nguyên liệu được tuyển chọn.",
  "category": { "slug": "banh-cung-dinh", "name": "Bánh Cung Đình" },
  "productType": "CAKE",
  "images": [
    { "id": "IMG-001", "url": "https://...", "alt": "Bánh ngũ sắc trong hộp quà" }
  ],
  "ocopCertification": { "stars": 4, "label": "OCOP 4 sao" },
  "foodInformation": {
    "ingredients": "Đậu xanh, bột năng, đường, nước cốt dừa.",
    "allergenStatement": "Có chứa dừa.",
    "storageInstructions": "Bảo quản nơi khô ráo, tránh nắng trực tiếp.",
    "manufacturingDatePolicy": "Ngày sản xuất được in trên bao bì sản phẩm.",
    "shelfLifeDescription": "45 ngày kể từ ngày sản xuất."
  },
  "skus": [
    {
      "skuId": "SKU-001-250",
      "label": "Hộp 250g",
      "weightGrams": 250,
      "flavor": "Ngũ sắc truyền thống",
      "packageType": "Hộp giấy",
      "priceVnd": 185000,
      "isAvailable": true,
      "unavailableReason": null,
      "imageUrl": "https://..."
    }
  ]
}
```

Validation response assumptions:

- `priceVnd`: integer `>= 0`, VND.
- `weightGrams`: integer `>= 1`.
- `images`: ít nhất một ảnh product đã approved; SKU image optional.
- `skus`: có thể gồm available và unavailable SKU đã được phép public; không gồm draft/internal SKU.
- Optional: `ocopCertification`, `allergenStatement`, SKU `flavor`, SKU `imageUrl`.
- Các field food bắt buộc để publish do nghiệp vụ/backend enforce; scenario thiếu optional không được dùng để hợp thức hóa việc thiếu field bắt buộc.

## HTTP/error matrix

| HTTP | Code | Khi nào | UI state |
| --- | --- | --- | --- |
| `200` | — | Product public được tìm thấy | Detail hoặc all-unavailable state. |
| `400` | `INVALID_PRODUCT_SLUG` | Slug/query không hợp lệ tại BFF | Error state an toàn + link về catalog. |
| `404` | `PRODUCT_NOT_FOUND` | Không tồn tại, unpublished hoặc không được phép public | Not-found state; không phân biệt nguyên nhân nội bộ. |
| `500` | `PRODUCT_DETAIL_ERROR` | Upstream lỗi | Error + retry. |
| `503` | `CATALOG_UPSTREAM_UNAVAILABLE` | BFF không kết nối upstream | Error + retry, hướng dẫn dev kiểm tra Mockoon. |

## Mock scenarios

Thêm `?mockScenario=<name>` vào URL page trong local development. BFF chuyển thành `X-Mock-Scenario` tới Mockoon port `4013`.

| Scenario | HTTP | Dữ liệu/state |
| --- | --- | --- |
| mặc định / `banh-ngu-sac-cung-dinh` | `200` | Nhiều SKU, một SKU unavailable, chưa auto-select. |
| slug `tra-sen-tinh-tam` | `200` | Chỉ một SKU available, auto-select. |
| `all-unavailable` | `200` | Tất cả SKU unavailable, CTA disabled. |
| `missing-optional-food-info` | `200` | Không có allergen/certification optional. |
| `product-not-found` | `404` | Not found/unpublished semantics. |
| unknown slug | `404` | Không có dữ liệu public. |
| `product-error` | `500` | Retry state. |
| `product-slow` | `200` sau khoảng 3 giây | Loading state. |

## Browser → BFF → Mockoon mapping

| Browser | BFF validation/transform | Upstream |
| --- | --- | --- |
| `GET /api/catalog/products/banh-ngu-sac-cung-dinh` | Allow exact `products/{slug}`, validate slug | `GET http://127.0.0.1:4013/api/v1/catalog/products/banh-ngu-sac-cung-dinh` |
| `?mockScenario=product-slow` (dev only) | Remove query; set allowlisted header | `X-Mock-Scenario: product-slow` |
| Arbitrary query/nested path/method | Reject locally | Không gọi upstream. |

## Public, mock-internal và forbidden data

| Phân loại | Field/dữ liệu |
| --- | --- |
| Public | Product identity, approved descriptions/gallery/category/type, approved OCOP summary, food policy, SKU label/weight/flavor/package/price/availability/public reason/image. |
| Mock index/internal | `internalStatus`, `publicationStatus`, `curatedRank`, fixture selector. Có thể tồn tại trong Mockoon để test nhưng BFF phải strip. |
| Không được lộ browser | Exact stock quantity, warehouse/location, cost price/margin, audit fields, internal reason, Batch/Lot ID, actual manufacturing/expiry date, search index fields, upstream URL. |

BFF phải project payload thay vì pass-through mù quáng: xóa root internal fields và SKU `availableQuantity`, `warehouseCode`, `costPriceVnd`, `internalUnavailableReason`, `batch`, `lot` trước khi trả browser.

## UI validation và hành vi selection

- Multi-SKU không tạo default purchase selection; CTA hiển thị validation nếu người dùng chưa chọn.
- Single available SKU auto-select.
- Unavailable SKU control có `disabled` và reason; không thể trở thành `selectedSkuId`.
- UI tìm SKU bằng `skuId` từ payload, không xây SKU từ tổ hợp label.
- UI không tự tính promotion, stock hoặc price ngoài việc tìm min/max để trình bày trước selection.
- Cart integration không phát request trong MVP này.
