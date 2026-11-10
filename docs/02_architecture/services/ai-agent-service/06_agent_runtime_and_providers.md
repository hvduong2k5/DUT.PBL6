# 06 — Runtime và provider

## Controlled runtime

Thứ tự bước của một lượt chat do **code workflow** quyết định; LLM chỉ làm hai việc: trích xuất nhu cầu thành cấu trúc và diễn đạt câu trả lời từ dữ kiện đã có. Ngân sách mỗi lượt nằm ở `platform/runtime.py` (`TurnState`): mặc định tối đa **8 bước** và **10 lần gọi tool** (cấu hình `AI_MAX_STEPS` / `AI_MAX_TOOL_CALLS`), vượt ngân sách ném `BudgetExceeded` và lượt kết thúc bằng fallback danh mục. Việc đếm tool call đi qua cùng một hàm `call_tool` duy nhất nên không có đường gọi tool “ngoài luồng”.

## Tool Execution Gateway

Mọi tool call bắt buộc qua `platform/gateway.py`, theo đúng thứ tự kiểm soát:

1. **Allowlist:** tên tool phải có trong registry, ngược lại lỗi `tool_not_allowed`.
2. **Yêu cầu khách:** tool gắn `requires_customer` bị chặn với khách vãng lai (`customer_required`).
3. **Validate đối số:** schema Pydantic riêng cho từng tool; sai schema → `invalid_arguments`.
4. **Timeout:** `asyncio.wait_for` với `AI_TOOL_TIMEOUT_SECONDS` (mặc định 2 giây) → `tool_timeout`.
5. **Kiểm tra kết quả:** handler phải trả object; khác → `invalid_tool_result`.
6. **Metric + audit:** mọi kết quả (kể cả lỗi) đều ghi counter/histogram Prometheus và gọi recorder (run id, tên tool, args, trạng thái, độ trễ) phục vụ ghi `tool_calls`.

Registry đợt 1 — cả năm tool đều chỉ đọc:

| Tool | Đối số | Bắt buộc đăng nhập | Nguồn |
| --- | --- | --- | --- |
| `search_products` | `query`, `max_price_vnd?`, `limit` 1–6 (mặc định 5) | Không | Catalog (chỉ sản phẩm `published`) |
| `check_stock` | `sku` | Không | Inventory |
| `get_order_status` | `order_id` | **Có** | Order (chuyển tiếp Authorization của khách; đối chiếu chủ đơn trong handler) |
| `estimate_shipping` | `province`, `weight_grams` 1–100.000 (mặc định 500) | Không | Shipping |
| `get_recommendations` | `placement`, `product_id?`, `sku_codes[]`, `limit` 1–6 (mặc định 4) | Không | Analytics |

## Lớp provider

Kiểu chuẩn dùng chung (hình dạng theo Bedrock Converse): `LLMRequest` (messages, system prompt, tool specs, cấu hình sinh), `LLMResponse` (content, tool calls, finish reason, usage), `LLMStreamEvent` (`text_delta` / `tool_call` / `usage` / `completed`). Cấu hình sinh mặc định: temperature 0.3, tối đa 700 token (`AI_MAX_REPLY_TOKENS`). Code nghiệp vụ không rẽ nhánh theo provider.

| Provider | Dùng cho | Ghi chú triển khai |
| --- | --- | --- |
| `FakeProvider` | Test, CI, dev mặc định | Xác định hoàn toàn: trích xuất slot bằng luật (regex ngân sách “triệu/k/đ”, danh sách người nhận/dịp/vị), trả lời từ khối `[GROUNDING]`; giả lập được lỗi, timeout và độ trễ cho test khả năng chịu lỗi |
| `GeminiProvider` | Dev thủ công (tùy chọn) | Qua endpoint tương thích OpenAI của Gemini (`AI_GEMINI_MODEL`, mặc định `gemini-2.0-flash`), timeout 30 giây. Không phải đường production |
| `BedrockProvider` | **Production** | Converse API + `converse_stream` qua aioboto3; model `AI_MODEL_ID`, mặc định `amazon.nova-pro-v1:0`; region mặc định `ap-southeast-2` (quyết định 10/11/2026 — xem ghi chú region bên dưới bảng biến môi trường) |

Embedding có cùng mẫu trừu tượng: `FakeEmbedder` (băm SHA-256 có hạt giống, chuẩn hóa, 1024 chiều — tính liên quan trong test do nửa từ khóa của hybrid gánh), `GeminiEmbedder`, và `TitanEmbedder` (Bedrock `invoke_model`, model mặc định `amazon.titan-embed-text-v2:0`, xin đúng số chiều cấu hình và chuẩn hóa). **Đổi model embedding bắt buộc nhúng lại toàn bộ corpus** vì vector của hai model không cùng không gian; từng chunk đã mang `embedding_model`/`embedding_version` để phục vụ việc này.

## Cấu hình (biến môi trường)

| Biến | Mặc định | Ý nghĩa |
| --- | --- | --- |
| `AI_DATABASE_URL` | `postgresql://ai_agent:ai_agent@localhost:54329/ai_db` | Chuỗi kết nối ai_db |
| `AI_LLM_PROVIDER` | `fake` | `fake` / `gemini` / `bedrock` |
| `AI_MODEL_ID` | `amazon.nova-pro-v1:0` | Model LLM production |
| `GEMINI_API_KEY`, `AI_GEMINI_MODEL`, `AI_GEMINI_BASE_URL` | — / `gemini-2.0-flash` / endpoint v1beta | Chỉ cho dev Gemini |
| `AWS_REGION` | `ap-southeast-2` | Region Bedrock |

**Ghi chú region (10/11/2026):** mặc định ban đầu là `ap-southeast-1` (Singapore), nhưng kiểm chứng thực tế cho thấy Titan Text Embeddings V2 **không được AWS cung cấp tại Singapore** và Nova Pro tại đó chỉ gọi được qua inference profile xuyên region (`apac.amazon.nova-pro-v1:0`). Bao chốt chuyển toàn bộ sang **ap-southeast-2 (Sydney)** — region gần nhất phục vụ cả hai model ngay trong region bằng model ID trực tiếp (đã kiểm chứng bằng lời gọi thật: Nova Pro 63/17 token, Titan V2 trả vector 1024 chiều).
| `AI_EMBEDDING_PROVIDER` / `AI_EMBEDDING_MODEL` / `AI_EMBEDDING_DIM` | `fake` / `amazon.titan-embed-text-v2:0` / 1024 | Embedding |
| `AI_FIXTURES` | `1` (bật) | `1` = adapter fixture + store in-memory; `0` = HTTP + PostgreSQL |
| `AI_CATALOG_BASE_URL` / `AI_INVENTORY_BASE_URL` / `AI_ORDER_BASE_URL` / `AI_SHIPPING_BASE_URL` / `AI_ANALYTICS_BASE_URL` | cổng 8005 / 8001 / 8004 / 8012 / 8011 | Địa chỉ các service bạn (theo cổng đăng ký trong service boundary) |
| `AI_JWT_ISSUER` / `AI_JWT_AUDIENCE` / `AI_JWKS_URL` | trống | Xác thực JWT khách |
| `AI_TOOL_TIMEOUT_SECONDS` / `AI_MAX_STEPS` / `AI_MAX_TOOL_CALLS` / `AI_MAX_REPLY_TOKENS` | 2.0 / 8 / 10 / 700 | Ngân sách runtime |
| `AI_RETENTION_CUSTOMER_DAYS` / `AI_RETENTION_GUEST_DAYS` | 30 / 15 | Retention phiên |
| `AI_MIGRATIONS_DIR` | `migrations/postgres` (image Docker: `/app/migrations/postgres`) | Thư mục migration |
