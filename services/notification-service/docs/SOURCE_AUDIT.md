# Source audit — 2026-11-06

## Cách đọc và độ tin cậy

Audit read-only từ Git objects/local checkout; không gọi provider, không fetch, không kiểm tra remote tip. `VERIFIED SOURCE` nghĩa là đã đọc code/schema tại revision được ghi; không có nghĩa runtime đã deploy hoặc test lại. README của service khác chỉ là evidence mô tả của owner, không kế thừa kết quả PASS vào Notification.

Thứ tự phân xử: yêu cầu hiện tại của user → EPIC 28 về nghiệp vụ → contract/schema được pin **và** code producer/caller thực tế → kiến trúc tổng quan → README lịch sử. Nếu schema và code lệch nhau: ghi gap, giữ adapter disabled; không tự tuyên bố một phía đã được team phê duyệt.

## Baselines đã kiểm tra

| Ref local | Commit đầy đủ | Phần đã đọc / ý nghĩa |
|---|---|---|
| `origin/dev` | `a24492b63c364f93ad19d880798137e554898c3f` | Base nhánh mới; EPIC 28, boundaries, event catalog/schema/examples, producer Order/Shipping, auth Profile, proto Profile, Identity source search, CI |
| `origin/feat/order-service` | `5479e198e753e4b5d181cf6a069827fcd89257c0` | README; so sánh service tree với dev; dev có thay đổi integration bổ sung, không thay bằng feature tip |
| `origin/feat/profile-service` | `a36afa222cd0af2aa43fb5c38ca22d3e0727b22e` | README hardening, Chi/pgx/encryption/outbox pattern; dev có Identity integration mới hơn ở một số file |
| `origin/feat/shipping-service` | `a415bbf3a9424e222dbd29eeb59bfa995fba1e79` | README provider/callback gates; dev giữ cohort v2 và hardening bổ sung |
| `origin/feat/inventory-service` | `5131c13282baec61db5c5fd36be9ac924af5c01c` | `internal/infrastructure/worker/outbox_publisher.go`: worker ports/config/retry; không sao chép nguyên default policy |
| `origin/feat/content-service` và local `feat/content-service` | `50f35e5f753ad67234501aa95e11c3013f553f42` | `AGENTS.md`, `docs/MASTER_PLAN.md`, `go.mod`; mẫu cô lập source/docs toàn bộ trong service; Go/Chi/pgx/Zerolog |
| `origin/feat/care-service` | `94d8731e969cc6a3966df63784b26d4632e5d4a8` | README và `internal/worker/outbox.go`; SLA metadata-only events, audit/metrics; không coi Care đã tích hợp Email/Push |
| `origin/codex/fulfillment-order-shipping` | `4c877d96da8a5a0fb73481055af34b3d58db5b03` | README: durable inbox, event collision ledger, lease fencing, live scope auth, private-media; implementation ở ref riêng, không merge vào nhánh này |

Đọc lại bằng `git show <commit>:<path>`; bảng pin commit tránh drift của branch name. Không có nhánh Notification trước lúc tạo nhánh hiện tại trong inventory local.

## Nguồn và kết luận kiểm chứng

Các path bên dưới mặc định thuộc base dev ở bảng trên.

| ID | Nguồn chính xác | Kết luận / tác động thiết kế |
|---|---|---|
| S01 | `docs/01_requirements/epics/EPIC_28_Notification.md` | Email OTP/Order/invoice là MVP; SMS/Zalo/Push/Internal Alert phase 2; có FR-NOTI-01…32 và quyết định mở; Notification không xác thực OTP hay rollback nguồn |
| S02 | `docs/02_architecture/service_boundary.md`, mục MS-17 và service matrix | BC-17, port định danh 8017, PostgreSQL `notification_db`, Redis `notif:queue:*`, Kafka và REST Admin; không chỉ Redis như một số HLD giản lược |
| S03 | `docs/02_architecture/bounded_context.md`, BC-17; `system_design.md` mục async/DLQ; `high_level_design.md` MS-17 | Mô tả tổng quan, có enum SENT giản lược và retry 5 lần; không đủ làm implementation contract |
| S04 | `docs/03_api_specs/02_kafka_event_catalog.md` so với `packages/events/README.md` | Catalog tổng quan còn Order v1, Shipping key tracking_code; README hiện hành phân cohort Order v1/v2/v3, Shipping key shipment_id. Retry catalog 3 lần khác system design 5 lần. Phải chốt policy theo loại lỗi |
| S05 | `services/order-service/internal/app/base.go` hàm Emit; `internal/app/runtime.go`; `internal/app/core_workers.go`; `internal/persistence/outbox_runtime.go` | Có emitter v2 và publisher v3 riêng, stable bytes/id/schema pins, lease fencing; không gọi v3 là chỉ có schema. Enable flags và deploy thực tế chưa kiểm tra |
| S06 | `packages/events/schemas/order/v3/{placed,paid,cancelled,completed}.event.json`, `packages/events/examples/order/v3/` | Event aggregate/version/ordinal/status/proof, không có destination Email hoặc đủ buyer/recipient policy. Ví dụ riêng lẻ dùng lặp ID: fixtures integration phải có ID riêng cho từng event, không replay cả thư mục như một stream hợp lệ |
| S07 | `services/shipping-service/internal/domain/shipment.go` NewDomainEvent; `internal/worker/outbox.go`; migration `000006_shipping_event_cohort.up.sql`; `packages/events/schemas/shipping/v2/` | Event type `vn.omama.shipping.shipment.<name>.v2`, subject `shipment_id:<id>`, v2 có resource_version/ready_generation; producer chọn version theo shipment cohort. Không có Email |
| S08 | `packages/events/profiles/mvp/order-source.event.json`; `packages/contracts/mvp/registry.yaml`; `services/order-service/internal/app/shipping.go` | Normalized ingress cho Order khác CloudEvents producer; registry ghi các capability gates, không phải chứng nhận runtime. ProcessShippingNotification là Order xử lý event Shipping, không phải MS-17 gửi thông báo |
| S09 | `packages/proto/profile/v1/profile.proto`; `services/profile-service/internal/transport/http/middleware/auth.go` | GetCustomerProfile có optional email; không chứng minh verified contact, consent hoặc quyền gửi cho Guest. Bearer và forwarded identity có trust boundary; không copy cách nhận header thành public auth |
| S10 | `services/identity-service/src/main/java/` targeted search OTP/notification/reset; `UserLifecycleServiceImpl.java`, `KeycloakAdminClient.java` | Có reset password qua Keycloak; chưa xác định hợp đồng `OTP dispatch` cho Notification trong phạm vi đã đọc. Không tự tạo OTP hay gửi email auth song song với IdP |
| S11 | `packages/events/schemas/finance/v1/invoice_issued.event.json` | Có schema invoice với invoice_pdf_url, customer/order; chưa có `services/finance-service` trên base này. URL trong schema không chứng minh private, được phép fetch hoặc đúng recipient |
| S12 | `.github/workflows/web-identity.yml`, step “Shipping regression and notification verification” | Chỉ chạy Shipping và Order shippingprovider tests; không có Notification module tests. Không dùng tên step để tuyên bố MS-17 đã tồn tại |
| S13 | `services/notification-service/src/.gitkeep`; Go modules Order/Profile/Shipping/Content | Notification chỉ scaffold. Go services tham chiếu khai báo Go 1.22.0, Chi v5/pgx v5/kafka-go/Zerolog ở các module tương ứng; đây là baseline repo, không đánh giá phiên bản được hỗ trợ/an toàn hiện tại |

## Áp dụng pattern có chọn lọc

- Content: docs/AGENTS nằm cùng service → dùng cho isolation. Không mang CMS state machine sang Notification.
- Order core/Fulfillment: transaction ngắn, stable message identity, lease epoch, separate collision evidence → dùng cho delivery và inbox. Không import saga/payment code.
- Profile: domain/usecase/transport, encryption port và fail-closed identity → dùng; không đọc trực tiếp Profile DB.
- Shipping: provider interface, fake provider và signed callback → dùng; không copy thói quen giữ SQL transaction xuyên network trong outbox worker hiện có.
- Inventory: cấu hình background worker/bounded retry → tham khảo; timeout gửi Email cần UNKNOWN, không retry mù.
- Care: metadata-only SLA/audit và private access → dùng cho Internal Alert phase sau; không suy đoán event payload khi chưa pin code/schema của owner.

## Giới hạn

Không audit toàn repository hoặc mọi branch. Các nhánh được chọn theo dependencies/pattern của Notification. Không xác minh broker, DB, gateway routes chạy thật, quyền provider, email receipt, OTP policy hoặc consent authority. Cần refresh có chủ đích khi user/owner yêu cầu; kế hoạch hiện tại gắn với snapshot đã pin.
