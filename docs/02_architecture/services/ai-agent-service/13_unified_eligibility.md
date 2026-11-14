# 13 — Unified Product Eligibility (đợt 1.5)

**Trạng thái: đã triển khai ngày 10/11/2026** trên nhánh `feat/ai-agent-service` — thành phần `platform/eligibility.py`, nối vào cả bốn đường sinh Product Card, kèm 14 test tại `tests/test_eligibility.py` (full suite 82 test đạt, ruff sạch). Nội dung chương này là kết quả tranh luận ba vòng giữa Muse và ChatGPT trên dữ kiện code tại baseline `7af8b52` (kết luận “Approved with conditions”); bản kế hoạch đầy đủ nằm ngoài repo. [Chương 09](09_shopping_workflow_design.md) và [chương 11](11_recommendation_consumption.md) đã cập nhật theo hiện trạng mới.

## Vấn đề

Ba đường sinh Product Card đang kiểm tra không đều nhau:

| Đường | Published | Ngân sách | Tồn kho |
| --- | --- | --- | --- |
| Luồng shopping ([chương 09](09_shopping_workflow_design.md)) | Có (tool chỉ trả `published`) | Có (lọc cứng `max_price_vnd` khi tìm) | Có, từng SKU nhưng **tuần tự** |
| RecommendationService ([chương 11](11_recommendation_consumption.md)) | Có (resolve lại qua Catalog) | **Không** | **Không** |
| Intent `basket` trong chat | Qua RecommendationService | **Không** | **Không** |

Hệ quả đã xác nhận bằng code: gợi ý mua kèm có thể đưa sản phẩm hết hàng hoặc vượt ngân sách của phiên vào Product Card. Đây là lỗi đúng-sai nghiệp vụ (correctness), không phải rủi ro giả định, nên được ưu tiên trước contract test 4 tầng với MS-11 (schema v1 của Analytics còn nằm trong PR #62 chưa merge — test contract lúc này chưa có nền ổn định).

## Thiết kế: một lớp hội tụ duy nhất

Thêm một thành phần dùng chung trong nền tảng — lớp eligibility — mà **mọi** Product Card đều phải đi qua, bất kể nguồn: kết quả tìm kiếm của luồng shopping, intent `basket`, cả ba placement của RecommendationService và cả danh sách phổ biến dùng làm fallback. Card nào không qua lớp này là vi phạm kiến trúc, kể cả khi sản phẩm thực tế còn hàng.

Pipeline theo đúng thứ tự, kiểm theo **SKU** (không chỉ productId):

1. **Chuẩn hóa + khử trùng** theo SKU, giữ ánh xạ về card và nguồn gợi ý.
2. **Resolve qua Catalog** — chỉ giữ sản phẩm đang `published` tại thời điểm kiểm tra.
3. **Lọc ngân sách cứng** — giữ nguyên semantics hiện hành của luồng shopping: giá lớn hơn `budget_vnd` của phiên thì loại (giá bằng đúng ngân sách thì giữ). Ngân sách là trần, không phải gợi ý; mô hình ngân sách mềm (target/range) chỉ xét khi có tín hiệu sử dụng thật.
4. **Kiểm tồn kho song song** — phát các lời gọi `check_stock` đồng thời có giới hạn concurrency (semaphore), timeout riêng cho từng SKU và một **deadline toàn cục** cho cả bước eligibility, truyền deadline/cancellation xuống adapter. Số lượng cụ thể là điểm khởi đầu để tinh chỉnh: latency thật của Inventory chưa từng được đo và lát này phải đo.
5. **Kết quả tường minh** — mỗi SKU hoặc thành card hợp lệ, hoặc bị loại kèm outcome ở mục dưới; không có trạng thái “không rõ mà vẫn hiển thị”.

## Chính sách lỗi: fail-closed theo từng SKU

| Tình trạng tồn kho | Xử lý |
| --- | --- |
| Số lượng > 0 (đã xác minh còn hàng) | Đủ điều kiện hiển thị |
| Số lượng = 0 (đã xác minh hết hàng) | Loại, outcome `OUT_OF_STOCK` |
| Timeout khi kiểm | Loại SKU đó, outcome `INVENTORY_TIMEOUT` |
| Lỗi/không lấy được dữ liệu | Loại SKU đó, outcome `INVENTORY_UNKNOWN` |

Lỗi ở một SKU **không** hủy các SKU còn lại: lượt gợi ý trả các sản phẩm đã xác minh được và loại phần chưa xác minh. Không bao giờ biến lỗi kỹ thuật thành “hết hàng” trong số liệu, và không bao giờ coi chưa xác minh được là còn hàng.

Điều kiện kèm theo của lát — **spike đã thực hiện ngày 10/11/2026**, đọc code Inventory trên `dev`: RPC `GetStockLevel` trả `available_qty` đúng bằng **khả dụng để bán** — `Q_avail = Q_phy − Q_res` (quy tắc INV-BI-01). Đường production của AI không gọi REST Inventory không tồn tại: eligibility advisory đọc `stock_available` từ durable Catalog projection đã được Inventory cập nhật. Giá trị lạ/thiếu projection bị ép 0 (fail-closed); checkout vẫn revalidate trực tiếp với Inventory trước khi quyết định bán (xem [chương 07](07_integrations_and_contract_gaps.md)).

## Outcome taxonomy

Phân loại kết quả để quan trắc và debug, thay cho cách gom chung hiện tại (Analytics lỗi và Analytics trả rỗng đang cùng rơi về fallback mà không phân biệt):

- **Cấp recommendation:** `SUCCESS`, `EMPTY`, `NOT_CONSENTED`, `TIMEOUT`, `UNAVAILABLE`, `INVALID_RESPONSE`. Riêng `NOT_CONSENTED` chỉ dùng fallback tổng hợp — fallback hiện tại vốn chỉ dùng độ phổ biến tổng hợp của Catalog, không dùng dữ liệu cá nhân, nên đây là yêu cầu quan trắc chứ chưa phải lỗ hổng riêng tư.
- **Cấp sản phẩm:** `ELIGIBLE`, `NOT_PUBLISHED`, `OVER_BUDGET`, `OUT_OF_STOCK`, `INVENTORY_TIMEOUT`, `INVENTORY_UNKNOWN`, `PRODUCT_NOT_FOUND`, `INVALID_SKU`.

Trường `strategy` của Analytics được **ghi log** cho observability. Không tự sinh reason code và không để LLM bịa lời giải thích cho khách: contract Analytics hiện không có `reason_codes` hay `score` công khai; muốn có tính năng giải thích thì đó là yêu cầu thay đổi contract phía MS-11, không phải việc MS-19 tự suy diễn.

## Quyết định kiến trúc (ADR) ghi kèm lát này

- **ADR-01 — Ownership của eligibility:** MS-19 điều phối việc kiểm tra eligibility nhưng không sở hữu dữ liệu publication, giá hay tồn kho; các dữ kiện vẫn thuộc Catalog/Inventory và được đọc qua contract của service nguồn.
- **ADR-02 — Ngữ nghĩa availability:** tồn kho trong chat là thông tin quan sát tại thời điểm kiểm tra (*advisory*), **không phải reservation**. Câu chữ kèm Product Card chỉ được nói “còn hàng tại thời điểm kiểm tra”, không ngụ ý giữ chỗ.
- **ADR-03 — Failure policy:** không xác minh được tồn kho thì không hiển thị sản phẩm như một lựa chọn có thể mua; lỗi từng phần xử lý theo SKU (fail-closed từng SKU, không fail toàn lượt).
- **ADR-04 — Ranh giới checkout:** reservation và finalize thuộc Order/Inventory tại thời điểm đặt đơn; MS-19 không tham gia giao dịch của checkout và không có trách nhiệm xử lý tranh chấp tồn kho đồng thời (TOCTOU). Nợ hiện hữu phải ghi rõ: đường finalize của checkout phía Inventory/Order chưa hoàn thiện — đây là rủi ro cấp hệ thống theo dõi riêng, lớp eligibility **không** giải quyết nó.

## Những gì lát này không thay đổi

Ranh giới service MS-19 trong service boundary, hợp đồng OpenAPI bên ngoài (cùng endpoint, cùng schema thẻ — chỉ nội dung thẻ được lọc chặt hơn), schema `ai_db`, ranh giới “AI chỉ gợi ý” và ownership dữ liệu của các service bạn đều giữ nguyên. Thay đổi nằm ở cấu trúc nội bộ (thêm một thành phần nền tảng dùng chung), chính sách lỗi và vốn từ quan trắc.

## Nghiệm thu

Bộ ca kiểm thử bắt buộc của lát liệt kê tại [chương 12](12_evaluation_and_verification.md) (nhóm eligibility): đủ bốn đường sinh card, các ca biên ngân sách (bằng trần / vượt 1đ), quantity 0/1/không-xác-minh, ngân sách thay đổi giữa phiên, phân loại outcome của Analytics, và test chứng minh kiểm tồn thực sự song song. Thứ tự việc sau lát này: contract test 4 tầng khi PR #62 merge, shadow reranking khi có retrieval ứng viên thật, feedback forwarding fire-and-forget có idempotency — xem [roadmap](08_implementation_roadmap.md).
