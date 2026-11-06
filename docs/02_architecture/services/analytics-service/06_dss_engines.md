# 06 — DSS engines Phase 3

Các công thức và thuật toán dưới đây là mặc định đề xuất để review cùng chủ nghiệp vụ. Mỗi run đóng băng scope, input generation, `data_cutoff`, loại và lý do loại bỏ bản ghi, model/rule/threshold version, feature version, lịch và kết quả quality gate. Input lịch sử phải có event time và thời điểm dữ liệu thực sự khả dụng để kiểm backtest theo mốc quá khứ. Thiếu nguồn hoặc không đạt ngưỡng thì trả `INSUFFICIENT_DATA`, kèm lý do.

## Customer RFM và churn

- **RFM.** Với khách có ID ổn định, lấy đơn hoàn thành eligible trong cửa sổ tính đến cutoff. Đề xuất loại hủy, test, fraud và hoàn toàn phần. `R` là số ngày nghiệp vụ từ lần mua eligible gần nhất, `F` là số order ID distinct và `M` là giá trị hàng sau chiết khấu, trừ phần refund đã settle theo policy và currency. Chờ owner duyệt việc phân bổ hoàn tiền. Mỗi khách nhận một segment theo rule ưu tiên có version, lưu điểm và căn cứ. Guest/claim chỉ nối bằng mapping có thẩm quyền và effective time.
- **Churn.** Định nghĩa label theo việc có quay lại mua trong horizon đã duyệt. Đề xuất bắt đầu bằng rule recency so với chu kỳ mua và lịch sử tối thiểu. Với mô hình sau này, chia tập theo thời gian, theo dõi precision/recall theo ngưỡng, độ hiệu chỉnh xác suất và drift; đối chiếu với baseline theo từng nhóm. Xuất audience criteria hoặc segment đã được cấp quyền cho Marketing theo consent hiện hành.
- **Clickstream.** FR-25 bao gồm hành vi. Thiết kế lane riêng với event taxonomy cho view/search/add-to-cart dạng tối thiểu, ID phiên giả danh và bộ lọc bot/test. Chờ phê duyệt thu thập, thời hạn và quy tắc trộn với giao dịch. Chỉ dùng dữ liệu đã thu thập hợp lệ cho phân tích hành vi.

## Product performance và velocity

- Chốt product, SKU, channel và snapshot Catalog tại thời điểm giao dịch. Gộp SKU theo product ID ổn định; giữ bản mô tả lịch sử. Rank theo metric riêng (lượng bán ròng, số order đủ điều kiện hoặc doanh thu do Finance phê duyệt). Hiển thị số kỳ đủ và coverage.
- Đề xuất velocity = (lượng bán eligible − lượng return được xác nhận) / số ngày quan sát đủ dữ liệu. Trả kèm stockout days, unknown days và độ tin cậy. Nếu cần velocity chỉ tính ngày có hàng thì đăng ký một metric version khác, tính tử và mẫu trên cùng tập ngày. Kỳ quan sát bằng 0 trả insufficient.
- Hủy và refund tiền có ý nghĩa riêng với return vật lý. Chỉ trừ số lượng khi có return line hợp lệ theo rule. Supply nhận snapshot velocity có cutoff, formula version, đơn vị và scope để lập kế hoạch.

## Demand forecast và expiry risk

- Khóa series theo SKU hoặc product, channel/region và granularity. Phân biệt ngày zero sales có coverage với ngày thiếu dữ liệu hoặc hết hàng. Dùng seasonal naïve làm baseline. Đánh giá Prophet với lịch Tết, promotion và stockout sau khi có đủ dữ liệu. Dùng rolling-origin backtest và chỉ dùng feature đã biết tại mỗi cutoff. Chọn model theo MAE/WAPE, interval coverage và ngưỡng đã duyệt. WAPE không xác định nếu tổng actual bằng 0.
- Lưu horizon, cutoff, model, p50 và khoảng dự báo, giả định và sai số theo segment. Khi actual về, đánh giá theo cùng scope/metric version. Thiếu lịch sử trả insufficient hoặc baseline đã duyệt có nhãn riêng.
- Với expiry risk, lấy snapshot kho/SKU/batch có HSD và `available` hợp lệ. Mô phỏng phân bổ nhu cầu trước HSD theo FEFO trên các batch cùng kho/SKU; mỗi đơn vị demand chỉ dùng một lần, bỏ batch hết hạn hoặc bị chặn tại ngày mô phỏng. Lượng rủi ro = available còn lại tại mốc HSD trong horizon. Ghi giả định bổ sung hàng, stockout, timezone của HSD và uncertainty. Batch thiếu HSD/revision vào quality issue. Peer nhận insight cùng snapshot, không nhận lệnh thay tồn.

## Market basket và mùa vụ

- Mỗi Order eligible tạo một basket set SKU distinct, có điều chỉnh return line theo policy; loại test/fraud/hủy/hoàn toàn phần. Với A → B, support = số basket có cả A, B / tổng basket eligible, confidence = số basket có A, B / số basket có A, lift = confidence / support(B). Chặn mẫu số 0 và cỡ mẫu nhỏ. Đề xuất FP-Growth khi cặp đơn giản đạt baseline; các ngưỡng cần Sales phê duyệt. Kiểm SKU còn bán trước khi chuyển snapshot sang Promotion.
- So sánh mùa vụ theo khoảng ngày tương đối với ngày sự kiện trong calendar version, gồm Tết âm lịch theo từng năm. Hiển thị tập kỳ so sánh, coverage, Promotion/stockout context. Tác động quan sát được đi kèm điều kiện và giả định.

## Khuyến nghị chiến lược

Insight gồm mục tiêu, đối tượng và phạm vi, run/source links, căn cứ, trade-off, ước lượng tác động, confidence, `valid_from` và `expires_at`. Rule publish chỉ chọn kết quả đủ dữ liệu, giải thích được và còn hiệu lực. Quyết định của Executive lưu lý do và audit. Theo dõi KPI sau xử lý bằng outcome link và method đánh giá. Nếu chỉ so sánh quan sát, trình bày dưới dạng tương quan. [Vòng đời](02_usecases_and_lifecycle.md)
