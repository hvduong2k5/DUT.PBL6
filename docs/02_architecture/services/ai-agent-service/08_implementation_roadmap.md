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

## Đợt 1.5 — Unified Eligibility (đã triển khai 10/11/2026)

Lát correctness đặt trước contract test và trước kích hoạt production, vì lỗi nó sửa đang tồn tại trong code đợt 1: đường recommendation/basket có thể hiển thị sản phẩm hết hàng hoặc vượt ngân sách. Thiết kế đầy đủ tại [chương 13](13_unified_eligibility.md); tóm tắt phạm vi:

- Một lớp eligibility dùng chung cho cả bốn đường sinh Product Card (shopping, basket, ba placement của RecommendationService, fallback phổ biến): khử trùng theo SKU → resolve `published` qua Catalog → lọc ngân sách cứng → kiểm tồn song song có deadline toàn cục, fail-closed theo từng SKU.
- Outcome taxonomy hai cấp (recommendation và sản phẩm) cho quan trắc; ghi log `strategy` của Analytics thay vì suy diễn lời giải thích.
- Spike đã xong: ngữ nghĩa `available_qty` = khả dụng để bán sau giữ hàng (INV-BI-01) — khớp interface adapter; Inventory thật là gRPC và đã có batch (xem [chương 13](13_unified_eligibility.md)). Đo latency tồn kho thật để tinh chỉnh timeout để lại cho bước nối dây production.
- Nghiệm thu bằng nhóm ca eligibility tại [chương 12](12_evaluation_and_verification.md).

Thứ tự việc sau lát này: (1) contract test 4 tầng với MS-11 khi PR #62 merge, dùng artifact schema v1 của Analytics làm nguồn sự thật; (2) shadow reranking — chỉ đo, không ảnh hưởng kết quả — khi có retrieval ứng viên thật từ Catalog và một rerank contract; (3) feedback forwarding: chuyển tiếp signal sang `POST /api/v1/recommendations/events` của Analytics kiểu fire-and-forget kèm idempotency (hướng xử lý của khoảng trống #6 tại [chương 07](07_integrations_and_contract_gaps.md), thực hiện khi nối dây thật).

## Kích hoạt production (không phải viết code mới, là nối dây + nghiệm thu)

1. **IAM Bedrock:** cấp quyền, đặt `AI_LLM_PROVIDER=bedrock`, `AI_EMBEDDING_PROVIDER=titan`; chạy lại **toàn bộ golden set trên Nova Pro** — lần chạy này là nghiệm thu production, vì FakeProvider không chứng minh được chất lượng model thật. **Tiến độ 10/11/2026:** IAM đã được cấp; kiểm chứng thật ở **ap-southeast-2** đạt cho cả hai model (Nova Pro gọi trực tiếp 63/17 token, Titan V2 trả vector 1024 chiều — chi tiết ở ghi chú region, chương 06). **Đã xong cùng ngày:** golden set trên Nova Pro đạt 25/30 ở lần chạy đầu; năm ca fail được phân loại (một lỗi nghiệp vụ thật S06, một lỗi bao phủ nội dung R02, ba ca do hợp đồng test) và xử lý ở đợt 1.6 — cổng bằng chứng ngân sách ([chương 14](14_budget_provenance_gate.md)), hiệu chỉnh khẳng định và kiến trúc Suite A/B tại [chương 12](12_evaluation_and_verification.md); kiểm tra lại có mục tiêu đạt 4/5, R02 còn là lỗi bao phủ đã biết của model. Việc còn lại của kích hoạt production: PR #62 merge và nối dây API thật (gồm adapter gRPC cho Inventory).
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
