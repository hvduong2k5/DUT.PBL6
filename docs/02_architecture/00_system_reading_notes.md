# Ghi chú tổng quan hệ thống Mè Xửng O Mạ

Ngày: 2026-10-17. Nhánh: `feat/traceability-service`, nền `origin/dev` tại `2291ef5`.
Đây là ghi chú đọc và đối chiếu tài liệu, không phải thay đổi yêu cầu hay ADR được team phê duyệt.

## Phạm vi và giới hạn

Vòng rà soát đi qua các nhóm tài liệu hiện có trên checkout: Product Backlog,
Sprint Planning, FR/NFR, Use Case, 28 Epic, Domain Design, Bounded Context,
Service Boundary, System Design, HLD, 5 tài liệu sơ đồ HLD, Profile LLD/TDD,
3 tài liệu API và README thư viện sơ đồ. Thư mục lúc bắt đầu có 51 Markdown,
trong đó 1 là bản readiness review mới tạo, không phải nguồn yêu cầu độc lập.
Các phần mục tiêu/phạm vi, requirements, quy tắc dữ liệu/vòng đời, quan hệ,
quyết định mở và contract được đối chiếu để hình thành bức tranh tổng thể.

Đọc bổ sung nội dung Word kế hoạch đội ngũ, schema REST/GraphQL và nhãn nguồn
Draw.io. Không thực hiện visual QA toàn bộ PNG hoặc xác minh từng tọa độ sơ đồ.
Rà soát này không phải chứng nhận mọi acceptance scenario/contract đã được
kiểm định đầy đủ; tài liệu trên nhánh tính năng khác có thể khác base hiện tại.

## Sản phẩm và lộ trình

Hệ thống phục vụ thương mại thực phẩm OCOP Huế: D2C Web/Mobile, B2B,
Marketplace, bán trực tiếp, vận hành kho/đóng gói/giao hàng và phân tích DSS.
Đặc thù xuyên suốt là SKU, lô/HSD, không oversell, bằng chứng đóng gói,
nguồn gốc có xác nhận, snapshot lịch sử và trách nhiệm thao tác có Audit.

Sprint Planning phân thành MVP v1.0 (mua hàng), v1.1 (vận hành), v1.2
(nguồn gốc, hậu mãi, Audit). Giai đoạn 2 mở rộng kênh, CRM, ưu đãi và Marketing;
giai đoạn 3 mở rộng Procurement, Finance, Analytics/DSS. Không mặc định toàn bộ
18 service phải xong để có một lát cắt MVP chạy được.

## Bản đồ 18 service theo thiết kế miền

| MS | Service | Sở hữu chính |
| --- | --- | --- |
| 01 | inventory | Số dư SKU/Batch, reserve/release, biến động, NSX/HSD, FEFO |
| 02 | fulfillment | Packing Task, phân công, checklist, kiện/seal, bằng chứng |
| 03 | traceability | Hồ sơ nguồn gốc/chứng nhận, revision công khai, QR |
| 04 | order | Cart, checkout, Order snapshot/lifecycle, Saga, module Payment/B2B/Gift |
| 05 | catalog | Product, Category, SKU, giá theo kênh, khả năng công khai |
| 06 | care | Ticket/hội thoại/SLA, Return/Refund Case, Review và moderation |
| 07 | promotion | Coupon, Combo, reservation ưu đãi và sổ điểm Loyalty |
| 08 | procurement | Supplier, Agreement, Supply Plan, PO, Receipt/variance |
| 09 | finance | Sổ/báo cáo tài chính, đối soát, hóa đơn và nghĩa vụ tài chính |
| 10 | content | Article, revision/publication, SEO, URL và media reference |
| 11 | analytics | Dataset/metric/model version, BI, RFM, forecast, insight |
| 12 | shipping | Báo phí, Shipment/waybill, tích hợp 3PL, tracking |
| 13 | channel | ACL Marketplace, mapping listing, POS, báo cáo/đối soát kênh |
| 14 | marketing | Campaign/revision/execution, budget, KPI và attribution |
| 15 | profile | Hồ sơ Customer/Employee, sổ địa chỉ; tách credentials/quyền |
| 16 | identity | Identity, credentials/session/JWT, Role/Permission/assignment |
| 17 | notification | Template/version, routing, consent, delivery attempt/status |
| 18 | audit | Bản ghi kiểm toán, tính toàn vẹn, retention, tra cứu theo quyền |

Epic là nhóm yêu cầu, không đồng nghĩa mỗi Epic là một service. Ví dụ Epic
05/06/07/08/18/26 được gom trong miền Commerce; 14/15/16 trong Care.
Mô tả Payment service độc lập còn trong tài liệu cũ và kế hoạch Word; HLD
và Service Boundary chủ yếu gom Payment vào MS-04, có hướng tách về sau.

## Giao tiếp và luồng đầu cuối

- Web/Admin/POS qua REST Gateway; Mobile có GraphQL BFF gom dữ liệu màn hình.
- Query/command cần kết quả ngay dùng gRPC theo HLD; Boundary còn cho phép
  REST internal non-critical. Không truy cập trực tiếp DB của service khác.
- Domain facts/workflow hậu kỳ qua Kafka, CloudEvents, outbox và consumer chống lặp.
- Business, Audit và Observability là ba mặt phẳng riêng. Audit qua Kafka;
  telemetry qua OTLP Collector, không dùng log kỹ thuật thay bằng chứng Audit.
- Gateway xác minh JWT bằng cached JWKS, strip header giả; service chỉ tin
  identity context qua đường truyền/nguồn được xác thực và kiểm tra quyền tại chỗ.

Luồng trả trước: chọn SKU → cart (chưa giữ tồn) → checkout kiểm tra lại giá,
địa chỉ, phí và ưu đãi → Order Saga giữ tồn → chờ Payment được xác minh →
OrderPaid → Inventory commit và Fulfillment nhận việc → PackingCompleted →
Shipping tạo vận đơn, Traceability kích hoạt QR theo thiết kế → tracking → hoàn tất.
Các mốc Order/Payment/reservation/Packing/Shipment phải giữ vòng đời riêng.

Marketplace: Channel chuẩn hóa webhook → Kafka → Order Saga tạo đơn/giữ tồn;
Channel không tự reserve rồi tạo Order. POS có `CreatePOSOrder` đồng bộ.
Return: Care xác minh Case/bằng chứng → phê duyệt → kiểm hàng và nghĩa vụ
Refund riêng; hàng về không tự thành tồn bán được, duyệt không đồng nghĩa tiền đã hoàn.

## Vị trí của Traceability và điều chỉnh nhận định trước

Traceability thuộc niềm tin/tuân thủ MVP v1.2, không nằm trên critical path
checkout. Public QR nên dựa trên dữ liệu đã xác nhận và lịch sử được bảo toàn;
không làm giao dịch mua hàng chờ EXT-10 hoặc chờ truy xuất nguồn gốc.

Catalog/Inventory cung cấp danh tính sản phẩm/lô; Procurement cung cấp nguồn
và snapshot Supplier; Fulfillment cung cấp lô thực đóng gói; Content trình bày
câu chuyện nhưng không xác nhận thay nguồn gốc. Identity/Gateway/Audit là
contract nền tảng cần kế thừa. Không tự thêm Order/Payment/Shipping thành peer
đồng bộ của Traceability chỉ vì cùng tham gia chuỗi vận hành.

Danh sách 14 điểm readiness cần được đọc như danh sách quyết định/độ lệch:
có phần đã có nguyên tắc chung, phần thiết kế nội bộ, phần cần thống nhất
contract và phần thuộc giai đoạn sau. Không phải 14 điều kiện chặn MVP.

## Các độ lệch có ảnh hưởng đến cách đọc tài liệu

- README/Word/Sprint risk còn RabbitMQ; System Design/HLD định hướng Kafka.
- Có sơ đồ/kế hoạch còn tách Cart/Payment và nói 9/17 service; Domain/Boundary/HLD có 18.
- Epic Inventory đặt FEFO nâng cao ở giai đoạn 2, trong khi kiến trúc yêu cầu FEFO ngay.
- Epic Packing đặt video ở giai đoạn 2, trong khi một số sơ đồ yêu cầu video cho mọi PACKED.
- COD nằm trong MVP, nhưng nhiều sequence vẫn chỉ mô tả luồng trả trước.
- Content và Dashboard có các mốc phát hành chưa đồng nhất giữa Epic Map/story/Sprint.
- GraphQL/REST còn địa chỉ có district; Profile LLD/TDD mô tả mô hình địa chỉ hai cấp.
- Header permission, RPC name, event name/topic, retry và một số field khác nhau giữa tài liệu.
- HLD PackingCompleted mô tả ngày sản xuất/hết hạn theo lô; JSON schema hiện chưa có các field đó.

Khi thiết kế cần truy ngược từ yêu cầu nghiệp vụ → luồng → owner → contract;
không mặc định tài liệu dài hơn, file mới đọc hơn hay đoạn có chữ SSOT là nguồn
ưu tiên tuyệt đối. Xung đột chưa có ADR/quyết định chung phải được ghi nhận.
