# Đồng bộ tồn Inventory → Catalog (07/11/2026)

## Lỗi và thay đổi
Catalog nghe inventory.stock-changed.v1, payload JSON phẳng gồm sku_code, sellable_quantity, inventory_version, source_updated_at. Inventory trước đây chỉ phát CloudEvents nghiệp vụ trên inventory.events.v1, nên projection Catalog không nhận được snapshot.

Migration 000004_stock_projection bổ sung version tăng theo SKU và deferred triggers trên inventory_items/batches. Khi transaction thành công, DB tính snapshot từ trạng thái cuối: tồn bán được lấy giá trị nhỏ hơn giữa tồn khả dụng item và tổng khả dụng của các batch ACTIVE/NEAR_EXPIRY chưa hết hạn. Item không ACTIVE trả 0. Snapshot và tăng version cùng transaction; rollback không để lại event. Cả use case legacy và terminal đều đi qua bảng này.

Outbox worker định tuyến snapshot tới inventory.stock-changed.v1, key SKU, giữ nguyên JSON phẳng. CloudEvents cũ vẫn phát như trước. Producer dùng hash key để cùng SKU vào cùng partition. Khi không cấu hình Kafka, worker giữ PENDING thay vì đánh dấu đã gửi bằng log publisher.

Một transaction có thể phát nhiều version cùng SKU do nhiều dòng batch thay đổi; tất cả phản ánh trạng thái cuối. Catalog áp dụng version mới hơn nên retry/đảo thứ tự không cộng/trừ lặp. Không kỳ vọng đúng một snapshot cho mỗi command.

## Áp dụng
1. Provision topic inventory.stock-changed.v1 trên Kafka dùng chung, partition theo SKU, theo quy trình vận hành hiện có.
2. Apply migrations/000004_stock_projection.up.sql bằng role migration sau migration 000003_packing_reads của dev. Không chạy seed/TRUNCATE trên DB đang dùng.
3. Migration tự enqueue snapshot mọi SKU hiện có. Đặt KAFKA_BROKERS cho Inventory và chạy consumer Catalog trên cùng broker/topic.
4. Rebuild: SELECT inventory_emit_stock_snapshot(sku) FROM inventory_items ORDER BY sku; hàm tăng version và enqueue snapshot, không reset version.
5. Down migration gỡ trigger/hàm nhưng cố ý giữ cột inventory_version để redeploy không gửi version thấp hơn Catalog đã nhận.

API GetStockLevel và snapshot cùng dùng inventory_sellable_quantity(sku, observed_at). API loại batch hết hạn ngay lúc đọc, không cần chờ worker. Physical/reserved vẫn được trả riêng để quản lý kho. Warehouse-scoped reads chưa hỗ trợ sẽ bị từ chối rõ ràng thay vì bỏ qua bộ lọc.

Worker expiry chạy mỗi phút và quét ngay khi khởi động. Query loại các lô NEAR_EXPIRY chưa hết hạn khỏi danh sách cần xử lý lại, tránh 50 lô đầu chặn các lô tiếp theo. Projection Kafka vẫn eventual: sau mốc hết hạn cần worker commit và broker/consumer nhận event, không bảo đảm cập nhật tức thời.

Credential: cấu hình INVENTORY_CATALOG_TOKEN riêng, ít nhất 32 bytes và khác INVENTORY_ORDER_TOKEN. File CATALOG_INVENTORY_TOKEN_FILE phải chứa đúng Catalog token; Order tiếp tục dùng token riêng. Token Catalog được đọc tồn/batch, không reserve/finalize/release. Khi bỏ trống Catalog token, chỉ Order credential có quyền đọc. Các kiểm tra này áp dụng trên terminal server TLS; không tự mở legacy server ra public.

Khi rollout, cập nhật secret Inventory và file token Catalog cùng đợt; fixture Web đã dùng hai secret riêng. Không dùng token người mua cho kết nối service. Startup terminal yêu cầu migration 000004 đã apply.

## Kiểm tra

- Inventory qua `go vet ./...` và `go test -race -count=1 ./...`.
- `python scripts/test-inventory-stock-projection.py` qua PostgreSQL 16 thật với migration packing 000003 và stock 000004. Kiểm bootstrap, reserve 10→8, release 8→10, deduction, rollback, quarantine, expiry, status, rebuild và down/up giữ version.
- `python scripts/test-order-inventory.py` qua Order–Inventory TLS/PostgreSQL với race detector. Stock 100→94, reserved=0, đúng 3 commit outbox dù retry/mất ACK. Cùng bài kiểm chạy lifecycle PostgreSQL gồm cả quyền đọc Catalog và kiểm phân bổ lô Fulfillment.
- Catalog qua 43 test. Phép đọc snapshot thật đã phát hiện mapping `inventory_version` dùng kiểu wrapper trong khi record nhận primitive. Mapper dùng alias `_long` để khớp constructor và phục vụ được API public sau khi nhận snapshot Kafka.
- Test PostgreSQL thật kiểm GetStockLevel trả cùng sellable quantity với snapshot. Quarantine và hết hạn trả 0 trong khi physical vẫn 8. Token Catalog đọc được và bị chặn reserve, finalize và release. Token thiếu, sai hoặc lặp header bị chặn.
- Kiểm hợp đồng MVP và codegen qua 41 OpenAPI spec, 52 event schema và 13 file protobuf. Public contract sinh lại khớp bản đã lưu.

## Kiểm tích hợp qua Kafka và Web

`scripts/test-web-identity.py --fulfillment` chạy publisher outbox Inventory thật, Kafka và consumer Java Catalog. Bài kiểm đọc `stock_available` qua API public khi Catalog đang dùng projection, rồi khởi động lại Catalog để đọc trực tiếp Inventory qua TLS bằng token Catalog riêng.

Các ca quarantine, hết hạn, khóa SKU và thay đổi lượng giữ chỗ phải cho cùng kết quả giữa API và projection. Bài kiểm chờ version phía Catalog bắt kịp version Inventory, với thời hạn 90 giây để bao gồm thời gian rebalance khi Catalog khởi động lại.

Sau đó chạy tiếp Web checkout, hủy/nhả tồn, thanh toán TEST/chốt tồn, Fulfillment/video và Shipping. Tồn sau chốt là 98 và projection phải bắt kịp đúng version đó. Môi trường dùng PostgreSQL, Redis, Kafka, Keycloak, các service thật và dữ liệu tổng hợp. Hãng vận chuyển dùng FAKE, statement thanh toán dùng TEST.
