# Migration cho release mvp-2026-10-26

Xem [đề xuất chốt 6 điểm](TEAM-DECISIONS.md) cho lý do, phân công và thứ tự mở chức năng. Phương án đề xuất cho MVP là Shipping dual-serve v1/v2, consumer v2 sẵn sàng trước producer, mỗi đơn giữ một luồng xử lý. Các lựa chọn này còn cần provider/consumer triển khai và nghiệm thu.

## Public API và client

`mvp-customer.openapi.yaml` là target cho client/BFF triển khai. Frontend Customer v1 snapshots vẫn giữ nguyên làm reference; không tiếp tục coi chúng là public target ở các route trùng MVP.

| Trước | Target | Owner cập nhật |
| --- | --- | --- |
| Checkout gửi items/address, nhận Order trực tiếp | Cart revision → quote → checkout → CheckoutOperation → Order/payment instructions | Web/Mobile/BFF + Order |
| `X-Session-Id` hoặc identity headers tự khai | Credential được Identity/Gateway xác minh và ownership phía server | Identity/Gateway + clients |
| Header key khác nhau | Public `Idempotency-Key` → Order `X-Idempotency-Key`; không phát sinh key mới cho retry cùng lệnh | BFF |
| Error DTO riêng từng service | Sanitize thành PublicError; giữ operation ID để phục hồi outcome chưa rõ | BFF |
| Address ward/province dạng text, wrapper `{addresses}` | DTO Profile với code/name canonical, array, version và ETag/If-Match | Profile/BFF + clients |
| Sửa email qua Profile | Email chỉ đọc; Identity chịu trách nhiệm workflow đổi email | Identity + clients |

Không che việc async bằng cách đổi `202` thành `201` hoặc tạo Order ID giả. Nếu BFF có chờ có giới hạn, hết hạn chờ vẫn phải trả operation để client tiếp tục theo dõi. Giao dịch SePay xác nhận từ canonical provider API; webhook 200 chỉ xác nhận observation bền vững.

Catalog public mapper đọc protobuf JSON theo field naming cấu hình của transcoder, map `products` → `items`, `page_info` → tổng/trang và giữ Money object của public DTO. Parse int64 string chính xác, kiểm VND/nanos, không đổi unavailable stock thành 0/sold-out. Nếu public DTO cần boolean `in_stock` mà projection chưa biết thì trả lỗi availability đã mô tả, không tự bịa kết quả. Các filter category/price/sort và channel phải được chuyển rõ sang Catalog ProductFilter; metadata/HTTP annotations không tự tạo REST server.

## Protobuf Shipping: xử lý xung đột field numbers

Order và Shipping từng mở rộng cùng `omamx.shipping.v1` độc lập:

| Message | Đề xuất Order cũ | Provider Shipping v1 |
| --- | --- | --- |
| CalculateShippingFeeRequest tag 6 | contract_version:string | parcel_dimensions:message |
| CalculateShippingFeeRequest tag 7 | request_hash:string | service_code:string |
| CalculateShippingFeeResponse tags 6–9 | quote_id, valid_until, quote_version, request_hash | service_code, quote_id, quote_expires_at, estimated_delivery_time |

Chọn provider Shipping v1 làm baseline v1 vì đây là wire contract của nhánh triển khai. Đề xuất quote Order chuyển sang **package mới** `omamx.shipping.v2`, giữ cấu trúc v1 provider và thêm binding:

- Request: contract_version tag 9, request_hash tag 10.
- Response: contract_version tag 10, request_hash tag 11, quote_version tag 12; quote_expires_at giữ tag 8 theo profile mới.
- CreateShipment: generation/digest/Order snapshot version mới dùng tags 16–18.
- TrackShipment: resource_version tag 9, ready_generation tag 10; hỗ trợ lookup theo shipment ID để truy vấn authority.

`contract_version` phải là `order-3.2`; canonical hash dùng quy tắc ở `order-3.2.md`, bao gồm toàn bộ địa chỉ/weight/dimensions/carrier/service/warehouse/fragile. Quote phải được lưu bền, có expiry và monotonic revision. Order kiểm cả validity và echo hash trước khi nhận terms; không dùng TTL tự tính thay cho validity provider.

Đây là migration có chủ ý của proposal chưa được provider nghiệm thu. Verification **loại riêng package Shipping v1 khỏi baseline Order cũ**, đồng thời kiểm nguyên package đó với baseline Shipping và dev. Ngoại lệ được ghi thẳng trong evidence, không tuyên bố tương thích wire với stub quote v1 của Order cũ. Các package khác của Order giữ tag/type/signature hiện có.

Dual-serve v1/v2 hoặc triển khai adapter version rõ ràng. Không đổi endpoint v1 sang schema v2, không alias method path, không reuse v1 request bytes. Regenerate Go theo go_package/import-mode; Catalog public có Go package khác Catalog internal dù cùng thư mục source proto.

## Order ↔ Shipping snapshot

GetShippingSnapshot được thêm vào OrderService bằng RPC mới; các method/field cũ giữ nguyên. Shipping hiện đã có client gọi RPC này nhưng Order phải triển khai provider.

Order phải kiểm service caller là Shipping, package thật sự thuộc Order đã đủ điều kiện đóng gói, local hold và generation. Response lấy từ immutable checkout snapshot; không đọc address book hiện tại. Bind `snapshot_version`, `ready_generation`, `items_digest`, `warehouse_id`, recipient, checkout fee, collect amount và carrier/service. Không tin `requesting_service` trong body như credential. Invalid/unknown proof trả lỗi hoặc eligible=false và không tạo shipment.

## Events và ingress adapter

| Topic/type | Key | Semantics |
| --- | --- | --- |
| fulfillment.events.v1 / packing.completed.v1 | order_id | Giữ nguyên schema; không thêm required package/parcel |
| fulfillment.events.v2 / packing.completed.v2 | order_id | package_id + parcel bắt buộc; chỉ bật sau khi consumer opt in |
| shipping.events.v1 / shipment.*.v1 | shipment_id theo provider hiện có | Giữ payload provider v1, chưa đủ version/generation cho Order proof |
| shipping.events.v2 / shipment.*.v2 | shipment_id | Thêm required resource_version/ready_generation; producer/consumer cùng migrate |
| order.events.v3 / order.*.v3 | order_id/aggregate_id cùng giá trị | Fact stream Order 3.2, không kích hoạt legacy stock/voucher deduction |

Các schema v2 là target chưa có producer thật được nghiệm thu. Không publish cùng effect vào hai cohort để “tương thích”; pin mỗi Order một effect authority.

Order hiện nhận normalized Event, không nhận trực tiếp mọi CloudEvent của Shipping/Fulfillment. Adapter phải:

1. Validate envelope/schema/source và lưu event ID + hash để dedup/conflict. Gắn type/source có xác thực, không nhận event công khai tự khai source.
2. Map `created` → SHIPMENT_CREATED; `handed-over` → SHIPMENT_DISPATCHED; `delivered` → SHIPMENT_DELIVERED; `delivery-failed` → SHIPMENT_FAILED. Giữ Order/package/shipment đúng identity.
3. Query owning provider để lấy **evidence của revision gốc**: Order core so equality version/kind/generation/seal. Current/latest projection không đủ để replay một event cũ. Provider hoặc adapter cần giữ lịch sử authority theo revision, không tự suy version từ timestamp/offset.
4. Chuyển thành schema `packages/events/profiles/mvp/order-source.event.json`; ký theo đúng codec Order `domain.EventSignature`, không ký JSON tùy ý. Signature hiện tính HMAC-SHA256 trên Go JSON của Event với signature rỗng và thứ tự field đã pin ở source; cần shared codec/test vectors trước khi triển khai bằng ngôn ngữ khác.
5. Đưa vào ingress Kafka/HTTP internal đã xác thực; preserve durable inbox/retry/DLQ trước khi commit offset. Chưa có proof thì retry/park, không force Order state.

Core facade `/order-core/resources/{id}?source=...&version=...` là giao diện adapter đã được đặc tả; nó không tự tồn tại chỉ vì provider có TrackShipment. Dùng RPC mới `Shipping.v2.GetShipmentRevision` và `Fulfillment.GetTaskRevision` để đọc revision gốc; provider phải lưu kind/time/seal/generation gốc, không dựng từ current state. Packing v2 có thêm task_id/resource_version/ready_generation để định vị đúng proof.

`/terms` trả CommercialTerms, chỉ gồm dữ liệu thương mại mà adapter có authority. Các ID Order/operation/snapshot, owner, provenance, merchant, channel và placed_at do Order bổ sung tại chỗ; adapter không bịa identity để điền một snapshot đầy đủ. `/revalidate`, `/admission-barrier`, `/completion` cũng phải dùng owning-service authority. Completion không được giả Care=false hoặc settlement=true khi chưa có provider.

## Feature gates và triển khai

- Inventory terminal protocol, Catalog snapshot, Shipping v2, Fulfillment permit, Identity mapping và adapter facade là các task bắt buộc trước owning-service MVP acceptance.
- Reverse committed stock, paid cancellation, Care returns/refund, vouchers và loyalty vẫn gated. Không quay về paid-event consumer để bù cho FinalizeReservation chưa có.
- Bổ sung provider/consumer tests trước khi bật route/cohort. Thực hiện rollout consumers trước producer cho event version mới.
- Mọi thay đổi breaking tiếp theo phải có package/route/cohort version hoặc migration rõ ràng; không sửa baseline để làm CI xanh.
