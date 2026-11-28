# API Specifications — `docs/03_api_specs/`

Thư mục này là **Single Source of Truth** cho tất cả contract giao tiếp kỹ thuật của hệ thống.

> [!IMPORTANT]
> **6 file frontend handoff bên dưới là nguồn chính thức duy nhất cho CI contract check.**
> Mọi thay đổi REST API (thêm/sửa/xóa endpoint, schema, auth, status code) **bắt buộc** cập nhật các file này trước hoặc song song với code. Xem quy tắc đầy đủ tại [`AGENTS.md`](../../AGENTS.md).

---

## 6 File Frontend Handoff (nguồn chính thức)

### Customer-facing API

| File | Vai trò |
|---|---|
| [`customer-stable.openapi.yaml`](./customer-stable.openapi.yaml) | **Customer API — sẵn sàng triển khai.** Bao gồm toàn bộ luồng Customer: auth (OIDC/PKCE), catalog/product discovery, giỏ hàng, checkout, đơn hàng, profile, địa chỉ, voucher, review. Breaking change yêu cầu thay đổi phiên bản contract. |
| [`customer-draft-planned.openapi.yaml`](./customer-draft-planned.openapi.yaml) | **Customer API — đề xuất/chờ duyệt.** Các capability đang chờ quyết định về policy, security hoặc workflow (ví dụ: return/refund customer-facing, loyalty). Không được import bởi `stable`. |
| [`customer-api-state-matrix.md`](./customer-api-state-matrix.md) | **Bảng trạng thái UI bắt buộc cho Customer** (success / loading / empty / error / auth) cho tất cả endpoint stable. Cập nhật khi thêm/sửa endpoint hoặc thay đổi phân quyền. |

### Admin / Staff-facing API

| File | Vai trò |
|---|---|
| [`admin-stable.openapi.yaml`](./admin-stable.openapi.yaml) | **Admin/Staff API — sẵn sàng triển khai.** Bao gồm: quản lý đơn hàng, catalog, payment/reconciliation, media evidence download-url, fulfillment staff. Dùng permission-and-scope based auth (không dùng role name làm check duy nhất). |
| [`admin-draft-planned.openapi.yaml`](./admin-draft-planned.openapi.yaml) | **Admin API — đề xuất/chờ duyệt.** Các luồng chưa policy-stable: payment mutations, refund orchestration, reconciliation batch, evidence full workflow. Không được import bởi `stable`. |
| [`admin-api-state-matrix.md`](./admin-api-state-matrix.md) | **Bảng trạng thái UI bắt buộc cho Admin** (success / loading / empty / error / auth) cho tất cả endpoint stable. |

### Vòng đời contract

```
candidate → draft-planned → stable (breaking change → version bump)
```

- **stable**: Backend có thể deploy ngay; thay đổi phá vỡ yêu cầu tăng version.
- **draft-planned**: Hình dạng endpoint đủ cụ thể để review, nhưng ít nhất một policy/workflow chưa được giải quyết.

---

## Lint / Validation

```powershell
# Validate cả 4 file OpenAPI (chạy từ repo root)
npx @redocly/cli lint .\docs\03_api_specs\customer-stable.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\customer-draft-planned.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\admin-stable.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\admin-draft-planned.openapi.yaml
```

CI chạy tự động tại `.github/workflows/ci.yml` trên mỗi PR — xem bước `Lint OpenAPI handoff contracts`.

---

## Các File Hỗ Trợ Khác Trong Thư Mục

| File / Thư mục | Mô tả |
|---|---|
| [`01_grpc_contracts.md`](./01_grpc_contracts.md) | Đặc tả 8 gRPC Services nội bộ, SLA timeout, Retry Idempotent |
| [`02_kafka_event_catalog.md`](./02_kafka_event_catalog.md) | Danh bạ Kafka Topics (100% 18 microservices), CloudEvents 1.0, Dead-Letter-Queue |
| [`schema.graphql`](./schema.graphql) | GraphQL Schema cho Mobile BFF (NFR-02 Mobile First) |
| [`iam-consumer-read.openapi.yaml`](./iam-consumer-read.openapi.yaml) | IAM read endpoints (identity service, internal consumer) |
| [`profile-service.openapi.yaml`](./profile-service.openapi.yaml) | Profile service internal contract |
| [`traceability-service.openapi.yaml`](./traceability-service.openapi.yaml) | Traceability (OCOP QR) service contract |
| [`fulfillment-service.candidate.openapi.yaml`](./fulfillment-service.candidate.openapi.yaml) | Fulfillment staff REST (candidate) |
| [`analytics-service.candidate.openapi.yaml`](./analytics-service.candidate.openapi.yaml) | Analytics Dashboard API (candidate) |
| [`customer-access.support.openapi.yaml`](./customer-access.support.openapi.yaml) | Support spec cho Web OIDC/Identity/Gateway (owner: Profile) |
| [`customer-onboarding.support.openapi.yaml`](./customer-onboarding.support.openapi.yaml) | Support spec đăng ký / địa chỉ (owner: Identity/Profile) |
| `fulfillment-service/` | Fulfillment gRPC proto, packing schemas |

---

## Kiến Trúc Giao Tiếp Đa Giao Thức

```text
                         CLIENTS & ĐỐI TÁC NGOÀI
       (Web, Mobile, POS Xưởng Hương Thủy, VietQR, Shopee/TikTok)
                                    │
                                    ▼  [North–South Edge]
                             API GATEWAY (KONG)
              ┌──────────────────────┼──────────────────────┐
              │ REST (OpenAPI 3.0)   │ GraphQL (/graphql)   │ Webhook HTTPS
              ▼                      ▼                       ▼
    MS-04: ORDER SERVICE      MOBILE BFF (Node)       MS-13: CHANNEL
              │                                               │ Kafka Async
              ├──► MS-01: inventory  (gRPC)          MarketplaceOrderImported
              ├──► MS-05: catalog    (gRPC)
              ├──► MS-07: promotion  (gRPC)
              ├──► MS-12: shipping   (gRPC)
              ├──► MS-15: profile    (gRPC)
              ├──► MS-16: identity   (gRPC ABAC)
              └──► MS-19: media      (gRPC)
                                    │
                            APACHE KAFKA BROKER
                                    │
            ┌───────────────────────┼───────────────────────┐
            ▼                       ▼                       ▼
   MS-02: fulfillment      MS-11: analytics      MS-17: notification
   MS-08: procurement      MS-09: finance        MS-18: audit
```

> **POS offline (Quầy Hương Thủy):** MS-13 gọi gRPC sync `CreatePOSOrder` → MS-04 để in hóa đơn tức thời.
> **Đơn sàn (Shopee/TikTok):** MS-13 chỉ nhận Webhook → phát `MarketplaceOrderImportedEvent` lên Kafka; MS-04 Saga consume và điều phối.

---

## Bảng Mã Lỗi Thống Nhất (Unified Error Model)

| `ErrorCode` | gRPC Status | HTTP | Thông điệp người dùng |
|---|:---:|:---:|---|
| `ERR_INVENTORY_INSUFFICIENT_STOCK` | `FAILED_PRECONDITION` | 400 | Sản phẩm đã hết hàng hoặc số lượng không đủ |
| `ERR_INVENTORY_RESERVATION_EXPIRED` | `DEADLINE_EXCEEDED` | 410 | Phiếu giữ hàng hết hạn sau 15 phút, vui lòng đặt lại |
| `ERR_CATALOG_PRICE_TAMPERED` | `INVALID_ARGUMENT` | 400 | Giá sản phẩm đã thay đổi, hệ thống đã cập nhật mức giá mới |
| `ERR_PROMOTION_VOUCHER_EXPIRED` | `FAILED_PRECONDITION` | 400 | Mã giảm giá đã hết hạn sử dụng |
| `ERR_PROMOTION_MIN_ORDER_AMOUNT_NOT_MET` | `FAILED_PRECONDITION` | 400 | Giá trị đơn chưa đạt mức tối thiểu để áp dụng mã này |
| `ERR_PROMOTION_VOUCHER_ALREADY_LOCKED` | `ABORTED` | 409 | Mã đang được dùng ở phiên thanh toán khác |
| `ERR_PAYMENT_HMAC_VERIFICATION_FAILED` | `UNAUTHENTICATED` | 401 | Xác thực chữ ký Webhook ngân hàng thất bại |
| `ERR_IDEMPOTENCY_CONFLICT` | `ALREADY_EXISTS` | 409 | Yêu cầu đang xử lý hoặc đã hoàn tất trước đó |
| `ERR_CIRCUIT_BREAKER_OPEN` | `UNAVAILABLE` | 503 | Hệ thống kho đang quá tải, vui lòng thử lại sau |

---

## Mock Server (Phát Triển Song Song)

```bash
# REST mock từ customer-stable (port 4010)
npx @stoplight/prism-cli mock docs/03_api_specs/customer-stable.openapi.yaml -p 4010

# GraphQL mock (Mobile BFF, port 4000)
npx @graphql-tools/mock-cli docs/03_api_specs/schema.graphql --port 4000

# gRPC mock (ví dụ inventory, port 8001)
npx @improbable-eng/grpc-web-fake-server --proto packages/proto/inventory/v1/inventory.proto
```

## 4. Traceability theo lô — bộ tài liệu dự án

Target **0.4.0-draft** theo canonical Catalog/Media của Dương. Bắt đầu tại [mục lục workflow](traceability-workflow-README.md); các phần Inventory/IAM/Fulfillment/Order/Web đã có trên nhánh Traceability, tránh giao lại công việc đã triển khai.

| Nội dung | Nguồn có thẩm quyền |
| --- | --- |
| Nghiệp vụ, actor và quyết định | [EPIC19](../01_requirements/epics/EPIC_19_OCOP_Traceability.md), [service spec](traceability-workflow-service-spec.md) |
| Giao tiếp giữa service | [Guide gRPC hiện hành](traceability-workflow-grpc.md), canonical `packages/proto/`; không khôi phục internal REST hoặc GetSkuBinding đã gỡ |
| REST scoped/admin và Web | [Admin v2](traceability-scoped-admin-draft.openapi.yaml), [Web BFF](traceability-web-bff-draft.openapi.yaml), [Identity cookie transport](traceability-workflow-bff-draft.openapi.yaml), [Order customer](traceability-workflow-draft.openapi.yaml) |
| Bàn giao triển khai/tích hợp | [Task/OUT](traceability-workflow-peer-handoff.md), [sáu file Markdown theo owner](traceability-owner-handoffs/README.md), [ranh giới plan Dương](traceability-workflow-supplement.md) |
| Kiểm thử và bằng chứng | [Acceptance](traceability-workflow-acceptance.md), [testing index](../04_testing/traceability-service/README.md), [report13 toàn luồng sau PR176](../04_testing/traceability-service/13_after_pr176_2026-11-28.md) |
| Trạng thái UI | [Admin matrix](admin-api-state-matrix.md), [customer matrix](customer-api-state-matrix.md) |
| Kafka | [Event catalog](02_kafka_event_catalog.md); packing không tự lập/công bố hồ sơ hoặc kích hoạt QR |

Nguồn Catalog nằm trong `docs/03_api_specs/` theo AGENTS.md. Sửa REST/quyền/UI states phải cập nhật đúng OpenAPI và state matrix; không sửa tay generated rows, không promote stable chỉ vì lint hoặc CI xanh. Các checkpoint cũ giữ làm lịch sử. PR176 đã merge vào dev@05047000 ngày28/11; khi ghép phải giữ pricing projection/checkout của Dương và kiểm lại tích hợp trên baseline mới.
