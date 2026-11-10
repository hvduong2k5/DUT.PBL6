# MS-19 AI Agent Service

**Tài liệu thiết kế và triển khai v1.0 · 10/11/2026 · baseline nhánh `feat/ai-agent-service` tại `7af8b52`.**

MS-19 điều phối các tác nhân AI của hệ thống trên một agent runtime dùng chung: tư vấn mua hàng cho khách (US-AI-01), gợi ý cá nhân hóa tiêu thụ kết quả từ MS-11 (US-AI-04), và ở các đợt sau là copilot CSKH (US-AI-02), hỏi đáp tri thức nội bộ (US-AI-06), hỗ trợ nội dung/SEO (US-AI-03) và khuyến nghị điều hành (EPIC 25). Đây là **một service duy nhất** — các module nghiệp vụ bên trong là tổ chức code, không phải các microservice riêng.

Nguyên tắc cứng xuyên suốt: **AI chỉ gợi ý và soạn nháp; con người xác nhận mọi hành động ghi dữ liệu.** Service không tự thêm giỏ hàng, đặt đơn, hoàn tiền, đổi giá, đăng nội dung hay chạy chiến dịch. Mọi dữ kiện nghiệp vụ (giá, tồn kho, trạng thái đơn, phí vận chuyển) trong câu trả lời đều phải truy vết được về kết quả tool của chính lượt chat đó.

Đợt 1 đã triển khai xong trên nhánh `feat/ai-agent-service` (7 commit, WP0–WP7): nền tảng runtime + `shopping_advisor` + `recommendation` phía khách hàng. Runtime tại [services/ai-agent-service](../../../../services/ai-agent-service/README.md). Đọc [09 — Workflow tư vấn mua hàng](09_shopping_workflow_design.md) để nắm luồng chính, rồi [06 — Runtime và provider](06_agent_runtime_and_providers.md) cho cơ chế kiểm soát LLM.

| Tài liệu | Nội dung |
| --- | --- |
| [Master index](ms19_ai_agent_service_design.md) | Nguồn, ưu tiên và truy vết yêu cầu |
| [00 — HLD](00_ai_agent_service_hld.md) | Ranh giới, thành phần và cô lập lỗi |
| [01 — Domain và module](01_domain_and_modules.md) | Sáu module nghiệp vụ, phân kỳ và ranh giới “chỉ gợi ý” |
| [02 — Use case và vòng đời](02_usecases_and_lifecycle.md) | Phiên chat, consent, retention và dọn dẹp |
| [03 — Persistence](03_database_and_persistence.md) | Schema `ai_db`, store và migration |
| [04 — Messaging và recovery](04_messaging_and_recovery.md) | Đồng bộ tri thức, lỗi và hành vi dự phòng |
| [05 — API và bảo mật](05_api_contracts_and_security.md) | REST/SSE, xác thực khách, quyền sở hữu phiên |
| [06 — Runtime và provider](06_agent_runtime_and_providers.md) | Controlled runtime, Tool Gateway, LLM/embedding provider |
| [07 — Tích hợp và khoảng trống](07_integrations_and_contract_gaps.md) | Adapter tới các service bạn, contract còn thiếu |
| [08 — Roadmap](08_implementation_roadmap.md) | Đợt 1 đã xong, điều kiện kích hoạt production, đợt 2–3 |
| [09 — Workflow mua hàng](09_shopping_workflow_design.md) | Định tuyến intent, slot, kiểm tra tồn bắt buộc, grounding |
| [10 — Tri thức và retrieval](10_knowledge_retrieval_design.md) | Chunking, hybrid pgvector + FTS, ingestion và thu hồi |
| [11 — Tiêu thụ Recommendation](11_recommendation_consumption.md) | Ba placement, resolve qua Catalog và fallback phổ biến |
| [12 — Đánh giá và kiểm chứng](12_evaluation_and_verification.md) | Golden set 30 ca tiếng Việt, test phòng thủ và CI |

- [DDL PostgreSQL](ai_db.candidate.sql) là bản chụp của migration `services/ai-agent-service/migrations/postgres/001_ai_agent.sql` — file migration mới là nguồn thực thi.
- [OpenAPI](../../../03_api_specs/ai-agent-service.candidate.openapi.yaml) mô tả toàn bộ endpoint đợt 1.
- Nguồn yêu cầu gốc: [đề xuất AI Agent](../../ai_agent_specification_and_proposal.md) và [service boundary MS-19](../../service_boundary.md#ms-19-ai-agent-service).
- Phần gợi ý phía Analytics xem [MS-11 chương 12–15](../analytics-service/12_recommendation_engine_design.md) và [OpenAPI Recommendation](../../../03_api_specs/analytics-recommendation.candidate.openapi.yaml) (hiện trên nhánh của PR #62, chờ merge vào `dev`).

Trạng thái cần nhớ khi đọc bộ này: code đợt 1 chạy đầy đủ ở chế độ fixture (dữ liệu mẫu trong repo) và có sẵn adapter HTTP cho chế độ thật; việc nối dây production còn chờ IAM Bedrock, PR #62 merge và đối chiếu API công khai của Catalog/Inventory/Order/Shipping — chi tiết ở [chương 07](07_integrations_and_contract_gaps.md) và [chương 08](08_implementation_roadmap.md).
