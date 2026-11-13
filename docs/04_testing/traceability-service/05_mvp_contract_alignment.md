# Traceability — đối chiếu MVP trên dev

## Kết luận
Tại dev `239c8db`, registry MVP chưa giao vai trò owner/consumer cho Traceability và Gateway chưa có route `/trace`. Contract Catalog public dùng REST. Các RPC packing trong registry phục vụ tích hợp giữa Order, Fulfillment và Inventory.

Nhánh đã merge dev. Giữ workflow QR/service-local OpenAPI hiện có và chặn integration startup khi chưa có adapter thật. Đã thêm regression test: fixture sandbox không được dùng thay Catalog/Inventory/Fulfillment/Shipping, không tạo batch giả, không chấp nhận binding SKU sai và tôn trọng cancellation.

## Những gì vẫn chưa thể ghép thật
- DependencySnapshot của Traceability cần product_id, SKU, batch identity, catalog_version, batch_version, dates, restriction và checked_at.
- Inventory GetBatchFEFODetails trả batch_code/dates/remaining/quality nhưng chưa có batch UUID, monotonic version hay binding product. Không dùng batch_code như batch_id hoặc tự suy version từ timestamp.
- Inventory `GetCommittedPackingAllocation` trả batch ID và eligibility revision trong phạm vi reservation đã commit. Traceability cần tra cứu lô độc lập với đơn hàng và cần cả phiên bản Catalog, ngày sản xuất cùng trạng thái hạn chế tại thời điểm đọc.
- Catalog/Inventory phải thống nhất source snapshot contract có identity/revision và authenticated lookup trước khi nối adapter thật.
- Event Traceability hiện là draft service-local; chưa được consumer/Audit nghiệm thu. Không tự nối event packing v2 chỉ vì có schema mới.
- Shared Identity permissions mapping và source access policy còn cần owning team thống nhất.

## Kiểm tra và mức tin cậy
- `./scripts/dev.ps1 -Action Test` PASS sau merge `239c8db`, với PostgreSQL và broker Redpanda chạy trong Docker. Coverage tổng 79,2%. Lệnh cũng chạy `go vet ./...` và `go build ./...`.
- `Init → Start -Relay → Smoke` build image, khởi động API đến trạng thái healthy và xác minh fixture sandbox hiện có qua HTTP. Git cố định LF cho migration SQL để Windows và Docker dùng cùng checksum. Lần kiểm tra này tái sử dụng database và fixture hiện có.
- Smoke test riêng trên PostgreSQL mới và API container trả `full_flow_passed`. Test tạo hồ sơ, gửi duyệt, phê duyệt, publish, đọc QR public, sửa bản nháp mới và xác minh QR trả HTTP 410 sau khi đình chỉ.
- `./scripts/dev.ps1 -Action Race` PASS trong Linux container với PostgreSQL và Redpanda. `Dockerfile.test` được đưa vào Git để lệnh chạy được từ checkout mới.
- Service OpenAPI/event schema validator được chạy tĩnh riêng; không thay bằng public MVP schema của Order.
- Registry runtime acceptance vẫn PENDING. Không có tuyên bố integration-ready/production-ready.

## Quyết định cần chốt để tích hợp
1. Theo registry hiện tại, kiểm thử Traceability như luồng QR độc lập. Khi đưa vào MVP, chốt route public với Gateway và quyền nhân viên với Identity.
2. Đề xuất Catalog xác thực product–SKU và trả phiên bản Catalog. Inventory cần hợp đồng tra cứu lô theo SKU và batch ID, trả mã lô, phiên bản, ngày sản xuất, hạn dùng, trạng thái hạn chế và thời điểm đọc. Hai service cần chốt cách cấp quyền cho Traceability trước khi bật adapter.
3. Nếu nhóm cần truy vết kiện hàng, chốt một contract liên kết package ID với lô đã đóng gói cùng Fulfillment. Phê duyệt và publish QR tiếp tục theo workflow của Traceability.
