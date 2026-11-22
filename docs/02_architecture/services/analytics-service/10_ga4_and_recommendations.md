# 10 — Phễu GA4 và Recommendation Engine

**Thiết kế đề xuất · 06/11/2026 · Nhánh `feat/analysis-service`.**

Kế hoạch triển khai ngày 07/11/2026 đã được cụ thể hóa thành [bộ thiết kế Recommendation 12–15](12_recommendation_engine_design.md). Bộ này xác định contract triển khai khi có khác biệt với phác thảo bên dưới.

Thiết kế gồm hai lát có thể phát hành độc lập. MS-14 `marketing-service` sở hữu đo chuyển đổi bằng GA4. MS-11 `analytics-service` sở hữu Recommendation Engine và API gợi ý cho Storefront. Product Discovery là tên năng lực nghiệp vụ trong EPIC 03. Tín hiệu bán chạy từ Supply được dùng khi nguồn đạt coverage và phạm vi phù hợp với kênh D2C.

## 1. Ranh giới và nguồn dữ liệu

| Thành phần | Trách nhiệm |
| --- | --- |
| Marketing / MS-14 | Sở hữu GA4 property, quy ước tracking, phễu và KPI theo chiến dịch. Dùng GA4 Explore trong pilot. Tích hợp báo cáo vào Web Admin khi MS-14 có runtime và quyền đọc riêng. |
| MS-05 `catalog-service` | Sở hữu danh mục, trạng thái công khai và dữ liệu hiển thị của Product/SKU. Cung cấp danh sách SKU được duyệt để làm fallback. |
| MS-11 `analytics-service` | Sở hữu Supply Analytics, tính điểm, xuất bản danh sách theo phạm vi và phục vụ API gợi ý. Recommendation có pipeline và ngân sách tài nguyên riêng trong MS-11. |
| Inventory / Order | Inventory xác nhận khả dụng theo vùng phục vụ. Order xác nhận giao dịch và cung cấp dòng đơn nếu cần phân tích mua hàng theo SKU/khách. |

[Ranh giới MS-11](00_analytics_service_hld.md#phạm-vi-và-ownership) · [Hợp đồng nguồn](07_integrations_and_contract_gaps.md) · [US-AI-04](../../../01_requirements/epics/EPIC_03_Product_Discovery.md#us-ai-04--product-recommendation-ai)

```mermaid
flowchart LR
  Web["Storefront"] -->|Ecommerce events theo consent| GA["GA4"]
  GA --> Explore["Marketing · Funnel Exploration"]
  Web <-->|Gợi ý sản phẩm| Gateway["API Gateway"]
  subgraph MS11["MS-11 analytics-service"]
    Supply["Supply Analytics"]
    Rank["Recommendation Engine"]
    Rec["Recommendation API"]
    Supply -.->|Tín hiệu bán chạy đã duyệt| Rank
    Rank -->|Danh sách theo ngữ cảnh| Rec
  end
  Gateway <--> Rec
  Catalog["Catalog"] -->|SKU công khai và fallback| Rank
  Inv["Inventory"] -->|Khả dụng theo vùng| Rank
  Ord["Order"] -->|Bán và trả theo ngày| Supply
  Inv -->|Lịch sử hết hàng và snapshot lô| Supply
  Web -.->|Tương tác first-party được phép ở v2| Rank
```

## 2. GA4 pilot

| Event | Điểm phát và dữ liệu tối thiểu |
| --- | --- |
| `view_item` | Trang chi tiết đã hiển thị sản phẩm. Gửi SKU mặc định thực sự hiển thị nếu xác định được SKU. |
| `add_to_cart` | Cart API xác nhận thêm hàng. Gửi SKU và số lượng vừa thêm. |
| `begin_checkout` | Khách vào bước checkout với giỏ hợp lệ. Gửi các dòng trong lần checkout đó, kiểm trùng theo checkout attempt. |
| `purchase` | Trong pilot web, Storefront phát khi đọc được xác nhận thanh toán từ Order và có dữ liệu mua hàng đã xác thực. Gửi `transaction_id` ổn định theo đơn, `currency`, `value` và `items`. |
| `view_item_list`, `select_item` | Khi dãy gợi ý xuất hiện trong vùng nhìn hoặc khách chọn mục. Gửi `item_list_id` theo vị trí gợi ý, SKU và thứ hạng. |

Dùng `items[].item_id = skuCode` cho các dòng SKU. Trong pilot, trang chưa xác định được variant chỉ ghi nhận thiếu coverage SKU, không phát `view_item` với SKU suy đoán. Chuẩn hóa thời gian phễu theo `Asia/Ho_Chi_Minh`. GA4 property dev tách riêng production. Tracking áp dụng consent và chỉ gửi các trường trong allowlist. Lọc URL/query chứa dữ liệu nhận dạng, thông tin khách hàng hoặc token. [GA4 Ecommerce](https://developers.google.com/analytics/devguides/collection/ga4/ecommerce) · [Chính sách dữ liệu GA4](https://support.google.com/analytics/answer/6366371)

Pilot giới hạn đơn chuyển khoản D2C. Chỉ phát `purchase` từ trạng thái đã xác nhận qua backend và dùng giá trị dòng hàng của Order theo quy ước GA4. Kiểm tra nguồn dữ liệu dòng đơn trước khi bật bước này. Trường hợp đóng tab trước khi xác nhận được ghi nhận là giới hạn coverage của pilot. Nếu cần bắt được các đơn đó, chuyển riêng `purchase` sang worker hậu thanh toán qua Measurement Protocol. Worker cần dòng đơn xác thực, `client_id` và `session_id` đã được phép thu thập, retry theo ID giao dịch và một bên phát event trên mỗi môi trường. [Trạng thái thanh toán web](../../../../apps/web-user/src/lib/payment/types.ts) · [Order PAID](../../../../packages/events/schemas/order/v3/paid.event.json) · [Measurement Protocol](https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag) · [Transaction ID](https://support.google.com/analytics/answer/12313109)

Marketing dùng phễu đóng `view_item → add_to_cart → begin_checkout → purchase` cho pilot, theo dõi số người qua bước và tỷ lệ rời, với breakdown theo thiết bị hoặc chiến dịch thu hút người dùng khi dữ liệu đủ. Đối chiếu giao dịch GA4 với tập Order D2C pilot đã xác nhận có consent và phát thành công. Ghi riêng phần chênh do thiếu consent, tab đóng hoặc tracking bị chặn. Nếu cần báo cáo nội bộ, adapter của Marketing đọc báo cáo tổng hợp qua `runFunnelReport` (`v1alpha`), đóng gói contract nội bộ và cache theo bộ lọc. [GA4 Funnel Exploration](https://support.google.com/analytics/answer/9327974) · [GA4 Data API](https://developers.google.com/analytics/devguides/reporting/data/v1/funnels)

## 3. Recommendation Engine v1

[Cơ sở chọn thuật toán, công thức pilot và cách đánh giá](11_recommendation_algorithm_proposal.md).

Pilot một vị trí trên trang chi tiết sản phẩm. Ưu tiên SKU cùng danh mục và SKU do Catalog duyệt, rồi xếp hạng theo tín hiệu bán chạy nếu dữ liệu đã đủ phạm vi. Fallback là danh sách SKU Catalog duyệt cho vị trí đó. MS-11 lọc bỏ sản phẩm đang xem, SKU trùng, chưa công khai hoặc chưa xác nhận bán được trong phạm vi giao hàng. Áp dụng cùng điều kiện cho fallback. Nếu phạm vi giao hàng chưa xác định, chỉ dùng danh sách có khả dụng đã xác nhận cho phạm vi mặc định của Storefront. Storefront ẩn dãy gợi ý khi danh sách rỗng hoặc API không phản hồi trong thời hạn. Cart và Checkout vẫn kiểm giá và tồn theo luồng nghiệp vụ.

**API đề xuất** `GET /api/v1/products/recommended` hỗ trợ `product_detail`, `cart` và `home`. MS-11 đối chiếu context và scope cấu hình server, trả recommendationId/generation/strategy cùng thẻ sản phẩm lấy từ projection Catalog đã xác minh. Response, errors và SKU mốc đã chọn được mô tả ở [chương 14](14_recommendation_api_and_storefront.md). Gateway chuyển riêng exact route này đến MS-11, ưu tiên hơn prefix `/api/v1/products` của Catalog. [API trong Service Boundary](../../service_boundary.md) · [Gateway hiện tại](../../../../services/api-gateway/config/kong.yml)

**Tái sử dụng Supply.** `order_day` chỉ có số lượng bán/trả theo SKU, kho và ngày. Engine đọc tổng hợp nội bộ sau khi Analytics xác nhận phạm vi kênh/vùng D2C, kỳ đo, coverage, `asOf` và version. Snapshot ghi `scope`, `from`, `to`, `asOf`, `version`, `coverage` và số bán/trả theo SKU để truy vết. Chỉ dùng pilot một kho nếu phạm vi recommendation cũng giới hạn đúng kho/vùng và cohort bán. `inventory_batches` là snapshot phục vụ ước lượng rủi ro. Trạng thái bán được trên Storefront cần contract khả dụng của Inventory có ngưỡng độ mới riêng. [Hợp đồng Supply v1](09_supply_analytics_v1.md)

**Cô lập nguồn.** Job xếp hạng của MS-11 đọc projection Catalog, Inventory và Supply theo revision/cutoff. Tái sử dụng cơ chế kiểm generation rồi cập nhật con trỏ, với pipeline/scope riêng cho Recommendation. Request đọc danh sách đã tính và lọc SKU theo projection công khai/khả dụng mới nhất. Nếu tín hiệu xếp hạng quá hạn, dùng fallback đã duyệt khi Catalog và Inventory còn đủ độ mới. Nếu không xác nhận được SKU đủ điều kiện, trả `EMPTY`. Đường phục vụ gợi ý đọc dữ liệu nội bộ đã xuất bản, tách khỏi các job Supply và GA4/BigQuery. [Cơ chế snapshot và recovery](04_messaging_and_recovery.md)

## 4. Cá nhân hóa v2

Chỉ mở khi có chính sách sử dụng dữ liệu theo mục đích và dữ liệu đủ kiểm chứng. Thu sự kiện first-party theo allowlist cho khách đăng nhập được phép cá nhân hóa. Collector của MS-11 xác định subject từ phiên xác thực, ghi `eventId`, `eventTime`, `skuCode`, `eventType` và consent; xử lý trùng theo `eventId`. Job định kỳ tạo tín hiệu theo danh mục/SKU dựa trên lượt xem hoặc thêm giỏ gần đây. Chỉ thêm lịch sử mua khi Order cung cấp dòng đơn và liên kết khách hàng đã được duyệt. Khi quyền thay đổi, ngừng phục vụ kết quả cá nhân hóa và thu hồi dữ liệu/cache liên quan theo chính sách. Khách chưa đăng nhập hoặc thiếu tín hiệu nhận kết quả v1. API cá nhân hóa xác thực subject phía server và không dùng chung cache với danh sách công khai.

GA4 dùng để đo lượt thấy, lượt chọn và chuyển đổi theo vị trí. Nếu Marketing cần phân tích chi tiết hơn, BigQuery Export là nguồn bất đồng bộ theo hợp đồng riêng. Nếu dùng GA4 làm tín hiệu xếp hạng, Marketing có thể xuất số lượt xem/thêm giỏ tổng hợp theo ngày và SKU cùng `asOf`, coverage và version. MS-11 thử tín hiệu này như một yếu tố bổ sung trong cùng danh mục. Số lượt phát sinh từ dãy gợi ý được tách theo placement để đo riêng hiệu ứng hiển thị. Lượt nhập theo ngày phải xử lý các phân vùng GA4 còn có thể nhận sự kiện muộn và công bố cutoff. Báo cáo GA4 và số giao dịch Order giữ định nghĩa và phạm vi riêng. [GA4 BigQuery Export](https://support.google.com/analytics/answer/9358801) · [Schema và dữ liệu đến muộn](https://support.google.com/analytics/answer/7029846)

## 5. Thứ tự triển khai và nghiệm thu

| Lát | Có thể bắt đầu khi | Nghiệm thu |
| --- | --- | --- |
| Supply pilot hiện tại | Có staff, ClickHouse dev và bộ dữ liệu đã đối soát | Hoàn tất các tiêu chí Order, Inventory và FEFO của pilot Supply. |
| GA4 trên dev | Marketing cấp property dev, chốt consent và điểm phát event | DebugView thấy đúng SKU và chuỗi thao tác. Mua đã xác nhận xuất hiện một lần dù reload. Đơn chưa xác nhận và phiên từ chối consent được kiểm theo policy. Ghi nhận coverage khi khách đóng tab. |
| Gợi ý theo ngữ cảnh v1 | MS-11 có projection Catalog, khả dụng Inventory và route đã kiểm thử | SKU công khai và còn bán được. Dùng fallback khi thiếu tín hiệu bán chạy. Ẩn dãy khi dữ liệu eligibility hết hạn hoặc API gặp lỗi. Có event hiển thị/chọn theo placement. |
| Xếp hạng theo tín hiệu Supply | Nguồn tự động đã đối soát và có phạm vi D2C phù hợp | Revision mới đổi đúng bảng xếp hạng. Snapshot cũ hoặc thiếu coverage kích hoạt fallback. |
| Cá nhân hóa v2 | MS-11 có contract dữ liệu first-party, consent và tiêu chí thử nghiệm | Cùng một ngữ cảnh, hồ sơ hành vi khác nhau cho gợi ý phù hợp. Kiểm tách dữ liệu giữa khách, thu hồi quyền và fallback. So sánh với v1 bằng CTR và tỷ lệ thêm giỏ theo placement trước khi mở rộng. |

**Cần chốt trước khi triển khai:** người sở hữu GA4 dev property; nguồn dòng đơn cho `purchase`; nguồn Catalog/Inventory theo vùng; phạm vi và ngưỡng độ mới của tín hiệu Supply. Các tên trường và endpoint mới trong tài liệu là đề xuất để review.
