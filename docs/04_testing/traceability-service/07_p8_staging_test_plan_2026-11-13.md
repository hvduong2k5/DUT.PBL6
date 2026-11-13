# Kế hoạch kiểm thử Traceability P8 trên staging

**Ngày lập** 13/11/2026 (UTC+7)
**Phạm vi** PR #22, từ xác minh lô đến quét QR qua Web và Kong.

E2E trên Docker local đã đạt. Kế hoạch này kiểm tra bản deploy trên staging, gồm Kong dùng PostgreSQL và cấu hình kết nối thật của môi trường. Trạng thái staging hiện tại là **chưa chạy**. Xem [báo cáo E2E local](06_p8_implementation_2026-11-13.md).

## Chuẩn bị

- Ghi SHA đang chạy của Traceability, Catalog, Inventory, Identity, Gateway và Web. Xác nhận CI của bản cần kiểm thử đã đạt.
- Áp dụng migration Traceability ở mode `integration`, chạy API bằng runtime role riêng, áp dụng Identity V12 và Inventory migrations 000002–000004. Kiểm tra DNS nội bộ, Inventory TLS CA, OIDC issuer/audience và credential đọc nguồn.
- Cấu hình route public `GET /api/v2/trace/{qr_code}` và route admin `/api/v1/admin/trace/records` trên Kong staging. BFF trỏ `TRACEABILITY_UPSTREAM_URL` tới prefix public v2. Ghi URL Web và Kong trong biên bản.
- Chuẩn bị các lô test riêng cho luồng chính, cách ly và đổi revision. Có `product_id`, SKU, batch UUID và dữ liệu chứng nhận test. Chuẩn bị hai tài khoản GLOBAL độc lập cho người soạn và người duyệt, cùng tài khoản kiểm tra thiếu quyền nếu cần.
- Với mỗi thao tác ghi, dùng `X-Idempotency-Key` riêng và lấy `expected_version` từ phản hồi mới nhất. Khi kiểm tra retry, gửi lại đúng body và key của thao tác gốc.

## Ca kiểm thử

| ID | Thao tác | Kết quả mong đợi |
| --- | --- | --- |
| ST-01 | Gọi `/readyz` qua mạng nội bộ. Gọi danh sách admin qua Kong khi thiếu token. | Readiness `200`. Admin `401`. |
| ST-02 | Người soạn tạo hồ sơ từ lô hợp lệ với `source_namespace=inventory`, rồi gửi duyệt. Người duyệt phê duyệt và publish. | Tạo `201`. Gửi duyệt, duyệt và publish `200`. Audit ghi đúng hai actor. |
| ST-03 | Mở `/trace/{qrCode}` trong cửa sổ ẩn danh. Mở hồ sơ có trường nghệ nhân hoặc ATTP rỗng. | Public API `200`, `Cache-Control: no-store`. Web hiện đúng lô, revision, xác minh thủ công và trường thiếu. Payload không lộ tham chiếu bằng chứng. |
| ST-04 | Đình chỉ hồ sơ rồi tải lại cùng QR. Tra thêm mã chưa từng tồn tại. | API và BFF giữ `410` cho hồ sơ đã đình chỉ, `404` cho mã lạ. Web hiển thị đúng từng trạng thái. |
| ST-05 | Cho người soạn có quyền approve thử tự duyệt chính hồ sơ mình đã gửi. Thu hồi quyền của tài khoản test rồi thử lại bằng token còn hạn. | Cả hai thao tác bị từ chối `403`. |
| ST-06 | Tạo với sai SKU hoặc batch UUID không tồn tại. Gửi duyệt hồ sơ của lô cách ly. | Trả `409`. Hồ sơ không được publish. |
| ST-07 | Duyệt lô test riêng, đổi dữ liệu lô để tăng revision rồi publish. | Publish `409`. QR chưa từng công bố trả `404`. Không tạo event publish. |
| ST-08 | Gửi lại yêu cầu publish thành công với cùng key và body. | Trả lại kết quả gốc với `X-Idempotency-Replayed: true`. Audit và outbox không ghi thêm bản publish. |
| ST-09 | Trong môi trường test cô lập, làm gián đoạn lần lượt Catalog, Inventory hoặc Identity. Sau khi khôi phục, chạy lại trên hồ sơ test. | Thao tác cần dependency trả `503`, không ghi bản công bố dở dang. Thao tác hoạt động lại khi dependency phục hồi. |

## Ghi nhận và quyết định triển khai

Chạy ST-01 đến ST-04 ngay sau deploy, rồi chạy ST-05 đến ST-09 trước khi mở production. Chỉ dùng hồ sơ và lô dành riêng cho kiểm thử. Sau test, đình chỉ các hồ sơ test đã publish.

Ghi ngày giờ (UTC+7), môi trường, SHA, người chạy, ID ca, kết quả, mã HTTP, request ID và link bằng chứng đã loại thông tin nhạy cảm. Nếu một ca không đạt, ghi lỗi, giữ production chưa mở và chạy lại ca bị lỗi cùng smoke test sau khi sửa. Hoàn tất khi mọi ca đạt trên staging.
