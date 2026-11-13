# Internal Employee API State Matrix

Updated: 2026-11-07

This matrix and the OpenAPI files are the backend handoff. It is intentionally independent from test-fixture implementation.

## State semantics

- **Loading:** client state while a request is pending; test with controlled latency. It has no dedicated HTTP status.
- **Empty:** successful `200` response with an empty collection and zero totals, never `404`.
- **Unauthorized:** `401` when credentials/session/grant are missing, invalid or expired.
- **Forbidden:** `403` when identity is known but capability, permission, ownership or scope is insufficient.
- **Error:** `4xx` is actionable client/business failure; `5xx` is server/dependency failure. All errors use the shared `Error` schema.
- **Success:** only documented `2xx` codes are compatible. Mutations requiring asynchronous processing use `202`.

Employee APIs require authentication, effective permission and server-side data-scope enforcement. A hidden menu is not authorization.

## Provisional JWT consumer rollout (parts 3–4)

Direct Profile contract is version 2.0.0: staff JWT v1 with GLOBAL HR permission;
department is a filter. Legacy employee tokens cannot use `/users/me` grants as
a fallback. Order retains each permission's GLOBAL/WAREHOUSE scope. Both retain
the old signed snapshot until token/grant expiry; role ADMIN alone never bypasses.
Issuer tasks 1–2 are pending; see [handoff](../06_operations/profile-order-jwt-handoff.md).

The entries below supplement the existing UI-format rows, without rewriting
historical inferred examples. Producer responses and BFF adaptation are defined
in [consumer read OpenAPI](iam-consumer-read.openapi.yaml), referenced by draft.

| Flow | Success / empty | Loading | Auth / scope | Error |
|---|---|---|---|---|
| Order GET list/queue/detail projection | 200; empty items, next_cursor null; unknown provider state → UNKNOWN; absent SLA → unavailable | Pending request, load-more disabled while fetching | 401 invalid/missing token; 403 no ORDER_VIEW; 404 foreign detail; recipient null without same-warehouse sensitive permission | 400 malformed query/cursor; 502 invalid projection/oversized proxy body; 503 upstream unavailable; no mock fallback |
| Web `/api/admin/orders[/{id}]` | UI DTO, counts explicitly over loaded rows; cursor loads next page | Initial skeleton; load-more progress; detail loading | Opaque OMAMA_SESSION only; backend snapshot controls authority | Friendly retry state; no fabricated dates or SLA |
| Identity GET `/api/v1/admin/order-api/orders[/queue|/{id}]` | Fixed upstream projection and server bearer; no-store | Upstream bounded by 8 seconds and 1 MiB | Existing ACTIVE employee gate plus Order's JWT policy; no browser identity headers | Denial propagated; no redirects or arbitrary targets |
| IAM GET `/api/v1/admin/access/{employees,roles,permissions}[/{id}]` | Versioned envelope, real calculatedAt, known lifecycle; empty arrays/counts | UI skeleton / detail pending | Required permission checked by task 2 producer; accessReview controls review fields | Producer absent → Web 503; malformed/version mismatch → 502; marked missing resource → 404; no mock fallback |
| Web `/api/admin/access/...` | Validated UI DTO; unknown permission catalog codes preserved, not treated as grants | Retry and loading states | Cookie only; no live `/users/me` permission substitution | Mutations 405; no credential exposure |

Access Review shows **current configured access**, not an inventory of unexpired
JWT permissions. Missing task 2 lifecycle/projection fields are an unavailable
capability, never invented dates, revision, impact or status. Existing CRUD APIs
are not silently aliased to the new projections. Role mutations remain owned by
task 2. Common BFF errors include code, safe message, requestId and errors.

## Procurement service authorization (internal API; not yet a frontend route)

Procurement's `/api/v1/admin/procurement/*` endpoints accept an Identity-issued
RS256 bearer JWT and require `principal_type=EMPLOYEE`, `user_id`, and the signed
`permission_grants` claim. Procurement no longer treats internal-token or user
role headers as user authentication. Its BFF/Gateway must forward the bearer
token and keep the service inaccessible to public clients. The service audience
is configurable; align it with Identity (currently expected to be `omama-api`).

| Procurement route family | Required grant | Scope/result |
|---|---|---|
| Supplier, material, offer, PO, and supply-plan reads; supply-plan preview | `PROCUREMENT_VIEW` | `GLOBAL` |
| Supplier, material, offer, contract reads and changes; PO and supply-plan changes excluding approvals | `PROCUREMENT_MANAGE` | `GLOBAL` |
| PO and supply-plan approve/reject actions | `PROCUREMENT_APPROVE` | `GLOBAL`; creator cannot approve their own submission |
| Goods receipt create/inspect; receipt read for warehouse staff | `PROCUREMENT_RECEIVE` | Matching `WAREHOUSE#<warehouseCode>`; operational admins may receive `GLOBAL` |
| Goods receipt variance resolution | `PROCUREMENT_VARIANCE_RESOLVE` | Matching `WAREHOUSE#<warehouseCode>`; operational admins may receive `GLOBAL` |

Missing/invalid credentials produce 401. A valid employee token missing the
required permission or matching warehouse scope produces 403. These operations
remain absent from the frontend OpenAPI paths until a BFF route is explicitly
approved; the corresponding policy summary is recorded in the draft OpenAPI
metadata meanwhile.
## Permission sync semantics

- The IAM database is the source of truth; Keycloak user attributes are a projection used to mint tokens.
- `GET .../effective-permissions` returns the *configured* grants recomputed from the database. An access token already issued is immutable and only reflects changes after the next refresh; the response includes `tokenNote` so the UI can label "configured" vs "in token".
- `POST .../sync-permissions` is the reconciliation path after a failed sync. Failures are audited as `IAM_PERMISSION_SYNC_FAILED` with the trigger and error code. A `502` means Keycloak rejected/failed the update and the call can be retried.
- Revocation paths (revoke assignment, reduce role permissions, disable role, suspend account) sync and then force re-login: Keycloak sessions are revoked and BFF sessions removed.

## Stable operations

| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `GET /api/v1/admin/users` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/users/{principalId}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/users/{principalId}/password/reset` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/users/{principalId}/sessions/revoke` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/users/{principalId}/status` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/employee-invitations` | EPIC_22 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/admin/employee-invitations/{invitationId}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `DELETE /api/v1/admin/employee-invitations/{invitationId}` | EPIC_22 | stable | 204 | N/A | —/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/role-assignments` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/role-assignments` | EPIC_22 | stable | 200/201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PATCH /api/v1/admin/role-assignments/{assignmentId}` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/role-assignments/{assignmentId}` | EPIC_22 | stable | 200/204 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/iam/principals/{principalId}/effective-permissions` | EPIC_22 | stable | 200 | 200 (empty array) | —/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/iam/principals/{principalId}/sync-permissions` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/502/503 |
| `GET /api/v1/admin/roles` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/roles` | EPIC_22 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/admin/roles/{roleId}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/roles/{roleId}` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/roles/{roleId}` | EPIC_22 | stable | 200/204 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/roles/{roleId}/permissions` | EPIC_22 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PUT /api/v1/admin/roles/{roleId}/permissions` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/roles/{roleId}/permissions` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/roles/{roleId}/permissions/{permissionCode}` | EPIC_22 | stable | 204 | N/A | —/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests` | EPIC_18 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}` | EPIC_18 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/b2b/quote-requests/{requestId}/actions` | EPIC_18 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}/files` | EPIC_18 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}/versions` | EPIC_18 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/catalog/products` | EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/catalog/products` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PUT /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable (legacy) | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/transition` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/skus` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}/skus/{skuId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` | EPIC_04 | stable | 200 | 200 (empty list) | 400/— | 401 | 403 | 404 | — | — | 500/503 |
| `PUT /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/images` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}/images/{imageId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/catalog/products/{productId}/images/{imageId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/inventory/skus` | EPIC_09 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/inventory/skus/{skuId}` | EPIC_09 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/inventory/workbench` | EPIC_09 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/inventory/workbench/adjustments` | EPIC_09 | stable | 202 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/admin/inventory/workbench/issues` | EPIC_09 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/admin/inventory/workbench/receipts` | EPIC_09 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/admin/orders` | EPIC_08 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/orders/{orderId}` | EPIC_08 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/packing/tasks` | EPIC_10 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/packing/tasks/{taskId}` | EPIC_10 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/packing/workbench/tasks` | EPIC_10 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/packing/workbench/tasks/{taskId}` | EPIC_10 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/packing/workbench/tasks/{taskId}/checklist/{itemId}` | EPIC_10 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/packing/workbench/tasks/{taskId}/complete` | EPIC_10 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/payments/cases` | EPIC_07, EPIC_24 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/payments/cases/{caseId}` | EPIC_07, EPIC_24 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/returns/cases` | EPIC_14 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/returns/cases/{caseId}` | EPIC_14 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/session` | EPIC_22, EPIC_23 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/shipping/shipments` | EPIC_11 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/shipping/shipments/{shipmentId}` | EPIC_11 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/shipping/workbench/shipments` | EPIC_11 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/shipping/workbench/shipments/{shipmentId}` | EPIC_11 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/support/tickets` | EPIC_16 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/support/tickets/{ticketId}` | EPIC_16 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |

## Draft and planned operations

These rows are proposals. Backend may refine them, but must record the decision and update the OpenAPI contract before frontend integration.

| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `POST /api/v1/admin/orders/{orderId}/actions` | EPIC_08 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/packing/tasks/{taskId}/assignments` | EPIC_10 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/shipping/shipments/{shipmentId}/actions` | EPIC_11 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/payments/cases/{caseId}/actions` | EPIC_07, EPIC_24 | draft | 202 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/returns/cases/{caseId}/actions` | EPIC_14 | draft | 202 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/actions` | EPIC_16 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/messages` | EPIC_16 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/internal-notes` | EPIC_16 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/inventory/approval-requests/{approvalRequestId}/decision` | EPIC_09 | planned | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |

## Catalog admin write contract

Nguồn chi tiết: [`admin-stable.openapi.yaml`](./admin-stable.openapi.yaml) cho tất cả thao tác catalog admin (đọc và ghi). Catalog admin chỉ dùng REST; gRPC chỉ internal.

### Envelope và quyền

- Mọi response dùng `ApiResponse`: `success`, `code`, `message`, `data`, `errors`, `requestId`, `timestamp`.
- `CATALOG_VIEW`: đọc danh sách/chi tiết/lịch sử giá.
- `CATALOG_MANAGE`: tạo/sửa Product, SKU, giá, ảnh.
- `CATALOG_APPROVE` (hoặc role `ADMIN`): `POST /products/{productId}/transition`.
- Actor audit lấy từ JWT `sub`; `X-Actor-Principal-Id` chỉ là fallback nội bộ, không cấp quyền.

### PATCH Product — toàn bộ nội dung

`PATCH /api/v1/admin/catalog/products/{productId}` cho phép sửa mọi nội dung:

| Field | Kiểu / giới hạn | Ghi chú |
| --- | --- | --- |
| `name` | string ≤ 500 | Không rỗng nếu gửi |
| `slug` | string ≤ 500 | Không rỗng; unique toàn catalog |
| `categoryId` | uuid | Danh mục phải tồn tại |
| `description` | string ≤ 5000 | Optional |
| `ingredients` | string ≤ 2000 | Optional |
| `storageGuide` | string ≤ 1000 | Optional |
| `ocopStar` | integer 3–5 | Optional |
| `ocopCertificateNo` | string ≤ 100 | Optional |
| `story` | string ≤ 5000 | Câu chuyện thương hiệu/làng nghề |
| `expectedVersion` | integer ≥ 0 | **Bắt buộc** |

Quy ước PATCH:

- Không gửi hoặc gửi `null` → giữ nguyên.
- Chuỗi rỗng `""` → xóa field optional (`description`, `ingredients`, `storageGuide`, `ocopCertificateNo`, `story`).
- `name`/`slug` nếu gửi lên thì không được rỗng.
- `expectedVersion` lệch revision hiện tại → `409 CATALOG_VERSION_CONFLICT`.
- `PUT /products/{productId}` là legacy (deprecated); client mới dùng PATCH.

Ví dụ cập nhật `story`:

```http
PATCH /api/v1/admin/catalog/products/6f1b1a2e-0f8e-4d5d-8f0b-9a8c7d6e5f40
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "story": "Câu chuyện làng nghề mè xửng Cồn Hến — hơn 200 năm gìn giữ hương vị Huế.",
  "expectedVersion": 3
}
```

Ví dụ xóa mô tả cũ và cập nhật OCOP:

```json
{
  "description": "",
  "ocopStar": 5,
  "ocopCertificateNo": "OCOP-HUE-2026-001",
  "expectedVersion": 4
}
```

### PATCH SKU

`PATCH /api/v1/admin/catalog/products/{productId}/skus/{skuId}`

| Field | Ghi chú |
| --- | --- |
| `variantName` | Không rỗng nếu gửi |
| `weightValue` | > 0 |
| `weightUnit` | `GRAM` / `KILOGRAM` |
| `flavor` | Chuỗi rỗng để xóa |
| `packagingType` | `HOP_GIAY_KRAFT`, `HOP_THIEC`, `GOI_HUT_CHAN_KHONG`, `HOP_QUA_TANG`, `DANG_ROI` |
| `shelfLifeDays` | > 0 |
| `saleStatus` | `ON_SALE` / `PAUSED` / `DISCONTINUED` (chấp nhận `ACTIVE` / `SUSPENDED` / `ARCHIVED`) |
| `expectedVersion` | **Bắt buộc** |

- `skuCode` bất biến, không đổi qua PATCH.
- PATCH SKU không đổi giá; giá dùng endpoint `/prices`.

### Giá theo kênh

- `GET .../skus/{skuId}/prices` trả mọi bản ghi giá (ACTIVE, SUPERSEDED, EXPIRED) của SKU.
- `PUT .../skus/{skuId}/prices` supersede giá ACTIVE cùng kênh rồi tạo bản ghi mới với `effectiveFrom = now`.
- Kênh: `D2C_WEB`, `MOBILE_APP`, `B2B_WHOLESALE`, `POS_QUAY`, `SHOPEE`, `TIKTOK`.
- VND dùng `amountUnits`, `amountNanos = 0`; chống chồng lấn khoảng hiệu lực bằng `tstzrange` + `&&`.

### Ảnh sản phẩm

- Lưu `objectKey` (MinIO/S3); response trả thêm `url = https://cdn.omama.vn/ + objectKey` (giữ nguyên nếu objectKey là URL tuyệt đối).
- `role`: `COVER` | `GALLERY` | `THUMBNAIL` (mặc định `GALLERY`).
- `mediaStatus`: `PENDING_SCAN` | `APPROVED` | `REJECTED` (mặc định `PENDING_SCAN`).
- Bất biến: tối đa một ảnh `COVER` + `APPROVED`. Khi thêm/sửa thành cover APPROVED, cover cũ tự hạ xuống `GALLERY`.
- `DELETE` chỉ xóa metadata; object trên storage do job dọn dẹp riêng xử lý.
- PATCH ảnh yêu cầu `expectedVersion` của ảnh.

### Vòng đời

- `approvalStatus`: `DRAFT` → `PENDING` → `APPROVED` / `REJECTED`.
- `listingStatus` (DB): `ACTIVE` / `SUSPENDED` / `ARCHIVED`; `saleStatus` (contract): `ON_SALE` / `PAUSED` / `DISCONTINUED`.
- Public chỉ trả sản phẩm `APPROVED` + `ACTIVE`.

| `transition` | Kết quả |
| --- | --- |
| `SUBMIT_FOR_REVIEW` | `DRAFT`/`REJECTED` → `PENDING` |
| `APPROVE` | `PENDING` → `APPROVED` |
| `REJECT` | `PENDING` → `REJECTED` |
| `SUSPEND` | `APPROVED` → tạm ngừng bán |
| `ARCHIVE` | Ngừng kinh doanh vĩnh viễn |

### Mã lỗi Catalog

| Code | HTTP | Mã nội bộ | Khi nào |
| --- | --- | --- | --- |
| `CATALOG_PRODUCT_NOT_FOUND` | 404 | 203 | `productId` không tồn tại |
| `CATALOG_VARIANT_NOT_FOUND` | 404 | 204 | `skuId` không tồn tại |
| `CATALOG_CATEGORY_NOT_FOUND` | 404 | 205 | `categoryId` không tồn tại |
| `CATALOG_SKU_DUPLICATE` | 409 | 207 | `skuCode` đã tồn tại |
| `CATALOG_PRODUCT_INVALID_STATE` | 422 | 208 | Transition không hợp lệ với trạng thái hiện tại |
| `CATALOG_SLUG_DUPLICATE` | 409 | 210 | `slug` đã được dùng |
| `CATALOG_VERSION_CONFLICT` | 409 | 211 | `expectedVersion` lệch revision hiện tại |
| `CATALOG_IMAGE_NOT_FOUND` | 404 | 212 | `imageId` không tồn tại hoặc không thuộc sản phẩm |
| `CATALOG_IMAGE_INVALID` | 400 | 213 | `role` / `mediaStatus` không hợp lệ |
| `CATALOG_VARIANT_INVALID` | 400 | 214 | `weightUnit` / `packagingType` / `saleStatus` không hợp lệ |
| `CATALOG_VARIANT_MISMATCH` | 400 | 202 | `skuId` không thuộc `productId` |

### BFF `apps/web-user`

- Route `/api/admin/catalog/[...path]` hỗ trợ `GET`, `POST`, `PUT`, `PATCH`, `DELETE`.
- Forward query `q`, `status`, `page`, `pageSize`, body JSON và header `Idempotency-Key`.
- Đọc yêu cầu `PRODUCT_VIEW`/`CATALOG_VIEW`; ghi yêu cầu `CATALOG_MANAGE`/`PRODUCT_MANAGE` hoặc role `ADMIN`.
- Lỗi upstream chuẩn hóa về `{ code, message, errors, requestId }`.

## Acceptance rules

1. Every list endpoint has both non-empty and empty contract tests.
2. Every protected endpoint has separate `401` and `403` tests.
3. Every detail endpoint returns `404` without leaking whether an out-of-scope resource exists.
4. Every mutation validates `400/422`, transition or revision conflict `409`, and retry behavior.
5. Every endpoint has controlled latency and `500/503` UI tests.
6. Error responses preserve `code` and `requestId`; clients do not branch on translated `message`.
