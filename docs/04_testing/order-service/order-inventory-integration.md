# Nối Order với Inventory thật — 31/10/2026

## Đã nối được phần nào?

Order có thể gọi Inventory thật để **giữ hàng → đọc lại bằng chứng → chốt trừ kho**, hoặc **trả phần hàng đang giữ**. Adapter này dùng gRPC qua TLS, credential riêng của Order và profile `order-3.2`. Inventory cùng bản triển khai được đưa vào nhánh tích hợp; không chỉ cập nhật tài liệu contract.

```mermaid
sequenceDiagram
    participant O as Order checkout worker
    participant D as PostgreSQL của Order
    participant I as Inventory qua TLS
    participant S as PostgreSQL của Inventory
    O->>D: Lưu yêu cầu đặt hàng và công việc cần làm
    O->>I: ReserveStock: giữ hàng, cùng mã yêu cầu khi thử lại
    I->>S: Giữ tồn + lưu quyết định trong một transaction
    I-->>O: Reservation và bằng chứng giữ hàng
    O->>I: GetReservationByOrder: xác minh quyết định gốc
    O->>D: Tạo đúng một đơn từ bằng chứng hợp lệ
    Note over O,I: VIETQR: chốt kho khi payment worker đủ điều kiện; COD: trước khi tạo đơn
    O->>I: FinalizeReservation: chốt trừ kho
    I->>S: Trừ tồn + lưu quyết định + outbox atomic
    I-->>O: COMMITTED và quyết định bền vững
```

**Giữ hàng** giảm số lượng có thể bán nhưng chưa giảm tồn vật lý. **Chốt kho** trừ tồn vật lý. **Release** chỉ bỏ phần đang giữ, không hoàn lại hàng đã chốt. **Mất ACK** là Inventory đã xử lý nhưng Order không nhận được phản hồi; Order phải tra quyết định cũ thay vì tạo thao tác mới.

## Những lỗi nối hai bên đã sửa

| Điểm | Adapter sandbox cũ | Provider thật |
|---|---|---|
| Credential | `x-internal-token` | Một metadata `authorization: Bearer …`, token riêng ít nhất 32 bytes |
| Kết nối | gRPC plaintext trong sandbox | TLS, kiểm CA và tên server |
| Mã kiểm tra nội dung (`request_hash`) | Hash của struct riêng trong Order | Hash canonical của các field protobuf đã normalize theo contract |
| Tên quyết định | `RESERVE`, `FINALIZE`, `RELEASE` | `ReserveStock`, `FinalizeReservation`, `ReleaseReservationByOrder` |
| Query chưa có quyết định của key | Simulator có thể trả resource hiện tại | `NOT_FOUND`; adapter gửi mutation có cùng key, provider serialize và kiểm registry |

Mỗi hồ sơ kết nối được chọn bằng cấu hình; lỗi TLS/auth/hash không tự chuyển sang sandbox. Query `NOT_FOUND` không được dùng làm bằng chứng rằng Inventory không có reservation.

Finalize/release bind `expected_reservation_id`, `items_digest` và `warehouse_id`. Adapter **không gửi optional `expected_version`**: khi mất ACK, checkout đọc lại có thể thấy version mới; đưa version vừa đọc vào cùng key sẽ đổi payload/hash. Payload của mutation giữ nguyên qua retry; Inventory khóa theo Order, kiểm các binding bất biến và trạng thái terminal. Adapter kiểm version phản hồi không thấp hơn proof đã biết. Đây là cách sử dụng field optional, không thay đổi wire contract.

## Chạy bài kiểm tra chung

Tại thư mục gốc repo, cần Docker đang chạy, Python 3 và Go 1.27.3 (phiên bản CI):

```sh
python scripts/test-order-inventory.py
# Nếu Go không nằm trong PATH:
python scripts/test-order-inventory.py --go /path/to/go
```

Script tạo PostgreSQL dùng riêng, credential/chứng chỉ tạm, chạy server bằng `TerminalServer` thật của Inventory và client thật của Order. Mỗi bên có schema riêng; script chỉ xóa container/schema nó vừa tạo. Không đọc credential deployment và không gọi API ngân hàng.

Bằng chứng local ngày 31/10/2026:

- Durable acceptance → checkout worker → reserve mất ACK/thử lại → tạo và đọc lại Order từ PostgreSQL → finalize mất ACK/khôi phục thành công. Gửi lại acceptance trả cùng operation và Order.
- Reserve/query/finalize/release, replay giữ decision ID; thử lại finalize sau khi Lookup đã thấy version 2 vẫn thành công.
- Release hàng đã chốt bị từ chối; quyết định finalize gốc vẫn truy xuất được sau đó. Cùng key đổi nội dung bị từ chối.
- Credential sai và tên server TLS sai bị chặn.
- Ba đơn mua 2 sản phẩm: tồn item và batch đều **100 → 94**, phần giữ còn **0**, đúng **3 event chốt kho trong outbox** dù có gửi lại. Đơn reserve rồi release không trừ tồn vật lý.
- `TestTerminalPostgresLifecycle` của Inventory chạy trên PostgreSQL thật, gồm concurrent finalize và tombstone chặn reserve đến muộn.
- Regression local: Order `go vet` và `go test -race` với PostgreSQL/Redis/Kafka thật có 250 test/subtest PASS; hai SKIP là peer fixture và Keycloak live (peer được chạy riêng ở bài chung). Inventory `go vet` và `go test ./...` có 177 test/subtest PASS; test PostgreSQL được chạy riêng ở bài chung. Shared MVP contract validator PASS; registry runtime acceptance vẫn PENDING.

CI `.github/workflows/order-inventory.yml` chạy lại bài chung này trên PR. Peer test trong `go test ./...` thông thường sẽ SKIP khi thiếu fixture; không dùng SKIP đó để kết luận đã nghiệm thu provider.

## Cấu hình hai service khi ghép môi trường TEST

Inventory cần database riêng, schema `000001_init_inventory_schema.up.sql` và `000002_terminal_profile.up.sql` đã migrate; dùng dữ liệu test có SKU/batch hợp lệ. Không chạy seed có TRUNCATE trên DB đang dùng.

```dotenv
INVENTORY_TERMINAL_ENABLED=true
INVENTORY_ORDER_TOKEN=<credential Order riêng, ít nhất 32 bytes>
INVENTORY_TLS_CERT=/run/secrets/inventory-server.pem
INVENTORY_TLS_KEY=/run/secrets/inventory-server-key.pem
```

Order đọc adapter này trong runtime TEST hiện có: hoàn tất cấu hình `APP_MODE=sandbox`, `SEPAY_ENABLED=true`, `ORDER_CORE_ENABLED=true` và các file SePay/merchant theo [SEPAY_TEST.md](../../../services/order-service/SEPAY_TEST.md), sau đó đặt:

```dotenv
SEPAY_INVENTORY_GRPC=inventory-service:8001
SEPAY_INVENTORY_PROFILE=terminal
SEPAY_INVENTORY_TOKEN_FILE=/run/secrets/order-inventory-token
SEPAY_INVENTORY_CA_FILE=/run/secrets/inventory-ca.pem
SEPAY_INVENTORY_SERVER_NAME=inventory-service
```

Tên server phải thuộc SAN của chứng chỉ. Nội dung token file phải trùng `INVENTORY_ORDER_TOKEN`. Inject secret khi chạy; không commit token/private key. `sandbox` hoặc profile rỗng giữ kết nối simulator cũ cho regression; không đưa file TLS/credential terminal vào profile đó. Cấu hình terminal thiếu endpoint/credential/CA/tên server làm startup lỗi.

Rollout một cohort Order mới vào provider mới, giữ phục hồi cho obligations cũ. Không gửi paid event legacy để trừ kho lần nữa; không chuyển Order legacy vào ledger terminal. Identity người mua và credential Order gọi Inventory là hai lớp khác nhau.

## Giới hạn và việc tiếp theo

Bài chung chứng minh ranh giới **Order–Inventory** và worker đặt đơn thật. Catalog/Shipping terms dùng fixture được kiểm schema; chưa chứng minh UI → Identity → Profile → Catalog → Shipping → Order → SePay thực tế. Bài worker dùng VIETQR; không coi đây là nghiệm thu COD hoặc khoản thanh toán ngân hàng.

Outbox đã kiểm trong PostgreSQL, chưa chứng minh broker/consumer nhận event Inventory trong bài chung. Reversal/refund, paid cancellation, tombstone từ Order trước khi reserve chưa được bật bởi adapter này. Cần owner nghiệm thu rotation/ACL/service identity và thử fault/expiry/restart/rollout trước deployment. Không đổi registry thành `verified` cho toàn bộ MVP.

Bước tiếp theo: ghép Catalog checkout snapshot và Shipping quote thật, rồi chạy một kịch bản UI đặt hàng trên cùng bộ dữ liệu test; nghiệm thu payment worker với SePay TEST và các nhánh timeout/compensation riêng.
