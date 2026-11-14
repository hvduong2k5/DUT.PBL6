# 07 — Tích hợp và khoảng trống contract

## Mẫu adapter

Mỗi service bạn có một interface nội bộ nhỏ (đó là contract mà agent phụ thuộc) và hai hiện thực: fixture (dữ liệu mẫu trong repo) và HTTP. Chọn hiện thực bằng `AI_FIXTURES` lúc dựng app trong `main.py`. Nếu API công khai của một service bạn khác với hình dạng adapter đang giả định, **chỉ sửa `integrations/http_clients.py`** — workflow, tool và test không đổi. Client HTTP dùng timeout 1,8 giây, nằm dưới timeout 2 giây của gateway.

## Bảng tích hợp đợt 1

| Service bạn | Interface nội bộ | Adapter HTTP đang giả định | Fixture | Trạng thái bằng chứng |
| --- | --- | --- | --- | --- |
| Catalog | `search(query, max_price_vnd, limit)`, `get(id)`, `popular(limit)` → `Product{id, sku, name, price_vnd, status, description}` | Stable `GET /api/v1/products`, `GET /api/v1/products/search`, `GET /api/v1/products/{id_or_slug}`; ánh xạ money/variant snake_case | 7 sản phẩm (6 published, 1 discontinued), tìm theo chồng từ khóa + độ phổ biến | **Đã đối chiếu** controller và customer stable contract ngày 14/11/2026 |
| Inventory | `check(sku) → số lượng khả dụng` | Đọc `stock_available` từ durable Catalog projection; checkout vẫn revalidate tại Inventory source-of-truth | Bảng tồn theo SKU (có SKU = 0 phục vụ ca kiểm thử) | **Đã đóng đường advisory**; không gọi REST Inventory không tồn tại |
| Order | `get(order_id_or_code, authorization) → OrderInfo?` | Mã UUID gọi stable detail; mã hiển thị được resolve qua danh sách đơn của chính caller rồi gọi detail, media type customer MVP; 401/403/404 → không tìm thấy | 2 đơn mẫu của 2 khách khác nhau | **Đã đối chiếu** snake_case/snapshot; Order là nơi enforce ownership |
| Shipping | `estimate(province, weight_grams) → ShippingQuote` | Không gọi quote khi thiếu recipient + parcel đầy đủ; trả kết quả unavailable để workflow chuyển khách sang checkout | Luật phí fixture chỉ phục vụ dev | **Fail-closed** theo contract thật `/api/v1/internal/shipping/quotes`; không bịa địa chỉ/phí |
| Analytics (MS-11) | `recommended(placement, context, limit, authorization)` | `GET /api/v1/products/recommended` đúng contract đã kiểm chứng | Fixture mô phỏng contract (related/basket theo bảng ánh xạ, home = popular) | **Đã kiểm chứng** trên nhánh PR #62 (xem dưới) |

## Contract Analytics — phần đã chắc

Contract tiêu thụ lấy từ code của nhánh `feat/analysis-service` tại `a0058d0`: ba placement `product_detail` (productId, skuCode tùy chọn), `cart` (skuCodes tối đa 20), `home` (cá nhân hóa, cần auth); `limit` 1–6; flag tắt theo placement trả 503 `RECOMMENDATION_DISABLED`; consent do Analytics sở hữu (ETag/If-Match + Idempotency-Key); feedback phía Analytics qua `POST /api/v1/recommendations/events`. Adapter HTTP của MS-19 truyền đúng các tham số này và chuyển tiếp Authorization của khách cho placement `home`.

Analytics recommendation API đã có trên `dev`; bước nghiệm thu còn lại là chạy đầu-cuối `home` có consent trong môi trường triển khai thật.

## Khoảng trống còn mở

| # | Khoảng trống | Ảnh hưởng | Việc cần làm / chủ sở hữu |
| --- | --- | --- | --- |
| 1 | IAM Bedrock chưa có (tài khoản AWS của nhóm) | Chưa bật được Nova Pro/Titan thật | Xin IAM có quyền `bedrock:Converse` / `bedrock:InvokeModel`, kiểm tra model khả dụng tại region (hoặc dùng inference profile xuyên region qua `AI_MODEL_ID`), rồi chạy lại golden set trên model thật làm nghiệm thu production |
| 2 | (Đã đóng) Adapter lệch contract Catalog/Inventory/Order/Shipping | Có thể trả sai field hoặc gọi endpoint không tồn tại | **Đã đóng 14/11/2026** bằng stable Catalog/Order mapping, Catalog stock projection và Shipping fail-closed |
| 3 | (Đã đóng phần code) Route/deploy MS-19 | Web/mobile chưa gọi qua gateway chung | Kong và CD đã có; vận hành phải tạo Coolify app rồi đặt repository variable `COOLIFY_APP_UUID_AI` |
| 4 | (Đã đóng đợt 1) Endpoint ingestion tri thức mở | Caller ngoài ý muốn có thể ingest tài liệu | Endpoint yêu cầu credential nội bộ từ secret file và không được expose qua Kong; permission grant nhân viên chỉ cần khi có UI knowledge assistant |
| 5 | Care không có contract API trong repo (chỉ còn 2 file `.exe` đã biên dịch ở `services/care-service` trên `dev`) | Chặn `support_copilot` đợt 2 | Owner của Care chốt contract đọc ticket (`support_tickets` thuộc MS-06 theo service boundary) trước khi code copilot |
| 6 | (Đã chốt) Feedback chat và behavior event Analytics khác ngữ nghĩa | Nếu đẩy `helpful/not_helpful/dismissed` vào contract behavior sẽ làm sai dữ liệu Analytics | Giữ feedback chất lượng AI trong `ai_db`; chỉ storefront thật mới phát `VIEW_ITEM/ADD_TO_CART` theo contract Analytics, không nhân bản signal |
| 7 | (Đã đóng) Metric LLM đã định nghĩa nhưng chưa phát | Thiếu quan sát lỗi và token theo provider | `ObservedProvider` phát `ai_llm_calls` và `ai_llm_tokens` cho mọi provider mà không đưa logic metric vào workflow |
| 8 | Đồng bộ tri thức bằng Kafka chưa làm | Cập nhật tri thức đang qua API/gieo tay | Consumer theo 4 sự kiện của đề xuất ở đợt 2 ([chương 04](04_messaging_and_recovery.md)) |
| 9 | (Đã đóng) Eligibility chưa thống nhất giữa các đường sinh Product Card | Gợi ý mua kèm có thể hiển thị sản phẩm hết hàng/vượt ngân sách | **Đã đóng 10/11/2026** bằng lớp eligibility dùng chung tại [chương 13](13_unified_eligibility.md) |
| 10 | Catalog stock projection có độ trễ sự kiện so với Inventory | Tư vấn có thể chậm hơn tồn nguồn trong một khoảng ngắn | Chỉ dùng cho advisory card; checkout bắt buộc revalidate Inventory và quyết định bán cuối cùng |
