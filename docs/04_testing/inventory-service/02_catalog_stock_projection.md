# Đồng bộ tồn Inventory → Catalog (04/11/2026)

## Lỗi và thay đổi
Catalog nghe inventory.stock-changed.v1, payload JSON phẳng gồm sku_code, sellable_quantity, inventory_version, source_updated_at. Inventory trước đây chỉ phát CloudEvents nghiệp vụ trên inventory.events.v1, nên projection Catalog không nhận được snapshot.

Migration 000003_stock_projection bổ sung version tăng theo SKU và deferred triggers trên inventory_items/batches. Khi transaction thành công, DB tính snapshot từ trạng thái cuối: tồn bán được lấy giá trị nhỏ hơn giữa tồn khả dụng item và tổng khả dụng của các batch ACTIVE/NEAR_EXPIRY chưa hết hạn. Item không ACTIVE trả 0. Snapshot và tăng version cùng transaction; rollback không để lại event. Cả use case legacy và terminal đều đi qua bảng này.

Outbox worker định tuyến snapshot tới inventory.stock-changed.v1, key SKU, giữ nguyên JSON phẳng. CloudEvents cũ vẫn phát như trước. Producer dùng hash key để cùng SKU vào cùng partition. Khi không cấu hình Kafka, worker giữ PENDING thay vì đánh dấu đã gửi bằng log publisher.

Một transaction có thể phát nhiều version cùng SKU do nhiều dòng batch thay đổi; tất cả phản ánh trạng thái cuối. Catalog áp dụng version mới hơn nên retry/đảo thứ tự không cộng/trừ lặp. Không kỳ vọng đúng một snapshot cho mỗi command.

## Áp dụng
1. Provision topic inventory.stock-changed.v1 trên Kafka dùng chung, partition theo SKU, theo quy trình vận hành hiện có.
2. Apply migrations/000003_stock_projection.up.sql bằng role migration. Không chạy seed/TRUNCATE trên DB đang dùng.
3. Migration tự enqueue snapshot mọi SKU hiện có. Đặt KAFKA_BROKERS cho Inventory và chạy consumer Catalog trên cùng broker/topic.
4. Rebuild: SELECT inventory_emit_stock_snapshot(sku) FROM inventory_items ORDER BY sku; hàm tăng version và enqueue snapshot, không reset version.
5. Down migration gỡ trigger/hàm nhưng cố ý giữ cột inventory_version để redeploy không gửi version thấp hơn Catalog đã nhận.

Expiry tự nhiên phản ánh khi worker cập nhật batch hoặc khi phát rebuild; vẫn phụ thuộc lịch expiry worker hiện có. Chưa thay lịch worker trong bản sửa này. Cần chủ động chạy expiry/rebuild trước demo nếu dữ liệu đã qua hạn.

## Kiểm tra
- go test ./... và go vet ./...: PASS.
- Test worker kiểm đúng topic, SKU key và nguyên payload phẳng.
- python scripts/test-inventory-stock-projection.py: PASS trên PostgreSQL 16 thật, container dùng riêng; kiểm bootstrap, reserve 10→8, release 8→10, deduction, rollback, quarantine, expiry, status, rebuild và down/up giữ version.
- Bài Order–Inventory TLS/PostgreSQL chạy lại cùng migration mới: PASS; stock 100→94, reserved=0, đúng 3 commit outbox dù retry/mất ACK.
- Chưa tuyên bố E2E Kafka → consumer Java Catalog thực tế trong bài test này. CI đã thêm bài PostgreSQL snapshot để chạy lại.
