# Nối Fulfillment thật vào MVP

Nhánh gốc: `dev@a24492b` sau merge PR #46. Bốn workflow chạy trên commit
này đều thành công: Contracts `41812670003`, Order–Inventory `41812669876`,
Mobile `41812669906`, Web–Identity–Gateway `41812669908`.
Đây là kết quả **sau merge**, không phải kết quả của nhánh PR trước merge.

## Luồng được nối

```mermaid
sequenceDiagram
    participant C as Customer / Web hoặc Mobile
    participant O as Order
    participant I as Inventory
    participant F as Fulfillment
    participant E as Employee ADMIN
    participant M as Kho video + ffprobe + ClamAV
    participant S as Shipping
    C->>O: Checkout VIETQR
    O->>I: Reserve; commit sau khi ghi nhận thanh toán
    O-->>F: fulfillment_ready.v3 qua Kafka
    F->>O: Xin permit và snapshot đơn đã chốt
    F->>I: Đọc đúng lô đã COMMITTED
    F-->>O: Task accepted; Order chuyển PROCESSING
    E->>F: Nhận việc, đếm hàng, hoàn thành checklist
    E->>M: Upload video qua URL ký, lưu phiên bản object
    F->>M: Kiểm SHA256, định dạng/thời lượng, quét nội dung
    E->>F: Niêm phong kiện kèm video VERIFIED
    F->>I: Kiểm lại lô còn được phép xuất
    F-->>O: packing.completed.v2
    O->>F: Đọc revision task và kiện đúng phiên bản
    O->>O: Lưu yêu cầu tạo vận đơn đã mã hóa
    O->>S: Tạo vận đơn từ kiện thực, khóa chống trùng cố định
    S-->>O: Revision/Kafka cập nhật vận chuyển
```

Provider TEST `/pack` bị chặn khi bật Fulfillment thật. Order sử dụng địa chỉ,
phí và dịch vụ vận chuyển đã chốt lúc checkout, cộng cân nặng/kích thước,
seal và package ID của kiện được niêm phong. Thay đổi địa chỉ trong Profile
sau checkout không thay đổi kiện đang giao.

## Quyền và nguồn dữ liệu

| Bên gọi | API / thao tác | Phạm vi và quyền |
|---|---|---|
| Customer | Cart, checkout, đơn của mình | Theo Identity trên `dev`; không được vào API đóng gói |
| Employee | `/api/v1/admin/fulfillment/*` qua Gateway | JWT RS256 đúng issuer/audience; Identity `/api/v1/users/me` xác nhận ACTIVE EMPLOYEE và ADMIN |
| Employee ADMIN GLOBAL | Queue/task, nhận việc cho mình, start, picks, checklist, blocks, seal, media | Các kho; không được mua hàng; chưa bật giao việc cho người khác |
| Employee ADMIN WAREHOUSE | Các API staff nêu trên | Chỉ đúng `scopeId` kho đã cấp; STORE/REGION không được tự đổi thành quyền kho |
| Fulfillment | Order `GetFulfillmentPermit`, `ValidateFulfillmentPermit`, `GetPackingSnapshot` | Credential riêng; Order kiểm PAID/CONFIRMED_COD, COMMITTED, generation, hold/case và permit gốc |
| Fulfillment | Inventory `GetCommittedPackingAllocation`, `ValidatePackingAllocation` | Credential chỉ đọc riêng; không dùng credential reserve/commit/release của Order |
| Order | Fulfillment `GetTaskRevision`, `GetPackageRevision` | Credential riêng; đọc đúng phiên bản, kiểm order/task/generation/seal/video; không suy trạng thái từ payload Kafka |
| Order | Shipping v2 `CreateShipment`, `GetShipmentRevision` | Credential Shipping hiện tại; yêu cầu gốc và idempotency key được giữ khi retry |

Các mã `fulfillment.task.*` / `fulfillment.media.*` là kiểm tra nội bộ của
Fulfillment, ánh xạ từ ADMIN hiện có. PR này không tự thêm claim permission
vào token của Dương và không seed danh sách IAM mới. Khi Identity có role
đóng gói chuyên biệt, thay bảng ánh xạ có kiểm thử trước khi mở quyền rộng hơn.

Order là nguồn order-line ID. Inventory chứng minh SKU/lô/số lượng đã commit;
line ID được truyền để ghép với snapshot của Order, không phải Inventory tự
phát hành line ID. Migration `000003_packing_reads.up.sql` bổ sung revision
lô tăng theo thay đổi; mỗi lần seal đọc lại trạng thái/hạn dùng hiện tại.

## Cấu hình bật tích hợp

Mặc định không bật tự động trên môi trường hiện hữu. Chạy migration của Order
(bao gồm `000008`), Inventory (`000003`) và `fulfillment-migrate` trước khi bật.
Tạo các Kafka topic `order.events.v3`, `fulfillment.task.events.v1`,
`fulfillment.events.v2`, `shipping.events.v2` theo quy định broker hiện có.

| Service | Cấu hình |
|---|---|
| Order | `ORDER_FULFILLMENT_GRPC`, `ORDER_FULFILLMENT_TOKEN_FILE`, `ORDER_FULFILLMENT_CALLER_TOKEN_FILE`, `ORDER_PACKING_POLICY_FILE`, `ORDER_FULFILLMENT_CA_FILE`, `ORDER_FULFILLMENT_SERVER_NAME` |
| Fulfillment | `IDENTITY_URL`, `FULFILLMENT_JWT_ISSUER`, `FULFILLMENT_JWT_PUBLIC_KEY`, `FULFILLMENT_STAFF_AUDIENCE=omama-api`; file public RSA phải thuộc issuer thực |
| Fulfillment | `FULFILLMENT_ORDER_GRPC_ENDPOINT`, `FULFILLMENT_ORDER_TOKEN_FILE`, `FULFILLMENT_ORDER_CA_FILE`, `FULFILLMENT_ORDER_SERVER_NAME` |
| Fulfillment | `FULFILLMENT_INVENTORY_GRPC_ENDPOINT`, `FULFILLMENT_INVENTORY_TOKEN_FILE`, `FULFILLMENT_INVENTORY_CA_FILE`, `FULFILLMENT_INVENTORY_SERVER_NAME`, `FULFILLMENT_ORDER_CALLER_TOKEN_FILE` |
| Inventory | Terminal profile hiện có + `INVENTORY_FULFILLMENT_TOKEN_FILE` |
| Fulfillment | Admission, seal, packing-v2, candidate-peers, media-upload flags đều `true`; database, Kafka, private S3 có versioning, ClamAV và ffprobe phải hoạt động |

Ghép credential theo đúng hai đầu: Order `TOKEN_FILE` = Fulfillment
`ORDER_CALLER_TOKEN_FILE`; Fulfillment `ORDER_TOKEN_FILE` = Order
`CALLER_TOKEN_FILE`; Fulfillment `INVENTORY_TOKEN_FILE` = Inventory
`FULFILLMENT_TOKEN_FILE`. Mỗi liên kết dùng bí mật riêng, tối thiểu 32 byte;
không đưa bí mật/private key vào Git hoặc gửi qua Gateway.

Packing policy là file JSON riêng, có `policy_version`,
`evidence_mode=VIDEO_REQUIRED`, checklist và giới hạn cân nặng/kích thước/video.
Ví dụ fixture nằm trong `scripts/mvp_fulfillment_checks.py`; không tự coi
policy fixture là quy định vận hành đã duyệt. SLA MVP hiện là 24 giờ từ lúc
đặt đơn, deadline revision 1. Permit gốc sống 2 phút; replay không gia hạn.

Production yêu cầu TLS và CA/server name đúng cho gRPC; Order mở TLS bằng
`ORDER_GRPC_TLS_CERT`/`ORDER_GRPC_TLS_KEY`; Identity URL phải HTTPS.
Local cleartext chỉ bật rõ trong fixture cô lập. Public RSA được pin bằng file;
đổi key phải cập nhật file và triển khai lại. Production cần database chữ ký
ClamAV được cập nhật, bucket private/versioning/ACL và chính sách lưu bằng chứng.

## Kiểm chứng có thể chạy lại

```sh
docker build -t omamx/fulfillment-service:integration services/fulfillment-service
python scripts/test-web-identity.py --gradle gradle --flutter flutter --fulfillment
```

Fixture chạy các service thật trên database dùng một lần, upload MP4 thực vào
RustFS có versioning, kiểm bằng ffprobe và engine ClamAV thực. Database chữ ký
ClamAV chỉ chứa dấu kiểm thử vô hại để kết quả tái lập được; kết quả này không
chứng minh độ bao phủ chống mã độc production. Kiểm cả Customer bị chặn,
video chứa dấu bị từ chối, scanner dừng thì không được seal, seal retry không
tạo kiện mới, Order tự nối Shipping, giao hàng và lịch sử Customer.

Kết quả máy đọc ở `.system_generated/web-identity/result.json`; không commit
token, presigned URL hay dữ liệu đăng nhập. Workflow Web chạy cùng cờ
`--fulfillment`; workflow Fulfillment chạy riêng test race/PostgreSQL/Kafka và
kiểm payload theo contract. `go test` trực tiếp thiếu env có thể SKIP test
database, nên không dùng riêng kết quả đó để kết luận E2E.

SePay statement vẫn TEST, carrier vẫn FAKE. Chưa mở COD settlement/completion,
paid cancellation/reversal, nhiều kiện, reopen, giao việc cho staff khác hoặc
Customer xem video. Không dùng boolean mặc định để tuyên bố các luồng này xong.

Nếu peer/scanner lỗi, giữ job/handoff để retry. Handoff giữ nguyên request
đã mã hóa và khóa `fulfillment:shipment:<package_id>` sau timeout/restart.
Theo dõi job MANUAL_REVIEW, tuổi outbox và handoff chưa có shipment; permit
đã hết hạn cần xử lý có kiểm soát, không tự đổi key để né kết quả cũ.
