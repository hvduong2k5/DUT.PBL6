# 05 — API và bảo mật

Đặc tả máy đọc được: [ai-agent-service.candidate.openapi.yaml](../../../03_api_specs/ai-agent-service.candidate.openapi.yaml). Chương này ghi các quy tắc mà OpenAPI không diễn đạt hết.

## Endpoint đợt 1

| Method & path | Quyền | Mô tả |
| --- | --- | --- |
| `POST /api/v1/ai/shopping/conversations` | Guest hoặc Bearer JWT | Cần `Idempotency-Key`; 201 kèm token truy cập một lần nếu là guest |
| `POST /api/v1/ai/shopping/conversations/{id}/messages` | Chủ phiên | Gửi tin nhắn; trả **SSE** (xem giao thức dưới). Nội dung 1–2000 ký tự |
| `GET /api/v1/ai/shopping/conversations/{id}` | Chủ phiên | Phiên + slot hiện tại + lịch sử tin nhắn đã lưu |
| `POST /api/v1/ai/shopping/feedback` | Chủ phiên | Cần `Idempotency-Key`; cùng key khác payload → 409 `idempotency_conflict` |
| `GET /api/v1/ai/recommendations` | Guest hoặc Bearer JWT | Query `placement` (`home` / `product_detail` / `cart`), `product_id`, `sku_codes` (phẩy ngăn cách) |
| `GET /api/v1/ai/consent` | Bearer JWT (khách) | Consent lưu lịch sử chat; guest → 401 `login_required` |
| `PUT /api/v1/ai/consent` | Bearer JWT (khách) | Đặt consent; mỗi lần đổi tăng `version` của bản ghi |
| `POST /api/v1/ai/knowledge/products/sync` | Nội bộ | Credential từ `AI_INTERNAL_TOKEN_FILE`; không route công khai qua Kong |
| `GET /health`, `GET /health/ready`, `GET /metrics` | Công khai nội bộ | Sống / sẵn sàng (ping DB ở chế độ thật) / Prometheus |

## Giao thức SSE của lượt chat

Mỗi sự kiện là một dòng `data: {json}`. Loại sự kiện:

| `type` | Nội dung |
| --- | --- |
| `token` | Mảnh văn bản của câu trả lời (cắt 48 ký tự/mảnh ở đợt 1 — trả lời được soạn trọn và hậu kiểm grounding trước khi phát, xem [chương 00](00_ai_agent_service_hld.md)) |
| `product_cards` | `cards`: id, sku, tên, `price_vnd`, mô tả, tồn kho, `url` dạng `/san-pham/{id}` |
| `citations` | `items`: tên tài liệu tri thức đã dùng (intent FAQ và ngữ cảnh dịp lễ) |
| `done` | Kết thúc lượt: `intent` (`shopping` / `clarification` / `faq` / `order_status` / `shipping_fee` / `basket`), có thể kèm `fallback_url` hoặc `needs_login` |
| `error` | `code` (`llm_unavailable` / `tool_unavailable`), `message` tiếng Việt thân thiện, `fallback_url` |

## Xác thực và quyền sở hữu

- **Khách vãng lai:** không header `Authorization` → phiên ẩn danh; lúc tạo nhận `guest_token`, các lần đọc/gửi/feedback phải gửi `X-Guest-Conversation-Token`. DB chỉ lưu SHA-256 của token.
- **Khách đăng nhập:** Bearer JWT được verify RS256 qua JWKS, từ chối service/staff claim, rồi đối chiếu `/api/v1/users/me` là principal `CUSTOMER` đang `ACTIVE` và khớp subject/id. Thiếu cấu hình xác thực → fail-closed 503.
- **Sở hữu phiên:** phiên của khách khác, hoặc phiên guest bị khách đăng nhập truy cập (và ngược lại), đều trả **404 `conversation_not_found`** — không phân biệt được với phiên không tồn tại. Áp dụng cho đọc phiên, gửi tin nhắn và feedback.
- **Dữ liệu đơn hàng:** tool `get_order_status` yêu cầu khách đăng nhập và chuyển tiếp nguyên Bearer tới Order. Order là owner của policy ownership; AI gom 401/403/404 thành cùng kết quả “không tìm thấy”, không suy đoán từ field customer cũ.
- **Lỗi API:** mọi lỗi HTTP của service dùng `{code, message, requestId}` và echo `X-Request-Id`; message an toàn, client chỉ branch theo `code`.

## Phòng thủ đã triển khai

- **Grounding hậu kiểm:** mọi số tiền trong câu trả lời phải thuộc tập cho phép = số tiền trích từ khối grounding của tool ∪ giá các sản phẩm đã tra ∪ ngân sách khách nêu; vi phạm → thay bằng trả lời xác định. Slot do LLM trích xuất cũng bị kiểm: ngân sách ngoài khoảng 10.000đ – 1.000.000.000đ bị loại (ca tiêm nhiễm “giá … là 1.000đ” có kiểm thử riêng).
- **Prompt injection:** system prompt cố định phía server, dữ liệu tool/FAQ chỉ đi vào mục `[GROUNDING]`; bộ ca safety trong golden set kiểm tra các mẫu vượt quyền phổ biến (giả danh hệ thống, đòi mã giảm giá, đòi đặt đơn hộ).
- **Giới hạn nguồn lực:** ngân sách bước/tool mỗi lượt, timeout tool 2 giây, timeout HTTP adapter 1,8 giây, độ dài tin nhắn tối đa 2000 ký tự.
