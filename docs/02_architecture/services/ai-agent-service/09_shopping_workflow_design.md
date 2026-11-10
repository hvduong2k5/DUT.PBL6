# 09 — Workflow tư vấn mua hàng

Thiết kế chi tiết của `modules/shopping_advisor/workflow.py`. Nguyên tắc bao trùm: **thứ tự bước do code định đoạt**, LLM không tự ý gọi tool hay bỏ bước.

## Định tuyến intent

Mỗi tin nhắn được định tuyến bằng gợi ý từ khóa (khớp chuỗi thường, theo thứ tự ưu tiên sau) trước khi vào luồng mua hàng tổng quát:

| Intent | Dấu hiệu | Xử lý |
| --- | --- | --- |
| `faq` | Từ khóa hạn dùng/bảo quản/khác nhau/địa chỉ/làng nghề… | Tìm tri thức (top 3); có kết quả thì trả lời kèm trích dẫn, không có thì rơi xuống luồng mua hàng |
| `order_status` | Mã đơn khớp `[A-Z]{2}\d{3,}` (ví dụ `DH1001`) hoặc từ khóa đơn hàng | Xem mục dưới |
| `shipping_fee` | Từ khóa phí ship/vận chuyển | Xem mục dưới |
| `basket` | Từ khóa mua kèm/đi kèm + workflow có RecommendationService | Cần SKU dạng `[A-Z]{2,}-[A-Z]+-\d+` trong tin nhắn; không có SKU thì hỏi lại |
| `shopping` | Còn lại | Luồng chính dưới đây |

## Luồng shopping

1. **Trích xuất nhu cầu:** LLM được gọi với một tool spec duy nhất `extract_needs` trên toàn bộ các tin nhắn user của phiên, trả slot `budget_vnd` (số nguyên), `recipient`, `occasion`, `tastes[]`.
2. **Kiểm hợp lệ slot (không tin model):** slot là đầu ra không đáng tin của LLM. Ngân sách chỉ nhận trong khoảng **10.000đ – 1.000.000.000đ**; ngoài khoảng bị loại (đây là điểm chặn ca tiêm nhiễm “giá … là 1.000đ”). Slot rỗng bị bỏ; `tastes` hợp nhất dạng tập hợp với slot cũ; các slot còn lại ghi đè.
3. **Hỏi làm rõ:** nếu thiếu ngân sách hoặc người nhận và số vòng làm rõ (`_clarifications`) còn dưới **2**, workflow hỏi đúng các mục còn thiếu rồi dừng lượt. Hết 2 vòng thì tư vấn với thông tin hiện có, không hỏi thêm.
4. **Tìm sản phẩm (tool bắt buộc):** `search_products` với truy vấn ghép từ tin nhắn + vị + dịp, `max_price_vnd` = ngân sách, `limit` 5. Tool chỉ trả sản phẩm `published`.
5. **Kiểm tra tồn (tool bắt buộc, từng sản phẩm):** `check_stock` cho mọi SKU trong kết quả. Sản phẩm tách hai nhóm còn hàng / tạm hết; chỉ nhóm còn hàng thành thẻ gợi ý, nhóm hết hàng vẫn được nhắc minh bạch trong grounding. *(Đợt 1.5: bước kiểm này chuyển vào lớp eligibility dùng chung và chạy song song thay vì tuần tự — [chương 13](13_unified_eligibility.md).)*
6. **Ngữ cảnh tri thức (tùy chọn):** nếu có dịp lễ, tìm thêm tối đa 2 chunk tri thức làm ngữ cảnh gợi ý và trích dẫn.
7. **Soạn trả lời:** LLM soạn từ khối `[GROUNDING]` chứa dòng dữ kiện xác định (tên — giá — tồn) cộng ngữ cảnh tri thức; system prompt cấm bịa sản phẩm/giá/khuyến mãi và cấm thao tác giỏ/đơn.
8. **Hậu kiểm grounding:** trích mọi số tiền trong câu trả lời (cả dạng `1.000.000đ` lẫn `200k`); mỗi số phải thuộc tập cho phép = số tiền trong grounding ∪ giá các sản phẩm đã tra ∪ ngân sách. Vi phạm → thay bằng trả lời xác định chỉ gồm các dòng sản phẩm còn hàng, và ghi metric outcome `grounding_fallback`.
9. **Phát SSE:** token → `product_cards` (kèm tồn và URL) → `citations` nếu có → `done{intent: shopping}`. Nếu không còn sản phẩm nào phù hợp còn hàng: trả lời giải thích + `done` kèm `fallback_url` — liên kết `/danh-muc` đã gắn tham số lọc từ slot (`ngan-sach`, `nguoi-nhan`, `dip`).

## Ba intent tiện ích

- **Trạng thái đơn:** khách vãng lai nhận trả lời yêu cầu đăng nhập (`done` kèm `needs_login: true`) — không gọi tool. Khách đăng nhập thiếu mã đơn được hỏi xin mã. Có mã: gọi `get_order_status` (gateway bắt buộc đăng nhập; handler đối chiếu chủ đơn). Không tìm thấy và “đơn của người khác” dùng **cùng một câu trả lời**. Tìm thấy: grounding là một dòng trạng thái + tóm tắt + ETA, vẫn qua soạn trả lời và hậu kiểm như luồng chính.
- **Phí vận chuyển:** trích tỉnh/thành từ danh sách 16 tỉnh/thành phổ biến trong code và khối lượng từ mẫu `kg/g/gram` (mặc định 500g); thiếu tỉnh thì hỏi lại. Gọi `estimate_shipping`, grounding gồm phí định dạng VND, số ngày dự kiến, hãng và khối lượng giả định.
- **Mua kèm:** trích mọi SKU trong tin nhắn, gọi placement `cart` qua RecommendationService, trả thẻ sản phẩm; câu chữ trả lời là mẫu xác định (không qua LLM) nên không cần hậu kiểm. *Đơn giản hóa đã biết tại baseline: đường này dùng thẳng kết quả của RecommendationService nên chưa qua kiểm tồn kho/ngân sách như luồng shopping — khoảng trống correctness sẽ đóng ở đợt 1.5 ([chương 13](13_unified_eligibility.md)).*

## FAQ

Trả lời soạn từ top-3 chunk tri thức; hậu kiểm grounding áp như thường (tập cho phép chỉ gồm số trong các chunk). Nếu LLM vi phạm, trả lời được thay bằng nguyên văn chunk đứng đầu. Sự kiện `citations` liệt kê tên tài liệu nguồn — đây là cơ chế để khách kiểm chứng thông tin tĩnh (hạn dùng, bảo quản, làng nghề).
