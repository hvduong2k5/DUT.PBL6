
**Trạng thái: đã triển khai ngày 10/11/2026** trên nhánh `feat/ai-agent-service` — thành phần `platform/budget_evidence.py`, nối vào bước trích xuất nhu cầu của luồng shopping ([chương 09](09_shopping_workflow_design.md)), kèm bộ ca đối kháng 36 ca và bản ghi trích xuất thật của Nova Pro tại `tests/evaluation/`. Nội dung chương này là kết quả tranh luận hai vòng giữa Muse và ChatGPT về năm ca fail của lần chạy golden set trên Nova Pro (ADR-01 tại chương này; ADR-02→04 về đánh giá nằm ở [chương 12](12_evaluation_and_verification.md)).

## Vấn đề

Nghiệm thu production đầu tiên trên Nova Pro (10/11/2026, golden set 25/30) lộ một lỗi đúng-sai ở ca S06: khách viết “Ngân sách của tôi là 5đ”, model trích xuất `budget_vnd=5000000` — nó đã *sửa* 5 đồng thành 5 triệu. Bộ kiểm tra hợp lệ lúc đó chỉ hỏi “số có nằm trong khoảng 10.000đ–1.000.000.000đ không” — 5.000.000 nằm gọn trong khoảng nên lọt, và workflow đi mua sắm với một ngân sách bịa. FakeProvider trích xuất nguyên văn nên bộ test xác định không bao giờ thấy lỗi này.

Gốc rễ là kiểm tra sai câu hỏi. Validator kiểm tra **miền giá trị** của con số model đưa ra; invariant đúng phải là **nguồn gốc**: con số được chấp nhận chỉ khi nó thuộc về tập giá trị mà chính lời của khách đã nêu (*provenance violation*). Việc này không sửa được bằng prompt — cùng một model, cùng prompt đã tăng cường nguyên văn, vẫn “sửa” tiếp ở ca sửa ngân sách giữa hội thoại (xem bằng chứng ở mục Nghiệm thu).

## Thiết kế: cổng bằng chứng ngân sách

Một parser xác định, phạm vi hẹp theo miền (tiền tệ tiếng Việt trong ngữ cảnh mua quà), đọc **toàn bộ các tin nhắn của khách theo thứ tự** và phân loại bằng chứng ngân sách thành bốn trạng thái. Lượt phát biểu ngân sách mới nhất thắng — kể cả khi lượt mới là một đính chính xuống một giá trị không hợp lệ.

| Trạng thái | Nghĩa | Ví dụ | Hành xử của cổng |
| --- | --- | --- | --- |
| `EXACT` | Số tiền tường minh ở vị trí ngân sách (đi kèm “ngân sách”, “tối đa”, “khoảng”, “tầm”, “dưới”…) hoặc dạng trần (“không quá 200k”) | “ngân sách 200k”, “2 triệu rưỡi”, “dưới 100k” | Parser là **thẩm quyền duy nhất**: dùng giá trị của parser, bỏ qua số của model. Dạng trần dùng chính giá trị trần vì workflow vốn coi ngân sách là giá tối đa. |
| `INVALID` | Ngân sách tường minh nhưng ngoài miền (dưới 10.000đ hoặc trên 1 tỷ) | “5đ”, “500đ”, “5 tỷ” | **Cấm model ghi đè tuyệt đối.** Ngân sách bị gỡ, workflow hỏi lại. Một phát biểu không hợp lệ là bằng chứng, không phải sự vắng mặt của bằng chứng. |
| `AMBIGUOUS` | Có phát biểu ngân sách nhưng không có một giá trị xác định | “2–3 trăm nghìn”, “vài trăm”, “ngân sách 500” (không đơn vị) | Không đoán: gỡ ngân sách, hỏi lại để khách chốt con số. |
| `ABSENT` | Không có số tiền nào ở vị trí ngân sách | “ngân sách sinh viên thôi”, “giá hợp lý là được” | Số của model chỉ được dùng khi **khớp một số tiền khách đã viết nguyên văn ở vị trí không phải giá sản phẩm và không bị phủ định/giả định**. Model được quyền *nhận diện* số nào trong lời khách là ngân sách; không bao giờ được *tạo ra* một sự kiện tiền tệ từ suy luận. |

Các quy tắc biên đã chốt bằng ca kiểm thử, không phải bằng cảm tính:

- **Neo theo câu:** marker ngân sách/giá chỉ có hiệu lực trong cùng một câu — “Tôi có ngân sách 500k. Hộp quà đó giá 200k” không cho số 200k mượn neo ngân sách của câu trước (ca B21).
- **Phủ định và giả định bị triệt tiêu:** số tiền đi sau “không phải”, “nếu”, “giả sử”… không thành bằng chứng — và cũng không được dùng làm căn cứ hỗ trợ cho số của model ở trạng thái `ABSENT` (ca B35: “Giả sử ngân sách 500k thì mua được gì?”).
- **Giá sản phẩm không phải ngân sách:** số tiền ở vị trí giá (“giá 200k”, đi sau tên sản phẩm) bị loại khỏi tập hỗ trợ — Nova đã thật sự lấy giá hộp quà 169.000đ làm ngân sách ở ca B22 và cổng chặn đúng ca này.
- **Đính chính là điểm bắt đầu bằng “À nhầm”:** từ “nhầm” *không* phải dấu phủ định (ca B28); hiệu lực đính chính đến từ thứ tự lượt, không từ từ khóa.
- **Provenance tối thiểu:** phiên lưu kèm quyết định của cổng (`_budget_source`: decision, trạng thái bằng chứng, chỉ số tin nhắn nguồn, đoạn văn bản gốc của riêng số tiền đó) và phát metric `ai_budget_gate{decision}` với sáu outcome: `evidence`, `invalid_blocked`, `ambiguous_clarify`, `model_supported`, `model_unsupported`, `no_budget`.

Prompt trích xuất vẫn được tăng cường như một lớp phụ (yêu cầu trích nguyên văn, không sửa số), nhưng không ai coi nó là hàng rào: bằng chứng ở mục Nghiệm thu cho thấy prompt một mình không đủ.

## Những gì lát này không thay đổi

Hợp đồng bên ngoài, schema `ai_db` (provenance nằm trong JSON slot sẵn có), ranh giới “AI chỉ gợi ý” và lớp eligibility của [chương 13](13_unified_eligibility.md) giữ nguyên. Cổng chỉ quyết định **giá trị `budget_vnd` nào được phép vào slot**; mọi kiểm tra phía sau (trần cứng cho thẻ, grounding số tiền trong câu trả lời) hoạt động như cũ. Hàm `extract_money` mù ngữ cảnh vẫn giữ đúng vai trò của nó — hậu kiểm grounding cho câu trả lời — và không dùng chung cho trích ngân sách nữa.

Giới hạn đã biết: số viết bằng chữ (“năm trăm nghìn”) chưa được parse; danh sách từ chỉ sản phẩm để nhận diện vị trí giá là đặc thù miền hàng của shop này; parser chưa được chứng minh trên tiếng Việt thực tế ngoài bộ ca đối kháng tự xây — đó là rủi ro mở lớn nhất của thiết kế, theo dõi khi có traffic thật.

## Nghiệm thu

Ba lớp bằng chứng, tách bạch phạm vi hiệu lực (đây là điểm tranh luận được chốt rõ nhất):

1. **Unit test của parser** trên bộ 36 ca đối kháng (`tests/evaluation/budget_cases.json`: exact 5, invalid 5, ambiguous 5, absent 4, va chạm ngữ cảnh 7, đính chính 6, phủ định/giả định 4) — chứng minh parser xử lý đúng các biểu thức đã định nghĩa. Tại `tests/test_budget_evidence.py`.
2. **Replay ngoại tuyến trên bản ghi thật:** trích xuất của Nova Pro được ghi lại đúng một lần cho 36 ca (`tests/evaluation/budget_recordings.json`, kèm model/prompt/schema version và hash hội thoại) rồi phát lại qua cả ba biến thể cổng tại `tests/evaluation/test_budget_replay.py` — chứng minh cổng xử lý đúng cùng một tập đầu ra của extractor, và cho phép so sánh biến thể với chi phí model bằng không. Kết quả trên bản ghi ngày 10/11/2026:

   | Biến thể cổng | Sai | Các ca sai |
   | --- | --- | --- |
   | A — chỉ prompt + kiểm khoảng (cách cũ) | 7/36 | B11, B12, B15, B22, B29, B30, B35 |
   | B — cổng cứng “số của model phải nằm trong mọi số khách đã viết” | 3/36 | B22 (giá thành ngân sách), B30 (ngân sách cũ trước đính chính), B35 (giả định thành thật) |
   | C — cổng bằng chứng lai (đã triển khai) | **0/36** | — |

   Ba ca đáng ghi nhất đều là hành vi thật của Nova trên bản ghi: B29 — khách đính chính ngân sách *xuống* “5đ”, model “sửa” thành 5.000.000 (đúng lỗi S06 ở dạng đính chính; prompt nguyên văn không chặn được, cổng chặn); B30 — khách hạ ngân sách 1 triệu → 700k, model giữ 1 triệu cũ (parser theo lượt mới nhất thắng); B22 — model lấy giá sản phẩm 169.000đ làm ngân sách (chặn nhờ loại vị trí giá khỏi tập hỗ trợ).
3. **Chạy thật trên model:** Suite B (xem [chương 12](12_evaluation_and_verification.md)) — sau khi nối cổng, kiểm tra lại đúng năm ca từng fail cho kết quả F01/S05/S06/S07 đạt; S06 đạt ở hành vi quan sát được (hỏi lại ngân sách) nhờ cổng, không nhờ model tự sửa.

Replay là bằng chứng để **chọn cổng**, không phải nghiệm thu production: chất lượng extractor trên dữ liệu thật vẫn cần quan trắc qua metric `ai_budget_gate` khi có traffic, và bản ghi phải ghi lại mỗi khi đổi model hoặc prompt trích xuất.
