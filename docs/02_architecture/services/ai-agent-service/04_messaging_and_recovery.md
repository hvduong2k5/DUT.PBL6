# 04 — Messaging và recovery

## Trạng thái thật của đợt 1

Đợt 1 **chưa có consumer Kafka**. Đồng bộ tri thức đi qua hai đường đồng bộ:

1. **Gieo dữ liệu (fixture/dev):** lúc khởi động ở chế độ fixture, service tự gieo corpus khởi điểm — 5 tài liệu FAQ tĩnh (hạn dùng/bảo quản, dẻo–giòn, làng nghề, địa chỉ xưởng, gợi ý quà Tết) và “câu chuyện sản phẩm” sinh từ mô tả của catalog fixture ([chương 10](10_knowledge_retrieval_design.md)).
2. **Ingestion API (đường production đợt 1):** `POST /api/v1/ai/knowledge/products/sync` nhận `{title, source, text, product_id?, visibility?}`, chunk + nhúng + ghi vào kho, trả số chunk đã tạo. Thu hồi sản phẩm dùng thao tác `deactivate_product` của knowledge store (đặt `is_active = FALSE` cho mọi chunk của sản phẩm) — retrieval lọc chunk inactive ngay trong truy vấn nên hiệu lực thu hồi là tức thì.

Đề xuất gốc dự kiến đồng bộ tri thức bằng 4 sự kiện Kafka (gồm thu hồi/ngừng bán). Hướng đó giữ cho đợt 2 khi có consumer framework dùng chung và khi Catalog chốt sự kiện vòng đời sản phẩm; interface của knowledge store (`ingest` / `deactivate_product`) đã có hình dạng đúng với hai thao tác mà consumer sẽ gọi, nên việc bổ sung consumer không đổi schema hay retrieval.

## Hành vi khi thành phần ngoài hỏng

| Sự cố | Hành vi đã triển khai |
| --- | --- |
| LLM lỗi hoặc timeout | Lượt chat kết thúc bằng sự kiện SSE `error` mã `llm_unavailable`, kèm `fallback_url` — liên kết danh mục đã lọc theo slot đã biết (ngân sách/người nhận/dịp) để khách tự xem tiếp. |
| Tool lỗi, timeout (mặc định 2 giây/lượt gọi) hoặc vượt ngân sách bước/tool | Sự kiện SSE `error` mã `tool_unavailable`, cùng `fallback_url`. |
| Analytics không trả được gợi ý | `RecommendationService` nuốt lỗi và suy biến về danh sách sản phẩm phổ biến của Catalog; không có gợi ý nào cũng không làm hỏng lượt chat. |
| Sản phẩm gợi ý đã ngừng bán/không còn `published` | Bị loại ở bước resolve qua Catalog trước khi trả khách ([chương 11](11_recommendation_consumption.md)). |
| LLM bịa số tiền (giá/khuyến mãi không có trong tool) | Hậu kiểm grounding thay toàn bộ câu trả lời bằng bản tóm tắt xác định chỉ gồm dữ kiện tool ([chương 09](09_shopping_workflow_design.md)). |
| Database mất kết nối (chế độ thật) | `/health/ready` trả 503 `not_ready`; các truy vấn store ngắn và nằm ngoài đoạn soạn trả lời nên lỗi lộ diện sớm theo request thay vì treo stream. |

## Nguyên tắc recovery

- Mọi lượt chat là độc lập và idempotent ở phía đọc: gửi lại tin nhắn tạo lượt mới, không có trạng thái “nửa vời” ngoài slot đã lưu.
- Không retry tự động trong workflow; độ tin cậy dựa vào timeout ngắn ở gateway/HTTP client (2 giây / 1,8 giây) và đường fallback xác định, để tổng thời gian một lượt luôn bị chặn.
- Dữ liệu tri thức luôn tái tạo được từ nguồn: corpus là projection phục vụ đọc, mất `ai_db` chỉ mất lịch sử chat và phải gieo/ingest lại tri thức — không mất dữ kiện nghiệp vụ nào.
