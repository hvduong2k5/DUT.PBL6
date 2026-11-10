# 12 — Đánh giá và kiểm chứng

## Cách chạy

```bash
docker compose -f services/ai-agent-service/compose.test.yml up --build --abort-on-container-exit
```

Compose test dựng PostgreSQL pgvector dùng một lần (dữ liệu trên tmpfs) và chạy toàn bộ pytest trong image test. CI: workflow `.github/workflows/ai-agent-service.yml` chạy đúng lệnh trên cho mọi push/PR chạm vào `services/ai-agent-service/**`. Ruff (line-length 100, bộ luật E4/E7/E9/F/I trong `pyproject.toml`) là cổng phong cách.

## Bản đồ kiểm thử

| File test | Phủ |
| --- | --- |
| `tests/test_health.py` | `/health`, `/health/ready` |
| `tests/test_auth.py` | JWT khách: token hợp lệ/không hợp lệ, claim `user_id`, phân vai guest/customer |
| `tests/test_gateway.py` | Thứ tự kiểm soát của Tool Gateway: allowlist, requires_customer, schema, timeout, audit |
| `tests/test_providers.py` | Ba LLM provider + embedding trên kiểu chuẩn; FakeProvider giả lập lỗi/timeout |
| `tests/test_knowledge.py` | Chunking, hybrid retrieval, lọc hiệu lực, deactivate |
| `tests/test_shopping_flow.py` | Luồng chat đầu-cuối trên fixture: làm rõ, gợi ý, hết hàng, fallback |
| `tests/test_defenses.py` | LLM bịa số tiền (grounding thay trả lời), tiêm nhiễm qua slot, timeout tool, consent không lưu tin nhắn, retention hết hạn |
| `tests/test_postgres_stores.py` | Tích hợp store PostgreSQL + pgvector thật (đánh dấu `integration`, tự bỏ qua khi không có DB) |
| `tests/evaluation/test_evaluation.py` | Chạy golden set và đối chiếu kỳ vọng từng ca |

## Golden set — 30 ca tiếng Việt (`tests/evaluation/golden_cases.json`)

Mỗi ca gồm các lượt hội thoại mẫu, vai (guest/customer) và kỳ vọng máy kiểm được: intent cuối, sản phẩm phải có/không có trong thẻ, trần giá thẻ, chuỗi phải chứa/không được chứa, có trích dẫn hay có `fallback_url`/`needs_login` hay không.

| Nhóm | Số ca | Kiểm tra tiêu biểu |
| --- | --- | --- |
| `functional` (F01–F12) | 12 | Tư vấn theo ngân sách/người nhận/dịp qua nhiều lượt; trạng thái đơn; phí ship (kể cả ca 2kg); mua kèm; lọc đúng trần ngân sách |
| `retrieval` (R01–R05) | 5 | FAQ hạn dùng, bảo quản, làng nghề, địa chỉ xưởng, dẻo–giòn — kèm trích dẫn đúng tài liệu |
| `grounding` (G01–G05) | 5 | Sản phẩm hết hàng không xuất hiện trong thẻ; sản phẩm ngừng bán bị loại; không số tiền lạ (169.000đ/1.000đ) trong trả lời; không bịa khuyến mãi 50% |
| `safety` (S01–S08) | 8 | Xem đơn người khác; guest tra đơn; tiêm nhiễm đổi giá; giả danh hệ thống đòi dữ liệu khách khác; đòi mã giảm giá; ngân sách vô lý (5đ) bị hỏi lại; hỏi sản phẩm ngoài catalog (iPhone); đòi đặt đơn trừ tiền hộ |

Kết quả tại baseline: **toàn bộ 68 test đạt** (gồm golden set và tích hợp PostgreSQL), ruff sạch; CI GitHub Actions trên nhánh đạt ngay lần chạy đầu.

## Quan trắc

`/metrics` phơi Prometheus: `ai_chat_turns{outcome}` (shopping/clarification/faq/order_status/shipping_fee/basket/grounding_fallback/shopping_fallback/llm_error/tool_error), `ai_tool_calls{tool,result}` và `ai_tool_seconds{tool}` từ gateway, `ai_retrieval_results` (histogram số kết quả truy hồi). Hai counter `ai_llm_calls` / `ai_llm_tokens` đã định nghĩa nhưng provider chưa phát — khoảng trống vận hành ghi tại [chương 07](07_integrations_and_contract_gaps.md).

## Nghiệm thu production

Đổi provider không đổi test: golden set chạy trên FakeProvider để khóa hành vi xác định trong CI. Khi bật Bedrock, **chạy lại nguyên golden set với Nova Pro** và coi đó là nghiệm thu phát hành — ca nào lệch do model thật thì xử lý ở prompt/workflow, không nới kỳ vọng âm thầm. Cùng nguyên tắc khi đổi model embedding: nhúng lại toàn bộ corpus rồi mới đánh giá retrieval.
