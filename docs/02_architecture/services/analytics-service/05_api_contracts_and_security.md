# 05 — API, quyền và bảo mật

[OpenAPI v0.1](../../../03_api_specs/analytics-service.candidate.openapi.yaml) mô tả Dashboard. Gateway giữ prefix `/api/v1/admin/analytics` và chỉ mở với audience/permission đã nghiệm thu.

Recommendation bổ sung `GET /api/v1/products/recommended`, consent API và event API dưới `/api/v1/recommendations`. Public context và Customer personalization có policy riêng với route staff trong chương này. Payload, status, Gateway/BFF và quyền ghi bảng mới được mô tả ở [chương 14](14_recommendation_api_and_storefront.md). Đây là thiết kế API cần triển khai và tạo candidate OpenAPI riêng.

| Method / path | Quyền đề xuất | Hành vi |
| --- | --- | --- |
| `GET /api/v1/admin/analytics/dashboard` | `analytics.dashboard.read` | Sáu card theo kỳ và `comparison=previous_period` tùy chọn. Phần dữ liệu thiếu được mô tả theo card. |
| `GET /api/v1/admin/analytics/metrics/{metricId}/series` | `analytics.dashboard.read` | Hai metric Order theo ngày, cùng `snapshotId` và range. |
| Các nhóm Phase 3 | Quyền Customer, Supply, Product, Insight, export và quyết định riêng | Chốt contract theo từng capability trước khi mở route. |

Lát đầu dùng scope cohort Order v3 do server cấu hình cho staff được cấp quyền tổng quan. Mỗi generation gắn scope server-side. Nếu bổ sung các chiều kênh/kho, API phải kiểm từng chiều và pin cả scope vào query, snapshot và cache key. Chỉ mở card khi người gọi có quyền với nguồn/metric tương ứng; ở candidate v0.1 các card đã cấp đều thuộc quyền Dashboard và quyền riêng có thể mở theo contract mới.

- Xác minh chữ ký JWT, issuer, audience, thời hạn và loại principal staff theo cơ chế Identity/Gateway đã duyệt. Loại bỏ header danh tính từ client ở biên; service tự áp authorization. Giữ mã lỗi ổn định và `requestId`; dùng `Cache-Control: private, no-store`.
- `400` với kỳ/metric ID sai, `401` với token thiếu/sai, `403` với quyền Dashboard thiếu. Snapshot không tồn tại, nằm ngoài scope, hết tuổi hoặc đã thu hồi đều trả `404 SNAPSHOT_NOT_FOUND`. Client refresh Dashboard. Query bị giới hạn trả `429`. Kho dữ liệu phục vụ mất và không có fallback hợp lệ trả `503`. Dashboard thiếu một nguồn vẫn có thể trả `200` với chất lượng theo card.
- Chỉ cung cấp `series` cho hai metric Order. Trong series, bucket ngoài coverage có giá trị `null`. Bảng Dashboard dùng cùng grain và cùng manifest. Mỗi khoảng thời gian tối đa 90 ngày; giới hạn của tổng kỳ hiện tại và kỳ trước phụ thuộc coverage thực.
- Phase 3 dùng khóa khách hàng giả danh theo namespace và key version, kiểm consent/mục đích cả khi build và lúc xuất. Không ghi email, số điện thoại, địa chỉ, bank reference, query text thô hoặc JWT vào journal/log/cache. Chặn lát cắt có thể nhận diện lại khi số mẫu dưới ngưỡng do policy quy định.
- Export và bàn giao cần quyền, mục đích, TTL, audit và kiểm lại quyền khi tải. Audit ghi actor, scope, tham số đã lọc, đối tượng, phiên bản, quyết định và correlation ID. Thu hồi quyền/policy làm vô hiệu export/cache/snapshot có dữ liệu bị thu hồi. Thiết kế Phase 3 giữ artifact trong kho private và theo dõi việc xóa.
