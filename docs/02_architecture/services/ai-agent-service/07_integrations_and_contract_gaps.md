# 07 — Tích hợp và khoảng trống contract

## Mẫu adapter

Mỗi service bạn có một interface nội bộ nhỏ (đó là contract mà agent phụ thuộc) và hai hiện thực: fixture (dữ liệu mẫu trong repo) và HTTP. Chọn hiện thực bằng `AI_FIXTURES` lúc dựng app trong `main.py`. Nếu API công khai của một service bạn khác với hình dạng adapter đang giả định, **chỉ sửa `integrations/http_clients.py`** — workflow, tool và test không đổi. Client HTTP dùng timeout 1,8 giây, nằm dưới timeout 2 giây của gateway.

## Bảng tích hợp đợt 1

| Service bạn | Interface nội bộ | Adapter HTTP đang giả định | Fixture | Trạng thái bằng chứng |
| --- | --- | --- | --- | --- |
| Catalog | `search(query, max_price_vnd, limit)`, `get(id)`, `popular(limit)` → `Product{id, sku, name, price_vnd, status, description}` | `GET /api/v1/products` (`status=PUBLISHED`, `search`, `maxPrice`, `limit`); `GET /api/v1/products/{id}` | 7 sản phẩm (6 published, 1 discontinued), tìm theo chồng từ khóa + độ phổ biến | Hình dạng HTTP là **giả định hợp lý, chưa đối chiếu** với controller thật của Catalog — việc đầu tiên khi nối dây |
| Inventory | `check(sku) → số lượng khả dụng` | `GET /api/v1/stock/{sku}` (404 → 0) | Bảng tồn theo SKU (có SKU = 0 phục vụ ca kiểm thử) | Chưa đối chiếu; nếu Inventory chỉ có gRPC thì chỉ thay file adapter |
| Order | `get(order_id, authorization) → OrderInfo?` | `GET /api/v1/orders/{id}` kèm Authorization của khách; 401/403/404 → không tìm thấy | 2 đơn mẫu của 2 khách khác nhau | Chưa đối chiếu hình dạng `customerId/summary/eta` |
| Shipping | `estimate(province, weight_grams) → ShippingQuote` | `POST /api/v1/shipping/estimate` `{province, weightGrams}` | Luật phí: nội tỉnh Huế 22.000đ, tỉnh khác 32.000đ, +5.000đ mỗi 500g vượt 1kg; hãng mặc định GHN | Chưa đối chiếu |
| Analytics (MS-11) | `recommended(placement, context, limit, authorization)` | `GET /api/v1/products/recommended` đúng contract đã kiểm chứng | Fixture mô phỏng contract (related/basket theo bảng ánh xạ, home = popular) | **Đã kiểm chứng** trên nhánh PR #62 (xem dưới) |

## Contract Analytics — phần đã chắc

Contract tiêu thụ lấy từ code của nhánh `feat/analysis-service` tại `a0058d0`: ba placement `product_detail` (productId, skuCode tùy chọn), `cart` (skuCodes tối đa 20), `home` (cá nhân hóa, cần auth); `limit` 1–6; flag tắt theo placement trả 503 `RECOMMENDATION_DISABLED`; consent do Analytics sở hữu (ETag/If-Match + Idempotency-Key); feedback phía Analytics qua `POST /api/v1/recommendations/events`. Adapter HTTP của MS-19 truyền đúng các tham số này và chuyển tiếp Authorization của khách cho placement `home`.

Điều kiện kích hoạt đường thật: **PR #62 merge vào `dev`** để Analytics có mặt trên nhánh chính, sau đó chạy thử đầu-cuối một lượt `home` có consent.

## Khoảng trống còn mở

| # | Khoảng trống | Ảnh hưởng | Việc cần làm / chủ sở hữu |
| --- | --- | --- | --- |
| 1 | IAM Bedrock chưa có (tài khoản AWS của nhóm) | Chưa bật được Nova Pro/Titan thật | Xin IAM có quyền `bedrock:Converse` / `bedrock:InvokeModel`, kiểm tra model khả dụng tại region (hoặc dùng inference profile xuyên region qua `AI_MODEL_ID`), rồi chạy lại golden set trên model thật làm nghiệm thu production |
| 2 | Hình dạng API công khai của Catalog/Inventory/Order/Shipping chưa đối chiếu | Adapter HTTP có thể lệch field | Khi nối dây: đọc controller thật của từng service, sửa riêng `http_clients.py`, bổ sung test contract |
| 3 | Chưa có route Kong cho MS-19 | Web/mobile chưa gọi qua gateway chung `api-pbl6` | Thêm route + cấu hình auth tương ứng khi triển khai lên VPS (dự án `pbl6` trên Coolify hiện mới có Kafka/Keycloak/Kong) |
| 4 | Endpoint ingestion tri thức chưa có cổng quyền | Ai gọi được endpoint đều ingest được tài liệu public | Đợt 2: khóa bằng permission grant nhân viên (vd `AI_KNOWLEDGE_USE`) theo baseline 7-B, đồng thời là lúc `knowledge_assistant` cần lọc quyền trong retrieval |
| 5 | Care không có contract API trong repo (chỉ còn 2 file `.exe` đã biên dịch ở `services/care-service` trên `dev`) | Chặn `support_copilot` đợt 2 | Owner của Care chốt contract đọc ticket (`support_tickets` thuộc MS-06 theo service boundary) trước khi code copilot |
| 6 | Feedback trong chat đang ghi vào `recommendation_feedback` của ai_db | Chưa chảy về kênh feedback của Analytics (`POST /api/v1/recommendations/events`) | Quyết định khi nối dây thật: chuyển tiếp signal sang Analytics hay giữ bản ghi cục bộ cho đánh giá chất lượng — không nhân bản cả hai nơi mà không có chủ đích |
| 7 | Metric LLM đã định nghĩa nhưng chưa phát | Counter `ai_llm_calls` / `ai_llm_tokens` tồn tại trong `observability.py` nhưng provider chưa tăng | Bổ sung phát metric trong lớp provider (hoặc bọc provider ở runtime) ở đợt vận hành hóa |
| 8 | Đồng bộ tri thức bằng Kafka chưa làm | Cập nhật tri thức đang qua API/gieo tay | Consumer theo 4 sự kiện của đề xuất ở đợt 2 ([chương 04](04_messaging_and_recovery.md)) |
