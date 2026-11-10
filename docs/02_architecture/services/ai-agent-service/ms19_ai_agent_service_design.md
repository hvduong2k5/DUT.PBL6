# MS-19 — Nguồn và truy vết thiết kế

Đọc [HLD](00_ai_agent_service_hld.md) để nắm phạm vi, [tích hợp và khoảng trống](07_integrations_and_contract_gaps.md) để kiểm tra mức bằng chứng của từng contract, rồi [roadmap](08_implementation_roadmap.md) để lập PR. Các đường dẫn nguồn dưới đây thuộc baseline ghi trong [README](README.md).

## Thứ tự ưu tiên khi có khác biệt

| Câu hỏi | Nguồn quyết định | Cách áp dụng |
| --- | --- | --- |
| AI Agent gồm những nghiệp vụ nào, phục vụ ai? | [Đề xuất AI Agent](../../ai_agent_specification_and_proposal.md), backlog US-AI-01/02/03/04/06 và EPIC 25 | Theo story và tiêu chí chi tiết. Cả sáu nhóm nghiệp vụ nằm trong **một** service MS-19 (quyết định 09/11/2026), phân kỳ theo đợt. |
| Ranh giới MS-19 với MS-11 ở đâu? | [Service boundary](../../service_boundary.md#ms-19-ai-agent-service) | MS-11 sở hữu tính toán phân tích (KPI/RFM/forecast/association, Recommendation Engine) trên dữ liệu của nó; MS-19 sở hữu hội thoại, runtime và diễn giải kết quả phân tích thành gợi ý cho người dùng. |
| LLM và kho vector dùng gì? | Quyết định 09/11/2026 (ghi trong [README service](../../../../services/ai-agent-service/README.md)) | LLM production: Amazon Bedrock Converse API, model duy nhất `amazon.nova-pro-v1:0` (đổi được bằng cấu hình). Embedding production: Titan Embeddings V2, 1024 chiều. Vector store: PostgreSQL `ai_db` riêng + pgvector (phương án B của đề xuất). |
| Runtime dùng LangGraph/Redis không? | Quyết định 09/11/2026 trong service boundary MS-19 | **Không, ở đợt này.** Đề xuất gốc (§3.2) kê LangGraph + RedisSaver; bản triển khai trì hoãn cả hai: workflow xác định do service tự quản, trạng thái phiên ở PostgreSQL của chính service. Cần xác nhận lại với nhóm khi mở PR. |
| Consent gợi ý cá nhân hóa thuộc ai? | Contract Analytics đã kiểm chứng trên nhánh PR #62 | Analytics sở hữu (GET/PUT `/api/v1/recommendations/consent`). MS-19 **không** nhân bản consent này; consent riêng của MS-19 chỉ bao phủ việc lưu lịch sử chat (xem [chương 02](02_usecases_and_lifecycle.md)). |
| Dữ kiện giá/tồn/đơn lấy ở đâu? | Nguyên tắc grounding của đợt 1 ([chương 09](09_shopping_workflow_design.md)) | Chỉ từ tool gọi service sở hữu dữ liệu, trong cùng lượt chat. RAG không bao giờ là nguồn giá/tồn. |

## Story, chương và gate

| Story | Đợt | Thiết kế | Gate chính |
| --- | --- | --- | --- |
| US-AI-01 Tư vấn mua hàng | 1 — đã xong | [Workflow](09_shopping_workflow_design.md), [Runtime](06_agent_runtime_and_providers.md) | Kiểm tra tồn kho bắt buộc trước khi gợi ý; grounding hậu kiểm mọi số tiền |
| US-AI-04 Gợi ý cá nhân hóa | 1 — đã xong (phía tiêu thụ) | [Tiêu thụ Recommendation](11_recommendation_consumption.md) | Engine thuộc MS-11 (PR #62); MS-19 chỉ tiêu thụ + fallback |
| Tiện ích sau mua (trạng thái đơn, phí ship) | 1 — đã xong | [Workflow §đơn hàng/phí ship](09_shopping_workflow_design.md) | Tool bắt buộc; đơn chỉ trả cho đúng chủ sở hữu |
| Lớp FAQ tĩnh | 1 — đã xong | [Tri thức và retrieval](10_knowledge_retrieval_design.md) | Corpus FAQ gieo sẵn; trích dẫn nguồn trong trả lời |
| US-AI-06 Hỏi đáp tri thức nội bộ | 2 | [Roadmap](08_implementation_roadmap.md) | Cần permission `AI_KNOWLEDGE_USE` phía Identity + lọc quyền ngay trong retrieval |
| US-AI-02 Copilot CSKH | 2 | [Roadmap](08_implementation_roadmap.md) | Care chưa có contract API ticket trong repo (chỉ còn file biên dịch) — cần owner chốt contract đọc |
| US-AI-03 Nội dung & SEO | 3 | [Roadmap](08_implementation_roadmap.md) | Bước con người xác nhận trước khi đăng |
| EPIC 25 Khuyến nghị điều hành | 3 | [Roadmap](08_implementation_roadmap.md) | Đọc kết quả DSS của MS-11; quyền sở hữu quyết định/hậu quả còn mở |

Mọi mô tả trong bộ này bám theo code tại baseline `7af8b52`. Chỗ nào là thiết kế dự kiến cho đợt sau, tài liệu ghi rõ “đợt 2/3” hoặc “chưa triển khai”; chỗ nào code còn đơn giản hóa so với thiết kế đầy đủ, ghi ở mục “Đơn giản hóa đã biết” của chương tương ứng.
