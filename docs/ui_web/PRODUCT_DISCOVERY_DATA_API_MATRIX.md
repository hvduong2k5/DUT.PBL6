# Product Discovery MVP — Data & Mock API Matrix

## Public UI API

Frontend chỉ gọi Next.js BFF cùng origin tại `/api/catalog/*`. BFF ánh xạ query công khai sang Mockoon port `4012`; khi Backend sẵn sàng chỉ cần thay upstream.

| Capability | Method | BFF route | Upstream mock route |
| --- | --- | --- | --- |
| `DISC-C01` | GET | `/api/catalog/products` | `/api/v1/catalog/products` |
| `DISC-C02` | GET | `/api/catalog/discovery-config` | `/api/v1/catalog/discovery-config` |

## `DISC-C01` query

| Query | Kiểu | Quy tắc |
| --- | --- | --- |
| `q` | string | Sau trim dài 2–100; optional |
| `category` | slug | Một category; optional |
| `productType` | enum | `CAKE`, `CANDY`, `TEA`, `GIFT_SET`; optional |
| `minPrice`, `maxPrice` | integer VND | `>= 0`, min không vượt max |
| `weights` | CSV integer | Các giá trị được config cho phép, ví dụ `150,250` |
| `inStock` | boolean | `true` để chỉ lấy SKU có thể bán |
| `sort` | enum | `CURATED`, `RELEVANCE`, `PRICE_ASC`, `PRICE_DESC`, `NAME_ASC` |
| `page` | integer | Từ 1 |
| `pageSize` | integer | 1–24; UI dùng 9 |

Response là `ProductSummary[]`. Tổng số phần tử sau lọc nằm trong header `X-Filtered-Count`; tổng catalog trước lọc nằm trong `X-Total-Count`.

```json
[
  {
    "id": "PRD-001",
    "slug": "banh-ngu-sac-cung-dinh",
    "name": "Bánh Ngũ Sắc Cung Đình",
    "shortDescription": "Năm sắc bánh thanh nhã từ đậu xanh và hạt sen.",
    "categorySlug": "banh-cung-dinh",
    "categoryName": "Bánh Cung Đình",
    "productType": "CAKE",
    "imageUrl": "https://...",
    "ocopStars": 4,
    "badges": ["Di sản Cố Đô"],
    "matchedOffer": {
      "skuId": "SKU-001-250",
      "label": "Hộp 250g",
      "priceVnd": 185000,
      "weightGrams": 250,
      "packageType": "BOX",
      "isAvailable": true
    }
  }
]
```

## `DISC-C02` response

Trả metadata hiển thị cho filter: categories, product types, weight options và price presets. UI không hard-code nhãn nghiệp vụ khi chúng có thể đến từ catalog service.

## Error semantics

| HTTP | Code | UI behavior |
| --- | --- | --- |
| 400 | `INVALID_QUERY` | Thông báo query không hợp lệ và cho reset |
| 500 | `CATALOG_ERROR` | Error state + retry |
| 503 | `CATALOG_UPSTREAM_UNAVAILABLE` | Hướng dẫn kiểm tra Mockoon/API Gateway + retry |

## Mock scenarios

Trong development, thêm `?mockScenario=<name>` vào URL UI. BFF chuyển thành header `X-Mock-Scenario` tới Mockoon.

| Scenario | Route | Kết quả |
| --- | --- | --- |
| mặc định | Cả hai | Happy path |
| `catalog-error` | products | 500 |
| `catalog-slow` | products | Happy path sau 3 giây |
| `config-error` | discovery-config | 500 |
| `config-slow` | discovery-config | Happy path sau 3 giây |

## Mapping BFF → Mockoon CRUD

| Public query | Mockoon query |
| --- | --- |
| `q` | `search` |
| `category` | `categorySlug_eq` |
| `productType` | `productType_eq` |
| `minPrice` / `maxPrice` | `priceVnd_gte` / `priceVnd_lte` |
| `weights=150,250` | `weightGrams_like=^(150|250)$` |
| `inStock=true` | `isAvailable_eq=true` |
| public sort enum | `sort=<flat field>&order=<asc|desc>` |
| `page`, `pageSize` | `page`, `limit` |

Các field phẳng `priceVnd`, `weightGrams`, `isAvailable`, `curatedRank` và số lượng tồn chính xác chỉ là dữ liệu nội bộ của mock. Payload public dùng `matchedOffer`, chỉ công bố `isAvailable`; API Contract thật không bắt Backend duy trì hai nguồn dữ liệu trùng lặp hoặc lộ số lượng tồn kho vận hành.
