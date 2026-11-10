# 08 — Roadmap

## Đợt 1 — đã hoàn thành (nhánh `feat/ai-agent-service`, 7 commit `f6a286c` → `7af8b52`)

| WP | Nội dung | Bằng chứng |
| --- | --- | --- |
| WP0 | Đăng ký MS-19 / cổng 8019 trong service boundary; MS-11 thu hẹp còn “Analytics & DSS” | Commit trên nhánh; [service boundary](../../service_boundary.md#ms-19-ai-agent-service) |
| WP1 | Khung service: FastAPI, cấu hình env, compose dev/test, migration ai_db | `compose.dev.yml` chạy API :8019 + pgvector |
| WP2 | Provider (Fake/Gemini/Bedrock + embedding), Tool Gateway, runtime ngân sách, store, xác thực JWT khách | Unit test gateway/auth/provider |
| WP3 | Adapter fixture + HTTP, retrieval hybrid, gieo FAQ, RecommendationService | Test knowledge/recommendation |
| WP4 | Workflow mua hàng xác định: trích xuất slot có kiểm hợp lệ, hỏi làm rõ tối đa 2 vòng, kiểm tra tồn bắt buộc, grounding hậu kiểm, SSE, fallback liên kết lọc | Test flow đầu-cuối trên fixture |
| WP5+WP6 | Lưu trữ theo consent, job retention, phòng thủ tiêm nhiễm/timeout tool, tiêu thụ contract Analytics + fallback phổ biến | Test defenses/consent/retention |
| WP7 | Golden set 30 ca tiếng Việt + test tích hợp PostgreSQL, README bàn giao | Toàn suite: 68 test đạt trong Docker (gồm tích hợp pgvector thật), ruff sạch; smoke trực tiếp trên máy dev: health/ready đạt, một lượt chat thật trả thẻ sản phẩm có grounding, phiên guest hết hạn đúng +15 ngày; CI GitHub Actions chạy đầu tiên trên nhánh đạt (run #1, commit `7af8b52`) |

## Kích hoạt production (không phải viết code mới, là nối dây + nghiệm thu)

1. **IAM Bedrock:** cấp quyền, đặt `AI_LLM_PROVIDER=bedrock`, `AI_EMBEDDING_PROVIDER=titan`; chạy lại **toàn bộ golden set trên Nova Pro** — lần chạy này là nghiệm thu production, vì FakeProvider không chứng minh được chất lượng model thật.
2. **Analytics thật:** chờ PR #62 merge vào `dev`; tắt fixture cho riêng adapter Analytics trước để kiểm chứng placement `home` có/không consent.
3. **Đối chiếu adapter:** đọc controller thật của Catalog/Inventory/Order/Shipping, sửa `http_clients.py`, bổ sung test contract cho từng service ([chương 07](07_integrations_and_contract_gaps.md)).
4. **Triển khai:** đưa service vào dự án `pbl6` trên Coolify (VPS đã có Kong/Keycloak/Kafka), thêm route Kong, đặt lịch chạy job `cleanup`, cấu hình `AI_JWT_*` trỏ Keycloak production.

## Đợt 2 — phạm vi nhân viên (cùng service)

- `knowledge_assistant` (US-AI-06): hỏi đáp tri thức nội bộ trên chính knowledge platform đợt 1, cộng: gieo permission `AI_KNOWLEDGE_USE` phía Identity, xác thực nhân viên bằng JWT chứa `permission_grants` (baseline 7-B), lọc `visibility` theo quyền **ngay trong truy vấn retrieval**, và khóa endpoint ingestion bằng grant tương ứng.
- `support_copilot` (US-AI-02): soạn nháp phản hồi từ ticket. Tiền lệ bắt buộc: Care chốt contract đọc ticket (hiện chưa tồn tại trong repo). Copilot chỉ soạn nháp; nhân viên CSKH là người gửi.
- Consumer Kafka đồng bộ tri thức từ sự kiện vòng đời sản phẩm của Catalog, thay dần đường ingestion thủ công.

## Đợt 3

- `content_assistant` (US-AI-03): nháp nội dung/SEO có bước người duyệt trước đăng.
- `strategic_advisor` (EPIC 25): thẻ hành động chiến lược từ kết quả DSS của MS-11. Còn mở: ngưỡng churn/forecast/expiry, tiêu chí chất lượng khuyến nghị, quyền sở hữu quyết định và ghi nhận hậu quả của thẻ hành động — thuộc nhóm DSS/PO, không tự quyết trong service này.

## Tiêu chí “xong” cho mọi đợt

Golden set mở rộng theo module mới và đạt trên provider production; mọi số kiện trong trả lời truy vết được về tool; không đường ghi dữ liệu nào thiếu bước người xác nhận; tài liệu trong thư mục này cập nhật cùng PR với code.
