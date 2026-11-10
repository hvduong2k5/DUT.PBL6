# 11 — Tiêu thụ Recommendation của MS-11

## Nguyên tắc phân chia

Engine gợi ý (thuật toán, dữ liệu nguồn, consent cá nhân hóa) thuộc **MS-11 Analytics** — thiết kế tại [MS-11 chương 12](../analytics-service/12_recommendation_engine_design.md) và [chương 14](../analytics-service/14_recommendation_api_and_storefront.md). MS-19 chỉ **tiêu thụ**: gọi API của Analytics, resolve kết quả về dữ liệu Catalog hiện hành, và trình bày trong ngữ cảnh hội thoại/storefront. MS-19 không tính lại gợi ý và không nhân bản consent.

## Hai bề mặt tiêu thụ

| Bề mặt | Đường vào | Placement |
| --- | --- | --- |
| API gợi ý cho storefront | `GET /api/v1/ai/recommendations` | `home` (kèm Authorization nếu có), `product_detail` (`product_id` bắt buộc), `cart` (`sku_codes`) |
| Trong chat | Tool `get_recommendations` + intent `basket` của workflow | Chủ yếu `cart`; workflow gọi qua `RecommendationService` |

## Luồng xử lý (`modules/recommendation/service.py`)

1. Gọi Analytics qua adapter (`GET /api/v1/products/recommended` ở chế độ thật; fixture mô phỏng contract ở chế độ dev).
2. **Resolve từng item:** với mỗi `productId` trả về, đọc lại Catalog (`catalog.get`) và chỉ giữ sản phẩm đang `published`. Bước này bảo đảm gợi ý không bao giờ trỏ tới sản phẩm đã ngừng bán chỉ vì snapshot của Analytics cũ hơn Catalog.
3. **Fallback xác định:** Analytics lỗi, trả rỗng, hoặc mọi item bị loại ở bước resolve → dùng danh sách **phổ biến** của Catalog (`popular`), vẫn lọc `published`; riêng `product_detail` loại chính sản phẩm đang xem khỏi fallback.
4. Giới hạn mặc định: `home` 6, `product_detail`/`cart` 4 (khớp trần `limit` 1–6 của contract Analytics).

## Consent và cá nhân hóa

Placement `home` chuyển tiếp nguyên `Authorization` của khách cho Analytics; có consent thì Analytics trả gợi ý cá nhân hóa, không có thì trả gợi ý chung — quyết định nằm hoàn toàn phía Analytics. Consent của MS-19 (`/api/v1/ai/consent`) là một consent **khác**, chỉ về lưu lịch sử chat ([chương 02](02_usecases_and_lifecycle.md)); hai consent không được gộp hay suy ra nhau.

## Feedback

Signal từ storefront/chat (`helpful / not_helpful / clicked / dismissed`) hiện ghi vào bảng `recommendation_feedback` trong `ai_db` để phục vụ đánh giá chất lượng phía MS-19. Analytics có kênh feedback riêng (`POST /api/v1/recommendations/events`, 202) theo contract của MS-11; việc chuyển tiếp signal của MS-19 sang kênh đó là quyết định nối dây còn mở — ghi tại [chương 07, khoảng trống #6](07_integrations_and_contract_gaps.md).
