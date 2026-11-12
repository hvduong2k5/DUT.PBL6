# 11 — Đề xuất thuật toán gợi ý sản phẩm

**Thiết kế đề xuất · 06/11/2026 · MS-11 Analytics Service.**

Tài liệu này ghi căn cứ nghiên cứu. Kế hoạch ngày 07/11/2026 đã chốt đủ ba kiểu gợi ý, Customer đăng nhập và triển khai các producer nguồn. Công thức, threshold và interfaces triển khai theo [chương 12–15](12_recommendation_engine_design.md).

Chọn **content-based theo sản phẩm đang xem** cho pilot. Chạy đối chứng với xếp hạng có thêm tín hiệu bán ròng. Tách vị trí **Thường mua cùng** và **Dành cho bạn** vì mỗi vị trí có đầu vào và tiêu chí đánh giá riêng. Phân biệt sản phẩm thay thế với sản phẩm bổ sung cũng là một bài toán riêng trong nghiên cứu thương mại điện tử. [McAuley, Pandey & Leskovec](https://arxiv.org/abs/1506.08839) · [Kiến trúc phục vụ](10_ga4_and_recommendations.md#3-recommendation-engine-v1)

## 1. So sánh thuật toán

| Phương án | Tín hiệu tối thiểu | Quyết định cho MS-11 |
| --- | --- | --- |
| Phổ biến theo vùng | Số bán và trả theo SKU, ngày và cohort D2C | Dùng làm đối chứng và tiêu chí phụ. Có thể thiên về SKU được hiển thị nhiều hoặc có hàng lâu hơn. |
| Content-based | Sản phẩm đang xem, danh mục và thuộc tính Catalog | Chọn cho pilot. Có thể chấm SKU mới từ thuộc tính sản phẩm và giải thích từng kết quả. [Lops et al.](https://link.springer.com/chapter/10.1007/978-0-387-85820-3_3) |
| Luật kết hợp theo giỏ hàng | Tập SKU trong từng đơn đủ điều kiện | Chọn cho **Thường mua cùng** khi có dòng đơn. Đếm cặp trực tiếp trước. Cân nhắc Apriori hoặc FP-Growth nếu cần khai thác nhóm nhiều món. [Agrawal & Srikant](https://rsrikant.com/papers/vldb94.pdf) · [DSS Market Basket](06_dss_engines.md#market-basket-và-mùa-vụ) |
| Collaborative filtering item-to-item | Tương tác theo khách hoặc phiên được phép sử dụng | Thử khi có đủ tương tác lặp giữa các SKU. Có thể tính bảng SKU lân cận theo lô rồi xếp hạng từ các SKU khách từng tương tác. [Linden et al.](https://www.cs.umd.edu/~samir/498/Amazon-Recommendations.pdf) |
| Matrix factorization trên implicit feedback | Ma trận khách–SKU từ hành vi đủ mật độ và có lịch sử kiểm thử | Dùng implicit ALS làm ứng viên khi có dữ liệu đủ. LightFM là ứng viên khi có metadata có thể hỗ trợ SKU mới. So sánh với rule cá nhân hóa đơn giản và item-to-item trước khi mở. [Hu et al.](https://yifanhu.net/PUB/cf.pdf) · [Kula](https://arxiv.org/abs/1507.08439) |
| Mô hình sâu hai tầng | Dữ liệu hành vi lớn và lý do đo được để tách truy xuất ứng viên khỏi xếp hạng | Xem xét khi tăng quy mô catalog hoặc các baseline không đạt chất lượng và độ trễ mong muốn. [Covington et al.](https://research.google/pubs/deep-neural-networks-for-youtube-recommendations/) |

## 2. Công thức pilot cho Sản phẩm tương tự

1. **Chọn ứng viên.** Lấy SKU cùng danh mục của sản phẩm đang xem. Chỉ giữ SKU công khai, bán được trong vùng giao hàng và thuộc sản phẩm khác. Kiểm lại độ mới của nguồn Catalog và Inventory khi phục vụ. Chọn tối đa một SKU đại diện cho mỗi product ID và trả tối đa sáu sản phẩm.
2. **Tính điểm tương đồng.** Lấy SKU khách đang chọn trên trang làm mốc. Đề xuất cộng **2 điểm** nếu cùng loại bao bì, **1 điểm** nếu giá D2C gần nhau và **1 điểm** nếu khối lượng gần nhau. Với giá và khối lượng dương, quy ước thử nghiệm “gần” khi tỷ lệ giá trị nhỏ hơn trên giá trị lớn hơn đạt ít nhất **0,75**. Trường thiếu bỏ điểm tương ứng và ghi coverage. Khi chưa chọn SKU, dùng cùng danh mục và thứ tự fallback của Catalog. Trọng số và ngưỡng là giả thuyết để review trên tập SKU pilot.
3. **Phá hòa bằng bán ròng.** Trong nhóm có cùng điểm tương đồng, có thể xếp theo `sum(soldUnits) − sum(returnedUnits)` giảm dần. Chỉ bật tiêu chí này khi mọi SKU trong nhóm đủ 14 ngày cùng phạm vi D2C, nguồn mới trong 48 giờ và stockoutMinutes bằng 0 toàn kỳ. Ngày không có bản xuất khác ngày có số bán bằng 0. Nếu điều kiện chưa đạt, dùng thứ tự fallback của Catalog và khóa ổn định theo `skuCode`.
4. **Phục vụ và dự phòng.** MS-11 tính theo lô và ghi version thuật toán, scope, cutoff, revision nguồn và lý do xếp hạng. Điền vị trí trống từ danh sách Catalog đã duyệt sau khi lọc cùng điều kiện và bỏ trùng product ID. Nếu nguồn khả dụng quá hạn và chưa xác nhận được SKU nào, trả danh sách rỗng.

Catalog hiện có mô hình danh mục, SKU, quy cách và giá theo kênh. MS-11 cần contract projection để nhận cùng ngữ nghĩa giữa các SKU. Hợp đồng Supply hiện tổng hợp theo SKU/kho/ngày, nên dữ liệu 14 ngày chỉ dùng cho phạm vi pilot đã đối soát. [Catalog API](../../../../services/catalog-service/src/main/java/com/hvduong/catalog/api/CatalogHttpController.java) · [Projection giá và quy cách hiện có](../../../../services/catalog-service/src/main/java/com/hvduong/catalog/application/service/CheckoutSnapshotService.java) · [Hợp đồng Supply v1](09_supply_analytics_v1.md)

Ví dụ minh họa với sản phẩm mốc hộp 500 g, giá 100.000 đồng và hai sản phẩm cùng danh mục có cùng loại bao bì.

| Ứng viên | Khối lượng | Giá D2C | Điểm tương đồng | Bán ròng trong kỳ | Thứ tự |
| --- | --- | --- | --- | --- | --- |
| Mè xửng dẻo | 450 g | 95.000 đ | 4 | 12 | 1 |
| Mè xửng giòn | 250 g | 65.000 đ | 2 | 40 | 2 |

Nếu hai ứng viên cùng điểm tương đồng và cùng đủ coverage, ứng viên có bán ròng cao hơn đứng trước. Số liệu trong bảng là ví dụ.

## 3. Hai lát sau pilot

- **Thường mua cùng.** Order cấp tập SKU tại mốc COMPLETED theo order ID, kênh/kho và revision. Mỗi SKU xuất hiện một lần trong một đơn, có policy loại đơn test hoặc rút dữ liệu sai. Với cặp A → B, tính số đơn chứa cả hai, confidence `P(B|A)` và lift `P(B|A)/P(B)`. Chặn cặp quá ít đơn chung và mẫu số bằng 0. Xếp theo confidence sau khi lọc lift lớn hơn 1 và đủ mẫu. Basket Recommendation dùng policy ở [chương 12](12_recommendation_engine_design.md#6-thường-mua-cùng), còn [DSS Market Basket](06_dss_engines.md#market-basket-và-mùa-vụ) có policy return riêng.
- **Dành cho bạn.** Baseline đầu tiên là sở thích theo danh mục, ưu tiên hành vi gần đây và cho tín hiệu mua hoặc thêm giỏ trọng số cao hơn lượt xem. Chỉ dùng những sự kiện first-party được cấp quyền cho mục đích cá nhân hóa. Khi có đủ lịch sử, so sánh item-to-item, implicit ALS hoặc LightFM trên cùng tập ứng viên và scope. Khách chưa có lịch sử đủ điều kiện nhận gợi ý theo ngữ cảnh. [Thiết kế quyền sử dụng và dữ liệu](10_ga4_and_recommendations.md#4-cá-nhân-hóa-v2)

## 4. Cách chọn phiên bản thắng

| Giai đoạn | So sánh | Chỉ số và điều kiện |
| --- | --- | --- |
| Dev, chưa có đủ traffic | B0 danh sách Catalog duyệt, B1 content-only, B2 content + bán ròng phá hòa. Thêm baseline bán chạy cùng vùng khi đủ coverage. | Nhóm Catalog chấm độ liên quan của top 6 trên bộ trang sản phẩm giữ riêng. Đo tỷ lệ trang có gợi ý, số SKU không đủ điều kiện và số trang đổi thứ tự giữa B1/B2. Kiểm tất cả trường hợp hết hàng, ẩn sản phẩm, thiếu revision, phạm vi kho và SKU mới. |
| Đánh giá offline cho giỏ hàng và cá nhân hóa | Baseline và mô hình thử trên cùng ứng viên, phạm vi và mốc thời gian | Dùng Recall@6 để đo tìm được bao nhiêu mục tiêu và NDCG@6 để ưu tiên mục đúng ở vị trí cao. Tách train/validation/test theo thời gian. Tại mỗi cutoff chỉ dùng revision, thuộc tính và danh sách ứng viên đã khả dụng lúc đó. Nếu không có snapshot lịch sử của eligibility, ghi giới hạn của phép đo. [Ji et al.](https://arxiv.org/abs/2010.11060) |
| Thử nghiệm Storefront | B1 và B2 trên cùng vị trí, giữ nhóm ổn định theo phiên hoặc khách theo policy | Chỉ số chính đề xuất là tỷ lệ phiên đủ điều kiện thêm ít nhất một SKU được gợi ý vào giỏ. Theo dõi CTR của dãy, tỷ lệ hiển thị, fallback, lỗi và p95 latency. Chốt đơn vị chia nhóm, cỡ mẫu và kỳ đo trước. Traffic pilot nhỏ dùng để xác nhận đo lường và vận hành. |

Storefront gửi `view_item_list` khi dãy thật sự hiển thị và `select_item` khi khách bấm, theo quy ước đo của Marketing. Gắn ngữ cảnh gợi ý vào lần thêm đúng SKU đã chọn trong cùng phiên để đo theo placement và biến thể thử nghiệm. Đếm tối đa một phiên thành công cho chỉ số chính. Dùng `recommendationId` trong trace nội bộ để đối chiếu khi cần. GA4 đo hiệu quả của danh sách và có thể cung cấp số tổng hợp theo SKU khi được phê duyệt. [GA4 Ecommerce](https://developers.google.com/analytics/devguides/collection/ga4/ecommerce)
