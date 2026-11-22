# 01 — Domain và định nghĩa chỉ số

| Khái niệm | Định danh và ngữ nghĩa |
| --- | --- |
| Metric Definition | `metric_id + definition_version`. Lưu owner, đơn vị, nguồn, grain, kỳ, eligibility, phép cộng và ngày hiệu lực. |
| Serving Generation | `generation_id`, tập metric và source cutoffs bất biến sau publish. Một query đọc đúng một generation. |
| Coverage | Cohort, nguồn, phạm vi thời gian liên tục và phần thiếu. Một nguồn có broker lag bằng 0 vẫn cần bằng chứng coverage. |
| Analysis Run | Scope chuẩn hóa, snapshot, model/rule/feature version và lease. |
| Insight | Tham chiếu run, căn cứ, thời hạn, kết quả đánh giá và quyết định của người dùng. |

## Metric registry đề xuất

| Metric | Đơn vị | Quy tắc và quyền sở hữu |
| --- | --- | --- |
| `orders_paid` | ORDER | Đếm một lần cho mỗi `order_id` tại mốc PAID đầu tiên của cohort v3 đủ điều kiện. Kỳ theo `time` của event milestone. Cancel/refund sau đó giữ nguyên lịch sử mốc này. |
| `orders_completed` | ORDER | Đếm một lần cho mỗi `order_id` tại mốc COMPLETED. Đơn PAID và COMPLETED có thể ở hai kỳ. Hai card hiển thị riêng. |
| `active_customers` | CUSTOMER | Đề xuất đếm khách ổn định duy nhất có đơn hoàn thành đủ điều kiện trong kỳ. Chờ mapping Order–Profile, rule guest/claim, privacy và chính sách loại trừ. |
| `sellable_stock` | ITEM | Snapshot cuối kỳ theo scope kho/SKU đã chốt, đơn vị quy đổi đã duyệt và coverage đầy đủ. Lấy trạng thái gần nhất trước biên cuối kỳ, giới hạn tuổi snapshot. Chờ nguồn Inventory có revision, mốc chốt và đủ phạm vi. |
| `recognized_revenue` | VND | Nhận dữ liệu theo kỳ ghi nhận và phiên bản chính sách của Finance, giữ trạng thái chốt kỳ riêng với trạng thái kỹ thuật. |
| `profit` | VND | Nhận từ Finance cùng rule phân bổ chi phí, phạm vi, trạng thái đối soát và phiên bản. |

Order v3 có `items_digest` ở PAID và có decision ID ở COMPLETED. Các trường đủ cho chỉ số mốc theo cohort đã duyệt. GMV, Customer, Product/SKU và kênh cần dữ liệu khác. Không lấy số tiền receipt để điền doanh thu chính thức. Dùng số nguyên đơn vị nguồn cho tiền và lượng; API dùng chuỗi thập phân để giữ chính xác cho client.

## Ngày nghiệp vụ, trạng thái và so sánh

- API nhận ngày dương lịch `from` bao gồm và `to` loại trừ theo `Asia/Ho_Chi_Minh`. Ví dụ `2026-11-01..2026-11-02` là `[2026-10-31T17:00:00Z, 2026-11-01T17:00:00Z)`. Lưu UTC và giữ riêng thời điểm sự kiện, ingest, publish và kỳ ghi nhận tài chính.
- Kỳ so sánh liền trước có cùng số ngày. Chỉ tính delta nếu hai kỳ cùng định nghĩa, scope/coverage đủ và đều `FRESH`. Delta tuyệt đối bằng current trừ previous. Tỷ lệ bằng delta chia trị tuyệt đối của previous nhân 100, làm tròn hai chữ số half-up. Previous bằng 0 cho tỷ lệ `null`.
- `FRESH` có dữ liệu đủ theo scope, đạt freshness. Giá trị 0 chỉ hợp lệ sau khi chứng minh coverage toàn kỳ và tập eligible rỗng.
- `STALE` có snapshot đủ nhưng đã vượt mục tiêu freshness. `PARTIAL` có giá trị được phép công bố cho phần coverage đã kiểm chứng và lý do thiếu; khi vừa thiếu vừa cũ thì giữ `PARTIAL` và kèm cả hai lý do. `UNAVAILABLE` dùng khi chưa có giá trị an toàn để công bố và trả `null`.
- `sourceAsOf` mô tả mốc nguồn được kiểm chứng cho metric. `coveredPeriods` liệt kê các khoảng ngày đã đủ coverage cho scope. Việc vừa ingest hay vừa rebuild không làm dữ liệu nguồn mới hơn. Với lịch sử, freshness xét khả năng bao phủ kỳ và chính sách chốt. Kỳ hiện tại có thể `PARTIAL` khi mốc xác minh chưa tới biên `to`.
- Card drill-down ở lát đầu là breakdown theo ngày, cùng ngày nghiệp vụ và generation. Card Order cộng các bucket thuộc ngày có coverage đầy đủ. Khi thiếu một phần kỳ, ngày thiếu coverage trả `null`. Nếu toàn kỳ thiếu coverage, card trả `UNAVAILABLE`. Customer distinct, stock snapshot và Finance chưa duyệt breakdown theo ngày ở contract v0.1.
