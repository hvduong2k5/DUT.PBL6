# 01 — Domain và module

## Một service, sáu module

`ai-agent-service` là một tiến trình FastAPI duy nhất (cổng 8019). Sáu nhóm nghiệp vụ AI của hệ thống là sáu module code trong cùng service, dùng chung nền tảng (runtime, hội thoại, tri thức, tool gateway, provider, phân quyền và quan trắc). Việc gom chung là quyết định kiến trúc ngày 09/11/2026: các module chia sẻ gần như toàn bộ hạ tầng, khác nhau chủ yếu ở workflow và quyền truy cập dữ liệu.

| Module | Story | Người dùng | Đợt | Trạng thái tại baseline |
| --- | --- | --- | --- | --- |
| `shopping_advisor` | US-AI-01 | Khách (vãng lai + đăng nhập) | 1 | Đã triển khai: chat tư vấn quà/sản phẩm, FAQ, trạng thái đơn của chính khách, ước tính phí ship |
| `recommendation` | US-AI-04 | Khách | 1 | Đã triển khai phía tiêu thụ: ba placement qua API của MS-11 + fallback phổ biến |
| `knowledge_assistant` | US-AI-06 | Nhân viên | 2 | Chưa có code; nền tảng tri thức dùng chung đã sẵn sàng |
| `support_copilot` | US-AI-02 | Nhân viên CSKH | 2 | Chưa có code; chờ contract đọc ticket của Care |
| `content_assistant` | US-AI-03 | Nhân viên nội dung | 3 | Chưa có code |
| `strategic_advisor` | EPIC 25 | Điều hành | 3 | Chưa có code; đọc kết quả DSS của MS-11 |

## Nền tảng dùng chung (`platform/`, `providers/`)

| Khối | Vai trò |
| --- | --- |
| Agent runtime | Ngân sách bước/tool cho mỗi lượt; workflow (code) sở hữu thứ tự bước, LLM chỉ quyết trong phần được giao (trích xuất nhu cầu, diễn đạt). |
| Conversations | Phiên, tin nhắn, slot làm việc, consent lưu lịch sử, retention và dọn dẹp. |
| Knowledge | Kho tài liệu/chunk có vòng đời (draft/active/retracted), truy hồi hybrid có lọc quyền và cửa sổ hiệu lực ngay trong truy vấn. |
| Tool Gateway | Mọi tương tác ra ngoài (kể cả đọc) đều qua gateway: đăng ký allowlist, schema, timeout, audit. |
| Product Eligibility (đợt 1.5, chưa triển khai) | Kiểm tra đủ điều kiện hiển thị dùng chung cho `shopping_advisor` và `recommendation`: published → ngân sách cứng → tồn kho, fail-closed theo SKU. Thiết kế tại [chương 13](13_unified_eligibility.md). |
| Providers | LLM (Fake/Gemini/Bedrock) và embedding (Fake/Gemini/Titan) sau interface chuẩn; hoán đổi bằng cấu hình. |
| Auth | Khách đăng nhập xác thực bằng JWT RS256/JWKS như Order/Profile; khách vãng lai là phiên ẩn danh. Quyền nhân viên (đợt 2) sẽ theo `permission_grants` trong JWT theo baseline phân quyền 7-B. |

## Ranh giới “chỉ gợi ý”

- Không tool ghi: đợt 1 cả năm tool đều chỉ đọc (`search_products`, `check_stock`, `get_order_status`, `estimate_shipping`, `get_recommendations`).
- System prompt của workflow cấm tự thêm giỏ/đặt đơn, và hậu kiểm grounding loại bỏ mọi khẳng định số tiền không có căn cứ tool — kể cả khi LLM cố tình “bịa” khuyến mãi (có ca kiểm thử riêng, [chương 12](12_evaluation_and_verification.md)).
- Các module đợt 2–3 giữ nguyên tắc: copilot soạn nháp phản hồi ticket, content soạn nháp bài viết, strategic soạn thẻ hành động — người có thẩm quyền luôn là bên xác nhận và thực thi ở hệ thống đích.
