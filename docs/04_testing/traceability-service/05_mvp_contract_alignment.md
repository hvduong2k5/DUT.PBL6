# Traceability — đối chiếu MVP 31/10/2026

## Kết luận
Registry mvp-2026-10-26 có 12 điểm tích hợp nhưng không giao task owner/consumer cho Traceability. Public MVP chưa có route /trace. Không có căn cứ để tự thêm packing v2 consumer hoặc thay đổi workflow publish QR trong PR này.

Nhánh đã merge dev. Giữ workflow QR/service-local OpenAPI hiện có và chặn integration startup khi chưa có adapter thật. Đã thêm regression test: fixture sandbox không được dùng thay Catalog/Inventory/Fulfillment/Shipping, không tạo batch giả, không chấp nhận binding SKU sai và tôn trọng cancellation.

## Những gì vẫn chưa thể ghép thật
- DependencySnapshot của Traceability cần product_id, SKU, batch identity, catalog_version, batch_version, dates, restriction và checked_at.
- Inventory GetBatchFEFODetails trả batch_code/dates/remaining/quality nhưng chưa có batch UUID, monotonic version hay binding product. Không dùng batch_code như batch_id hoặc tự suy version từ timestamp.
- Catalog/Inventory phải thống nhất source snapshot contract có identity/revision và authenticated lookup trước khi nối adapter thật.
- Event Traceability hiện là draft service-local; chưa được consumer/Audit nghiệm thu. Không tự nối event packing v2 chỉ vì có schema mới.
- Shared Identity permissions mapping và source access policy còn cần owning team thống nhất.

## Kiểm tra và mức tin cậy
- go test ./..., go vet ./..., go build ./... PASS sau merge và regression mới.
- Service OpenAPI/event schema validator được chạy tĩnh riêng; không thay bằng public MVP schema của Order.
- Không chạy live DB/Kafka/E2E trong phiên này: Docker daemon chưa hoạt động. Các DB tests mặc định skip nếu thiếu TRACEABILITY_TEST_DATABASE_URL.
- Registry runtime acceptance vẫn PENDING. Không có tuyên bố integration-ready/production-ready.

## Việc cần hỏi nhóm (không chặn việc review nhánh)
1. Traceability có thuộc luồng đặt hàng MVP đợt này hay là QR độc lập?
2. Ai cung cấp authoritative Catalog/Inventory batch snapshot, contract và quyền truy cập nào?
3. Nếu cần event packing: đó là liên kết kiện hàng–lô hàng hay tự động publish chứng nhận? Phải chốt semantics, không tự biến packing thành phê duyệt OCOP.
