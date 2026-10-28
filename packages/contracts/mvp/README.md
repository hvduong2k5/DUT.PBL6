# MVP contracts — Git integration release mvp-2026-10-26

Đây là điểm vào chung để cập nhật các service cho luồng **Catalog → Profile/address → Cart/Quote → CheckoutOperation → SePay → Inventory COMMITTED → Fulfillment → Shipping**.

**Trạng thái: integration target.** Bộ contract được hợp nhất để team triển khai; chưa có sign-off owning-service hoặc bằng chứng chạy xuyên service thật. CI xanh chứng minh wire/schema/codegen, không chứng minh service đã hỗ trợ capability. Không tự bật feature sau khi chỉ merge contract.

## Bắt đầu theo vai trò

**Đọc trước:** [Đề xuất chốt 6 điểm cho MVP](TEAM-DECISIONS.md) giải thích luồng mua bằng sơ đồ, phương án lựa chọn, trách nhiệm và điều kiện nghiệm thu. Đây là đề xuất triển khai; chưa phải sign-off của các nhóm. Vòng đầu đề xuất khách đăng nhập + SePay + giao hàng; Guest/COD/automatic completion có điều kiện mở riêng.

| Thành viên | Contract cần đọc | Việc tiếp theo |
| --- | --- | --- |
| Web / Mobile / BFF | [MVP Customer OpenAPI](../../../docs/03_api_specs/mvp-customer.openapi.yaml), [migration](MIGRATION.md) | Dùng quote/revision, xử lý `202`, poll operation, map error/idempotency và Profile DTO |
| Catalog | [internal Catalog](../../proto/catalog/v1/catalog.proto), [public](../../proto/catalog/v1/catalog_public.proto), [admin](../../proto/catalog/v1/catalog_admin.proto) | Giữ hai service/package riêng, triển khai coherent checkout snapshot và adapter DTO |
| Inventory | [Inventory proto](../../proto/inventory/v1/inventory.proto), [semantics Order 3.2](../order-3.2.md) | Thêm terminal RPC/proof/tombstone, kiểm concurrency expiry/finalize/release |
| Profile | [direct-backend OpenAPI](../../../docs/03_api_specs/profile-service.openapi.yaml), [Profile proto](../../proto/profile/v1/profile.proto) | Kiểm integration ownership, version/address mapping và gateway identity |
| Order | [Order proto](../../proto/order/v1/order.proto), [core facade](../../../docs/03_api_specs/order-core-dependencies.mvp.openapi.yaml) | Triển khai GetShippingSnapshot và thay simulator bằng các adapter thật |
| Shipping | [provider v1](../../proto/shipping/v1/shipping.proto), [bound-quote v2](../../proto/shipping/v2/shipping.proto) | Dual-serve v2; persist quote/revision; cung cấp source evidence theo phiên bản gốc |
| Fulfillment | [Fulfillment proto](../../proto/fulfillment/v1/fulfillment.proto), [packing v2](../../events/schemas/fulfillment/v2/packing_completed.event.json) | Permit/accept/cancel barrier và kiện có package/parcel; chưa có provider được nghiệm thu |
| Identity / Gateway | [Identity proto](../../proto/identity/v1/identity.proto), [registry](registry.yaml) | Xác thực issuer/audience/key, subject mapping và trusted internal identity |

[registry.yaml](registry.yaml) là bảng owner–consumer–capability–task có thể kiểm bằng máy. Không tự đổi `provider-required` thành verified chỉ vì có stub được generate.

## Nguồn chuẩn và thứ tự ưu tiên

1. Public MVP: `docs/03_api_specs/mvp-customer.openapi.yaml`.
2. Internal provider calls: các protobuf được registry trỏ tới; HTTP integration facade được mô tả riêng trong `order-core-dependencies.mvp.openapi.yaml`.
3. Events: đúng version/topic trong [migration](MIGRATION.md), không dùng lẫn v1/v2/v3.
4. Các file `*.candidate.openapi.yaml`, D2C tổng và frontend snapshots vẫn được giữ để review/provenance; chúng không ghi đè public MVP ở các route trùng nhau.

Public contract là integration target mới, không sửa DTO service đang chạy một cách ngầm định. Gateway/BFF và client phải cùng migrate. Nếu một adapter chưa có, capability vẫn đóng; không thay bằng response từ simulator.

## Những quyết định đã ghi vào contract

- Checkout nhận `quote_id`, `cart_revision`, `payment_method`. `202` là nhận xử lý, chưa phải Order đã tồn tại; cả `201` và `202` trả CheckoutOperation. Poll `/api/v1/checkout-operations/{id}` rồi đọc Order/payment instructions khi thành công.
- Public dùng `Idempotency-Key`; adapter chuyển sang `X-Idempotency-Key` của Order. Giữ key 16–128 ký tự ASCII hiển thị và cùng payload khi retry.
- Public errors dùng `code`, `message`, `requestId`; giữ `retryable`, `operation_id` khi có. Không chuyển dev message/PII ra public.
- Order dùng integer VND. Protobuf Money phải là VND, nanos=0; không dùng float. Catalog public vẫn giữ Money object trong DTO Frontend, `units` là integer và phải chuyển từ proto JSON int64 string không mất precision.
- Profile giữ code địa giới, version/ETag, If-Match và list dạng array của backend; email thay đổi thuộc Identity. Client cần migrate khỏi DTO địa chỉ chỉ có tên ward/province.
- `CatalogService` (`omamx.catalog.v1`) cho internal Order và `CatalogPublicService`/`CatalogAdminService` (`hvduong.catalog.v1`) cùng tồn tại. Go codegen đặt package public riêng để tránh trùng symbol.
- Shipping `v1` giữ field number provider hiện có. Profile quote có binding của Order chuyển sang `omamx.shipping.v2`; không dùng stub v1 cũ của Order để gọi provider Shipping v1.
- Packing `v1` giữ nguyên. Package/parcel bắt buộc thuộc packing `v2`. Shipping proof events thuộc stream `v2`; paid facts của Order 3.2 vẫn thuộc Order `v3` và không kích hoạt consumer trừ kho legacy.

## Kiểm tra trước khi sửa/đẩy contract

Python 3.12+; không cần Docker hoặc credentials dịch vụ:

```sh
python -m venv .system_generated/contract-venv
# Activate the venv for your OS, then:
python -m pip install -r packages/contracts/mvp/requirements.txt
python packages/contracts/mvp/build_public_contract.py
python packages/contracts/mvp/verify.py
python packages/contracts/verify_order_design.py --descriptor .system_generated/mvp-contracts/descriptor.pb
```

CI bổ sung generate Go message + gRPC bằng protoc-gen-go v1.36.10 / protoc-gen-go-grpc v1.5.1. Có thể chạy cùng bước tại local bằng `verify.py --go-plugin <path> --go-grpc-plugin <path>`.

`verify.py` kiểm schema OpenAPI, generated public contract, event fixtures/negative cases, registry và wire compatibility với bốn baseline dev/Order/Catalog/Shipping đã pin. Các baseline nằm trong Git nên kiểm tra thông thường chạy offline. Việc tạo lại baseline chỉ dành cho thay đổi có chủ ý, phải review migration; lệnh thông thường không ghi lại baseline.

## Cập nhật service theo bộ này

1. Lấy commit contract đã review vào nhánh service; giải quyết xung đột protobuf theo migration, không chọn toàn bộ ours/theirs.
2. Regenerate stub theo đúng package/version. Có schema mới không đồng nghĩa provider đã triển khai RPC.
3. Triển khai task trong registry và contract tests provider/consumer: quyền sở hữu, retry cùng key, payload conflict, timeout/unknown outcome, version/expiry và dữ liệu thiếu proof.
4. Nối một stack thật, lưu evidence theo commit của cả provider và consumer. Chỉ sau đó cập nhật trạng thái implementation acceptance.

Hoàn tất đơn cần completion authority/Care/settlement có bằng chứng; không mặc định đủ điều kiện chỉ vì carrier báo delivered. Refund thật, paid cancellation, vouchers/loyalty, production carriers và CMS/analytics giữ ngoài phạm vi được nghiệm thu ở release này.

## Provenance

[sources.json](sources.json) pin commit nguồn trên Git. Bộ này lấy contract từ nhánh Order/Profile, Catalog, Shipping và Mobile/Web, không đồng bộ từ Drive. Ba snapshot Customer được giữ nguyên nội dung từ nhánh Mobile, chuẩn hóa xuống dòng LF và lưu cả SHA-256 gốc/bản Git; các Web candidate được giữ dưới `frontend-contracts/web-reference`.
