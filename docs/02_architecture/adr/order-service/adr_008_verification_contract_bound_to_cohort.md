# ADR008 Gắn verification contract với runtime cohort

Ngày24/10/2026. Accepted, implemented trong đợt kiểm thử toàn service theo yêu cầu user.

## Gap phát hiện

`scripts/verify-contracts.py` lint sandbox spec nhưng lấy response schemas từ `order-service.openapi.yaml` (target3.2). Khi kiểm actual sandbox Order, nó báo thiếu items_digest. Đây là lỗi chọn contract của verification harness: legacy transport chưa triển khai các target fields và không được ngầm chuyển cohort.

## Quyết định

Runtime checker phải dùng `order-service.sandbox-3.1.openapi.yaml` và ghi runtime_cohort/runtime_contract trong kết quả. Target3.2 vẫn được kiểm riêng bằng `packages/contracts/verify_order_design.py`; không xóa required fields hoặc làm lỏng target schemas để legacy pass.

Thêm `--compose-project` để queries chỉ chạm isolated test project và `--evidence-dir` để test lại không ghi đè baseline evidence cũ. Các source docs dùng để kiểm links vẫn lấy từ docs gốc, không nhầm với output directory. Không đổi HTTP business behavior hoặc schema migration do lỗi harness này.

## Evidence và giới hạn

Sau sửa, actual sandbox Order/CheckoutOperation/Cart responses và15 v2 outbox payloads thuộc5 event types đều pass. Target contract static checks vẫn pass3 specs/8 v3 schemas/9 valid event examples/32 negative cases. Protobuf compile và generated drift checks pass; compile không có baseline descriptor không chứng minh binary compatibility toàn lịch sử.

Full Go race130 test/subtests, Postman21 requests/36 assertions pass; actual provider2 receipts correlate với2 sandbox API transactions, nhưng chưa có active target Payment adapter. Không suy nghiệm thu target flow từ legacy/sandbox hoặc provider probe.

[Báo cáo đầy đủ](../../../04_testing/order-service/07_full_regression_2026_10_24.md), [checker](../../../../services/order-service/scripts/verify-contracts.py).
