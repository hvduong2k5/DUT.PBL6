# Kế hoạch tích hợp Traceability P8

**Ngày lập:** 13/11/2026 (UTC+7)
**Mốc đối chiếu:** `dev@239c8db`
**Ngày chốt:** 13/11/2026 (UTC+7) · **Trạng thái:** Đã chốt thiết kế P8, chờ triển khai và nghiệm thu

Traceability đã qua smoke test toàn bộ luồng QR độc lập và kiểm thử PostgreSQL/Redpanda. P8 đưa luồng này lên dữ liệu lô và danh tính nhân viên thật, rồi nối trang QR công khai qua Kong. Kết quả sandbox và phạm vi còn mở được ghi trong [bản đối chiếu MVP](../../../04_testing/traceability-service/05_mvp_contract_alignment.md).

| Tài liệu | Service triển khai | Kết quả cần có |
| --- | --- | --- |
| [08 · Snapshot sản phẩm–SKU–lô](08_batch_snapshot_2026-11-13.md) | Catalog, Inventory, Traceability | Contract có phiên bản, quyền gọi và fixture trên provider thật |
| [09 · Xác thực và phân quyền nhân viên](09_staff_authorization_2026-11-13.md) | Identity, Traceability | Mapping quyền và scope, token từ Identity và test hai actor |
| [10 · Kong và Web QR](10_gateway_web_qr_2026-11-13.md) | Gateway, Web, Traceability | Contract public thống nhất, route và kiểm thử trình duyệt |

## Trình tự triển khai

1. Triển khai contract nguồn và adapter Identity song song theo tài liệu 08–09.
2. Đồng bộ OpenAPI, state matrix và Web. Xác minh routing trong môi trường kiểm thử.
3. Chạy xuyên suốt bằng lô có thật trong môi trường test và hai tài khoản nhân viên riêng. Người soạn tạo và gửi duyệt. Người duyệt phê duyệt và publish. Trình duyệt mở QR ẩn danh qua Web và Kong.
4. Chạy ca lỗi với lô đổi phiên bản trước publish, tài khoản thiếu quyền, mã lạ và QR đình chỉ. Ghi SHA của các service, môi trường, ngày chạy, người xác nhận và kết quả.

## Nhật ký quyết định

| Ngày (UTC+7) | Mục | Trạng thái | Bằng chứng |
| --- | --- | --- | --- |
| 13/11/2026 | Snapshot Catalog/Inventory | Đã chốt thiết kế | Tài liệu 08 |
| 13/11/2026 | Identity và scope | Đã chốt thiết kế | Tài liệu 09 |
| 13/11/2026 | Kong, OpenAPI và Web | Đã chốt thiết kế | Tài liệu 10 |

Khi triển khai từng mục, ghi ngày, SHA/phiên bản contract và kết quả kiểm thử. P8 triển khai QR độc lập với đặt hàng. Packing và truy vết kiện–lô nằm ngoài đợt này.
