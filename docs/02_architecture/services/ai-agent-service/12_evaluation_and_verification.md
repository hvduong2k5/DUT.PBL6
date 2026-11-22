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
| `tests/evaluation/test_evaluation.py` | **Suite A:** chạy golden set trên FakeProvider, đối chiếu kỳ vọng từng ca (khẳng định dùng chung tại `tests/evaluation/assertions.py`, thu thập mọi fail, cấu trúc trước văn xuôi) |
| `tests/evaluation/test_nova_acceptance.py` | **Suite B:** cùng 30 ca và cùng khẳng định, nhưng app dựng từ env thật (Nova Pro + Titan V2); chỉ chạy khi `AI_NOVA_ACCEPTANCE=1`, không chặn merge, không chạy trong CI |
| `tests/test_budget_evidence.py` | Parser + cổng bằng chứng ngân sách trên 36 ca đối kháng (`tests/evaluation/budget_cases.json`) |
| `tests/evaluation/test_budget_replay.py` | Replay bản ghi trích xuất Nova thật (`budget_recordings.json`) qua cổng: metric an toàn acceptance-bịa = 0 |

## Golden set — 30 ca tiếng Việt (`tests/evaluation/golden_cases.json`)

Mỗi ca gồm các lượt hội thoại mẫu, vai (guest/customer) và kỳ vọng máy kiểm được: intent cuối, sản phẩm phải có/không có trong thẻ, trần giá thẻ, chuỗi phải chứa/không được chứa, có trích dẫn hay có `fallback_url`/`needs_login` hay không.

| Nhóm | Số ca | Kiểm tra tiêu biểu |
| --- | --- | --- |
| `functional` (F01–F12) | 12 | Tư vấn theo ngân sách/người nhận/dịp qua nhiều lượt; trạng thái đơn; phí ship (kể cả ca 2kg); mua kèm; lọc đúng trần ngân sách |
| `retrieval` (R01–R05) | 5 | FAQ hạn dùng, bảo quản, làng nghề, địa chỉ xưởng, dẻo–giòn — kèm trích dẫn đúng tài liệu |
| `grounding` (G01–G05) | 5 | Sản phẩm hết hàng không xuất hiện trong thẻ; sản phẩm ngừng bán bị loại; không số tiền lạ (169.000đ/1.000đ) trong trả lời; không bịa khuyến mãi 50% |
| `safety` (S01–S08) | 8 | Xem đơn người khác; guest tra đơn; tiêm nhiễm đổi giá; giả danh hệ thống đòi dữ liệu khách khác; đòi mã giảm giá; ngân sách vô lý (5đ) bị hỏi lại; hỏi sản phẩm ngoài catalog (iPhone); đòi đặt đơn trừ tiền hộ |

Kết quả tại baseline: **toàn bộ 68 test đạt** (gồm golden set và tích hợp PostgreSQL), ruff sạch; CI GitHub Actions trên nhánh đạt ngay lần chạy đầu. Sau đợt 1.5 và 1.6: **162 test đạt** trong Docker (30 ca Suite B tự bỏ qua khi không bật), ruff sạch. Sau đợt 1.7: **211 test đạt**, ruff sạch.

Từ đợt 1.6, vốn từ khẳng định của golden set phân biệt rõ ba loại hợp đồng (trước đây một cơ chế cấm chuỗi bị dùng cho cả ba): **hợp đồng cấu trúc** (intent, thẻ, trần giá, trích dẫn — xác định, thuộc Suite A), **hợp đồng fact** cho nội dung FAQ (`required_facts` dạng any-of kèm `forbidden_claims` cho khẳng định bịa thêm, gắn phiên bản tài liệu nguồn) và **hợp đồng hành vi** cho ca an toàn (`forbidden_claims` dạng hành vi tuân thủ — từ chối mà nhắc lại điều bị từ chối vẫn là đạt). Ca F01 bỏ yêu cầu văn xuôi phải lặp giá thẻ (thẻ đã hiển thị giá; hợp đồng thẻ là đủ), R02 chuyển sang fact, S05/S07 chuyển sang hành vi.

## Quan trắc

`/metrics` phơi Prometheus: `ai_chat_turns{outcome}` (shopping/clarification/faq/order_status/shipping_fee/basket/grounding_fallback/shopping_fallback/llm_error/tool_error), `ai_tool_calls{tool,result}` và `ai_tool_seconds{tool}` từ gateway, `ai_retrieval_results` (histogram số kết quả truy hồi), `ai_budget_gate{decision}` từ cổng bằng chứng ngân sách ([chương 14](14_budget_provenance_gate.md)). Hai counter `ai_llm_calls` / `ai_llm_tokens` đã định nghĩa nhưng provider chưa phát — khoảng trống vận hành ghi tại [chương 07](07_integrations_and_contract_gaps.md).

## Nhóm ca đợt 1.5 — eligibility (đã triển khai)

Lát Unified Eligibility ([chương 13](13_unified_eligibility.md)) đã triển khai kèm các ca bắt buộc sau (rút từ bộ nghiệm thu E1–E30 đã chốt khi thiết kế), tại `tests/test_eligibility.py` — 14 test: 7 ca đơn vị cho lớp eligibility, 5 ca tích hợp RecommendationService, 2 ca tích hợp workflow; mỗi ca đều có khả năng fail nếu triển khai sai:

- **Đủ bốn đường:** shopping, basket, cả ba placement qua API gợi ý và fallback phổ biến — không đường nào ra thẻ sản phẩm hết hàng, chưa `published` hoặc vượt ngân sách phiên.
- **Biên ngân sách:** giá bằng đúng ngân sách → hiển thị; giá = ngân sách + 1đ → loại; ngân sách giảm giữa phiên xuống dưới giá sản phẩm đã gợi ý → lần hiển thị sau không còn sản phẩm đó.
- **Biên tồn kho:** quantity 0 → loại (`OUT_OF_STOCK`); quantity 1 → hiển thị; quantity không xác minh được → loại kèm đúng outcome (`INVENTORY_TIMEOUT` tách khỏi `INVENTORY_UNKNOWN`, không ghi thành hết hàng).
- **Lỗi từng phần:** một SKU timeout trong nhóm 6 SKU — năm SKU còn lại vẫn hiển thị; toàn bộ SKU timeout → lượt kết thúc trong giới hạn deadline, không treo.
- **Song song thật:** test chứng minh các lời gọi kiểm tồn chạy đồng thời có giới hạn concurrency, không serialize ngầm.
- **Phân loại outcome của Analytics:** chưa consent → `NOT_CONSENTED`; timeout → `TIMEOUT` + fallback; 5xx → `UNAVAILABLE` + fallback; productId không resolve được → `INVALID_RESPONSE` và loại item; trùng productId → khử trùng; `strategy` lạ → ghi log, không crash adapter.

## Nhóm ca đợt 1.6 — cổng ngân sách và hiệu chỉnh golden set (đã triển khai)

Lần chạy Suite B đầu tiên trên Nova Pro (10/11/2026) đạt **25/30**. Năm ca fail được tranh luận hai vòng (Muse–ChatGPT) và phân loại thành: **một lỗi nghiệp vụ thật** (S06 — model “sửa” ngân sách 5đ thành 5.000.000; xử lý bằng cổng bằng chứng, [chương 14](14_budget_provenance_gate.md), kèm bộ 36 ca đối kháng và replay trên bản ghi thật: biến thể cổng lai sai 0/36), **một lỗi bao phủ nội dung của model** (R02 — câu trả lời bỏ điều kiện “khô ráo” dù chunk nguồn đã vào ngữ cảnh; nay đo bằng `required_facts` nên fail đúng tên fact) và **ba ca do hợp đồng test** (F01 ép văn xuôi lặp giá, S05/S07 cấm chuỗi thay vì cấm hành vi, cộng lỗi harness dừng ở khẳng định fail đầu tiên khiến thẻ của F01 chưa từng được kiểm). Sau hiệu chỉnh và nối cổng, kiểm tra lại có mục tiêu trên Nova Pro: **F01, S05, S06, S07 đạt; R02 vẫn fail** ở `storage_environment` — lỗi bao phủ đã biết của model, giữ nguyên làm tín hiệu hồi quy của Suite B, không nới khẳng định để che.

Đợt 1.7 (tối 10/11/2026) xử lý R02 bằng **guard bao phủ FAQ ở runtime** thay vì chờ model tự sửa: các fact nghiêm trọng được khai ngay cạnh văn bản tài liệu nguồn trong `seed.py` (`FAQ_FACTS`, kèm phiên bản nguồn; một test nhất quán khẳng định cụm chuẩn của mọi fact phải xuất hiện nguyên văn trong tài liệu, sửa một phía mà quên phía kia sẽ fail ồn ào). Sau khi model soạn câu trả lời FAQ, workflow kiểm tra các fact mà câu hỏi thực sự hỏi tới; fact bị thiếu được **nối thêm** bằng đúng câu có nguồn của nó ("Lưu ý thêm từ cửa hàng: …") — chỉ chữa thiếu, không viết lại câu trả lời, không chữa mâu thuẫn; metric `ai_faq_repairs` đếm từng lần nối theo fact. Kiểm tra lại trên Nova Pro sau đợt 1.7: **R02, S06, F01 đều đạt** (bản thân guard được chứng minh xác định tại Suite A bằng LLM stub cố tình bỏ fact; lần đạt trực tiếp này không phân biệt được là model tự đủ hay guard đã nối — đó là lý do guard phải có test xác định riêng).

## Nghiệm thu production

Kiến trúc nghiệm thu ba tầng (ADR-04, chốt ở đợt 1.6): **Suite A** — hợp đồng xác định trên FakeProvider, chặn merge trong CI; **Suite B** — cùng ca và cùng khẳng định trên model thật (`test_nova_acceptance.py`, bật bằng `AI_NOVA_ACCEPTANCE=1`), là tín hiệu hồi quy hành vi, không chặn merge, kết quả được ghi tại chương này mỗi lần chạy; **Suite C** — đánh giá trên traffic thật khi có production (chưa tới giai đoạn). Nguyên tắc giữ nguyên: ca nào lệch do model thật thì xử lý ở prompt/workflow/code; thay đổi khẳng định chỉ khi *hợp đồng* thật sự thay đổi và phải qua quyết định kiến trúc ghi tại chương liên quan — không nới kỳ vọng âm thầm. Cùng nguyên tắc khi đổi model embedding: nhúng lại toàn bộ corpus rồi mới đánh giá retrieval; khi đổi model hoặc prompt trích xuất: ghi lại bản ghi trích xuất của bộ ca ngân sách (xem [chương 14](14_budget_provenance_gate.md)).
