# UI Screen Specification for Stitch AI

## 1. Document Purpose

Tài liệu này chuyển toàn bộ requirement hiện có thành đặc tả màn hình Web D2C và Web Admin để dùng trực tiếp trong thiết kế UX/UI và Stitch AI. Tài liệu mô tả mục tiêu nghiệp vụ, actor, entry point, nội dung, dữ liệu, hành động, state, quyền, điều hướng và layout direction; không định nghĩa lại màu, typography, spacing, radius, shadow, grid, button size hoặc visual token.

Mọi màn hình phải sử dụng Master Design System **“O Mạ — Master Design System V1.0 (Desktop 1440px)”**. Các nhãn sau được dùng xuyên suốt:

- **[DOCUMENTED]**: được nêu trực tiếp trong requirement.
- **[DERIVED]**: suy ra chắc chắn từ nhiều requirement để nối flow.
- **[UX RECOMMENDATION]**: quyết định UX cần thiết, không phải business rule.
- **[OPEN QUESTION]**: thiếu hoặc mâu thuẫn, cần Product Owner xác nhận.

## 2. Sources Analyzed

- 28/28 Epic trong `docs/01_requirements/epics/` (175 User Story).
- 6 tài liệu gốc: `README.md`, `01_Product_Backlog.md`, `02_Sprint_Planning.md`, `03_Non_Functional_Requirements.md`, `04_Functional_Requirements.md`, `use_case_specification.md`.
- `docs/01_requirements/diagrams/README.md` và 17 tệp Draw.io có thể parse: master overview, 14 phân hệ và 2 workbook tổng hợp.
- 25 PNG trong thư mục diagrams được kiểm kê là bản render/ảnh tham chiếu; nội dung nghiệp vụ được đối chiếu từ nguồn Draw.io tương ứng khi có.
- Tổng số file nguồn nội dung đã đọc/parse: **52**; tổng số file trong cây requirement đã scan: **77**.

## 3. Assumptions & Analysis Rules

1. Epic không được ánh xạ cơ học thành screen. Story cùng entity/task được hợp nhất thành workspace; workflow nhiều bước hoặc khác actor được tách.
2. Không tạo screen riêng cho webhook, idempotency, reservation worker, CI/CD, observability hoặc xử lý nền. Ảnh hưởng user-facing của chúng nằm trong states/feedback của screen liên quan.
3. Priority dùng release plan: Must 1st = MVP v1.0; Must 2nd = MVP v1.1; Must 3rd = MVP v1.2; Should = Giai đoạn 2; Could = Giai đoạn 3. Nếu Epic ghi khác, tài liệu giữ conflict ở Mục 11.
4. Những field chưa được requirement định nghĩa nhưng cần cho thao tác cơ bản được ghi rõ **[UX RECOMMENDATION]**; không được coi là schema hay business rule.
5. Một màn hình Admin có thể là master-detail workspace với list, drawer và detail panel; các phần này không tạo Screen ID riêng trừ khi workflow/actor khác biệt đáng kể.
6. Mọi deep link, notification và search result phải kiểm tra lại authentication/authorization; ẩn nút không thay thế kiểm soát quyền.
7. Các trạng thái `PENDING_PAYMENT`, `PAID`, `CONFIRMED`, `PROCESSING`, `PACKED`, `SHIPPED`, `DELIVERED`, `COMPLETED`, `DELIVERY_FAILED`, `EXPIRED`, `CANCELLED`, `RETURN_REQUESTED`, `REVIEWING`, `APPROVED`, `REJECTED`, `RETURNED`, `REFUNDED` chỉ dùng đúng phạm vi đã được backlog mô tả; các trạng thái bổ sung trong Use Case được đánh dấu conflict.

## 4. System Screen Inventory

### 4.1 Web D2C

| Screen ID | Phân hệ | Tên màn hình | Primary Actor | Mục tiêu chính | Epic liên quan | Priority |
|---|---|---|---|---|---|---|
| D2C-001 | D2C / Discovery | Trang chủ & khám phá | Guest, Customer | Bắt đầu khám phá sản phẩm, câu chuyện và đề xuất | 03, 20 | MVP/GĐ2 |
| D2C-002 | D2C / Discovery | Danh mục & kết quả tìm kiếm | Guest, Customer | Duyệt, tìm, lọc catalog công khai | 03, 04, 15 | MVP |
| D2C-003 | D2C / Discovery AI | Trợ lý tìm sản phẩm | Guest, Customer | Diễn đạt nhu cầu tự nhiên và nhận gợi ý có căn cứ | 03 | GĐ3 |
| D2C-004 | D2C / Product | Chi tiết sản phẩm | Guest, Customer | Chọn đúng SKU, hiểu thông tin thực phẩm, quyết định mua | 04, 03, 15, 19 | MVP/GĐ2 |
| D2C-005 | D2C / Cart | Giỏ hàng | Guest, Customer | Kiểm tra SKU, số lượng, giá tạm tính trước checkout | 05, 04 | MVP |
| D2C-006 | D2C / Checkout | Checkout | Guest, Customer | Cung cấp nhận hàng, giao hàng, ưu đãi/quà tặng và xác nhận | 01, 02, 06, 17, 26 | MVP/GĐ2 |
| D2C-007 | D2C / Payment | Thanh toán & chờ xác nhận | Guest, Customer | Chọn/thực hiện QR, chuyển khoản hoặc COD và theo dõi kết quả | 07, 06 | MVP |
| D2C-008 | D2C / Order | Xác nhận đặt hàng | Guest, Customer | Nhận mã đơn, trạng thái thanh toán và bước tiếp theo | 07, 08, 28 | MVP |
| D2C-009 | D2C / Account | Đăng nhập | Customer | Xác thực bằng phương thức được hỗ trợ | 01 | MVP/GĐ2 |
| D2C-010 | D2C / Account | Đăng ký | Guest | Tạo tài khoản không trùng | 01 | MVP |
| D2C-011 | D2C / Account | Khôi phục quyền truy cập | Customer | Yêu cầu và hoàn tất đặt lại mật khẩu | 01, 28 | MVP |
| D2C-012 | D2C / Account | Hồ sơ cá nhân | Customer | Xem/cập nhật dữ liệu của chính mình | 02 | MVP |
| D2C-013 | D2C / Account | Sổ địa chỉ | Customer | Quản lý nhiều địa chỉ và chọn cho checkout | 02, 06 | MVP |
| D2C-014 | D2C / Order | Đơn hàng của tôi | Customer | Tra cứu lịch sử đơn | 08 | MVP |
| D2C-015 | D2C / Order | Chi tiết đơn & tracking | Customer | Theo dõi Order/Payment/Shipment, hủy hoặc reorder khi hợp lệ | 07, 08, 11, 17, 26 | MVP/GĐ2 |
| D2C-016 | D2C / Order | Tra cứu đơn Guest | Guest | Xác minh sở hữu và xem đơn/tracking | 07, 08, 11 | MVP |
| D2C-017 | D2C / Return | Tạo yêu cầu hủy/đổi/trả | Guest, Customer | Chọn dòng hàng, lý do, số lượng và bằng chứng | 14 | MVP v1.2 |
| D2C-018 | D2C / Return | Chi tiết yêu cầu hậu mãi | Guest, Customer | Theo dõi review, quyết định, hàng trả và refund | 14, 07, 28 | MVP v1.2 |
| D2C-019 | D2C / Support | Gửi yêu cầu hỗ trợ | Guest, Customer | Tạo Ticket có hoặc không liên kết Order | 16 | MVP v1.2 |
| D2C-020 | D2C / Support | Chi tiết Ticket | Guest, Customer | Xem trao đổi công khai, trạng thái và bổ sung thông tin | 16, 28 | GĐ2 |
| D2C-021 | D2C / Review | Viết/chỉnh sửa đánh giá | Customer | Đánh giá Order Item đủ điều kiện và tải ảnh | 15 | GĐ2 |
| D2C-022 | D2C / Promotion | Khuyến mãi & combo | Guest, Customer | Hiểu chương trình đang áp dụng và điều kiện | 17 | GĐ2 |
| D2C-023 | D2C / Loyalty | Ví điểm & lịch sử | Customer | Xem số dư, lịch sử, điểm giữ/chờ/hết hạn | 17 | GĐ2 |
| D2C-024 | D2C / B2B | Yêu cầu báo giá sỉ | B2B Customer | Gửi nhiều dòng nhu cầu, tùy biến và logo | 18 | GĐ2 |
| D2C-025 | D2C / B2B | Chi tiết báo giá & chấp nhận | B2B Customer | Xem đúng version, cung cấp hóa đơn và chuyển thành Order | 18 | GĐ2 |
| D2C-026 | D2C / OCOP | Trang truy xuất QR | Guest, Customer | Xem nguồn gốc/chứng nhận theo Product/SKU/Batch | 19 | MVP v1.2 |
| D2C-027 | D2C / Content | Trung tâm câu chuyện & bài viết | Guest, Customer | Duyệt nội dung thương hiệu đã công khai | 03, 20 | GĐ2 |
| D2C-028 | D2C / Content | Chi tiết bài viết | Guest, Customer | Đọc nội dung, media và liên kết sản phẩm/OCOP | 20 | GĐ2 |
| D2C-029 | D2C / Notification | Trung tâm & tùy chọn thông báo | Customer | Đọc thông báo và quản lý preference/consent được phép | 28 | GĐ2 |

### 4.2 Web Admin

| Screen ID | Phân hệ | Tên màn hình | Primary Actor | Mục tiêu chính | Epic liên quan | Priority |
|---|---|---|---|---|---|---|
| ADM-001 | Admin / Catalog | Product & SKU Workspace | Sales Manager | Tạo/cập nhật Product, SKU và trạng thái bán | 04 | MVP |
| ADM-002 | Admin / Pricing | Giá theo SKU/kênh | Sales Manager | Quản lý giá cơ bản và giá đa kênh | 04 | MVP/GĐ2 |
| ADM-003 | Admin / Customer | Customer Accounts | Nhân viên được cấp quyền | Tra cứu hồ sơ/trạng thái Customer | 02 | GĐ2 |
| ADM-004 | Admin / HR | Employee Profiles | HR Manager | Quản lý hồ sơ nhân sự, tách khỏi system access | 02 | GĐ2 |
| ADM-005 | Admin / Order | Order Operations | Sales Manager, Operations | Tìm/lọc và ưu tiên Order đa kênh theo SLA | 08 | MVP v1.1 |
| ADM-006 | Admin / Order | Order Detail | Sales Manager, Operations | Xem snapshot, timeline và chuyển trạng thái hợp lệ | 07, 08, 11 | MVP v1.1 |
| ADM-007 | Admin / Payment | Payment Reconciliation | Sales Manager, Finance | Đối chiếu Payment–Order và xử lý sai lệch theo quyền | 07 | MVP v1.1 |
| ADM-008 | Admin / Inventory | Inventory Overview | Warehouse Staff | Xem số dư theo SKU và drill-down Batch | 09 | MVP v1.1 |
| ADM-009 | Admin / Inventory | Batch Detail | Warehouse Staff | Quản lý lô, NSX/HSD, nguồn và lịch sử biến động | 09 | MVP v1.1 |
| ADM-010 | Admin / Inventory | Stock Receipt | Warehouse Staff | Nhập kho theo Batch/PO và ghi chênh lệch | 09, 27 | MVP/GĐ3 |
| ADM-011 | Admin / Inventory | Stock Issue & Adjustment | Warehouse Staff | Xuất, hỏng, thất thoát, kiểm kê với audit | 09 | MVP v1.1 |
| ADM-012 | Admin / Inventory | Expiry, FEFO & Consumption | Warehouse, Supply, Sales | Xử lý cận date, gợi ý FEFO và xem tiêu thụ | 09 | MVP/GĐ2 |
| ADM-013 | Admin / Packing | Packing Queue | Packing Staff | Nhận hàng đợi đủ điều kiện theo SLA | 10 | MVP v1.1 |
| ADM-014 | Admin / Packing | Packing Task | Packing Staff | Lấy đúng SKU/Batch, checklist, video và hoàn tất | 10, 26 | MVP/GĐ2 |
| ADM-015 | Admin / Packing | Packing Evidence Viewer | CSKH, Manager được cấp quyền | Tra cứu bằng chứng đúng Order/Package | 10, 14 | GĐ2 |
| ADM-016 | Admin / Shipping | Shipment Operations | Packing/Warehouse/Sales được cấp quyền | Tạo vận đơn, bàn giao và xử lý exception 3PL | 11 | MVP v1.1 |
| ADM-017 | Admin / Delivery | Delivery Workspace | Delivery Staff | Xem assignment và cập nhật trạng thái giao | 11 | GĐ2 |
| ADM-018 | Admin / Shipping | Delivery Performance | Sales Manager | Xem tỷ lệ thành công/thất bại theo kỳ | 11 | GĐ2 |
| ADM-019 | Admin / Marketplace | Marketplace Order Center | Marketplace Operator | Tiếp nhận, theo dõi sync và đưa đơn vào fulfillment | 12 | GĐ2 |
| ADM-020 | Admin / Marketplace | Listing–SKU Mapping | Marketplace Operator | Giải quyết listing chưa/không rõ mapping | 12, 04 | GĐ2 |
| ADM-021 | Admin / Marketplace | Settlement & Revenue | Marketplace Operator, Finance | Đối soát Order, phí, subsidy và net settlement | 12, 24 | GĐ2/GĐ3 |
| ADM-022 | Admin / Offline | Offline Receiving | Offline Sales Staff | Xác nhận hàng kho giao cho điểm bán | 13 | GĐ2 |
| ADM-023 | Admin / Offline | POS Sale | Offline Sales Staff | Ghi nhận giao dịch bán đúng tồn/giá/quyền | 13 | GĐ2 |
| ADM-024 | Admin / Offline | Offline Stock & Return | Offline Sales Staff | Kiểm đếm, hỏng, trả kho và Customer Return | 13, 14 | GĐ2 |
| ADM-025 | Admin / Offline | Shift Report & Reconciliation | Offline Staff, Sales, Finance | Nộp báo cáo và đối soát hàng/tiền | 13 | GĐ2 |
| ADM-026 | Admin / Return | Return Case Queue & Review | CSKH, Sales Manager | Thẩm định, ưu tiên và duyệt/từ chối Case | 14, 16 | MVP v1.2 |
| ADM-027 | Admin / Return | Returned Goods Inspection | Warehouse Staff | Ghi thực nhận, tình trạng và phân loại hàng trả | 14, 09 | MVP v1.2 |
| ADM-028 | Admin / Refund | Refund Approval & Execution | Sales Manager, Finance | Hoàn toàn phần/một phần theo quyết định hợp lệ | 14, 07, 17 | MVP v1.2 |
| ADM-029 | Admin / Review | Review Operations & Analytics | CSKH, Sales Manager | Xử lý review tiêu cực/vi phạm và xem xu hướng | 15 | GĐ2 |
| ADM-030 | Admin / Customer Service | Ticket Queue & Agent Workspace | Customer Service | Nhận, xử lý, trao đổi và xem ngữ cảnh giao dịch | 16 | GĐ2 |
| ADM-031 | Admin / Customer Service | Expiry Coordination | CSKH, Sales Manager | Phối hợp xử lý hàng cận date mà không sửa kho | 16, 09 | GĐ2 |
| ADM-032 | Admin / Promotion | Coupon Workspace | Sales Manager | Tạo, công khai, dừng coupon theo điều kiện | 17 | GĐ2 |
| ADM-033 | Admin / Promotion | Combo Workspace | Sales Manager | Quản lý thành phần, giá và khả năng mua Combo | 17 | GĐ2 |
| ADM-034 | Admin / B2B | B2B Quote Workspace | Sales Manager | Thẩm định request, version hóa và phát hành Quote | 18 | GĐ2 |
| ADM-035 | Admin / OCOP | Traceability Workspace | Manager được cấp quyền | Quản lý hồ sơ Product/SKU/Batch và nguồn | 19 | MVP/GĐ2 |
| ADM-036 | Admin / OCOP | Traceability Approval | Manager được cấp quyền | So sánh, xác minh và công khai/đình chỉ hồ sơ | 19 | GĐ2 |
| ADM-037 | Admin / Content | Content Workspace | Content Manager | Soạn/sửa Article, SEO, nguồn liên kết và revision | 20 | GĐ2 |
| ADM-038 | Admin / Content | Media, Approval & Publication | Content Manager, Approver | Quản lý media, review và lịch xuất bản | 20 | GĐ2 |
| ADM-039 | Admin / Marketing | Campaign Workspace | Marketing Staff | Lập kế hoạch, asset, audience, budget và KPI | 21 | GĐ2 |
| ADM-040 | Admin / Marketing | Campaign Approval & Calendar | Executive/Manager | Duyệt revision và theo dõi lịch execution | 21 | GĐ2 |
| ADM-041 | Admin / Marketing | Campaign Performance & Integration | Marketing, Executive | KPI, sync channel và ROI/ROAS có nguồn | 21, 24, 25 | GĐ2/GĐ3 |
| ADM-042 | Admin / Administration | Employee Account Administration | System Admin/HR được cấp quyền | Tạo/cập nhật/khóa system access | 22 | MVP v1.1 |
| ADM-043 | Admin / Administration | Role & Permission | System Administrator | Quản lý Role và permission matrix | 22 | MVP v1.1 |
| ADM-044 | Admin / Administration | Access Review | System Administrator | Xem quyền hiệu lực và nguồn assignment | 22 | MVP v1.1 |
| ADM-045 | Admin / Audit | Audit Explorer | Auditor, System Admin | Search, xem diff/source/reason/correlation | 23 | MVP/GĐ2 |
| ADM-046 | Admin / Audit | Integrity Verification & Export | Auditor/Security Operator | Kiểm chứng continuity, alert và export có kiểm soát | 23 | GĐ2 |
| ADM-047 | Admin / Finance | Finance Analytics | Executive, Finance | Doanh thu, chi phí, lợi nhuận, kênh, khu vực | 24 | GĐ3 |
| ADM-048 | Admin / Finance | Finance Operations | Finance | Đối soát, thuế, hóa đơn và export job | 24, 07, 12, 13 | GĐ3 |
| ADM-049 | Admin / DSS | Executive Dashboard | Executive | KPI tổng quan và drill-down có freshness | 25 | GĐ2? |
| ADM-050 | Admin / DSS | Customer Analytics | Marketing, Executive | RFM, churn và export phân khúc có kiểm soát | 25 | GĐ3 |
| ADM-051 | Admin / DSS | Product & Supply Analytics | Supply, Executive, Sales | Performance, velocity, forecast, expiry risk | 25 | GĐ3 |
| ADM-052 | Admin / DSS | Strategic Insights | Executive, Sales, Marketing | Association, seasonality, recommendation và outcome | 25 | GĐ3 |
| ADM-053 | Admin / Procurement | Supplier Directory | Supply Manager | Quản lý Supplier, hợp đồng, giá và hiệu lực | 27, 19 | GĐ3 |
| ADM-054 | Admin / Procurement | Supply Plan & Purchase Order | Supply Manager | Lập plan, scenario, PO revision/approval/issue | 27, 25 | GĐ3 |
| ADM-055 | Admin / Procurement | PO Receiving & Variance | Warehouse Staff | Đối chiếu expected–actual, chất lượng và Batch | 27, 09 | GĐ3 |
| ADM-056 | Admin / Notification | Internal Alert Center | Nhân viên được cấp quyền | Nhận/acknowledge alert SLA, expiry, order khẩn | 28 | GĐ2 |
| ADM-057 | Admin / Notification | Template, Route & Delivery Operations | Admin/Operator được cấp quyền | Quản lý version/template và điều tra delivery | 28 | GĐ2 |

Kiểm tra inventory: không có Screen ID trùng; D2C và Admin không bị gộp; background requirements không tạo screen giả. Các overlap có chủ ý: Order Detail là nguồn nghiệp vụ, còn Ticket/Return/Packing/Payment chỉ hiển thị snapshot hoặc deep link theo quyền.

## 5. Web D2C Screens

### D2C-001 — Trang chủ & khám phá

#### 1. Web Module & Screen

**Web Type:** Web D2C · **Module:** Discovery · **Primary Actor:** Guest/Registered Customer. **Entry:** URL gốc, campaign, search engine. **Previous:** external/referrer. **Next:** D2C-002, D2C-003, D2C-004, D2C-022, D2C-027.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Giúp khách nhanh chóng đi từ nhu cầu chưa cụ thể đến catalog, sản phẩm hoặc nội dung thương hiệu đã công khai; business đưa đúng sản phẩm/câu chuyện/khuyến mãi hợp lệ vào phễu mua. **2.2 User goals:** duyệt danh mục; mở sản phẩm; xem bán chạy/gợi ý; đọc câu chuyện Huế. **2.3 Preconditions:** không cần đăng nhập; chỉ dữ liệu public/đang bán/đã duyệt.

**2.4 Main flow:** (1) mở trang; (2) xem search và danh mục; (3) chọn collection/sản phẩm/nội dung; (4) đi tới listing/PDP/article. **2.5 Alternative:** không có dữ liệu nổi bật thì ẩn khối, không tạo placeholder giả; recommendation AI thiếu dữ liệu thì dùng gợi ý chung; nội dung hết hiệu lực không hiển thị.

**2.6 Sections:** 01 global search (query, mở D2C-002); 02 category navigation (tên, ảnh/nội dung đại diện đã duyệt); 03 featured/bestseller [GĐ2] (Product, SKU khả dụng, giá hiện hành, lý do gợi ý nếu cá nhân hóa); 04 promotion/combo đang hiệu lực; 05 câu chuyện thương hiệu và bài viết; 06 trust entry tới truy xuất OCOP. **2.7 Data:** product name, public media, price/range theo SKU, availability, promotion label, article title/summary. **2.8 Search/filter/sort:** search theo từ khóa; không đặt filter đầy đủ ở Home. **2.9 Actions:** search, mở category/PDP/promotion/article, bỏ qua recommendation. **2.10 Rules:** không hiển thị Product/SKU không public hoặc không thể mua; nội dung phải đã duyệt. Sources: US-DISC-01, 04, 05; US-AI-04; BR-PROD-04. **2.11 States:** loading skeleton, content partial, no-featured, recommendation fallback, error. **2.12 Feedback:** search suggestion panel [UX RECOMMENDATION]. **2.13 Permissions:** public; personalization chỉ với dữ liệu/consent hợp lệ. **2.14 Navigation:** D2C-001 → D2C-002/004/022/027. **2.15 Interaction:** ưu tiên search/category và khả năng mua; section dài không che đường tới catalog.

#### 3. Reference / UX Layout Direction

Không có visual reference được chỉ định. Dùng pattern homepage D2C theo hierarchy search → category → commerce → brand content, áp dụng Master Design System hiện có.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Trang chủ & khám phá**, Web D2C/Discovery, dùng “O Mạ — Master Design System V1.0 (Desktop 1440px)”, không đổi token. Primary user: khách mới/khách quay lại. Bố cục: search nổi bật; danh mục; sản phẩm đang bán/bestseller hoặc recommendation có fallback; promotion/combo; câu chuyện Huế; entry truy xuất OCOP. Hiển thị tên/media/giá/availability và chỉ nội dung đã công khai. Actions: tìm kiếm, mở listing/PDP/article/promotion. States: loading, partial, không có khối nổi bật, lỗi, recommendation fallback. Flow: external → D2C-001 → D2C-002/D2C-004/D2C-027. Không thêm nghiệp vụ ngoài requirement. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 03 / US-DISC-01, 04, 05, US-AI-04; EPIC 20; FR-21, FR-31; UC-DISC-01, 04; diagrams 03, 13.

### D2C-002 — Danh mục & kết quả tìm kiếm

#### 1. Web Module & Screen

**Web Type:** Web D2C · **Module:** Discovery · **Actor:** Guest/Customer. **Entry:** Home search/category, article/Product link. **Previous:** D2C-001/027/028. **Next:** D2C-004; D2C-003.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho khách thu hẹp catalog công khai theo nhu cầu thực và thấy rõ tiêu chí đang áp dụng. **2.2 Goals:** tìm theo từ khóa; lọc; sort [UX RECOMMENDATION nếu chưa chốt]; mở PDP. **2.3 Preconditions:** catalog có dữ liệu public. **2.4 Flow:** nhập query/chọn category → hệ thống trả kết quả → áp/bỏ filter → mở Product. **2.5 Alternative:** query rỗng/không hợp lệ; no results; Product biến mất/hết hàng giữa lúc xem.

**2.6 Sections:** 01 breadcrumb/query summary; 02 search field; 03 filter controls + applied chips; 04 result count/sort; 05 product grid/list; 06 no-result recovery. **2.7 Data:** Product name, media, category, SKU price/range, weight/package indicators khi phù hợp, availability, rating/count chỉ khi EPIC 15 có dữ liệu, promotion indicator. **2.8 Search by:** tên Product và phạm vi từ khóa cần PO chốt. **Filter by [DOCUMENTED]:** giá, khối lượng, loại sản phẩm, rating, còn hàng; semantics Product/SKU là open question. **Sort:** [UX RECOMMENDATION] relevance/giá; chưa là rule. **2.9 Actions:** apply/remove/reset filters, sort, open PDP, quay lại search. **2.10 Rules:** chỉ public/sellable; hết hạn không tính available; rating không được giả là 0 khi chưa có dữ liệu. Sources: US-DISC-01~03; US-PROD-06; BR-BATCH-05. **2.11 States:** loading, empty category, no search/filter results, partial availability, error. **2.12 Drawer:** filter drawer trên viewport hẹp [UX RECOMMENDATION]. **2.13 Permissions:** public. **2.14 Nav:** D2C-001 → D2C-002 → D2C-004. **2.15 Interaction:** applied filters luôn nhìn thấy; ưu tiên tên/giá/availability khi không đủ không gian.

#### 3. Reference / UX Layout Direction

Pattern PLP/search hiện đại: controls trước results, filter có trạng thái rõ, no-results có đường sửa query/reset filter.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Danh mục & kết quả tìm kiếm**, Web D2C. Bố cục: breadcrumb/query; search; filter + applied chips; result count/sort; product grid; no-result recovery. Filters được document: giá, khối lượng, loại, rating, còn hàng; rating chỉ xuất hiện khi dữ liệu Review khả dụng. Cards hiển thị Product, media, giá/range, availability và promotion hợp lệ. Actions: tìm, lọc, bỏ filter, mở PDP. States: loading, empty category, no results, partial, error. Flow D2C-001 → D2C-002 → D2C-004. Dùng Master Design System hiện có; không tự thêm filter/business rule. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 03 / US-DISC-01~03; EPIC 04; EPIC 15; FR-31; UC-DISC-01~03.

### D2C-003 — Trợ lý tìm sản phẩm

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Discovery AI · **Actor:** Guest/Customer · **Entry/Previous:** D2C-001/002 · **Next:** D2C-002/004.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Chuyển mô tả tự nhiên như mục đích tặng quà, khẩu vị, ngân sách thành gợi ý có thể kiểm tra, không cho AI thay đổi dữ liệu. **2.2 Goals:** hỏi; hiểu gợi ý/lý do; mở kết quả hoặc quay lại search thường. **2.3 Preconditions:** GĐ3, AI khả dụng và privacy/consent được đáp ứng. **2.4 Flow:** nhập nhu cầu → nhận gợi ý Product public/available → mở PDP hoặc chuyển tiêu chí sang listing. **2.5 Alternative:** không hiểu, thiếu dữ liệu, provider unavailable, gợi ý đã hết hàng. **2.6 Sections:** conversation/input; interpreted needs; recommendation cards; sources/limitations; conventional search link. **2.7 Data:** user prompt, interpreted constraints, Product/SKU, price, availability, explanation. **2.8 Search:** hội thoại; không tự thêm profile data. **2.9 Actions:** send, refine, open Product, dismiss, return to filters. **2.10 Rules:** AI chỉ gợi ý, không sửa giá/tồn/Order; dữ liệu ngoài quyền không được gửi. Sources: US-AI-01, 04; External AI rules. **2.11 States:** thinking, no-match, low-confidence, stale result, provider error. **2.12 Feedback:** cảnh báo AI/thiếu dữ liệu. **2.13 Permission/privacy:** consent và opt-out cần chốt. **2.14 Nav:** D2C-001/002 ↔ D2C-003 → D2C-004. **2.15 Interaction:** luôn có lối về search/filter chuẩn.

#### 3. Reference / UX Layout Direction

Conversational discovery kết hợp result cards; không trình bày suggestion như cam kết tồn/giá.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Trợ lý tìm sản phẩm**, Web D2C/GĐ3. Khu vực: hội thoại + input; nhu cầu đã hiểu; product recommendations với lý do, giá hiện hành và availability; giới hạn/nguồn; đường về tìm kiếm thường. Actions: gửi, tinh chỉnh, mở PDP, bỏ gợi ý. States: thinking, no match, low confidence, stale/out-of-stock, AI unavailable. Dán nhãn AI rõ; không cho AI thay đổi business data. Dùng Master Design System hiện có. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 03 / US-AI-01, US-AI-04; FR-31; UC-AI-01, 02.

### D2C-004 — Chi tiết sản phẩm

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Product · **Actor:** Guest/Customer · **Entry:** D2C-001/002/003/022/028 · **Next:** D2C-005, D2C-026, D2C-021.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Giúp khách hiểu đúng Product/SKU thực phẩm và chọn SKU có thể mua trước khi thêm giỏ. **2.2 Goals:** xem media/thông tin; chọn Variant/SKU; xem giá/tồn; thêm giỏ; đọc review/traceability. **2.3 Preconditions:** Product public; SKU dữ liệu hợp lệ. **2.4 Flow:** mở PDP → xem media/info → chọn SKU → price/availability cập nhật → chọn quantity → Add to Cart. **2.5 Alternative:** chưa chọn SKU, SKU hết hàng/tạm ngừng/ngừng bán, thông tin bắt buộc chưa public, price/stock thay đổi.

**2.6 Sections:** breadcrumb; media; purchase information; SKU selector; quantity; food information (thành phần, khối lượng/quy cách, bảo quản, thông tin NSX/HSD ở đúng cấp đã chốt); OCOP/traceability entry; description/story; promotion/combo; reviews; recommendations. **2.7 Data:** Product/SKU name/code khi public, media, selected attributes, price, availability, food info, rating/count, verified review + reply, traceability link. **2.8:** không phải listing; review filter/sort [UX RECOMMENDATION]. **2.9 Primary:** Add to Cart; **Secondary:** open traceability, promotion, review media, recommendation. **2.10 Rules:** SKU phải được chọn; giá/tồn theo SKU; hết hạn không bán; Product info phải duyệt. Sources: BR-PROD-01~04, BR-BATCH-05, US-PROD-03~04, US-CART-01, US-REV-03. **2.11 States:** loading, invalid SKU, out of stock, paused/discontinued, partial info, add success/error. **2.12 Feedback:** add-to-cart toast/mini-cart; SKU-required validation. **2.13 Permissions:** public; review creation requires eligible Customer. **2.14 Nav:** D2C-002/003 → D2C-004 → D2C-005; → D2C-026/021. **2.15 Interaction:** purchase panel luôn dễ tiếp cận khi đọc nội dung dài [UX RECOMMENDATION].

#### 3. Reference / UX Layout Direction

Modern D2C PDP: media và purchase information song song, nội dung/traceability/review phía dưới; không sao chép visual style nền tảng khác.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Chi tiết sản phẩm**, Web D2C/Product. Bố cục: breadcrumb; media; purchase panel với tên, SKU selector, price, availability, quantity, Add to Cart; food information; OCOP; description/story; promotion/combo; verified reviews; recommendations. States: chưa chọn SKU, out of stock, paused/discontinued, price/stock changed, loading/error, add success. Flow listing/AI → PDP → cart; PDP → traceability/review. Purchase panel dễ truy cập khi scroll. Dùng Master Design System; không invent SKU attribute hoặc HSD display rule chưa chốt. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 04 / US-PROD-03, 04, 06; EPIC 03 / US-DISC-04, US-AI-04; EPIC 15 / US-REV-03; EPIC 19; FR-04, 09, 16, 20.

### D2C-005 — Giỏ hàng

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Cart · **Actor:** Guest/Customer · **Entry:** PDP/mini-cart · **Previous:** D2C-004 · **Next:** D2C-006.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho khách kiểm soát SKU/số lượng và hiểu subtotal trước phí/ưu đãi. **2.2 Goals:** xem, đổi quantity, xóa, xử lý item không còn mua được, checkout. **2.3 Preconditions:** có cart session/account. **2.4 Flow:** xem lines → chỉnh quantity → hệ thống validate current purchasability → subtotal cập nhật → checkout. **2.5 Alternative:** empty; vượt tồn; SKU paused/out-of-stock/expired; price changed; guest/login cart conflict [OPEN].

**2.6 Sections:** cart lines; line warnings; subtotal summary; checkout action; continue shopping. **2.7 Fields/columns:** Product media/name, selected SKU/variant, unit price/current change, quantity, line total, availability, remove. **2.8:** không filter; [UX] grouping không cần. **2.9 Primary:** Checkout. Secondary: change quantity, remove, choose alternative/open PDP. Destructive: remove confirmation chỉ khi cần [UX]. **2.10 Rules:** quantity dương và không vượt khả năng mua; cart không reserve stock; subtotal không phải final total; current price used at validation. Sources: US-CART-01~04; EPIC 09. **2.11 States:** loading, empty, validation error, stale price/stock, partial invalid lines, network error. **2.12 Feedback:** line-level error; remove undo [UX RECOMMENDATION]. **2.13 Permissions:** owner/session isolation. **2.14 Nav:** D2C-004 ↔ D2C-005 → D2C-006. **2.15 Interaction:** summary/action dễ tiếp cận; preserve context on validation.

#### 3. Reference / UX Layout Direction

Cart lines + persistent order summary; invalid item xử lý tại dòng thay vì lỗi chung.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Giỏ hàng**, Web D2C. Sections: cart lines với Product/SKU/price/quantity/line total/availability/remove; line warnings; subtotal ghi rõ chưa gồm shipping/discount; actions checkout/continue shopping. States: empty, loading, quantity invalid, exceeds stock, SKU unavailable, price changed, partial invalid, network error. Flow PDP → Cart → Checkout. Không reserve stock tại Cart. Dùng Master Design System hiện có. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 05 / US-CART-01~04; BR-ORDER-02; FR-05; UC-CART-01, 02.

### D2C-006 — Checkout

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Checkout/Gifting/Promotion · **Actor:** Guest/Customer · **Entry:** Cart, Buy Now [nếu được hỗ trợ] · **Previous:** D2C-005/004 · **Next:** D2C-007/008.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Thu đủ dữ liệu nhận hàng, tính shipping, áp quyền lợi hợp lệ, review snapshot và tạo Order/reservation chống oversell. **2.2 Goals:** nhập/chọn địa chỉ; xem phí; chọn payment; coupon/loyalty; gift; xác nhận. **2.3 Preconditions:** cart có item hợp lệ. **2.4 Flow:** thông tin nhận → shipping quote → tùy chọn gift/ưu đãi → chọn payment → review breakdown → confirm; hệ thống kiểm tra lại giá/tồn và giữ tồn nguyên tử; tạo Order một lần. **2.5 Alternative:** invalid address, no shipping method/provider error, coupon invalid, insufficient points, one line out-of-stock, price/fee changed, reservation failure/expiry, duplicate confirm.

**2.6 Sections theo bước:** 01 contact/recipient; 02 saved/new address; 03 shipping method/fee; 04 gift preference (recipient, message, hide price) [GĐ2]; 05 coupon/loyalty [GĐ2]; 06 payment method; 07 items snapshot; 08 breakdown (merchandise, shipping, discounts, points, final total); 09 confirmation. **2.7 Data:** buyer/recipient, phone, address fields (schema OPEN), shipping quote/service, items/SKU/qty/price, promotion/point result, gift preference, payment choice. **2.8:** address selection/search [UX]; không filter. **2.9 Primary:** Confirm order. Secondary: edit section/apply-remove coupon/points/gift. **2.10 Rules:** Guest không bị ép đăng ký; address ownership; no oversell; snapshot Order; expired Batch excluded; gift recipient không phải owner; price hidden chỉ trên recipient artifact; coupon/points revalidate. Sources: US-AUTH-01; US-CHK-01~05; US-GIFT-01~03; US-PROMO-01; US-LOY-02. **2.11 States:** step loading, invalid fields, quote unavailable/changed, stock/price changed, reservation processing/failed/expired, confirm success/error. **2.12 Modals:** confirm changed total; leave with unsaved data [UX]; reservation expiry warning. **2.13 Permissions:** Customer chỉ dùng address/points của mình. **2.14 Nav:** Cart → Checkout → Payment/Confirmation; edit returns to section, không làm mất data hợp lệ. **2.15 Interaction:** summary luôn dễ tiếp cận; multi-step phải cho thấy bước và error source.

#### 3. Reference / UX Layout Direction

Step checkout với form chính và order summary; các phần gift/promotion là mở rộng có điều kiện, không làm rối MVP.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Checkout**, Web D2C. Flow rõ: nhận hàng → shipping/payment → review/confirm. Sections: buyer/recipient; saved/new address; shipping quote; optional gift message/hide-price; coupon/loyalty; payment method; items; full price breakdown; confirm. States: validation errors, no shipping, provider error, coupon/points invalid, stock/price/fee changed, reservation processing/failed/expired, duplicate-safe confirmation. Guest checkout phải rõ. Flow Cart → Checkout → Payment/Order confirmation. Dùng Master Design System; không chốt field/address/payment/provider chưa được PO xác nhận. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 01 / US-AUTH-01; EPIC 06 / US-CHK-01~05; EPIC 17 / US-PROMO-01, US-LOY-02; EPIC 26 / US-GIFT-01~03; UC-CHK-01~03, UC-GIFT-01~02, UC-ORD-01; FR-05, 17, 18, 28.

### D2C-007 — Thanh toán & chờ xác nhận

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Payment · **Actor:** Guest/Customer · **Entry:** successful Order creation · **Previous:** D2C-006 · **Next:** D2C-008/015/016.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Hướng dẫn thanh toán đúng amount/reference và phản ánh trạng thái có thẩm quyền; với COD giải thích chưa thu tiền. **2.2 Goals:** quét QR/chuyển khoản, chờ, retry/change method khi được phép, xem kết quả. **2.3 Preconditions:** Order/Payment Attempt hợp lệ. **2.4 Flow:** hiển thị method details → Customer thực hiện → hệ thống chờ xác nhận hợp lệ → success chuyển D2C-008. **2.5 Alternative:** pending, failed, expired, mismatch/needs reconciliation, provider timeout, late confirmation; COD đi thẳng xác nhận đặt hàng.

**2.6 Sections:** Order code/amount; QR or bank instruction/reference; remaining validity only theo policy; authoritative payment status; help/retry; security note. **2.7 Data:** Order code, amount, method, reference, attempt status/time; không lộ secret. **2.8:** none. **2.9 Primary:** Check status/continue after success; Secondary: retry/change method if allowed, copy amount/reference, contact support. **2.10 Rules:** redirect page không chứng minh Paid; duplicate callback không cộng tiền; mismatch không tự Paid; COD not collected. Sources: US-PAY-01~03; BR-ORDER-03. **2.11 States:** initializing, pending, success, failed, expired, mismatch, integration unavailable, offline. **2.12 Feedback:** expiry/retry confirmation. **2.13 Permissions:** Guest ownership verification on later revisit. **2.14 Nav:** D2C-006 → 007 → 008; error → support/order. **2.15 Interaction:** status/amount/reference ưu tiên cao; polling/realtime technology không được đặc tả.

#### 3. Reference / UX Layout Direction

Payment status pattern với một nguồn trạng thái rõ; không trình bày return URL như success.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Thanh toán & chờ xác nhận**, Web D2C/Payment. Hiển thị Order code, exact amount, QR/chuyển khoản reference, validity theo policy, authoritative status và help/retry. COD có trạng thái “thanh toán khi nhận”, không phải Paid. States: initializing, pending, success, failed, expired, mismatch/reconciliation, provider unavailable, offline. Actions: copy, check status, retry/change method nếu được phép, support. Flow Checkout → Payment → Confirmation/Order Detail. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 07 / US-PAY-01~03; EPIC 06 / US-CHK-03; UC-PAY-01, 02; UC-ORD-01; FR-06.

### D2C-008 — Xác nhận đặt hàng

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Order · **Actor:** Guest/Customer · **Entry:** payment/order success · **Previous:** D2C-007/006(COD) · **Next:** D2C-015/016/010.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Xác nhận Order đã được tạo đúng một lần, trạng thái Payment tách biệt, và cung cấp đường theo dõi. **2.2 Goals:** lưu mã đơn, hiểu bước tiếp theo, theo dõi hoặc tạo account/link order. **2.3 Preconditions:** Order created. **2.4 Flow:** hiển thị confirmation → summary/status → tracking instructions → destination. **2.5 Alternative:** Payment pending/mismatch/COD; notification email pending/failed không làm Order thất bại. **2.6 Sections:** success/pending status; Order code; buyer/recipient summary; item/total snapshot; Payment status; shipping next step; contact/help; Guest account/link prompt [GĐ2]. **2.7 Data:** code, created time, items, total, payment/Order status, masked contact/address. **2.9 Primary:** View order. Secondary: continue shopping, register/link Guest Order. **2.10 Rules:** status không hợp nhất mơ hồ; email delivery independent. **2.11 States:** confirmed, payment pending, COD confirmed, notification delayed, partial data/error. **2.12 Feedback:** copy code; no destructive modal. **2.13 Permissions:** only current confirmed session; later Guest lookup needs verification. **2.14 Nav:** 007 → 008 → 015/016. **2.15 Interaction:** mã đơn và bước tiếp theo phải nổi bật.

#### 3. Reference / UX Layout Direction

Order success page có summary cô đọng và CTA theo dõi; tránh giả định Paid khi COD/pending.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Xác nhận đặt hàng**, Web D2C. Sections: confirmation/status; Order code; items/total snapshot; buyer/recipient masked summary; Payment status riêng; next shipping step; tracking/help; Guest registration/link option. States: prepaid success, payment pending, COD confirmed-not-paid, notification delayed, partial/error. Actions: view order, copy code, continue shopping, optional register/link. Flow Payment/Checkout → Confirmation → Order Detail/Guest Lookup. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 07, 08, 28; US-PAY-02, 03; US-ORD-02, 03; US-AUTH-06; FR-06, 07, 30.

### D2C-009 — Đăng nhập

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Auth · **Actor:** Registered Customer · **Entry:** header/protected destination/checkout optional · **Next:** return URL, D2C-012/014.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Xác thực Customer và trở lại đúng hành trình mà không ép Guest checkout. **2.2 Goals:** login bằng email/password; recover; register. Social Login thuộc Giai đoạn 2 và không hiển thị trong MVP. **2.3 Preconditions:** account `ACTIVE`. **2.4 Flow:** nhập email/password → submit → nhận secure session cookie → success redirect. **2.5 Alternative:** invalid credentials, account unavailable, rate limited, server/offline error. **2.6 Sections:** login form; remember-me mặc định tắt; recover/register links; Guest continuation when invoked from checkout. **2.7 Data:** email, password, rememberMe, allowlisted returnUrl. **2.9 Primary:** Login; Secondary: recover, register, continue Guest. **2.10 Rules:** protect other Customer data; không lộ access/refresh token cho JavaScript. **2.11 States:** loading, field error, auth failed, account unavailable, rate limited, server error, offline. **2.12 Feedback:** generic secure error; no account enumeration [UX/NFR]. **2.13 Permissions:** public. **2.14 Nav:** any protected → 009 → return; 009 → 010/011/006. **2.15 Interaction:** preserve intended destination/cart; returnUrl chỉ là relative internal path hợp lệ.

#### 3. Reference / UX Layout Direction

Focused authentication form; optional methods visually secondary to the configured primary method.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Đăng nhập**, Web D2C/Auth MVP. Form email + password; remember-me mặc định tắt; links đăng ký/khôi phục; khi vào từ checkout phải có “tiếp tục Guest”. Không hiển thị Social Login trong MVP. States: submitting, invalid credential, account unavailable, rate limited, server error, offline. Preserve allowlisted internal return URL/cart. Dùng Master Design System và approved Auth decisions trong `SCOPE_TRACEABILITY.md`. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 01 / US-AUTH-01, 03, 04; FR-01; UC-AUTH-03, 04, 06.

### D2C-010 — Đăng ký

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Auth · **Actor:** Guest · **Entry:** login/header/order success · **Next:** verification/return destination/profile.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tạo Customer Account hợp lệ, không trùng và không tự động biến mọi Guest Order thành account. **2.2 Goals:** register; bắt buộc verify email; Guest Order linking thuộc Giai đoạn 2. **2.3 Preconditions:** none. **2.4 Flow:** nhập displayName/email/password/confirm password/terms → validate → tạo `PENDING_VERIFICATION` → kiểm tra email → mở magic link → `ACTIVE` + auto-login → returnUrl. **2.5 Alternative:** registration unavailable, invalid fields/password, delivery failure, link invalid/expired/used, rate limit. **2.6 Sections:** account form; password checklist; terms acknowledgement; verification-email feedback; login link. **2.7 Data:** displayName, email, password, passwordConfirmation client-only, termsVersion. Không thu phone/dateOfBirth/preferences trong Auth MVP. **2.9 Primary:** Create account. **2.10 Rules:** password 15–128 ký tự, có chữ hoa, chữ thường, số, ký tự đặc biệt; invalid rejected; Guest Order not auto-linked. **2.11 States:** form, validation, pending email verification, link invalid/expired/used, verified + auto-login, delivery/server/offline error. **2.12 Modal:** none; resend cooldown 60 giây theo server. **2.13 Permissions:** public. **2.14 Nav:** 009/008 → 010 → email link → returnUrl. **2.15:** không dùng OTP/SMS/Zalo trong MVP.

#### 3. Reference / UX Layout Direction

Progressive registration; only ask documented required data once PO confirms.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Đăng ký**, Web D2C/Auth MVP. Fields: displayName, email, password, confirm password, terms. Password checklist: 15–128 ký tự và có chữ hoa, chữ thường, số, ký tự đặc biệt. Sau submit luôn có trạng thái chờ email magic link; không dùng OTP/SMS/Zalo. States: registration unavailable, invalid fields/password, verification pending/invalid/expired/used, delivery/rate-limit error, verified + auto-login. Không thu phone/dateOfBirth/preferences và không link Guest Order trong MVP. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 01 / US-AUTH-02, 06; FR-01; UC-AUTH-02, 07.

### D2C-011 — Khôi phục quyền truy cập

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Auth · **Actor:** Customer · **Entry:** D2C-009 · **Next:** D2C-009.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Khởi tạo và hoàn tất reset password an toàn, giảm hỗ trợ thủ công. **2.2 Goals:** request email; exchange magic link; set new password. **2.3 Preconditions:** email/account hợp lệ nhưng UI không tiết lộ tồn tại. **2.4 Flow:** nhập email → neutral confirmation → mở magic link → exchange token → đặt password mới → thu hồi session → login. **2.5 Alternative:** invalid/expired/already-used link, resend/rate limit, email delivery failure. **2.6 Sections:** request form; neutral confirmation; magic-link landing/reset form; completion. **2.7 Data:** email, recoveryToken, resetProof, newPassword, newPasswordConfirmation client-only. **2.9 Actions:** submit, resend after server cooldown, return login. **2.10 Rules:** recovery link single-use và hết hạn 15 phút; password 15–128 ký tự, có chữ hoa, chữ thường, số, ký tự đặc biệt. **2.11 States:** requesting, sent, link invalid/expired/used, setting password, reset success/error, offline. **2.12 Feedback:** neutral anti-enumeration message [UX/security]. **2.13:** public with secure token/proof. **2.14:** 009 → 011 → email link → 011 reset → 009. **2.15:** no auto-login after reset; retain no sensitive data in UI history.

#### 3. Reference / UX Layout Direction

Focused recovery step flow; outcome copy must not expose account existence.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Khôi phục quyền truy cập**, Web D2C/Auth MVP. Steps: nhập email; neutral sent state; mở email magic link; đặt password mới; completion back to Login. Không dùng OTP/SMS/Zalo. Password checklist: 15–128 ký tự và có chữ hoa, chữ thường, số, ký tự đặc biệt. States: invalid/expired/used link, resend limited, email delivery failure, password validation, success/offline. Reset thành công không auto-login và thu hồi session cũ. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 01 / US-AUTH-05; EPIC 28 / US-NOTI-01; FR-01; UC-AUTH-05.

### D2C-012 — Hồ sơ cá nhân

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Account · **Actor:** Customer · **Entry:** account navigation · **Previous:** D2C-009 · **Next:** D2C-013/014/023/029.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho Customer xem/cập nhật đúng hồ sơ của mình để tái sử dụng cho mua hàng. **2.2 Goals:** view/edit/save. **2.3 Preconditions:** authenticated. **2.4 Flow:** load own profile → edit → validate → save. **2.5 Alternative:** invalid field, contact re-verification required [OPEN], concurrent/stale update, restricted account. **2.6 Sections:** account summary/status; personal/contact fields; privacy/account links; related navigation. **2.7 Data:** only documented profile data; exact required fields OPEN. **2.9 Actions:** edit/save/cancel; initiate contact verification if policy. **2.10 Rules:** cannot view/update another Customer; history orders retained across status change. **2.11 States:** view/edit/loading/validation/save success/error/permission denied. **2.12:** unsaved changes confirm [UX]. **2.13:** owner only. **2.14:** account hub links to addresses/orders/loyalty/notifications. **2.15:** form errors adjacent to fields.

#### 3. Reference / UX Layout Direction

Account settings form with clear read/edit mode; do not mix address or order entities into profile fields.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Hồ sơ cá nhân**, Web D2C/Account. Sections: account summary/status; personal/contact form; privacy links; navigation to Address, Orders, Loyalty, Notifications. States: view/edit, validation, verification required, saving/success/error, permission denied. Exact required fields remain open; only own profile. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 02 / US-USER-01; FR-01; UC-USER-01; NFR-04, 05, 08.

### D2C-013 — Sổ địa chỉ

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Account/Checkout · **Actor:** Customer · **Entry:** account or Checkout · **Next:** D2C-006.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý nhiều địa chỉ và chọn lại cho checkout mà không truy cập dữ liệu người khác. **2.2 Goals:** list/add/edit/delete/select. **2.3 Preconditions:** authenticated. **2.4 Flow:** view cards → add/edit valid address → save → optionally select for checkout. **2.5 Alternative:** invalid, delete address in pending order/default [OPEN], no addresses, stale administrative data. **2.6 Sections:** address list; add/edit form/drawer; selection context. **2.7 Data:** recipient/name, contact, administrative/address fields—exact schema OPEN; default marker only if policy confirmed. **2.8:** [UX] search only if many, not required. **2.9:** add/edit/delete/select. **2.10 Rules:** owner isolation; saved address use in checkout; deleting saved record does not alter Order snapshot. **2.11 States:** empty/loading/form validation/save/delete error. **2.12:** delete confirmation; address editor drawer [UX]. **2.13:** owner only. **2.14:** D2C-012/006 ↔ 013. **2.15:** selection and management modes clearly distinct.

#### 3. Reference / UX Layout Direction

Address cards + editor drawer; checkout entry returns selected address to the same step.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Sổ địa chỉ**, Web D2C. Address list/cards; add/edit form; delete confirmation; select-and-return mode when opened from Checkout. States: empty, loading, validation, save/delete error. Exact administrative fields/default/delete policy are open. Do not alter historical Order snapshots. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 02 / US-USER-02; EPIC 06 / US-CHK-01; FR-05; UC-USER-02.

### D2C-014 — Đơn hàng của tôi

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Order · **Actor:** Customer · **Entry:** account navigation · **Next:** D2C-015.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tự tra cứu lịch sử Order thuộc Customer. **2.2 Goals:** scan history, identify status, open detail. **2.3 Preconditions:** authenticated. **2.4 Flow:** load list → optional search/filter → open detail. **2.5 Alternative:** no orders, linked Guest Order appears, partial/stale status. **2.6 Sections:** page summary; filters; order list/cards; pagination. **2.7 Columns/data:** Order code, created time, item summary, total, Order status; Payment/Shipment summary when useful [DOCUMENTED from detail separation]. **2.8 Search:** Order code. **Filter:** status/time [UX supported by use case]; **Sort:** newest default [UX]. **2.9:** view detail; reorder where eligible may originate in detail. **2.10:** only own Orders; linked Guest only after verification. **2.11:** loading, empty, no results, partial, error. **2.12:** none. **2.13:** owner. **2.14:** D2C-012 → 014 → 015. **2.15:** list becomes cards on narrow view while preserving code/status/total.

#### 3. Reference / UX Layout Direction

Transaction-history pattern with status hierarchy and direct row/card navigation.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Đơn hàng của tôi**, Web D2C/Order. Sections: summary/filter; list with Order code, date, items summary, total, status; pagination; empty recovery. Search by code; filter status/time; newest first as UX recommendation. States: loading, empty, no results, partial/error. Click opens D2C-015. Only own/verified-linked Orders. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 08 / US-ORD-01; EPIC 01 / US-AUTH-06; FR-07; UC-ORD-03.

### D2C-015 — Chi tiết đơn & tracking

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Order/Shipping · **Actor:** Customer · **Entry:** D2C-014, notification deep link · **Next:** D2C-017/019/021/005.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho Customer hiểu riêng Order, Payment, Packing/Shipment và các action hợp lệ. **2.2 Goals:** view snapshot/timeline/tracking; cancel; reorder; request return/support/review. **2.3 Preconditions:** ownership. **2.4 Flow:** verify access → show current statuses + history → show eligible actions → navigate/action. **2.5 Alternative:** no Shipment, multiple Shipment, pending/multiple Payment Attempts, stale/out-of-order update, cancel no longer eligible, gift cutoff passed.

**2.6 Sections:** Order header/code/source/date; status trio; item snapshot; buyer/recipient/gift; price breakdown; Payment history/status; Shipment timeline per package; Order timeline; action panel; related support/return/review. **2.7 Data:** snapshot names/SKU/qty/price, discounts/shipping/total, masked addresses, state timestamps, tracking code/provider if available, gift message/preferences, refund/return summary. **2.8:** none. **2.9 Primary:** relevant next action. Secondary: cancel if eligible, reorder, track, support, return, review. **2.10 Rules:** cannot view other Customer; cancel only allowed window; statuses from source; reorder uses current price/stock/promotion; recipient not owner. Sources: US-ORD-02~04; US-SHIP-02; US-LOY-03; US-GIFT-01~03. **2.11 States:** loading, payment pending/failed, no tracking, shipment partial/multiple, cancellation processing/success/rejected, stale/partial/error. **2.12:** cancel confirmation explains stock/payment consequence; reorder preview. **2.13:** owner only. **2.14:** 014/notification → 015 → return/support/review/cart. **2.15:** timeline collapses detail but status/action remains accessible.

#### 3. Reference / UX Layout Direction

Order detail with separate status lanes and chronological timeline; no single ambiguous “order status”.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Chi tiết đơn & tracking**, Web D2C. Header; separate Order/Payment/Shipping statuses; item and financial snapshot; buyer/recipient/gift; payment attempts; package tracking timelines; Order history; eligible actions. Actions: cancel with confirmation, reorder using current data, return, support, review. States: payment pending/failed, no/multiple shipment, partial delivery, cancellation result, stale/partial/error. Flow My Orders/notification → Order Detail → Return/Support/Review/Cart. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 07 / US-PAY-03; EPIC 08 / US-ORD-02~04; EPIC 11 / US-SHIP-02; EPIC 17 / US-LOY-03; EPIC 26; FR-07, 12, 18, 28.

### D2C-016 — Tra cứu đơn Guest

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Guest Order · **Actor:** Guest · **Entry:** confirmation/email/public lookup · **Next:** verified D2C-015 view mode.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Guest xem Order/Payment/Shipment mà không lộ dữ liệu chỉ bằng mã đơn. **2.2 Goals:** prove ownership; view status; optionally link after account login. **2.3 Preconditions:** Guest Order. **2.4 Flow:** enter Order reference + ownership proof → verify → display limited detail/tracking. **2.5 Alternative:** invalid/expired proof, no match, rate limit, linked Order now requires login. **2.6 Sections:** secure lookup form; neutral result; verified Order summary/tracking; link-order prompt. **2.7 Data:** Order code plus verification fields OPEN. **2.9:** verify, resend/access help if policy, link after login. **2.10:** code alone insufficient; no pre-verification disclosure. **2.11:** loading, invalid, expired, not found, verified, permission denied. **2.12:** secure generic feedback. **2.13:** proof-scoped. **2.14:** 008/email → 016 → limited 015/009. **2.15:** minimize PII.

#### 3. Reference / UX Layout Direction

Secure lookup gate followed by the same Order hierarchy in limited mode.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Tra cứu đơn Guest**, Web D2C. Trước xác minh chỉ có form Order reference + ownership proof chưa chốt; sau xác minh hiển thị limited Order/Payment/Shipment detail và tracking. States: invalid/expired proof, not found, rate limited, linked order/login required, verified. Không lộ PII trước verification. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 08 / US-ORD-02; EPIC 07 / US-PAY-03; EPIC 11 / US-SHIP-02; EPIC 01 / US-AUTH-06.

### D2C-017 — Tạo yêu cầu hủy/đổi/trả

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Return · **Actor:** Guest/Customer · **Entry:** eligible Order Detail/Guest lookup · **Next:** D2C-018.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Thu một yêu cầu có phạm vi rõ tới từng Order line, số lượng, lý do, phương án và bằng chứng. **2.2 Goals:** understand eligibility; submit full/partial request; upload evidence. **2.3 Preconditions:** ownership và Order/line trong điều kiện policy. **2.4 Flow:** choose lines/qty → reason/desired outcome → upload evidence → review → submit once → receive Case code. **2.5 Alternative:** expired/ineligible; qty exceeds returnable; duplicate request; media invalid/upload failed; simple cancellation handled directly by D2C-015.

**2.6 Sections:** policy/eligibility; selectable Order lines; reason/outcome; evidence upload; request review; consent/contact. **2.7 Data:** Order/line/SKU, purchased/eligible/requested qty, reason, desired return/exchange/refund where supported, media status, note. **2.9 Primary:** Submit. Secondary: save/retry upload [UX], cancel. **2.10:** ownership; no duplicate same quantity; media must be stored before counted; exchange scope OPEN. Sources: US-RET-01~03. **2.11:** loading, ineligible, validation, upload progress/fail, submitting, success, duplicate/error. **2.12:** submit confirmation; media removal. **2.13:** owner/proof. **2.14:** 015/016 → 017 → 018. **2.15:** eligibility/errors at line level.

#### 3. Reference / UX Layout Direction

Guided return wizard with line-item selection and evidence status; keep policy visible before submission.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Tạo yêu cầu hủy/đổi/trả**, Web D2C. Sections: eligibility/policy; Order-line selection and quantity; reason/desired resolution; image/video upload with real stored status; review; submit. States: ineligible/expired, duplicate, quantity invalid, upload progress/fail, validation, submitting/success/error. Flow Order Detail → Request → Case Detail. Do not promise exchange/refund timing before policy is confirmed. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 14 / US-RET-01~03; UC-RET-01, 02; FR-15.

### D2C-018 — Chi tiết yêu cầu hậu mãi

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Return/Refund · **Actor:** Guest/Customer · **Entry:** submission, Order Detail, notification · **Next:** D2C-019/020/015.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho khách theo dõi Case minh bạch qua review, decision, returned goods và Refund. **2.2 Goals:** view status/timeline; provide requested evidence; understand result/refund. **2.3 Preconditions:** ownership/proof. **2.4 Flow:** open Case → view current step + request snapshot → respond/add evidence if open → receive decision/refund state. **2.5 Alternative:** partial approval, rejection, pending/failed refund, return shipment/inspection not defined, Case closed/no more upload. **2.6 Sections:** Case header/status; requested lines/reason; public timeline; evidence; requested-information action; decision/reason; return logistics/inspection when applicable; refund status/amount/method. **2.7 Data:** Case/Order codes, statuses/times, line quantities, public messages, media, approved scope, Refund status. **2.9 Actions:** add requested info, open Order/support, appeal/reopen only if policy later supports. **2.10:** customer sees refunded only after valid result; internal notes hidden; closed Case blocks upload. **2.11:** reviewing, waiting customer/internal, approved/rejected/partial, returned, refund pending/success/fail, partial/error. **2.12:** upload feedback. **2.13:** owner/proof. **2.14:** 017 → 018 ↔ 020/015. **2.15:** timeline distinguishes Case, goods and money.

#### 3. Reference / UX Layout Direction

Case tracker with three separate layers: decision, returned goods, refund.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Chi tiết yêu cầu hậu mãi**, Web D2C. Header/current step; requested items/reason; public timeline; evidence; request-for-information action; decision/reason; returned-goods/inspection summary; Refund amount/method/status. States: reviewing, waiting customer, approved/rejected/partial, returned, refund pending/success/fail, closed/partial/error. Hide internal notes. Flow Request/notification → Case Detail → Order/Support. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 14 / US-RET-04~06; EPIC 07; EPIC 28; UC-RET-03, UC-PAY-04.

### D2C-019 — Gửi yêu cầu hỗ trợ

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Customer Service · **Actor:** Guest/Customer · **Entry:** footer/help/Order/Return · **Next:** D2C-020.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tạo Ticket chính thức, có mã theo dõi và liên kết đúng Order thuộc khách nếu có. **2.2 Goals:** describe issue; attach/link context; submit once. **2.3 Preconditions:** Guest contact verification policy OPEN; Customer logged in for own Order. **2.4 Flow:** choose topic/input description/contact → optionally link owned Order → attach if supported → submit → Ticket code. **2.5 Alternative:** invalid required data, unauthorized Order link, duplicate network retry, Guest verification failure. **2.6 Sections:** topic/description; contact; Order lookup/selection; attachment if configured; consent; summary. **2.7 Data:** topic, subject [UX], description, contact, Order reference, attachment. **2.9:** submit, remove attachment, open related Order. **2.10:** idempotent submission; cannot link another owner’s Order. **2.11:** form, validation, upload, submitting, success/error. **2.12:** confirmation with Ticket code. **2.13:** public/owner. **2.14:** 015/018/help → 019 → 020. **2.15:** preserve draft on recoverable error [UX].

#### 3. Reference / UX Layout Direction

Support intake form prioritizing description and optional transaction context.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Gửi yêu cầu hỗ trợ**, Web D2C. Sections: topic; description; contact; optional owned Order; attachment if configured; review/submit. States: validation, unauthorized Order, upload progress/fail, duplicate-safe submitting, success with Ticket code, error. Guest follow-up verification is open. Flow Help/Order/Return → Support Form → Ticket Detail. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 16 / US-CS-01; FR-15; UC-CS-01.

### D2C-020 — Chi tiết Ticket

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Customer Service · **Actor:** Guest/Customer · **Entry:** Ticket success/account/notification · **Next:** related D2C-015/018.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho khách xem trạng thái và hội thoại công khai, bổ sung thông tin mà không lộ Internal Note. **2.2 Goals:** follow progress; reply/add evidence; open related transaction. **2.3 Preconditions:** ownership/proof. **2.4 Flow:** open → review status/timeline → reply if allowed → sent status appears. **2.5 Alternative:** outbound message failed, awaiting customer, resolved/closed, reopen policy OPEN. **2.6 Sections:** Ticket header/topic/status; SLA expectation if customer-facing policy; public conversation timeline; composer/attachments; related Order/Case. **2.7 Data:** Ticket code, status, messages with sender/channel/time/delivery state, attachments, related refs. **2.9:** reply, retry own failed upload, open related; reopen only if documented later. **2.10:** no Internal Note; ownership. **2.11:** loading, waiting, sending, send failed, resolved/closed, partial/error. **2.12:** attachment feedback. **2.13:** owner/proof. **2.14:** 019/notification → 020 ↔ Order/Case. **2.15:** distinguish system/agent/customer messages and failed delivery.

#### 3. Reference / UX Layout Direction

Customer support portal with chronological public timeline and persistent case context.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Chi tiết Ticket**, Web D2C. Header code/topic/status; public conversation timeline distinguishing customer/agent/system/channel/delivery; reply and attachment; related Order/Return. States: waiting customer/internal, sending/send failed, resolved/closed, loading/partial/error. Never show Internal Notes. Flow Support/notification → Ticket → related transaction. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 16 / US-CS-01, 03; EPIC 28; FR-15; UC-CS-02.

### D2C-021 — Viết/chỉnh sửa đánh giá

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Review · **Actor:** Customer · **Entry:** eligible Order Item in D2C-015/notification · **Next:** D2C-004.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Thu rating/review/media cho đúng Order Item đã mua và ngăn review trùng. **2.2 Goals:** review eligible SKU, attach images, edit within window. **2.3 Preconditions:** authenticated owner; eligibility after purchase state still OPEN. **2.4 Flow:** choose eligible item → rating 1–5 → content/media → preview → submit. **2.5 Alternative:** ineligible/refunded/expired edit window, duplicate, invalid rating/media, upload interruption, moderated media. **2.6 Sections:** eligible item context; rating; text; image uploader; preview/policy; submit. **2.7 Data:** Order Item/Product/SKU, rating, content, media, Verified indicator assigned by system. **2.9:** submit/edit/remove image. **2.10:** rating 1–5; Verified cannot be self-declared; one-review uniqueness needs final rule; edit window OPEN. **2.11:** draft/loading/upload/validation/submitted/edit-locked/moderated/error. **2.12:** leave-draft confirm [UX]. **2.13:** owner/eligible. **2.14:** D2C-015 → 021 → D2C-004. **2.15:** upload progress is not stored-success.

#### 3. Reference / UX Layout Direction

Single-purpose form anchored to an immutable purchase context.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Viết/chỉnh sửa đánh giá**, Web D2C. Sections: eligible Order Item/SKU; 1–5 rating; review text; image upload progress/result; preview/policy; submit. States: ineligible, duplicate, invalid rating/media, upload fail, submitted, edit window closed, moderated. Verified is system-assigned. Dùng Master Design System; do not invent eligibility/edit duration. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 15 / US-REV-01~03; BR-REVIEW-01, 02, 04, 06; FR-16; UC-REV-01, 02.

### D2C-022 — Khuyến mãi & combo

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Promotion · **Actor:** Guest/Customer · **Entry:** Home/PDP · **Next:** D2C-004/005.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho khách hiểu chương trình, thời hạn, phạm vi và điều kiện trước khi mua. **2.2 Goals:** view active/upcoming/ended; inspect combo; add eligible offer. **2.3 Preconditions:** promotion public. **2.4 Flow:** browse → open details → select product/combo → PDP/cart. **2.5 Alternative:** no eligible program, login/coupon required, ended while viewing, combo component unavailable. **2.6 Sections:** status tabs; offer cards; details/terms; combo components; related products. **2.7 Data:** name, type, validity, condition, scope, coupon/login/channel requirement, components/qty/benefit/availability. **2.8 Filter/tab:** active/upcoming/ended; type only if supported. **2.9:** view terms, copy coupon [UX], open/add combo when sellable. **2.10:** only active/public applicable; components each require stock; historical Order unaffected. **2.11:** loading, empty, ended, not eligible, component out of stock, error. **2.12:** terms drawer. **2.13:** public; individualized eligibility may require login. **2.14:** Home/PDP → 022 → PDP/Cart. **2.15:** condition not hidden behind visual badge.

#### 3. Reference / UX Layout Direction

Promotion discovery with explicit eligibility and component transparency.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Khuyến mãi & combo**, Web D2C. Status tabs active/upcoming/ended; offer cards; terms; coupon/login/channel condition; combo component SKU/qty/benefit/availability; related products. States: empty, not eligible, expired while viewing, component unavailable, loading/error. Actions: view terms, open Product, add eligible combo. Dùng Master Design System; do not invent stacking rules. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 17 / US-PROMO-03, 04; FR-17; UC-PROMO-02.

### D2C-023 — Ví điểm & lịch sử

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Loyalty · **Actor:** Customer · **Entry:** Account/Checkout · **Next:** D2C-006/014/015.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Minh bạch số dư và mọi giao dịch cộng/giữ/trừ/đảo điểm. **2.2 Goals:** view balances/history; understand expiry/source; use points in Checkout. **2.3 Preconditions:** authenticated; Loyalty account behavior OPEN. **2.4 Flow:** view balance buckets → inspect history → go checkout/related Order. **2.5 Alternative:** no account/history, pending/held/expired/reversed, delayed Order event. **2.6 Sections:** balance summary (available/pending/held/expiring); expiry notice; transaction history; usage guidance. **2.7 Columns:** time, type, points +/-/held/released, source Order/refund, status, expiry. **2.8 Filter:** type/status/time [UX]. **2.9:** open source Order, go shopping/use points. **2.10:** no earn merely on Order creation/payment unless milestone chosen; duplicate event no duplicate points; cancel/return reverses by policy. **2.11:** loading, empty, partial/stale, error. **2.12:** none. **2.13:** owner. **2.14:** Account → 023 ↔ Checkout/Order. **2.15:** signed quantities and balance buckets cannot be conflated.

#### 3. Reference / UX Layout Direction

Wallet ledger pattern emphasizing available versus non-spendable points.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Ví điểm & lịch sử**, Web D2C. Balance buckets: available, pending, held, expiring; expiry notice; ledger columns time/type/points/source/status/expiry; links to source Order. States: no account/history, pending/held/reversed/expired, stale/partial/error. Actions: open Order, go to Checkout/shop. Dùng Master Design System; earning milestone/rate/expiry remain open. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 17 / US-LOY-01, 02; BR-REFUND-03; FR-18; UC-LOY-01, 02.

### D2C-024 — Yêu cầu báo giá sỉ

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** B2B · **Actor:** B2B Customer · **Entry:** B2B landing/contact/PDP · **Next:** D2C-025.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Thu nhu cầu mua sỉ có nhiều dòng, giao/tùy biến/logo đủ để Sales thẩm định. **2.2 Goals:** build request; upload logo; submit once; get tracking code. **2.3 Preconditions:** B2B identity/company verification policy OPEN. **2.4 Flow:** business/contact → lines SKU/qty → delivery/customization → logo upload → review → submit. **2.5 Alternative:** invalid SKU/qty, duplicate, invalid/interrupted file, unauthorized company. **2.6 Sections:** company/requester; line items; delivery timing/location; packaging customization; logo version/upload; notes; review. **2.7 Fields:** requested SKU/product, qty, requested delivery, customization, file status; MOQ/wholesale threshold OPEN. **2.9:** add/remove line, upload/replace before issue, submit. **2.10:** idempotent; only own company request; file security. **2.11:** draft, validation, upload, submitted/error. **2.12:** file replace confirm. **2.13:** B2B Customer scope. **2.14:** Product/B2B entry → 024 → 025. **2.15:** multi-line table remains usable on narrow screens.

#### 3. Reference / UX Layout Direction

Multi-line quote request builder with clear upload version and review step.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Yêu cầu báo giá sỉ**, Web D2C/B2B. Sections: company/requester; product/SKU quantity lines; delivery; packaging customization; logo upload/version; notes; review/submit. States: invalid line/quantity, upload progress/fail, duplicate-safe submission, success code, permission error. MOQ/verification/required billing fields remain open. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 18 / US-B2B-01, 02; FR-19; UC-B2B-01.

### D2C-025 — Chi tiết báo giá & chấp nhận

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** B2B · **Actor:** B2B Customer · **Entry:** request list/notification/link · **Next:** D2C-007/015 when converted.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Hiển thị đúng Quote Version/điều khoản/hiệu lực và ghi nhận acceptance một lần trước khi tạo Order. **2.2 Goals:** inspect history/current version; provide invoice company snapshot; accept/reject/respond. **2.3 Preconditions:** company/request ownership; issued Quote. **2.4 Flow:** open current version → review lines/pricing/customization/shipping/tax/expiry → choose billing profile/edit allowed fields → accept → revalidate sellability → create Order. **2.5 Alternative:** superseded/expired, SKU/stock changed, duplicate accept, Provider invoice failure, unauthorized representative. **2.6 Sections:** request status/timeline; current version banner; lines and financial breakdown; customization/file; terms/expiry; billing profile; messages; acceptance. **2.7 Data:** version, unit price/discount/tax/shipping/total as issued, terms, expiry, company snapshot, Order link. **2.9:** accept, decline/request clarification, select billing profile, open Order. **2.10:** acceptance binds exact version; not equivalent to payment; historical versions retained. **2.11:** draft/waiting/issued/superseded/expired/accepted/conversion failed/success. **2.12:** acceptance confirmation names version/total. **2.13:** authorized company representative only. **2.14:** 024 → 025 → Order/Payment. **2.15:** version/status fixed near top.

#### 3. Reference / UX Layout Direction

Document-style quote viewer plus controlled transaction action.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Chi tiết báo giá & chấp nhận**, Web D2C/B2B. Header status/version/expiry; line pricing breakdown; customization/logo; delivery/payment terms; company invoice snapshot; history/messages; accept action. States: superseded, expired, stock/sellability changed, accepted, converting, duplicate-safe success/failure. Acceptance must name exact version and is not payment. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 18 / US-B2B-04, 05; FR-19; UC-B2B-03.

### D2C-026 — Trang truy xuất QR

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** OCOP Traceability · **Actor:** Guest/Customer · **Entry:** scanned QR/PDP · **Next:** D2C-004/019.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho người cầm sản phẩm kiểm tra hồ sơ nguồn gốc/chứng nhận công khai đúng Product/SKU/Batch và trạng thái xác minh. **2.2 Goals:** identify scope; view region/producer/process; verify certificates; understand incomplete/suspended data. **2.3 Preconditions:** QR/reference exists and public profile. **2.4 Flow:** resolve code → show identity/scope → origin/process → certificates → sources/media → related Product. **2.5 Alternative:** invalid QR, wrong Batch, suspended profile, certificate expired/out of scope, provider mismatch/unavailable, media missing. **2.6 Sections:** verification status/time; Product/SKU/Batch/NSX/HSD where applicable; ingredient-origin groups; producer/process timeline; certificates with issuer/scope/validity/provider result; public media/docs; support. **2.7 Data:** level label, revision/status, source/validity, no internal/private supplier fields. **2.9:** view certificate/media, open Product/support. **2.10:** only approved public revision; expired/suspended clearly marked; do not rewrite history after source changes. **2.11:** verified, partial/unverified, suspended, invalid, provider unavailable, media unavailable. **2.12:** document viewer. **2.13:** public. **2.14:** QR/PDP → 026 → Product/Support. **2.15:** mobile-first scanning context; evidence hierarchy over marketing copy.

#### 3. Reference / UX Layout Direction

Trust/traceability page organized identity → origin → process → certificate, with explicit verification state.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Trang truy xuất QR**, Web D2C/OCOP, ưu tiên mobile scanning context. Sections: verification status/time and scope level; Product/SKU/Batch/NSX/HSD as applicable; ingredient regions/producers; process; certificates with code/issuer/scope/validity/provider result; public media/docs; support/Product links. States: valid, partial/unverified, suspended, invalid/wrong Batch, provider/media unavailable. Dùng Master Design System; never expose internal supplier PII. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 19 / US-OCOP-01~03; BR-PROD-04; FR-20; UC-OCOP-01; diagram 03.

### D2C-027 — Trung tâm câu chuyện & bài viết

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Content · **Actor:** Guest/Customer · **Entry:** Home/navigation/search engine · **Next:** D2C-028/004.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Duyệt nội dung đã xuất bản về thương hiệu/văn hóa Huế và liên kết sản phẩm đúng nguồn. **2.2 Goals:** browse/find public article; open detail/product. **2.3 Preconditions:** published revision. **2.4 Flow:** open hub → browse category/tag if defined → open article. **2.5 Alternative:** empty/no result, hidden/expired content, partial media. **2.6 Sections:** hub intro; featured articles; article list; content taxonomy only after defined; related products. **2.7 Data:** title, summary, cover, publication time, category/tag if configured, related Product. **2.8 Search/filter:** keyword/category only when taxonomy confirmed; sort latest [UX]. **2.9:** open article/Product. **2.10:** only published; hidden removed from public index. **2.11:** loading, empty, no result, media error. **2.13:** public. **2.14:** D2C-001 → 027 → 028/004. **2.15:** accessible alternative text.

#### 3. Reference / UX Layout Direction

Editorial hub with clear distinction between story content and commerce cards.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Trung tâm câu chuyện & bài viết**, Web D2C/Content. Intro; featured published content; article list with title/summary/cover/date; optional confirmed taxonomy/search; related Products. States: loading, empty/no result, hidden content removed, media failure. Flow Home → Content Hub → Article/Product. Dùng Master Design System; do not invent categories. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 03 / US-DISC-05; EPIC 20 / US-CONTENT-01, 02; FR-21; UC-DISC-04.

### D2C-028 — Chi tiết bài viết

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Content/SEO · **Actor:** Guest/Customer · **Entry:** D2C-027/search engine · **Next:** related D2C-004/026.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Trình bày đúng Published Revision, media và dữ kiện Product/OCOP đã duyệt. **2.2 Goals:** read; consume media; follow related evidence/product. **2.3 Preconditions:** public revision/URL. **2.4 Flow:** URL resolve/canonical → render article → open related links. **2.5 Alternative:** old redirected URL, hidden/unpublished, media missing, linked source changed. **2.6 Sections:** breadcrumb; title/publication info; hero; body/media with alt/caption; source/related Product/OCOP; related articles. **2.7 Data:** published content and SEO metadata; no Draft label leakage. **2.9:** open related Product/traceability, share [UX]. **2.10:** approved revision only; old URL redirect history; private media protected. **2.11:** loading, not found/redirect, hidden, partial media, error. **2.13:** public. **2.14:** 027 → 028 → 004/026. **2.15:** long-form reading and alternative text.

#### 3. Reference / UX Layout Direction

Long-form editorial page with evidence/product links secondary to narrative.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Chi tiết bài viết**, Web D2C. Breadcrumb; title/publication metadata; hero; structured body; media with alt/caption; Product/OCOP sources; related articles. States: loading, redirected old URL, hidden/not found, partial media/error. Only Published Revision; no Draft/private media. Flow Content Hub/search → Article → Product/Traceability. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 20 / US-CONTENT-01~06; FR-21, 22; UC-CONTENT-01, 02.

### D2C-029 — Trung tâm & tùy chọn thông báo

#### 1. Web Module & Screen

**Web Type:** D2C · **Module:** Notification · **Actor:** Customer · **Entry:** header/account · **Next:** deep-linked authorized screen.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Xem in-app notifications và quản lý preference/consent theo classification mà không chặn thông báo bắt buộc. **2.2 Goals:** read/unread; open deep link; configure permitted channels/quiet hours. **2.3 Preconditions:** authenticated. **2.4 Flow:** open center → scan notification → open target with permission check; preferences tab saves changes. **2.5 Alternative:** expired/deleted target, provider/contact invalid, suppressed marketing, push token unavailable, partial delivery. **2.6 Sections:** unread/all list; notification detail; preference categories (transactional/security/marketing where policy allows); contact/endpoint status; quiet hours/frequency info when supported. **2.7 Data:** type, source, time, read state, delivery/channel status; consent record. **2.8 Filter:** unread/type/time [UX]. **2.9:** mark read, open, save preference, opt out marketing. **2.10:** deep link reauthorizes; marketing opt-out does not block mandatory transactional messages. **2.11:** empty/loading, unread, suppressed/delayed, expired link, permission denied, save error. **2.12:** toast on save. **2.13:** owner. **2.14:** 029 → Order/Ticket/Return/Promotion. **2.15:** notification content minimizes sensitive data.

#### 3. Reference / UX Layout Direction

Two-mode center: notification feed and preference settings, sharing classification semantics.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Trung tâm & tùy chọn thông báo**, Web D2C. Tabs: notifications (unread/all, type, time, deep link) và preferences (classification/channel/consent/contact status/quiet hours nếu supported). States: empty, loading, delayed/suppressed, expired link, permission denied, invalid endpoint, save error. Marketing opt-out không chặn transactional bắt buộc; deep links recheck quyền. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 28 / US-NOTI-01~03; EPIC 08 / US-ORD-03; FR-30; UC-NOTI-01.

## 6. Web Admin Screens

Quy ước chung cho mọi Admin screen: navigation theo module và current scope; list/table có loading, empty, no-result, partial/stale, error và permission-denied khi relevant; mọi mutation nhạy cảm hiển thị actor scope, impact, reason/confirmation và không dựa vào việc ẩn nút để thực thi quyền. Các screen dưới đây vẫn nêu thêm state/action riêng.

### ADM-001 — Product & SKU Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager · **Entry:** Catalog navigation · **Next:** ADM-002/008/035/037.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tạo/cập nhật Product mẹ, Variant/SKU và trạng thái bán trong một nguồn catalog. **2.2 Goals:** find Product; edit fields; add/update SKU; publish/pause/discontinue. **2.3 Preconditions:** catalog permission. **2.4 Flow:** search/list → select/create → edit Product/SKU → validate → save/status action. **2.5 Alternative:** duplicate/invalid SKU, missing required food info, concurrent edit, pause with carts/orders. **2.6 Sections:** product list; Product form/media reference; SKU table/editor; food information; status/history. **2.7 Columns:** Product, category, SKU count, public/sales status, updated time; SKU code, attributes, weight/package, price link, stock read-only, status. **2.8 Search/filter:** name/SKU; category/status. **2.9 Actions:** create/update SKU/Product, change status; no hard delete of used entity. **2.10 Rules:** price/stock per SKU; approved public data; historic Orders unchanged. Sources: US-PROD-01,02,04,06; BR-PROD-01~04. **2.11 States:** draft/validation/conflict/permission/save success. **2.12 Modal:** status-impact confirm. **2.13 Permissions:** Sales Manager; stock read-only. **2.14 Nav:** Catalog ↔ Pricing/Inventory/SEO/OCOP. **2.15:** dense master-detail with horizontal SKU table.

#### 3. Reference / UX Layout Direction

Shopify-style information architecture only as a master-detail pattern; no copied visual styling.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Product & SKU Workspace**, Web Admin. Product searchable list + detail editor; SKU table/editor; food info; status/history; links Pricing/Inventory/SEO/OCOP. Actions create/update/status change with validation and impact confirm. States duplicate SKU, missing required data, concurrent edit, draft/save/error/permission. Use existing Master Design System; no new SKU attributes or publish workflow. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 04 / US-PROD-01,02,04,06; FR-04,09; UC-PROD-01~03,05.

### ADM-002 — Giá theo SKU/kênh

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager · **Module:** Pricing · **Entry:** Catalog/Pricing · **Next:** ADM-001/019/023/034.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý giá từng SKU và override theo kênh mà không làm đổi snapshot Order cũ. **2.2 Goals:** view price matrix/effective period; create/update authorized price. **2.3 Preconditions:** SKU exists. **2.4 Flow:** select SKU/channel → enter price/effectivity → preview impact → save. **2.5 Alternative:** invalid/negative, overlapping/conflicting price, unauthorized channel, concurrent change. **2.6 Sections:** SKU search; base price; channel price table; effectivity/history; impact preview. **2.7 Columns:** SKU, channel, price, validity, status, last actor/time. **2.8:** search SKU; filter channel/status/effective date. **2.9:** create/update/end price. **2.10:** independent SKU price; Order snapshot immutable; precedence with promotion/B2B OPEN. **2.11:** no price, scheduled/active/expired, conflict/error. **2.12:** confirm current-price impact. **2.13:** authorized Sales Manager. **2.14:** Catalog ↔ Pricing; downstream Marketplace/POS/B2B. **2.15:** matrix can horizontally overflow/column-prioritize.

#### 3. Reference / UX Layout Direction

Pricing matrix + effective history, emphasizing channel and time.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Giá theo SKU/kênh**, Web Admin. SKU search; base price; channel/effectivity matrix; history; impact preview. Columns SKU/channel/price/from-to/status/actor/time. States missing price, scheduled/active/expired, overlap/conflict, permission/error. Actions create/update/end with confirmation. Do not invent precedence with coupon/B2B/Marketplace. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 04 / US-PROD-05,07; FR-04; UC-PROD-04; BR-AUDIT-01.

### ADM-003 — Customer Accounts

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** authorized customer-management staff · **Module:** Customer · **Entry:** Customer nav/Ticket · **Next:** ADM-005/030/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tra cứu Customer, xem hồ sơ/trạng thái và thay đổi trạng thái theo thẩm quyền mà không sửa lịch sử mua. **2.2 Goals:** search, inspect, status action. **2.3 Preconditions:** explicit permission. **2.4 Flow:** search → open detail → view profile/order summary/status → enter reason → update. **2.5 Alternative:** no result, restricted fields, concurrent status change, unauthorized transition. **2.6 Sections:** directory; identity/contact masked; status; address count; transaction/ticket links; status history. **2.7 Columns:** Customer identifier/name masked, contact, status, last activity [UX], orders count [UX]. **2.8:** search identifier/contact; filter status. **2.9:** view, change status; no editing Order. **2.10:** status matrix/effects OPEN; history retained. **2.11:** active/restricted, partial/permission/conflict. **2.12:** status confirm with reason/impact. **2.13:** scoped role. **2.14:** Customer → Order/Ticket/Audit. **2.15:** PII masking persists in exports/view.

#### 3. Reference / UX Layout Direction

Customer 360 summary with privileged status action separated from read-only transaction context.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Customer Accounts**, Web Admin. Searchable directory; detail with masked profile/contact, status, address summary, Order/Ticket links, status history. Action change status only with permission, reason and impact confirmation. States no result, restricted data, conflict, denied. Status taxonomy/effects remain open; never modify purchase history. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 02 / US-USER-05; NFR-04,05,08; use case UC-USER-04.

### ADM-004 — Employee Profiles

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** HR Manager · **Module:** HR · **Entry:** HR nav · **Next:** ADM-042/044.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý hồ sơ nhân sự tách khỏi tài khoản truy cập. **2.2 Goals:** create/update/search profile; link system access. **2.3 Preconditions:** HR authority. **2.4 Flow:** list → create/select → edit profile → validate/save. **2.5 Alternative:** missing/invalid, duplicate employee, restricted salary/contract fields, no Account. **2.6 Sections:** employee directory; employment/position/contract/salary fields per FR; contact; status; System Access reference. **2.7 Columns:** employee ref, name, position, employment status, account status. **2.8:** search; filter position/status. **2.9:** create/update; open account admin. **2.10:** profile is not credential/account; exact retention/view scopes OPEN. **2.11:** empty, validation, denied, save conflict. **2.12:** unsaved confirm. **2.13:** HR and field-level permissions OPEN. **2.14:** Employee Profile → Account/Access Review. **2.15:** sensitive HR fields grouped and masked.

#### 3. Reference / UX Layout Direction

HR directory + detail editor, with System Access shown as a linked but distinct object.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Employee Profiles**, Web Admin/HR. Directory; profile editor for documented salary/contract/position and employment data; System Access reference; status. States empty, validation, duplicate, restricted fields, no account, conflict/error. Actions create/update/open Account. Keep HR Profile separate from login access. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 02 / US-USER-03,04; FR-02; UC-USER-03.

### ADM-005 — Order Operations

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager/authorized Operations · **Module:** Order · **Entry:** Admin nav/alert · **Next:** ADM-006/007/013/016/026.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Một hàng đợi đa kênh thống nhất để tìm Order và ưu tiên công đoạn gần/quá SLA. **2.2 Goals:** search/filter, spot blocked/urgent, open detail. **2.3 Preconditions:** scoped order view. **2.4 Flow:** choose quick status/SLA tab → filter → inspect row → detail/action destination. **2.5 Alternative:** no result, stale status, out-of-scope channel, partial integration. **2.6 Sections:** KPI/quick tabs; search/filter; operational table; alert legend. **2.7 Columns:** Order ID/time/customer masked/items/source, Payment/Fulfillment/Shipping status, total, SLA/priority, assigned staff, sync issue, actions. **2.8 Search:** Order/source reference/customer permitted fields. **Filters:** time, source, Order/Payment/fulfillment/shipping/SLA; **Sort:** SLA/created. **2.9:** open; bulk actions only when requirement later defines—none now. **2.10:** only valid stage Orders; status source-separated. **2.11:** loading/empty/no result/partial/stale/denied. **2.13:** scoped view. **2.14:** queue → Order Detail → specialized workspace. **2.15:** wide operations table with horizontal overflow/column priority.

#### 3. Reference / UX Layout Direction

Information-dense data table with quick workflow tabs and SLA signals.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Order Operations**, Web Admin. KPI/quick tabs; search/filter; dense table columns Order/time/customer/items/source, separate Payment/Fulfillment/Shipping statuses, total, SLA, assignee, sync issue, actions. Filters time/source/status/SLA; sort SLA. States loading/empty/no result/partial/stale/permission. Row opens ADM-006. No invented bulk state changes. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 08 / US-ORD-05,06; FR-07; UC-ORD-05,06.

### ADM-006 — Order Detail

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager/authorized Operations · **Module:** Order · **Entry:** ADM-005 or linked entity · **Next:** ADM-007/013/016/026/028/030/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cung cấp snapshot và timeline thống nhất để thực hiện đúng transition, không sửa trực tiếp lịch sử. **2.2 Goals:** inspect full context; execute permitted state action; drill to source modules. **2.3 Preconditions:** permission/scope. **2.4 Flow:** load snapshot/current statuses → review eligibility → action with reason/confirm → timeline updates. **2.5 Alternative:** invalid transition, concurrent change, Payment/Shipment mismatch, late/out-of-order event, data partial. **2.6 Sections:** header/source refs; Order/Payment/Fulfillment/Shipping status; item snapshot; customer/address masked; financial breakdown; inventory reservation; packages/shipments; return/ticket; audit timeline. **2.7 Data:** IDs, timestamps, actor/source, before/after status, SLA, snapshot amounts. **2.9:** valid transition, cancel/escalate/deep link; no arbitrary edit. **2.10:** state machine/actor matrix; snapshot immutable; BR-ORDER-01~05. **2.11:** blocked/mismatch/concurrent/denied/partial/error. **2.12:** transition confirmation with impacts/reason. **2.13:** per action. **2.14:** master hub to specialized screens. **2.15:** master-detail; actions remain contextual.

#### 3. Reference / UX Layout Direction

Operational order detail with status lanes and entity timeline.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Order Detail**, Web Admin. Header/source refs; distinct Order/Payment/Fulfillment/Shipping statuses; item/customer/financial snapshot; reservation; packages; Return/Ticket links; timeline/audit. Only valid state actions, with reason/impact confirm. States invalid transition, mismatch, concurrent update, stale/partial/denied. Deep links to Payment/Packing/Shipping/Return/CS/Audit. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 08 / US-ORD-05,06; EPIC 07/11/14/16; BR-ORDER; FR-07; UC-ORD-05.

### ADM-007 — Payment Reconciliation

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager, Finance · **Module:** Payment · **Entry:** Order/Finance/alert · **Next:** ADM-006/028/048.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Đặt Payment cạnh Order để phát hiện khớp/thiếu/thừa/trùng/không rõ và xử lý theo quyền. **2.2 Goals:** search transaction; compare; resolve/assign. **2.3 Preconditions:** view financial permission. **2.4 Flow:** filter exceptions → open comparison → inspect attempts/transaction → choose allowed resolution + reason. **2.5 Alternative:** signature/source invalid, duplicate, mismatch, multiple success, late payment, no matching Order. **2.6 Sections:** exception queue; Payment–Order comparison; attempt timeline; provider/source data; resolution history. **2.7 Columns:** transaction/ref, Order, amount expected/actual/delta, method, Payment/Order status, source time, case status/owner. **2.8:** search IDs; filter method/status/delta/time. **2.9:** assign/resolve per authority; never arbitrary mark Paid. **2.10:** authoritative confirmation; idempotency; audit reason. **2.11:** matched/mismatch/unmatched/duplicate/pending/denied/provider unavailable. **2.12:** resolution confirmation. **2.13:** Sales view vs Finance resolve. **2.14:** Order ↔ Reconciliation ↔ Finance/Refund. **2.15:** expected/actual/delta visually adjacent.

#### 3. Reference / UX Layout Direction

Exception queue with side-by-side reconciliation detail.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Payment Reconciliation**, Web Admin. Exception table transaction/Order/expected/actual/delta/method/status/owner; detail compares Payment attempts, provider data and Order; resolution history. States matched, mismatch, unmatched, duplicate, late/pending, provider unavailable, denied. Actions assign/resolve only by authority with reason; no manual Paid shortcut. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 07 / US-PAY-04,05; UC-PAY-03; FR-06; BR-ORDER-03, BR-AUDIT-01.

### ADM-008 — Inventory Overview

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Warehouse Staff; view for Sales/Supply as permitted · **Module:** Inventory · **Entry:** Inventory nav · **Next:** ADM-009~012.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Hiển thị nhất quán on-hand/reserved/non-sellable/available theo SKU và drill-down Batch. **2.2 Goals:** search SKU; compare balances; inspect lot/movement. **2.3 Preconditions:** inventory view permission. **2.4 Flow:** filter/search → inspect quantities → open SKU/Batch/action. **2.5 Alternative:** expired physical stock but zero available, partial location data, concurrent reservation. **2.6 Sections:** balance summary; SKU table; filters; alert links. **2.7 Columns:** SKU/Product, on-hand, reserved, non-sellable, available, batch count, nearest HSD, updated time. **2.8:** search SKU/Product; filter availability/expiry/status/location only if supported. **2.9:** open Batch, receive/issue/adjust based permission. **2.10:** no direct balance edit; expired excluded; reservations cannot exceed eligible stock. **2.11:** empty/no result/stale/concurrent refresh/denied. **2.13:** Warehouse mutate, other roles view. **2.14:** Overview → Batch/Receipt/Movement/Expiry. **2.15:** numbers align; definition tooltips [UX].

#### 3. Reference / UX Layout Direction

Inventory operations table with balance components visible together.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Inventory Overview**, Web Admin. Summary + SKU table columns Product/SKU, on-hand, reserved, non-sellable, available, batch count, nearest HSD, freshness. Search Product/SKU; relevant filters. Actions drill to Batch/Receipt/Movement/Expiry by permission. States empty/no result/stale/partial/denied. No direct balance edit; expired stock not available. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 09 / US-INV-01; BR-BATCH-05, BR-ORDER-02; FR-08,09; UC-INV-01.

### ADM-009 — Batch Detail

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Warehouse Staff · **Module:** Inventory · **Entry:** ADM-008/012/010 · **Next:** ADM-011/035/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Truy vết một lô qua identity, dates, source, balances và immutable movement history. **2.2 Goals:** create/view/update allowed metadata; inspect movements/OCOP. **2.3 Preconditions:** SKU exists; create permission. **2.4 Flow:** create/select Batch → validate code/NSX/HSD/source → save → view balance/history. **2.5 Alternative:** duplicate/ambiguous code, invalid dates, expired, correction after movement requiring controlled process. **2.6 Sections:** identity/SKU/source; NSX/HSD/expiry status; balance; locations if supported; movement timeline; traceability link. **2.7 Data:** code, SKU, NSX, HSD, qty received/remaining, source, status, refs. **2.9:** create, permitted correction, movement/traceability drilldown. **2.10:** required fields; no erase history; scope uniqueness OPEN. **2.11:** active/near-expiry/expired/quarantined [only if policy], invalid/conflict. **2.12:** correction impact confirm. **2.13:** Warehouse; traceability manager linked. **2.14:** Inventory ↔ Batch ↔ OCOP/Audit. **2.15:** dates/status prominent.

#### 3. Reference / UX Layout Direction

Lot dossier with immutable ledger below operational facts.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Batch Detail**, Web Admin. Identity Product/SKU/code/source; NSX/HSD/status; balance; location if supported; movement ledger; OCOP/Audit links. States active/near-expiry/expired, invalid date, duplicate/ambiguous code, correction conflict, denied. Actions create/permitted correction only; never erase history. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 09 / US-INV-04,05; BR-BATCH-01~05; FR-08,09; UC-INV-02.

### ADM-010 — Stock Receipt

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Warehouse Staff · **Module:** Inventory/Receiving · **Entry:** Inventory or PO · **Next:** ADM-009/055.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Ghi nhận thực nhận dương vào đúng SKU/Batch và lưu reference/chênh lệch. **2.2 Goals:** select source/PO; enter actual lines; validate Batch/HSD; post receipt once. **2.3 Preconditions:** SKU exists; authorized receipt; PO optional until GĐ3. **2.4 Flow:** choose source → expected lines if PO → actual qty/Batch/date/quality → review variance → confirm. **2.5 Alternative:** invalid SKU/qty/Batch, duplicate receipt, partial/over/under/damaged. **2.6 Sections:** source/reference; expected-vs-actual table; Batch data; quality/evidence; variance summary; confirmation. **2.7 Columns:** item/SKU, expected, received, variance, batch, NSX/HSD, disposition/status. **2.9:** add line, save draft [UX], confirm. **2.10:** qty positive; required Batch fields; idempotent reference; movement created only on valid completion. **2.11:** draft/validation/partial/variance/duplicate/posting/success. **2.12:** post confirmation. **2.13:** Warehouse. **2.14:** Inventory/PO → Receipt → Batch/Variance. **2.15:** scan device details outside scope.

#### 3. Reference / UX Layout Direction

Receiving worksheet centered on expected versus actual.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Stock Receipt**, Web Admin. Source/PO; expected-vs-actual item table; Batch/NSX/HSD/source; quality/evidence; variance; review/confirm. States draft, invalid SKU/qty/date, partial/over/under/damaged, duplicate reference, posting/success/error. Confirm creates controlled movement. Dùng Master Design System. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 09 / US-INV-02; EPIC 27 / US-PROC-04; UC-INV-02, UC-PROC-03; FR-08,29.

### ADM-011 — Stock Issue & Adjustment

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Warehouse Staff/approver when defined · **Module:** Inventory · **Entry:** ADM-008/009 · **Next:** ADM-045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Ghi xuất hoặc adjustment có loại nghiệp vụ/reference/reason; không sửa số dư trực tiếp. **2.2 Goals:** issue for Order/non-Order authorized reason; record damage/loss/count difference. **2.3 Preconditions:** sufficient valid Batch qty and permission. **2.4 Flow:** choose type/reference/SKU-Batch → qty/reason/evidence → preview balance → submit/approval if threshold. **2.5 Alternative:** exceeds balance/negative, expired/not eligible, duplicate, approval required/denied. **2.6 Sections:** movement type/source; line table; reason/evidence; before/change/after; approval status; history. **2.7 Columns:** SKU/Batch, available, qty, expected after, reference. **2.9:** submit, approve only if role defined, reverse via new entry—not delete. **2.10:** no negative; idempotent; audit sensitive adjustment. **2.11:** draft/invalid/pending approval/approved/rejected/posted/conflict. **2.12:** impact confirmation. **2.13:** thresholds/approver OPEN. **2.14:** Inventory → Movement → Audit. **2.15:** sign and resulting balance explicit.

#### 3. Reference / UX Layout Direction

Controlled journal-entry form with before/change/after and evidence.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Stock Issue & Adjustment**, Web Admin. Movement type/reference; SKU/Batch lines; available qty; entered change; before/after; reason/evidence; approval status/history. States insufficient/negative, expired batch, duplicate, pending/rejected/posted, conflict. Actions submit/approve only by permission; corrections use reversal, no delete. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 09 / US-INV-03,09; BR-AUDIT-01; FR-08; UC-INV-05.

### ADM-012 — Expiry, FEFO & Consumption

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Warehouse, Supply Manager; Sales view · **Module:** Inventory · **Entry:** Inventory/alert · **Next:** ADM-009/031/051.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Nhận biết Batch cận/hết hạn, xem tiêu thụ và đề xuất thứ tự FEFO có giải thích. **2.2 Goals:** filter risk; inspect date/qty; follow FEFO; record controlled exception. **2.3 Preconditions:** Batch dates. **2.4 Flow:** open risk list → filter threshold/status → inspect Batch → use proposal in allocation or coordinate action. **2.5 Alternative:** expired blocked, insufficient data, alert duplicate/closed, FEFO exception, one Order needs multiple Batch. **2.6 Sections:** risk summary; expiry table; consumption trends [GĐ2]; FEFO recommendation; coordination status. **2.7 Columns:** SKU/Batch, qty, HSD/days remaining, status, consumption, suggested priority, alert state/updated. **2.8:** SKU/status/risk window/actor scope; sort nearest HSD. **2.9:** open Batch, acknowledge/coordinate, choose exception with reason if authorized. **2.10:** expired never sell; FEFO conditional; alert dedup. **2.11:** near/expired/no data/stale/partial/denied. **2.12:** exception warning/reason. **2.13:** role-specific view/actions. **2.14:** Alert → Expiry → Batch/CS/DSS. **2.15:** date and qty explicit, not color-only.

#### 3. Reference / UX Layout Direction

Risk queue + FEFO decision support, not an autonomous allocation screen.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Expiry, FEFO & Consumption**, Web Admin. Risk summary; table SKU/Batch/qty/HSD/days/status/consumption/priority/alert; FEFO recommendation and reason; coordination state. Filters risk/status/SKU; sort nearest expiry. States near/expired, insufficient data, stale/partial, duplicate alert, denied. Expired is blocked; exception requires permission/reason. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 09 / US-INV-06~08,10; BR-BATCH-03~06; UC-INV-03,04; FR-09.

### ADM-013 — Packing Queue

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Packing Staff · **Module:** Packing · **Entry:** nav/alert · **Next:** ADM-014.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Chỉ hiển thị Task đủ điều kiện và ưu tiên gần/quá SLA. **2.2 Goals:** filter; see blockers; claim/open assigned task. **2.3 Preconditions:** permission; eligible Order. **2.4 Flow:** quick status/SLA → select task → open/claim per assignment model OPEN. **2.5 Alternative:** blocked/unready, reassigned, concurrent claim, empty. **2.6 Sections:** queue summary/tabs; filter; task table; blocker legend. **2.7 Columns:** Order, source, item lines, assigned staff, deadline/SLA, priority, state/blocker. **2.8:** search Order; filter status/assignee/source/SLA; sort deadline. **2.9:** open/claim only if allowed. **2.10:** unready not shown as ready; priority recalculates. **2.11:** loading/empty/no result/blocked/stale/denied. **2.12:** claim conflict feedback. **2.13:** Packing Staff scope. **2.14:** Order → Queue → Task. **2.15:** dense queue with deadline always visible.

#### 3. Reference / UX Layout Direction

Operational queue pattern with quick status tabs and explicit blockers.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Packing Queue**, Web Admin. Summary/status tabs; filter; table Order/source/items/assignee/deadline/SLA/priority/state/blocker. Search Order; filters status/assignee/source/SLA; sort deadline. States empty/no result/blocked/stale/concurrent claim/denied. Row opens Packing Task. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 10 / US-PACK-01,07; EPIC 08 / US-ORD-06; FR-10; UC-PACK-01.

### ADM-014 — Packing Task

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** assigned Packing Staff · **Module:** Packing · **Entry:** ADM-013 · **Next:** ADM-016; ADM-015.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Lấy đúng SKU/Batch/số lượng, hoàn thành checklist/video/gift artifact và chỉ rồi mới đánh dấu Packed. **2.2 Goals:** verify pick; complete checks; record evidence; finish. **2.3 Preconditions:** assigned, task ready. **2.4 Flow:** review snapshot → confirm actual SKU/Batch → checklist → capture/upload required media → preview gift/packing artifact → complete. **2.5 Alternative:** mismatch/invalid Batch, incomplete checklist, upload fail, reassignment, duplicate complete, reopen request. **2.6 Sections:** Order/package/gift header; pick list; Batch allocation; checklist; media status; artifact preview; completion conditions. **2.7 Columns:** Product/SKU/ordered-picked qty/Batch/HSD/result. **2.9:** mark check pass/fail, record issue, upload video, complete. **2.10:** assigned staff only; snapshot not current catalog; required evidence stored before complete; idempotent. **2.11:** ready/in-progress/blocked/uploading/failed/incomplete/completed/conflict. **2.12:** finish confirmation lists missing/impact. **2.13:** assignee; evidence view separate. **2.14:** Queue → Task → Shipment. **2.15:** pick data adjacent checklist.

#### 3. Reference / UX Layout Direction

Task workspace, not a long generic form; pick list and checks stay in the same context.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Packing Task**, Web Admin. Header Order/package/source/gift; item pick table with snapshot SKU/qty/Batch/HSD; checklist; issue handling; video upload real status; gift artifact preview; completion conditions. States blocked, mismatch, checklist incomplete, upload fail, reassigned, duplicate-safe complete, conflict. Only assigned staff completes. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 10 / US-PACK-02~05,08; EPIC 26; BR-PACK-01~04; UC-PACK-01~03; FR-10,11,28.

### ADM-015 — Packing Evidence Viewer

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** authorized CSKH/Sales Manager/System Admin only when explicitly granted · **Module:** Packing Investigation · **Entry:** Return/Ticket/Order · **Next:** ADM-026/030/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Xác minh media đúng Order/Task/Package mà không cho sửa bằng chứng gốc hay lộ storage URL. **2.2 Goals:** select evidence; view metadata/video; relate to complaint. **2.3 Preconditions:** media permission and business context. **2.4 Flow:** open from Case/Order → choose package/video → authorized stream → return to investigation. **2.5 Alternative:** no media/not ready, wrong link, expired access, denied, playback error. **2.6 Sections:** context banner; package/task list; media player; capture/store metadata; access history/link. **2.7 Data:** Order/Package/Task, media ID, created actor/time, state, no raw path. **2.9:** play, open Case/Audit; download only if permission later defined. **2.10:** no modify; access is audited; scoped media security. **2.11:** not available/processing/failed/denied/expired. **2.13:** explicit media permission. **2.14:** Return/Ticket → Evidence → back. **2.15:** PII warning.

#### 3. Reference / UX Layout Direction

Evidence investigation viewer with read-only provenance.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Packing Evidence Viewer**, Web Admin. Case/Order context; package/task selector; secure player; media provenance actor/time/state; related Case/Audit links. States none, processing, failed, wrong link, expired access, denied/playback error. No raw storage URL, no edit; access audited. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 10 / US-PACK-06; EPIC 14 / US-RET-07; BR-PACK-04,05; NFR-09; UC-PACK-04.

### ADM-016 — Shipment Operations

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** authorized Packing/Warehouse/Sales · **Module:** Shipping · **Entry:** packed Order/Shipping nav · **Next:** ADM-006/017/018.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tạo vận đơn đúng recipient/package/COD, bàn giao và xử lý sự kiện/provider exception. **2.2 Goals:** create label/tracking; handoff; monitor sync. **2.3 Preconditions:** Order/package eligible. **2.4 Flow:** choose package → verify recipient/service/weight/COD/fee → create once → print label → handoff → monitor tracking. **2.5 Alternative:** unready, provider reject/timeout, duplicate request, unknown/out-of-order status, cancel shipment OPEN. **2.6 Sections:** ready queue; creation form; shipment table; tracking timeline; exception queue. **2.7 Columns:** Shipment/Order/Package/provider/service/tracking/COD/fee/state/last sync/error. **2.8:** search ref; filter provider/status/error/time. **2.9:** create, print/download label, confirm handoff, retry safe sync. **2.10:** idempotent creation; personal data limited; status mapping controlled. **2.11:** creating/created/ready/handoff/in transit/delivered/failed/provider unavailable/conflict. **2.12:** create/handoff confirm. **2.13:** action-specific. **2.14:** Packing → Shipping → Order/Delivery. **2.15:** distinguish label-created from handed-over/shipped.

#### 3. Reference / UX Layout Direction

Shipment master-detail plus exception monitor.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Shipment Operations**, Web Admin. Ready queue; creation form Order/Package/recipient/service/weight-dimensions/COD/fee; shipment table; timeline; exception queue. States created vs ready vs handed-over vs in-transit vs delivered/failed, provider timeout/reject, duplicate/out-of-order. Actions create once, print label, handoff, safe retry. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 11 / US-SHIP-01,06; FR-12; UC-SHIP-01,03; BR-AUDIT-01.

### ADM-017 — Delivery Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Delivery Staff · **Module:** Delivery · **Entry:** assignment/notification · **Next:** ADM-016/006.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cho Delivery Staff xem đúng chuyến được phân công và cập nhật progress hợp lệ. **2.2 Goals:** route/task list; contact recipient as allowed; update success/failure/COD result. **2.3 Preconditions:** assignment active. **2.4 Flow:** view assigned shipments → sort tasks → open → update allowed next status with evidence/reason → sync. **2.5 Alternative:** reassigned, invalid transition, offline/network error, duplicate update, delivery failure. **2.6 Sections:** assigned list; task detail; minimal recipient/contact; timeline; status update/COD capture. **2.7 Columns:** Shipment, sequence [UX], recipient/address masked, delivery state, COD due, assignment. **2.8:** filter status; sort planned order [UX]. **2.9:** allowed status update; failure reason; confirm delivery. **2.10:** cannot view other assignments; COD not Paid until valid confirmation. **2.11:** assigned/reassigned/offline/pending sync/success/failure/conflict. **2.12:** completion/failure confirm. **2.13:** assignment scope. **2.14:** Shipment → Delivery update → Order/Payment. **2.15:** task-first, PII minimal.

#### 3. Reference / UX Layout Direction

Mobile-friendly route/task pattern despite being Web Admin; operational actions dominate.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Delivery Workspace**, Web Admin responsive. Assigned shipment list; minimal recipient/address/contact; detail timeline; COD due; valid next-state actions and failure reason. States reassigned, offline/pending sync, duplicate, invalid transition, success/failure. Only own assignment; minimize PII. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 11 / US-SHIP-03,04; US-PAY-02; UC-SHIP-02,04.

### ADM-018 — Delivery Performance

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager · **Module:** Shipping Analytics · **Entry:** Shipping nav · **Next:** ADM-016/047.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** So sánh chất lượng giao theo kỳ/provider/khu vực mà không coi no-data là 0%. **2.2 Goals:** inspect rate/trend; drill to shipments. **2.3 Preconditions:** authorized aggregate data. **2.4 Flow:** set period/dimension → view denominator/rates → drill exceptions. **2.5 Alternative:** insufficient/partial/stale data, multi-shipment semantics OPEN. **2.6 Sections:** period/filter; KPI success/failure/attempt; trend; breakdown; underlying table. **2.7 Data:** counts, rate, denominator, provider/area/time, failure reasons where documented. **2.8:** period/provider/area/status; sort rate/count. **2.9:** drill shipment/export only if authorized. **2.10:** cancel-before-handoff excluded from delivery failure; formula OPEN. **2.11:** no data/partial/stale/denied. **2.13:** Sales view. **2.14:** Performance → Shipment detail/Finance. **2.15:** every rate shows sample size.

#### 3. Reference / UX Layout Direction

KPI + drill-down with definition/freshness visible.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Delivery Performance**, Web Admin. Period/provider/area filters; KPI success/failure/attempt with denominator; trends; breakdown; drill-down shipment table. States no data distinct from 0%, insufficient, partial/stale, denied. Exclude cancel-before-handoff; exact formula remains open. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 11 / US-SHIP-05; FR-12; UC-SHIP-06.

### ADM-019 — Marketplace Order Center

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Marketplace Operator · **Module:** Marketplace · **Entry:** nav/alert · **Next:** ADM-006/020/021.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Nhận biết Order nguồn, sync state và đưa vào fulfillment nội bộ chống trùng. **2.2 Goals:** filter by shop/platform; inspect original/internal state; resolve retryable issues. **2.3 Preconditions:** shop scope/integration. **2.4 Flow:** receive/list → match source ID once → validate mapping/stock → create/link internal Order → monitor sync. **2.5 Alternative:** duplicate, unmapped listing, insufficient stock, masked address, status conflict/out-of-order, cancellation. **2.6 Sections:** shop/platform tabs; order table; source/internal comparison; sync timeline; issue actions. **2.7 Columns:** platform/shop/source ID/internal ID/time, source/internal statuses, inventory/fulfillment, sync/error. **2.8:** search IDs; filter platform/shop/status/error/time. **2.9:** open, safe retry, route mapping; no replay with unknown impact. **2.10:** unique by source+ID; immediate reserve/release; actual platform value retained. **2.11:** new/processing/synced/partial/failed/conflict/cancelled. **2.12:** retry impact confirm. **2.13:** shop-scoped. **2.14:** Marketplace → Mapping/Order/Settlement. **2.15:** source versus internal always visually distinct.

#### 3. Reference / UX Layout Direction

Omnichannel operations table with dual-state columns and exception grouping.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Marketplace Order Center**, Web Admin. Platform/shop tabs; table source ID/internal ID/time/source state/internal state/inventory/fulfillment/sync/error; detail comparison and timeline; safe retry/mapping routes. States duplicate, unmapped, insufficient stock, masked data, out-of-order/conflict, partial/failed/cancelled. Keep source/internal meanings separate. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 12 / US-MKT-01~03,06; BR-MKTPLACE-01~07; FR-13; UC-MKT-01~03.

### ADM-020 — Listing–SKU Mapping

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Marketplace Operator · **Module:** Marketplace/Catalog · **Entry:** ADM-019 · **Next:** ADM-001/019.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Ánh xạ listing nguồn tới đúng SKU nội bộ; không tự áp dụng mapping mơ hồ. **2.2 Goals:** inspect listing/bundle; select SKU/components; approve change. **2.3 Preconditions:** scoped shop and catalog view. **2.4 Flow:** open unmapped item → compare source listing → search SKU → select/confirm → reprocess affected Order safely. **2.5 Alternative:** no match, ambiguous, combo/bundle, concurrent mapping change, listing inactive. **2.6 Sections:** unmapped queue; source listing snapshot; internal SKU search/results; mapping history/impact. **2.7 Columns:** platform/shop/listing/variant/source attributes/current mapping/state. **2.8:** search source/SKU; filter shop/state. **2.9:** map/change/unmap if allowed; retry affected Order. **2.10:** change audited; no guess; mapping scope OPEN. **2.11:** unmapped/mapped/ambiguous/conflict/stale/denied. **2.12:** change impact confirm. **2.13:** Marketplace + catalog view permissions. **2.14:** Order issue ↔ Mapping ↔ Catalog. **2.15:** comparison side-by-side.

#### 3. Reference / UX Layout Direction

Entity-matching workbench with source record left and internal candidates right.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Listing–SKU Mapping**, Web Admin. Unmapped queue; source listing/variant snapshot; internal SKU search/candidates; current mapping/history; affected Order impact. States unmapped, ambiguous, bundle/unsupported, stale/conflict, denied. No auto-map ambiguous data; changes confirmed/audited. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 12 / US-MKT-02,03; EPIC 04; FR-13; UC-MKT-01.

### ADM-021 — Marketplace Settlement & Revenue

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Marketplace Operator, Finance; Sales view · **Module:** Marketplace Finance · **Entry:** marketplace/finance nav · **Next:** ADM-048.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Đối soát từng thành phần Order/fee/subsidy/net settlement và phân biệt tạm tính/đã đối soát. **2.2 Goals:** import/view settlement; match; resolve difference; compare revenue. **2.3 Preconditions:** settlement data and permission. **2.4 Flow:** choose period/platform → view aggregate → exception rows → compare expected/actual → resolution. **2.5 Alternative:** adjustment later period, unmatched/duplicate, partial file/API, definition mismatch. **2.6 Sections:** period summary; gross/net KPIs; reconciliation table; exception detail; adjustment history. **2.7 Columns:** source Order/internal Order, GMV, discounts/subsidy, commission/fees/tax/refund, expected/received/delta, status. **2.8:** platform/shop/period/status/delta. **2.9:** assign/resolve by role; export only authorized. **2.10:** platform paid value is source; audit adjustments. **2.11:** matched/mismatch/unmatched/adjusted/partial/stale. **2.13:** Operator vs Finance permissions. **2.14:** Marketplace → Finance. **2.15:** money components adjacent.

#### 3. Reference / UX Layout Direction

Financial reconciliation table plus gross/net summary and exception drill-down.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Marketplace Settlement & Revenue**, Web Admin. Period/platform/shop filters; gross/net/unreconciled KPI; table Order IDs, GMV, seller discount, platform subsidy, fees/commission/tax/refund, expected/received/delta/status; exception detail/history. States matched/mismatch/unmatched/later adjustment/partial/stale. Actions role-based resolve/export. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 12 / US-MKT-04,05; BR-MKTPLACE-07; FR-13; UC-MKT-04.

### ADM-022 — Offline Receiving

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Offline Sales Staff · **Module:** Offline Sales · **Entry:** assigned transfer · **Next:** ADM-023/024.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Xác nhận hàng được giao đúng điểm/người/SKU/Batch và ghi chênh lệch trước bán. **2.2 Goals:** inspect transfer; count actual; accept/report variance. **2.3 Preconditions:** assignment/transfer. **2.4 Flow:** open document → count lines → flag variance/invalid Batch → confirm once. **2.5 Alternative:** wrong assignee, quantity mismatch, expired Batch, duplicate confirm. **2.6 Sections:** document/point/staff; expected-actual table; variance; confirmation. **2.7 Columns:** SKU/Batch/HSD/expected/actual/delta/eligibility. **2.9:** confirm, report discrepancy. **2.10:** assigned identity; expired not sellable; idempotent. **2.11:** pending/variance/blocked/confirmed/denied. **2.12:** confirm variance warning. **2.13:** assigned staff/location. **2.14:** Transfer → Receiving → POS stock. **2.15:** count-first UI.

#### 3. Reference / UX Layout Direction

Compact handover worksheet with expected/actual comparison.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Offline Receiving**, Web Admin. Transfer header point/staff; item table SKU/Batch/HSD/expected/actual/delta/eligibility; variance summary; confirm. States wrong assignment, mismatch, expired/invalid Batch, duplicate, confirmed/denied. Confirmation names responsible receiver. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 13 / US-OFF-01; BR-OFFLINE-01,02; FR-14; UC-OFF-01.

### ADM-023 — POS Sale

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Offline Sales Staff · **Module:** POS · **Entry:** point-of-sale nav · **Next:** ADM-025.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Ghi mỗi sale thành Order đúng SKU/Batch/tồn/giá/thu tiền; thao tác nhanh nhưng không vượt quyền. **2.2 Goals:** find/scan item; set qty; collect payment; complete receipt. **2.3 Preconditions:** identified staff, open selling context/kỳ if defined. **2.4 Flow:** add SKU → validate offline stock/expiry/current price → discount within authority → choose documented payment method → confirm once/print bill if in scope. **2.5 Alternative:** insufficient stock, expired, price/discount over authority, payment discrepancy, offline mode OPEN. **2.6 Sections:** SKU entry; basket; price/discount; customer/VAT minimum if requested; payment; receipt. **2.7 Data:** SKU/Batch/qty/price/discount/total/payment/order ref. **2.9:** add/update/remove/confirm/print. **2.10:** create Order; no negative stock; only authorized discount; methods OPEN. **2.11:** empty/validation/blocked/processing/success/duplicate/network error. **2.12:** sale confirm. **2.13:** staff/location. **2.14:** Receiving → POS → Shift report/Order. **2.15:** keyboard/scanner-friendly interaction [UX].

#### 3. Reference / UX Layout Direction

Fast POS split: item search/basket left, totals/payment right.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **POS Sale**, Web Admin. SKU scan/search; basket SKU/Batch/qty/price/discount; stock/expiry validation; customer/VAT minimal fields when requested; payment; total; confirm/receipt. States insufficient stock, expired, discount over authority, payment discrepancy, duplicate-safe processing, network error. Every sale creates Order. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 13 / US-OFF-02; BR-OFFLINE-05~07; FR-14; UC-OFF-02.

### ADM-024 — Offline Stock & Return

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Offline Sales Staff · **Module:** Offline Inventory · **Entry:** POS/period close · **Next:** ADM-025/027.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Phân biệt sellable count, damaged/quarantine, return-to-warehouse và Customer Return. **2.2 Goals:** count; record damage; initiate return transfer/customer case. **2.3 Preconditions:** responsible point/staff. **2.4 Flow:** select period/item → enter actual/disposition → attach reason/evidence → review variance → submit. **2.5 Alternative:** invalid qty, negative, expired, customer return requiring case, return transfer pending. **2.6 Sections:** inventory summary; count table; damage form; warehouse return; Customer Return link. **2.7 Columns:** SKU/Batch/system/actual/sellable/damaged/return qty/delta/reason. **2.9:** save count, report damage, return stock, open Return Case. **2.10:** no direct overwrite; central stock changes only after appropriate receipt. **2.11:** counting/pending transfer/variance/error. **2.12:** disposition confirm. **2.13:** point scope. **2.14:** POS → Stock → Reconciliation/Return. **2.15:** dispositions clearly separated.

#### 3. Reference / UX Layout Direction

Inventory count worksheet with explicit disposition lanes.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Offline Stock & Return**, Web Admin. Summary; table SKU/Batch/system/actual/sellable/damaged/return/delta; reason/evidence; warehouse-return workflow; separate Customer Return entry. States invalid/negative, expired, variance, pending transfer, error. No direct balance overwrite. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 13 / US-OFF-03; BR-OFFLINE-03,04; EPIC 14; UC-OFF-03.

### ADM-025 — Shift Report & Reconciliation

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Offline Staff; Sales/Finance reviewers · **Module:** Offline Reconciliation · **Entry:** period close/nav · **Next:** ADM-006/011/048.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Nộp báo cáo kỳ và đối soát riêng hàng hóa với tiền theo staff/point. **2.2 Goals:** review generated totals; complete missing data; submit; investigate/resolve delta. **2.3 Preconditions:** period definition OPEN. **2.4 Flow:** select point/period → view issued-sold-damaged-returned-ending equation and expected-actual cash → resolve missing → submit → reviewer closes. **2.5 Alternative:** no transaction, incomplete, difference, late correction, reopen closed period. **2.6 Sections:** period header; goods equation; money equation; source Orders; differences/evidence; review history; performance summary. **2.7 Columns:** SKU/Batch issued/sold/damaged/returned/expected/actual/delta; payment expected/actual/delta. **2.8:** point/staff/period/status. **2.9:** submit; assign/resolve/close/reopen by authority. **2.10:** no silent edit after submit; reason/audit. **2.11:** draft/incomplete/submitted/mismatch/reconciled/reopened. **2.12:** submit/close/reopen confirm. **2.13:** split duties. **2.14:** POS/Stock → Report → Inventory/Finance. **2.15:** goods and money never combined into one unexplained delta.

#### 3. Reference / UX Layout Direction

Period-close workbook with two reconciliations and source drill-down.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Shift Report & Reconciliation**, Web Admin. Point/staff/period/status; goods equation table; money equation; source Orders; differences/evidence; reviewer history; performance summary. States no transactions, incomplete, submitted, mismatch, reconciled, reopened. Staff submit; Sales/Finance resolve/close per permission with reason. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 13 / US-OFF-04~06; BR-OFFLINE-03,04,07; FR-14; UC-OFF-04.

### ADM-026 — Return Case Queue & Review

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** CSKH (triage), Sales Manager (decision) · **Module:** Return · **Entry:** nav/alert/Ticket · **Next:** ADM-015/027/028/030.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Ưu tiên Case, tập hợp Order/evidence và ghi quyết định đúng phạm vi. **2.2 Goals:** filter/assign; request info; review; approve/reject/partial by authority. **2.3 Preconditions:** Case exists; role permission. **2.4 Flow:** queue → Case detail/context → CS triage/request info → submit for review → Manager decision/reason → downstream goods/refund. **2.5 Alternative:** duplicate, insufficient evidence, concurrent decision, ineligible, partial approval, no packing video. **2.6 Sections:** SLA/status queue; Case/Order snapshot; requested lines; evidence; Payment/Shipment; communication; decision panel/timeline. **2.7 Columns:** Case/Order/customer masked/reason/amount/age/SLA/status/owner. **2.8:** search IDs; filter status/reason/age/SLA/owner. **2.9:** assign, request info, submit review, approve/reject/partial. **2.10:** CS cannot approve unless granted; decision audited; one decision state. **2.11:** waiting customer/reviewing/approved/rejected/partial/conflict. **2.12:** decision confirmation with impacts. **2.13:** split role. **2.14:** Queue → Evidence/Inspection/Refund/Ticket. **2.15:** Case and source data side-by-side.

#### 3. Reference / UX Layout Direction

Case-management queue plus investigation workspace.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Return Case Queue & Review**, Web Admin. SLA queue; table Case/Order/reason/amount/age/status/owner; detail requested lines/evidence/Order-Payment-Shipment/communication; decision timeline/panel. CS triages/requests info; Manager approves/rejects/partial with reason and impact confirm. States waiting, insufficient evidence, conflict, decided. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 14 / US-RET-04,05,07; EPIC 16; FR-15; UC-RET-03.

### ADM-027 — Returned Goods Inspection

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Warehouse Staff · **Module:** Return/Inventory · **Entry:** approved Case/return receipt · **Next:** ADM-028/009/011.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Ghi đúng expected–actual hàng trả và phân loại trước khi ảnh hưởng Inventory/Refund. **2.2 Goals:** receive; inspect SKU/Batch/qty/condition; set disposition. **2.3 Preconditions:** eligible approved/received return according to policy OPEN. **2.4 Flow:** scan/open Case → compare expected → record actual/condition/evidence → disposition → complete. **2.5 Alternative:** missing/extra/wrong item, damaged, no-return refund, multiple receipts. **2.6 Sections:** Case/return shipment; expected-actual table; condition/evidence; disposition; resulting inventory/refund handoff. **2.7 Columns:** Order line/SKU/Batch/expected/received/condition/disposition/delta. **2.9:** record receipt, classify sellable/quarantine/damaged/return supplier when defined. **2.10:** never auto-restock sellable; movement controlled. **2.11:** partial/mismatch/pending/completed/conflict. **2.12:** disposition confirm. **2.13:** Warehouse. **2.14:** Case → Inspection → Inventory/Refund. **2.15:** physical inspection workflow prioritized.

#### 3. Reference / UX Layout Direction

Receiving/inspection worksheet with downstream impact preview.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Returned Goods Inspection**, Web Admin. Case/return shipment; expected-vs-received line table; condition/evidence; disposition; inventory/refund handoff. States partial, missing/extra/wrong, damaged, multiple receipts, conflict/completed. No automatic sellable restock; confirm disposition. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 14; EPIC 09 return-stock open rules; FR-15,08; Sprint 10 state flow.

### ADM-028 — Refund Approval & Execution

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager; Finance where assigned · **Module:** Refund · **Entry:** approved Case/Payment · **Next:** ADM-007/048/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Phê duyệt/thực thi full/partial Refund với method hợp lệ, consent Loyalty khi dùng và audit đầy đủ. **2.2 Goals:** verify refundable amount; choose scope/method; authorize; track result. **2.3 Preconditions:** eligible paid/returned Order and reviewed Case. **2.4 Flow:** inspect original payment/Case → enter/confirm amount and method → additional verification if policy → execute once → track callback/result. **2.5 Alternative:** partial, loyalty with consent, provider pending/fail, insufficient refundable balance, duplicate, cancellation before collection. **2.6 Sections:** Case decision; original transactions; amount allocation; method/consent; impact; execution timeline. **2.7 Data:** original transaction, captured/refunded/remaining, proposed amount, lines, method, reason, approver/executor, result. **2.9:** approve/reject/execute/retry only safe. **2.10:** Sales Manager or higher approves; original method or Loyalty with agreement; audited; no refund for uncollected cancellation. **2.11:** draft/pending approval/executing/pending/success/fail/partial/duplicate. **2.12:** high-impact confirmation/OTP only where policy confirms. **2.13:** approve vs execute split OPEN. **2.14:** Case → Refund → Finance/Audit/Customer. **2.15:** refundable remaining and final customer result prominent.

#### 3. Reference / UX Layout Direction

Financial approval workspace with immutable source evidence and execution ledger.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Refund Approval & Execution**, Web Admin. Case decision; original Payment; captured/refunded/remaining; line/amount allocation; refund method and Loyalty consent; impact; approver/executor; timeline. States full/partial, pending approval, executing/pending, provider fail, duplicate, success. Actions approve/reject/execute by role with reason/confirmation. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 14 / US-RET-06; BR-REFUND-01~04; UC-PAY-04; FR-06,15.

### ADM-029 — Review Operations & Analytics

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** CSKH, Sales Manager · **Module:** Review · **Entry:** nav/alert · **Next:** ADM-030/004/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Theo dõi review tiêu cực, xử lý vi phạm/reply có lịch sử và xem phân bố/xu hướng. **2.2 Goals:** filter queue; inspect purchase verification/media/version; create Ticket/reply/hide media/review; analyze. **2.3 Preconditions:** scoped permission. **2.4 Flow:** queue or Analytics tab → select review → action with reason → result/audit. **2.5 Alternative:** Ticket already exists, review concurrently changed, hidden review, media violation, no data. **2.6 Sections:** moderation/negative queue; review detail; Order verification context; public reply; action history; analytics KPIs/trend/star distribution/Product-SKU breakdown. **2.7 Columns:** rating, Product/SKU, customer masked, verified, time, status, Ticket, flags. **2.8:** rating/time/Product/SKU/status/Ticket; analytics period. **2.9:** link/create Ticket, reply, hide/restore review/media by permission. **2.10:** no falsifying Verified; hidden handling audited; sample size shown. **2.11:** empty/no data/conflict/hidden/denied. **2.12:** moderation confirm/reason. **2.13:** CS support vs Manager moderation. **2.14:** Review → Ticket/Product/Audit. **2.15:** 0-star not used for no data.

#### 3. Reference / UX Layout Direction

Moderation inbox with an Analytics tab and contextual side panel.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Review Operations & Analytics**, Web Admin. Queue table rating/Product-SKU/customer masked/Verified/time/status/Ticket; detail content/media/purchase context/history/reply; moderation actions; analytics average+sample, star distribution, trend and filters. States no data, existing Ticket, hidden/media violation, concurrent conflict, denied. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 15 / US-REV-04~06; BR-REVIEW-03,05; FR-16; UC-REV-03,04.

### ADM-030 — Ticket Queue & Agent Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Customer Service · **Module:** CS · **Entry:** nav/alert/review · **Next:** Orders/Returns/Evidence/Audit.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Nhận đúng Ticket theo SLA, xử lý hội thoại đa kênh và xem ngữ cảnh giao dịch read-only. **2.2 Goals:** find/claim/transfer; converse; internal note; inspect context; use AI draft only as assistant. **2.3 Preconditions:** queue scope. **2.4 Flow:** prioritize queue → claim → read timeline/context → reply/note/request action → update status. **2.5 Alternative:** concurrent claim, send fail, unidentified inbound, stale source, no media permission, SLA overdue. **2.6 Sections:** queue/filter; Ticket header/SLA/assignee; conversation; composer/internal note; customer profile masked; Order/Payment/Shipment/Return context; AI summary/draft [GĐ3]. **2.7 Columns:** Ticket/time/topic/priority/status/SLA/customer/channel/assignee. **2.8:** search code/customer/order; filter priority/status/SLA/channel/assignee. **2.9:** claim/transfer/reply/retry/note/status/escalate; source entities read-only. **2.10:** internal note never customer-facing; AI never sends/acts; mask PII. **2.11:** unassigned/claimed/waiting/overdue/send failed/partial/denied. **2.12:** transfer/status confirms where impactful. **2.13:** queue/data/media scopes. **2.14:** Ticket → Order/Return/Evidence. **2.15:** queue and conversation/context can use master-detail layout.

#### 3. Reference / UX Layout Direction

Agent desktop: queue, conversation and contextual sidebar in one workstream.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Ticket Queue & Agent Workspace**, Web Admin. Queue Ticket/time/topic/priority/status/SLA/channel/assignee; claim/transfer; public conversation and delivery states; internal notes; composer; masked customer and read-only Order/Payment/Shipment/Return context; secure evidence link; AI summary/draft labeled and never auto-send. States concurrent claim, overdue, waiting, send failed, partial/denied. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 16 / US-CS-02~05, US-AI-02, US-AI-06; FR-15; UC-CS-01,02.

### ADM-031 — Expiry Coordination

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** CSKH, Sales Manager · **Module:** CS/Expiry · **Entry:** ADM-012/alert · **Next:** ADM-032/039.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Dùng cảnh báo Inventory đã xác minh để ghi hoạt động phối hợp mà không sửa kho/bán hàng hết hạn. **2.2 Goals:** view affected Batch/Product; assign coordination; link Promotion/Campaign if authorized. **2.3 Preconditions:** expiry alert and view permission. **2.4 Flow:** open alert → inspect current qty/HSD → record proposed/assigned action → follow status. **2.5 Alternative:** Batch expired/qty changed/alert resolved/duplicate. **2.6 Sections:** risk list; Batch current snapshot; coordination notes/owner; linked campaign/promotion; history. **2.7 Columns:** Product/SKU/Batch/qty/HSD/risk/source freshness/owner/status. **2.8:** risk/owner/status. **2.9:** acknowledge, assign, record activity, deep link only. **2.10:** no inventory edit; expired not sellable; alert dedup. **2.11:** open/ack/resolved/stale/expired/denied. **2.12:** none beyond assignment feedback. **2.13:** CS/Sales actions. **2.14:** Expiry → Coordination → Promotion/Campaign. **2.15:** source freshness visible.

#### 3. Reference / UX Layout Direction

Cross-functional action list linked to the authoritative Inventory record.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Expiry Coordination**, Web Admin. Risk table Product/SKU/Batch/qty/HSD/risk/freshness/owner/status; current Inventory snapshot; coordination notes/activity/history; Promotion/Campaign links. States stale/quantity changed, expired, resolved, duplicate, denied. No inventory edit and never offer expired stock. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 16 / US-CS-06; EPIC 09 / US-INV-10; BR-BATCH-04,05.

### ADM-032 — Coupon Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager · **Module:** Promotion · **Entry:** promotion nav · **Next:** ADM-006/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tạo/công khai/tạm dừng coupon với điều kiện, scope, hạn mức và lịch rõ. **2.2 Goals:** list/search; create; validate; publish/pause; inspect usage. **2.3 Preconditions:** permission. **2.4 Flow:** draft config → preview conditions → validate unique code/value → publish → monitor uses. **2.5 Alternative:** duplicate code, invalid discount, overlapping/changed conditions, usage exists, cap contention. **2.6 Sections:** list/status; basic config; value/conditions/scope; validity/limits; preview/usage/history. **2.7 Columns:** code/name/type/value/scope/from-to/usage/limit/status. **2.8:** search code; filter status/date/type. **2.9:** create/update/publish/pause; no destructive history delete. **2.10:** reservations/caps atomic; used coupon edits protect old Order snapshots; stacking OPEN. **2.11:** draft/scheduled/active/exhausted/paused/ended/validation/conflict. **2.12:** publish/pause impact confirm. **2.13:** authorized Sales. **2.14:** Coupon → Checkout/Order/Audit. **2.15:** conditions readable, not raw rule syntax.

#### 3. Reference / UX Layout Direction

Promotion rule builder with human-readable preview and lifecycle status.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Coupon Workspace**, Web Admin. List code/type/value/scope/date/usage-limit/status; editor configuration, conditions/scope, schedule/limits, readable preview, usage/history. States duplicate, invalid value, draft/scheduled/active/exhausted/paused/ended/conflict. Actions publish/pause with impact confirm. Stacking and exact offer types remain open. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 17 / US-PROMO-01,02; EPIC 06 / US-CHK-05; FR-17; UC-PROMO-01.

### ADM-033 — Combo Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager · **Module:** Promotion · **Entry:** promotion nav/DSS insight · **Next:** ADM-001/052.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Cấu hình Combo theo SKU/qty/benefit trong khi giữ tồn từng thành phần và Order snapshot. **2.2 Goals:** create/update/publish; inspect component sellability. **2.3 Preconditions:** component SKUs exist. **2.4 Flow:** name/schedule → add components/qty → price/benefit → validate inventory/policy → publish. **2.5 Alternative:** invalid/inactive component, insufficient stock, SKU price changes, concurrent edit. **2.6 Sections:** combo list; component builder; pricing/benefit; schedule/status; preview/history. **2.7 Columns:** Combo/status/components/current availability/benefit/validity. **2.9:** create/update/publish/pause. **2.10:** each component reserved; old Orders unchanged; whether Combo is SKU OPEN. **2.11:** draft/active/component unavailable/price changed/ended/conflict. **2.12:** change impact confirm. **2.13:** Sales. **2.14:** DSS insight → Combo → D2C Promotion/PDP. **2.15:** component breakdown always visible.

#### 3. Reference / UX Layout Direction

Bundle builder with component table and storefront preview.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Combo Workspace**, Web Admin. Combo list; component SKU+qty builder; price/benefit; schedule/status; availability validation; preview/history. States component inactive/out of stock, price changed, draft/active/ended/conflict. Actions create/update/publish/pause. Do not decide whether Combo is independent SKU until PO confirms. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 17 / US-PROMO-03,04; FR-17; UC-PROMO-02.

### ADM-034 — B2B Quote Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Sales Manager · **Module:** B2B · **Entry:** B2B nav/notification · **Next:** ADM-006/047/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Thẩm định request, yêu cầu bổ sung, lập version có breakdown và phát hành đúng thẩm quyền. **2.2 Goals:** prioritize queue; inspect company/request; build/revise/issue/reject Quote. **2.3 Preconditions:** B2B request. **2.4 Flow:** claim request → validate lines/customization/logo → request info or build version → preview → issue → monitor accept/expiry/conversion. **2.5 Alternative:** invalid/unsafe file, concurrent edit, superseded version, unauthorized issue, stock/price change. **2.6 Sections:** request queue/SLA; company; lines; files/customization; Quote builder; terms/expiry; version history/messages. **2.7 Columns:** request/company/time/lines/status/SLA/owner/current version/expiry. **2.8:** search request/company; filter status/SLA/owner. **2.9:** assign, request info, draft version, issue/reject/withdraw per policy. **2.10:** issued version immutable; acceptance exact version; pricing formula/authority OPEN. **2.11:** waiting info/draft/issued/superseded/accepted/rejected/expired/conflict. **2.12:** issue/withdraw confirm. **2.13:** issue permission. **2.14:** Request → Quote → Order/Finance/Audit. **2.15:** preview equals customer view.

#### 3. Reference / UX Layout Direction

Sales case queue and versioned quote builder in master-detail workspace.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **B2B Quote Workspace**, Web Admin. SLA queue; company/request lines; customization/logo; Quote Builder with unit price/discount/customization/shipping/tax/total/expiry; customer preview; version/messages/history. States waiting info, draft, issued, superseded, accepted/rejected/expired, conflict. Issue/withdraw by permission and exact version. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 18 / US-B2B-03; US-B2B-01~05 context; FR-19; UC-B2B-02.

### ADM-035 — Traceability Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Manager assigned (ACT-12/15 OPEN) · **Module:** OCOP · **Entry:** OCOP/Catalog/Batch · **Next:** ADM-036/053/009.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý versioned traceability record đúng Product/SKU/Batch, source/evidence và public/internal fields. **2.2 Goals:** create/edit draft; link supplier/material/process/certificate; submit review. **2.3 Preconditions:** actor mapping/permission. **2.4 Flow:** select scope → create revision → enter origins/process/cert/media/source → validate → submit. **2.5 Alternative:** wrong scope/link, overlapping public record, changed Supplier/Batch, invalid certificate, concurrent edit. **2.6 Sections:** Product–SKU–Batch tree; revision/status; origin/material/producer; process; certificates; media/docs; QR link; history. **2.7 Data:** source validity, scope, evidence, public flag; exact required fields OPEN. **2.8:** search Product/SKU/Batch/cert; filter status/scope. **2.9:** draft/edit/link/submit/correct via new revision. **2.10:** history snapshot; only public approved revision; no duplicate scope. **2.11:** draft/review/public/suspended/conflict/incomplete. **2.12:** submit/QR correction confirmation. **2.13:** creator role OPEN. **2.14:** Catalog/Batch/Supplier ↔ OCOP → Approval. **2.15:** tree + revision workspace.

#### 3. Reference / UX Layout Direction

Hierarchical Product–SKU–Batch record editor with revision/evidence panels.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Traceability Workspace**, Web Admin. Product-SKU-Batch tree; revision/status; origins/material regions/producers; process; certificates; media/docs; QR relation; source validity/history. States incomplete, wrong scope/link, duplicate public scope, stale source, conflict, draft/review/public/suspended. Actions draft/edit/submit; corrections create revision. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 19 / US-OCOP-04,05; FR-20; UC-OCOP-02.

### ADM-036 — Traceability Approval

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** authorized approver (ACT-12/15 OPEN) · **Module:** OCOP · **Entry:** review queue · **Next:** ADM-035/D2C-026/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** So sánh revision/internal sources/EXT-10 result và chỉ công khai hồ sơ đủ căn cứ. **2.2 Goals:** review diff/evidence; approve/reject/suspend. **2.3 Preconditions:** submitted revision and separation policy OPEN. **2.4 Flow:** queue → compare current/new/source/provider → approve/publish or reject reason → audit. **2.5 Alternative:** provider mismatch/unavailable, revision changes, expired certificate, unauthorized self-approval. **2.6 Sections:** queue; diff; evidence/source; provider verification; public preview; decision history. **2.7 Columns:** scope/revision/submitter/time/status/issues. **2.8:** scope/status/issue. **2.9:** approve, reject/request correction, suspend public record. **2.10:** approved exact revision only; reason required for reject/suspend; audit. **2.11:** pending/issue/mismatch/stale/approved/rejected/suspended. **2.12:** publish/suspend confirmation. **2.13:** approver role OPEN. **2.14:** Workspace → Approval → Public/Audit. **2.15:** public preview adjacent evidence.

#### 3. Reference / UX Layout Direction

Approval diff workspace with source reconciliation and public preview.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Traceability Approval**, Web Admin. Queue; exact revision diff; internal evidence; EXT-10 result; certificate issues; public preview; history. States provider mismatch/unavailable, revision changed, certificate expired, approved/rejected/suspended. Actions approve/publish, reject/correction, suspend with reason and audit. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 19 / US-OCOP-06; FR-20; UC-OCOP-03.

### ADM-037 — Content Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Content Manager · **Module:** Content/SEO · **Entry:** Content nav · **Next:** ADM-038/001/035.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Soạn Article/Product SEO dưới dạng revision, liên kết nguồn Product/OCOP và preview mà không tự publish. **2.2 Goals:** list/search content; create/edit/save draft; manage SEO/URL; request AI suggestion. **2.3 Preconditions:** permission. **2.4 Flow:** create/select → edit structured content/media refs/sources/SEO → save draft → preview → submit review. **2.5 Alternative:** missing source, slug duplicate/reserved, concurrent edit, AI/provider fail, Product changes. **2.6 Sections:** content list; editor; media/source references; SEO panel (title/description/slug/canonical/indexability); preview; revision history; AI assistant [GĐ2/3 conflict]. **2.7 Columns:** title/type/status/revision/owner/updated/publish schedule. **2.8:** search title/slug; filter type/status/owner. **2.9:** create/edit/save/preview/submit; accept AI suggestions selectively. **2.10:** AI cannot publish; public facts from approved source; old URL history. **2.11:** draft/unsaved/validation/conflict/AI unavailable/source stale. **2.12:** unsaved and slug-change warning. **2.13:** Content Manager. **2.14:** Content → Media/Approval/Public. **2.15:** editor/SEO/source separated but same revision context.

#### 3. Reference / UX Layout Direction

Editorial workspace with main editor and contextual SEO/source panels.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Content Workspace**, Web Admin. Searchable list; revision editor; media refs; Product/OCOP sources; SEO title/description/slug/canonical/indexability preview; public preview; history; AI suggestions labeled/selective. States missing source, duplicate slug, unsaved, concurrent conflict, stale Product, AI unavailable. Save/preview/submit only, no direct AI publish. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 20 / US-CONTENT-01,02,04~06, US-AI-03; FR-21,22; UC-CONTENT-01,02.

### ADM-038 — Media, Approval & Publication

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Content Manager/authorized approver · **Module:** Content Operations · **Entry:** ADM-037 · **Next:** D2C-027/028/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý media usage/alt text và duyệt/lên lịch/ẩn đúng revision. **2.2 Goals:** upload/reuse media; inspect usage; review diff; approve/reject; schedule/publish/hide. **2.3 Preconditions:** permissions and submitted revision. **2.4 Flow:** media upload/process → attach to revision → review diff/source/SEO → approve → schedule/publish; hide when needed. **2.5 Alternative:** invalid/upload fail, missing alt, media in use, self-approval restriction, changed revision, schedule invalid. **2.6 Sections:** media library; usage/details; approval queue/diff; public preview; publication calendar/history. **2.7 Columns:** media/type/state/usage/alt/owner/time; content/revision/status/schedule. **2.8:** search/filter media/status/date. **2.9:** upload/attach/hide media; approve/reject; schedule/publish/hide content. **2.10:** exact approved revision; no hard delete in-use; audit. **2.11:** processing/failed/in-use/pending/changed/approved/scheduled/published/hidden. **2.12:** delete/hide/publish confirm. **2.13:** creator/approver mapping OPEN. **2.14:** Editor → Approval/Calendar → Public. **2.15:** media and publication share revision context.

#### 3. Reference / UX Layout Direction

Media library combined with approval queue and calendar tabs.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Media, Approval & Publication**, Web Admin. Tabs Media Library (processing, usage, alt/caption), Approval Queue (revision diff, Product/OCOP source, SEO, public preview), Publication Calendar/history. States invalid/upload fail, media in use, missing alt, revision changed, pending/approved/rejected/scheduled/published/hidden. Confirm publish/hide; role separation. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 20 / US-CONTENT-02,03,07; FR-21,22; UC-CONTENT-03.

### ADM-039 — Campaign Workspace

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Marketing Staff · **Module:** Marketing · **Entry:** Marketing nav/DSS insight · **Next:** ADM-040/041.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Lập Campaign revision với objective, audience, channel, budget, schedule, owner, KPI và asset readiness. **2.2 Goals:** create/edit; link Content/Promotion/audience; submit. **2.3 Preconditions:** permission and source entities accessible. **2.4 Flow:** draft basics → audience/channel/budget/KPI → link briefs/assets/promotions → readiness → submit. **2.5 Alternative:** invalid period, missing asset, budget threshold, stale linked asset, concurrent edit. **2.6 Sections:** campaign list; objective/period/owner; audience; channel plan; budget; KPI targets; content/assets; readiness; revision. **2.7 Columns:** Campaign/status/period/owner/budget/readiness/revision. **2.8:** search; filter status/channel/owner/date. **2.9:** create/edit/link/submit/withdraw per rule. **2.10:** submitted revision frozen; asset changes visible; privacy consent. **2.11:** draft/incomplete/pending/changed/conflict. **2.12:** submit/withdraw confirm. **2.13:** Marketing create; audience/finance limited. **2.14:** DSS/Content/Promotion → Campaign → Approval. **2.15:** complex form separated into logical sections.

#### 3. Reference / UX Layout Direction

Campaign builder with readiness checklist and linked-asset dependencies.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Campaign Workspace**, Web Admin. List; editor objective/period/owner; audience; channels; planned budget; KPI targets; linked Brief/Article/Promotion/media; readiness; revision history. States invalid period, missing/stale asset, budget threshold, concurrent conflict, draft/pending. Submit exact revision. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 21 / US-MKTG-01~03; FR-23; UC-MKTG-01.

### ADM-040 — Campaign Approval & Calendar

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Executive/authorized Manager · **Module:** Marketing Governance · **Entry:** approval queue · **Next:** ADM-039/041/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Duyệt revision theo thẩm quyền ngân sách/audience/channel và theo dõi lịch execution. **2.2 Goals:** compare; approve/reject/request change; view calendar. **2.3 Preconditions:** submitted ready revision. **2.4 Flow:** queue → diff/readiness/risk → decision/reason → calendar status. **2.5 Alternative:** over authority, self-approval, asset changed, revision stale, cancellation/withdrawal. **2.6 Sections:** queue; revision diff; budget/KPI/audience/channel; readiness/risk; decision; calendar/execution drilldown. **2.7 Columns:** Campaign/revision/requester/budget/period/readiness/status. **2.8:** status/date/owner/threshold. **2.9:** approve/reject/request change; pause/cancel only after policy. **2.10:** exact revision; separation/threshold OPEN; audit. **2.11:** pending/changed/over-limit/approved/rejected/withdrawn. **2.12:** decision confirm. **2.13:** delegated authority. **2.14:** Campaign → Approval → Performance/Audit. **2.15:** calendar timezone visible.

#### 3. Reference / UX Layout Direction

Approval diff view plus campaign execution calendar.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Campaign Approval & Calendar**, Web Admin. Approval queue; exact revision diff; budget/KPI/audience/channel; asset readiness/risk; decision history; calendar with Approved/Scheduled/Active/Paused only when defined. States over authority, self-approval restricted, asset/revision changed, approved/rejected/withdrawn. Decision requires reason/audit. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 21 / US-MKTG-03,04; FR-23; UC-MKTG-02.

### ADM-041 — Campaign Performance & Integration

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Marketing Staff; Executive with financial permission · **Module:** Marketing Analytics · **Entry:** active/completed Campaign · **Next:** ADM-047/049/052.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** So target–actual KPI, theo dõi dữ liệu channel và ROI có định nghĩa/source/freshness. **2.2 Goals:** filter period/channel; inspect sync; resolve mapping; compare Campaigns. **2.3 Preconditions:** authorized data. **2.4 Flow:** select Campaign/period → KPI target/actual/variance → channel breakdown → inspect sync/mapping → ROI view. **2.5 Alternative:** no/late/partial data, provider disconnected, duplicate, currency/timezone mismatch, unattributed revenue/refund. **2.6 Sections:** KPI dashboard; channel breakdown; integration monitor; mapping errors; cost/revenue/ROI; source/freshness. **2.7 Data:** impression/reach/click/etc only after dictionary; target/actual, source, updated; planned/committed/actual cost; attributed revenue. **2.8:** Campaign/channel/period/data state. **2.9:** safe retry, fix mapping, compare/export if permitted. **2.10:** external vs internal distinguished; attribution formula/version visible; refunds reflected. **2.11:** empty/partial/stale/failed/disconnected/estimated/denied. **2.12:** retry impact confirm. **2.13:** financial field permissions. **2.14:** Campaign → Performance → Finance/DSS. **2.15:** missing data not zero.

#### 3. Reference / UX Layout Direction

Dashboard + integration operations tabs with common filters/freshness.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Campaign Performance & Integration**, Web Admin. Common Campaign/channel/period filters; target-actual-variance KPI; channel breakdown; data source/freshness; sync monitor/mapping errors/safe retry; cost and attributed revenue/ROI with formula version. States no data, partial/late/stale, duplicate, disconnected/provider fail, estimated, denied. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 21 / US-MKTG-05~07; FR-23; UC-MKTG-03.

### ADM-042 — Employee Account Administration

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** System Administrator/HR Manager where authorized · **Module:** Administration · **Entry:** admin/Employee Profile · **Next:** ADM-043/044/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tạo/cập nhật/khóa System Access gắn đúng Employee, không trộn HR fields. **2.2 Goals:** search; create invitation; update identity metadata; lock/schedule lock. **2.3 Preconditions:** Employee Profile requirement per policy OPEN. **2.4 Flow:** select Employee → configure identifier/status/Role baseline → invite → update/lock with impact/reason. **2.5 Alternative:** duplicate employee/identifier, invitation fail, no Role, concurrent update, last admin/self lock. **2.6 Sections:** directory; Profile reference; account state/invitation; Role assignments; sessions; change history. **2.7 Columns:** Employee, identifier, account status, Roles, invitation/last access [UX], scheduled lock. **2.8:** search; filter status/Role. **2.9:** create/update/resend invite/lock; unlock policy OPEN. **2.10:** no readable password; locked retains history; protect final admin; sessions handling OPEN. **2.11:** invited/active/no-role/locked/scheduled/error/conflict. **2.12:** lock impact/reason confirm. **2.13:** target/scope authority. **2.14:** Employee ↔ Account ↔ Role/Review/Audit. **2.15:** dangerous actions isolated.

#### 3. Reference / UX Layout Direction

Identity administration directory and account detail; HR profile is only a linked reference.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Employee Account Administration**, Web Admin. Directory Employee/identifier/status/Roles/invitation/scheduled lock; detail Profile reference, access status, invitation, Roles, sessions/history. States duplicate, no Role, invite fail, active/locked/scheduled, concurrent conflict. Actions create/update/invite/lock with reason and last-admin warning. Never show password. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 22 / US-ADM-01~03; FR-03; UC-ADM-01.

### ADM-043 — Role & Permission

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** System Administrator · **Module:** RBAC · **Entry:** administration nav · **Next:** ADM-044/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý Role và permission matrix theo resource/action/scope, bảo vệ Role hệ thống. **2.2 Goals:** create/update/disable Role; view/manage configurable permissions; assess impact. **2.3 Preconditions:** privileged permission. **2.4 Flow:** select/create Role → edit purpose/scope/matrix → preview affected Employees/effective rights → save/disable. **2.5 Alternative:** duplicate, invalid scope, system Role, in-use disable, concurrent edit, privilege escalation. **2.6 Sections:** Role list; metadata; permission catalog/matrix; assignments/impact; history. **2.7 Columns:** Role, status, system/custom, employees, updated; matrix module/resource/action/scope. **2.8:** search Role/permission; filter status/module. **2.9:** create/update/disable/assign permission; no delete with history. **2.10:** view ≠ update/approve/export; cannot exceed authority; final admin protected. **2.11:** active/disabled/system/conflict/denied. **2.12:** save/disable impact confirm. **2.13:** System Admin scoped. **2.14:** Role → Assignment/Review/Audit. **2.15:** matrix horizontally scrollable and groupable.

#### 3. Reference / UX Layout Direction

RBAC matrix pattern with effective-impact preview.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Role & Permission**, Web Admin. Role list; purpose/scope/status; permission matrix grouped module/resource/action/scope; affected Employees; effective-right preview/history. States duplicate, invalid scope, protected system Role, in-use disable, conflict/denied. Actions create/update/disable and permission assignment with impact confirm. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 22 / US-ADM-04,05; BR-AUTH-03,04; FR-03; UC-ADM-02.

### ADM-044 — Access Review

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** System Administrator · **Module:** Access Governance · **Entry:** administration nav/account detail · **Next:** ADM-042/043/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Trả lời Employee có quyền gì, từ Role/assignment nào, scope/thời hạn/status nào. **2.2 Goals:** search employee/sensitive permission; inspect effective rights; revoke/expire assignment when authorized. **2.3 Preconditions:** access-review permission. **2.4 Flow:** choose Employee/permission → calculate view-time result → inspect sources/duplicates/expiry → remediate. **2.5 Alternative:** multiple Roles same permission, disabled/expired Role, locked account, result changes while open. **2.6 Sections:** employee/account summary; role assignments; effective permission matrix; source/expiry/scope; review history. **2.7 Columns:** permission, source Role, scope, effective from/to, state, approval ref. **2.8:** employee/Role/permission/sensitive/expiry. **2.9:** open assignment, revoke if allowed, export only when defined. **2.10:** effective right reflects account lock/Role state; no silent direct permission. **2.11:** no access/expired/changed/stale/denied. **2.12:** revoke confirm. **2.13:** Access Review privilege. **2.14:** Account/Role ↔ Review → Audit. **2.15:** sources can expand per permission.

#### 3. Reference / UX Layout Direction

Entitlement review table with provenance, not a simple checkbox matrix.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Access Review**, Web Admin. Employee/account summary; assigned Roles; effective permission table Permission/source Role/scope/from-to/state/approval; sensitive-permission search; review history. States duplicate source, expired/disabled Role, locked account, stale/changed, denied. Actions open/revoke assignment by permission with confirm. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 22 / US-ADM-06,07; FR-03; UC-ADM-03.

### ADM-045 — Audit Explorer

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Auditor/System Administrator · **Module:** Audit · **Entry:** audit nav/entity deep link · **Next:** ADM-046.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tra cứu event theo actor/action/object/time/correlation và xem diff/source/reason mà không sửa log. **2.2 Goals:** search/filter; inspect timeline/detail; export route. **2.3 Preconditions:** audit privilege/data scope. **2.4 Flow:** set filter → result table → event detail/diff/correlation → optionally request export. **2.5 Alternative:** no result, late/out-of-order, sensitive masked, integrity issue, out-of-scope. **2.6 Sections:** filters/scope/timezone; result table; event metadata; source/reason; before-after diff; correlation timeline; annotation link. **2.7 Columns:** occurred/recorded time, actor, action, object, outcome, integrity status, correlation. **2.8:** actor/action/module/object/outcome/time/correlation. **2.9:** view detail, annotate/link incident if policy, go integrity/export; no delete/edit. **2.10:** log immutable/tamper evident; search/export audited; secrets masked. **2.11:** empty/no result/partial/masked/integrity invalid/denied. **2.12:** none. **2.13:** search/detail/sensitive scopes. **2.14:** source screens → Audit → Integrity. **2.15:** wide table and diff viewer.

#### 3. Reference / UX Layout Direction

Security event explorer with result table and forensic detail drawer/page.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Audit Explorer**, Web Admin. Scope/timezone; filters actor/action/module/object/outcome/time/correlation; table occurred/recorded time, actor, action, object, outcome, integrity; detail metadata/source/reason/before-after diff/correlation timeline. States no result, late/out-of-order, masked, integrity issue, denied. No edit/delete Audit Event. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 23 / US-AUDIT-01~04; BR-AUDIT-01~05; FR-27; UC-AUDIT-01.

### ADM-046 — Integrity Verification & Export

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Auditor/Security Operator · **Module:** Audit Integrity · **Entry:** Audit nav/alert · **Next:** ADM-045/056.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Xem coverage/kết quả kiểm chứng, khoanh vùng invalid/incomplete/error và tạo export có kiểm soát. **2.2 Goals:** run/view verification; inspect gaps; acknowledge/escalate; request export. **2.3 Preconditions:** verify/export permission. **2.4 Flow:** select range/scope → verification job → result summary/gaps → incident action; export filter/fields/reason → job/file. **2.5 Alternative:** late/duplicate events, partial run, verification error, repeated alert, export too large/expired. **2.6 Sections:** integrity KPI/last run; job list; result/gap detail; alert/escalation; export dialog/jobs. **2.7 Columns:** scope/time range/coverage/result/invalid point/run time/requester; export status/expiry. **2.8:** range/module/result/job. **2.9:** verify, acknowledge/escalate, request/download authorized export. **2.10:** no repair/delete; annotations separate; exports audited. **2.11:** valid/invalid/incomplete/error/running/export failed/expired. **2.12:** export scope/reason confirm. **2.13:** verify/export separate. **2.14:** Integrity ↔ Explorer/Alert. **2.15:** long-running jobs async status.

#### 3. Reference / UX Layout Direction

Integrity dashboard + job center; incident actions distinct from source log.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Integrity Verification & Export**, Web Admin. Coverage/last verify; job list scope/range/result; invalid/gap detail; alert/escalation; export request filter/fields/estimate/reason; export jobs/status/expiry. States valid/invalid/incomplete/error/running, duplicate alert, export failed/expired. Never repair or delete events. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 23 / US-AUDIT-05, US-AUDIT-01 export; NFR-07; UC-AUDIT-02.

### ADM-047 — Finance Analytics

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Executive, Finance · **Module:** Finance BI · **Entry:** Finance nav · **Next:** ADM-048/049.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Phân tích doanh thu/chi phí/lợi nhuận theo kỳ, kênh, Product/SKU, khu vực với định nghĩa/freshness. **2.2 Goals:** select dimensions; compare period; drill source. **2.3 Preconditions:** GĐ3, finance permission. **2.4 Flow:** set period/filters → view KPI/trends/breakdown → drill to reconciled/source rows. **2.5 Alternative:** late/partial inputs, refunds/corrections, missing cost, small geographic group, definition mismatch. **2.6 Sections:** definition/freshness; KPI revenue/cost/profit/margin; trends; channel/product/region tables; drilldown. **2.7 Data:** gross/net/cost/profit only per approved formulas; counts and statuses. **2.8:** period/channel/Product-SKU/region/data state. **2.9:** drill, compare, export route. **2.10:** historical dimensions; privacy aggregation; missing ≠ zero. **2.11:** empty/partial/stale/missing cost/denied. **2.13:** financial scope. **2.14:** Analytics → Finance Ops/source Order. **2.15:** KPI carries definition and updated time.

#### 3. Reference / UX Layout Direction

Dashboard KPI + drill-down with shared filters and reconciled-state context.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Finance Analytics**, Web Admin. Period comparison; common filters channel/Product-SKU/region/data state; KPI revenue/cost/profit/margin with formula/freshness; trends; breakdown tables; drilldown. States no/partial/stale, missing cost, late correction/refund, privacy-limited, denied. Use approved definitions only. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 24 / US-FIN-01~05; FR-24; UC-FIN-01.

### ADM-048 — Finance Operations

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Finance · **Module:** Reconciliation/Tax/Invoice/Export · **Entry:** Finance nav/exceptions · **Next:** ADM-007/021/025/028/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tổng hợp exception đối soát, nghĩa vụ thuế/hóa đơn và export job có Audit. **2.2 Goals:** reconcile sources; resolve; issue/adjust invoice; export report. **2.3 Preconditions:** Finance permissions/provider where used. **2.4 Flow:** choose period/source → exception comparison → resolve reason → tax/invoice action → create/download export. **2.5 Alternative:** provider timeout, mismatch, partial data, locked period OPEN, invoice failed/adjusted/cancelled, export too large/expired. **2.6 Sections:** reconciliation queue; expected/actual detail; tax period; invoice list/detail/history; export center. **2.7 Columns:** source/ref/expected/actual/delta/status/owner; invoice number/order/customer/status/provider; export requester/params/status/expiry. **2.8:** period/source/status/delta. **2.9:** resolve, issue/adjust/cancel only by policy, request export. **2.10:** law/tax rules OPEN; sensitive files protected; history immutable. **2.11:** matched/mismatch/pending/provider unavailable/export failed/denied. **2.12:** financial action confirmation. **2.13:** separate view/resolve/issue/export. **2.14:** source modules ↔ Finance ↔ Audit. **2.15:** long jobs async.

#### 3. Reference / UX Layout Direction

Tabbed finance operations center sharing period/source filters.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Finance Operations**, Web Admin. Tabs reconciliation exceptions (expected/actual/delta), tax period, invoices/provider history, export jobs. States matched/mismatch/partial, provider timeout/fail, invoice adjusted/cancelled, export queued/failed/expired, denied. Actions resolve/issue/adjust/export by separate permissions with reason/confirmation. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 24 / US-FIN-06,07; FR-24; UC-FIN-02; EPIC 07,12,13 source reconciliations.

### ADM-049 — Executive Dashboard

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Executive · **Module:** DSS · **Entry:** admin home · **Next:** ADM-047/050~052.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Nắm tình hình doanh thu, Order, Customer, Inventory và lợi nhuận theo cùng kỳ/định nghĩa. **2.2 Goals:** scan KPI/trends; compare period; drill. **2.3 Preconditions:** Dashboard release/permission. **2.4 Flow:** set period → load cards/trends/freshness → inspect anomalies → drill module. **2.5 Alternative:** missing/late source, Finance-DSS definition conflict, no data. **2.6 Sections:** period/scope; KPI cards; trend/comparison; source quality/freshness; attention items; drill links. **2.7 Data:** only defined KPIs, value/delta/definition/source/update. **2.8:** period/channel if authorized. **2.9:** drill, compare. **2.10:** no autonomous decisions; missing/estimated explicit. **2.11:** loading/empty/partial/stale/failed/denied. **2.13:** Executive. **2.14:** Dashboard → Finance/Customer/Supply/Insights. **2.15:** concise overview, not duplicating full analytics.

#### 3. Reference / UX Layout Direction

Executive KPI dashboard with drill-down and data-quality strip.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Executive Dashboard**, Web Admin. Period/scope; KPI revenue/orders/customers/inventory/profit only after definition; value/delta/source/freshness; trends; attention items; drill links. States empty, partial/stale, source failed, metric definition conflict, denied. Missing is not zero. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 25 / US-DSS-01; FR-25; UC-DSS-01.

### ADM-050 — Customer Analytics

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Marketing, Executive · **Module:** DSS Customer · **Entry:** DSS nav · **Next:** ADM-039/041.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Phân tích RFM/churn có version, explanation, privacy và chuyển insight sang Campaign. **2.2 Goals:** inspect segments/risk; filter; controlled export; create Campaign audience link. **2.3 Preconditions:** GĐ3, consent/privacy and model version. **2.4 Flow:** select run/window → view segment distribution → drill aggregate/customer permitted detail → handoff. **2.5 Alternative:** Guest not merged, refund/cancel correction, insufficient/stale/model drift, suppressed Customer. **2.6 Sections:** run/quality; RFM segments; churn risk; explanation; aggregate trends; governed export/handoff. **2.7 Data:** segment/risk, score components, model/config version, source period, confidence/freshness. **2.8:** segment/risk/period/channel/quality. **2.9:** drill, export/handoff per consent; no automatic outreach. **2.10:** privacy/consent; small groups protected; AI result advisory. **2.11:** insufficient/stale/partial/failed/denied. **2.13:** export permission separate. **2.14:** Analytics → Campaign. **2.15:** explainability near scores.

#### 3. Reference / UX Layout Direction

Segment analytics with model/version/quality context and controlled activation.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Customer Analytics**, Web Admin. Run/window/model quality; RFM distribution/table; churn risk and explanation; aggregate trend; filters; governed export/Campaign handoff. States Guest/unmerged, insufficient, stale/drift, partial/failed, privacy-suppressed, denied. No automatic campaign send. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 25 / US-DSS-02,03; FR-25,26; UC-DSS-02.

### ADM-051 — Product & Supply Analytics

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Supply Manager, Executive, Sales per view · **Module:** DSS Supply · **Entry:** DSS/Inventory · **Next:** ADM-012/054.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** So sánh Product performance, SKU velocity, forecast và Batch expiry risk để hỗ trợ planning. **2.2 Goals:** identify fast/slow/declining; view forecast; inspect risk; handoff to supply plan. **2.3 Preconditions:** GĐ3 and quality data. **2.4 Flow:** set period/level → performance/velocity → forecast intervals → expiry risk → create planning context. **2.5 Alternative:** new Product, stockout distortion, returns, missing HSD/history, stale model. **2.6 Sections:** Product ranking/trend; velocity windows; forecast/version/confidence; expiry Batch risk; data quality; handoff. **2.7 Data:** Product/SKU/Batch, sales/valid orders, inventory, velocity, forecast horizon/range, risk, freshness. **2.8:** period/channel/Product/SKU/risk/quality. **2.9:** drill Batch/Product; pass context to Supply Plan. **2.10:** analytics advisory; stockout/returns identified. **2.11:** insufficient/stale/partial/failed/denied. **2.13:** role-dependent details. **2.14:** DSS → Inventory/Procurement. **2.15:** confidence/assumptions visible.

#### 3. Reference / UX Layout Direction

Tabbed product/supply analytics sharing Product–SKU–Batch hierarchy.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Product & Supply Analytics**, Web Admin. Tabs product performance, SKU velocity, demand forecast, Batch expiry risk; common filters; ranking/trends; forecast horizon/range/confidence/version; data quality/freshness; links Batch/Supply Plan. States new/no history, stockout distortion, missing HSD, stale/partial/model failed, denied. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 25 / US-DSS-04~07; FR-25,26; UC-DSS-03.

### ADM-052 — Strategic Insights

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Executive; Sales/Marketing by insight · **Module:** DSS · **Entry:** DSS nav · **Next:** ADM-033/039/054.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Trình bày association/seasonality/recommendation với evidence, confidence, trade-off và decision/outcome; không tự hành động. **2.2 Goals:** inspect insight; accept/reject for planning; track outcome. **2.3 Preconditions:** GĐ3, valid data/model. **2.4 Flow:** choose type/period → view cards/evidence → drill → record decision → handoff to Combo/Campaign/Supply Plan → later outcome. **2.5 Alternative:** small sample, promotion confounder, conflicting goals, stale/unexplainable recommendation. **2.6 Sections:** insight queue; evidence/metrics/sample; recommendation/trade-offs/confidence/expiry; decision/reason; handoff/outcome. **2.7 Data:** Product associations, seasonal periods, source/version, confidence, expected/observed outcome. **2.8:** type/status/period/confidence. **2.9:** accept/reject, create draft context only, export if permitted. **2.10:** human decision required; no automatic coupon/message/price/inventory change. **2.11:** insufficient/stale/conflicting/unexplainable/accepted/rejected/expired. **2.13:** role. **2.14:** Insight → Combo/Campaign/Plan. **2.15:** evidence precedes CTA.

#### 3. Reference / UX Layout Direction

Decision-support cards plus evidence drill-down and outcome log.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Strategic Insights**, Web Admin. Insight queue; association/seasonality metrics and sample; recommendation evidence/trade-offs/confidence/expiry; accept/reject with reason; create draft handoff to Combo/Campaign/Supply Plan; outcome tracking. States insufficient, stale, conflicting, unexplainable, expired. Never execute business action automatically. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 25 / US-DSS-08~10; FR-26; UC-DSS-04.

### ADM-053 — Supplier Directory

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Supply Manager · **Module:** Procurement · **Entry:** procurement/OCOP · **Next:** ADM-054/035.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý Supplier/hộ/HTX, contacts, contracts, pricing policy, status và traceability source. **2.2 Goals:** search/create/update/suspend; inspect history/open PO. **2.3 Preconditions:** GĐ3 permission. **2.4 Flow:** list → create/select → enter legal/contact/contract/price → validate duplicate/effectivity → save/status action. **2.5 Alternative:** possible duplicate, expired contract, sensitive price restricted, suspend with open PO. **2.6 Sections:** directory; identity/legal/contact; contracts/documents; pricing policy; status/open PO; traceability fields/history. **2.7 Columns:** Supplier/type/status/contracts expiry/open PO/updated. **2.8:** name/type/status/contract expiry. **2.9:** create/update/suspend; merge policy OPEN. **2.10:** historical PO/Batch source retained; restricted prices/docs. **2.11:** active/suspended/contract expiring/duplicate candidate/denied. **2.12:** suspend impact confirm. **2.13:** field/document permissions. **2.14:** Supplier → PO/OCOP/Audit. **2.15:** sensitive panels isolated.

#### 3. Reference / UX Layout Direction

Supplier master-data directory with contracts and source-data tabs.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Supplier Directory**, Web Admin. List Supplier/type/status/contract expiry/open PO; detail legal/contact; contracts/docs; pricing policy; traceability fields; history. States duplicate candidate, contract expiring, suspended with open PO, restricted data, conflict. Actions create/update/suspend with impact; preserve historical source. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 27 / US-PROC-01; EPIC 19 / US-OCOP-05; FR-29; UC-PROC-01.

### ADM-054 — Supply Plan & Purchase Order

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Supply Manager · **Module:** Procurement · **Entry:** procurement/DSS · **Next:** ADM-055/053/045.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Lập scenario từ forecast/tồn/open PO, override có lý do và chuyển dòng thành versioned PO qua approval/issue. **2.2 Goals:** compare scenarios; create plan; generate/edit PO; submit/issue/cancel per authority. **2.3 Preconditions:** GĐ3 data/Supplier/item. **2.4 Flow:** load demand/inventory → scenario/net need → manual override → approve plan → create PO draft → validate Supplier/lines/terms → approve/issue → acknowledge/receive. **2.5 Alternative:** stale forecast, MOQ/pack conflict, missing item, Supplier inactive, partial received/cancel. **2.6 Sections:** planning assumptions/scenarios; need lines; approval; PO list/detail; line/terms; revision/timeline. **2.7 Columns:** item, forecast, stock, open PO, net need, proposed/override; PO/Supplier/status/total/delivery/received. **2.8:** horizon/scenario/status/Supplier/item. **2.9:** override reason, submit/approve, create/revise/issue/cancel PO. **2.10:** thresholds/workflow OPEN; issued revision preserved. **2.11:** stale/incomplete/draft/pending/issued/acknowledged/partial/closed/cancel conflict. **2.12:** issue/cancel confirm. **2.13:** separation OPEN. **2.14:** DSS → Plan/PO → Receiving. **2.15:** planning and PO tabs share lineage.

#### 3. Reference / UX Layout Direction

Scenario planning worksheet connected to a versioned PO lifecycle.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Supply Plan & Purchase Order**, Web Admin. Planning tab forecast/inventory/open PO/net need/scenario/override reason/approval; PO tab Supplier, lines, terms, delivery, revision, approval/issue/acknowledgement/receipt timeline. States stale/incomplete, MOQ conflict, Supplier inactive, draft/pending/issued/partial/closed/cancel conflict. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 27 / US-PROC-02,03; EPIC 25 / US-DSS-06; FR-29; UC-PROC-02.

### ADM-055 — PO Receiving & Variance

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** Warehouse Staff · **Module:** Procurement Receiving · **Entry:** issued PO · **Next:** ADM-010/009/054.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Đối chiếu nhiều đợt nhận với PO và xử lý shortage/overage/wrong/damaged/HSD trước nhập kho. **2.2 Goals:** select PO line; record receipt/quality/Batch; log variance; complete PO when valid. **2.3 Preconditions:** valid open PO. **2.4 Flow:** open PO → actual lines → quality/Batch/date/evidence → variance decisions → post Stock Receipt → update received/remaining. **2.5 Alternative:** partial, over/under/wrong unit, damaged/poor HSD, duplicate receipt, closed PO. **2.6 Sections:** PO header/remaining; expected-actual lines; quality/Batch; variance queue; receipt timeline; completion. **2.7 Columns:** item/ordered/prior received/this receipt/remaining/variance/Batch/HSD/quality. **2.9:** save/submit receipt; flag variance; complete when criteria. **2.10:** retry idempotent; no stock update until valid posting. **2.11:** partial/variance/quarantine/duplicate/posted/closed conflict. **2.12:** post/complete confirm. **2.13:** Warehouse; variance decision authority OPEN. **2.14:** PO → Receiving → Stock Receipt/Batch. **2.15:** accumulated quantities visible.

#### 3. Reference / UX Layout Direction

PO receiving workbook with cumulative receipt and variance queue.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **PO Receiving & Variance**, Web Admin. PO header/remaining; expected-vs-actual lines including prior/this/remaining; Batch/NSX/HSD; quality/evidence; variance queue; receipt timeline; completion. States partial, over/under/wrong/damaged, poor HSD, duplicate, posted, closed conflict. Post creates Stock Receipt once. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 27 / US-PROC-04; EPIC 09 / US-INV-02; FR-29; UC-PROC-03.

### ADM-056 — Internal Alert Center

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** authorized internal staff · **Module:** Notification · **Entry:** header/admin home · **Next:** source screen.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Tập trung alert Order khẩn/Ticket quá hạn/Batch cận date, acknowledge mà không giả là source đã giải quyết. **2.2 Goals:** filter; open source; acknowledge; see escalation. **2.3 Preconditions:** source permission. **2.4 Flow:** list by priority/SLA → inspect → open source/action there → acknowledge; source resolution closes alert. **2.5 Alternative:** duplicate/coalesced, source resolved, user loses permission, escalation overdue. **2.6 Sections:** counters/tabs; alert table; source preview; owner/SLA/escalation; history. **2.7 Columns:** priority/type/source/ref/created/SLA/owner/ack/resolution/escalation. **2.8:** unread/priority/type/owner/SLA/status. **2.9:** acknowledge, assign if allowed, open source. **2.10:** ack ≠ resolve; dedup; permission recheck. **2.11:** new/ack/escalated/resolved/expired/denied/partial. **2.12:** none. **2.13:** source scope. **2.14:** Alert → Order/Ticket/Expiry/Audit. **2.15:** priority not color-only.

#### 3. Reference / UX Layout Direction

Operational inbox with SLA/escalation and deep links.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Internal Alert Center**, Web Admin. Counters/tabs; table priority/type/source/ref/time/SLA/owner/ack/resolution/escalation; source preview/history. Filters unread/priority/type/owner/SLA. States new, acknowledged, escalated, resolved, duplicate-coalesced, permission lost. Ack is not resolution; deep links recheck permission. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 28 / US-NOTI-04; EPIC 08/09/16 source alerts; FR-30; UC-NOTI-02.

### ADM-057 — Template, Route & Delivery Operations

#### 1. Web Module & Screen

**Web Type:** Admin · **Actor:** authorized Notification Admin/Operator · **Module:** Notification · **Entry:** notification admin · **Next:** ADM-045/056.

#### 2. Business Requirements & User Flow

**2.1 Mục tiêu:** Quản lý version/template/route và điều tra delivery attempt mà không resend tùy tiện. **2.2 Goals:** search delivery; inspect attempts/provider callback; configure/preview/test approved template/route. **2.3 Preconditions:** distinct permissions; exact admin role OPEN. **2.4 Flow:** Template tab create revision/locale/preview/test/approve; Delivery tab search source/recipient masked/channel → timeline → safe retry/resend if authorized. **2.5 Alternative:** invalid contact, suppressed/opt-out, provider unavailable, duplicate/out-of-order callback, expired endpoint, template unapproved. **2.6 Sections:** template list/editor/version/preview/approval; routing/fallback; delivery search/table/detail attempts; provider health/error; audit. **2.7 Columns:** notification/source/recipient masked/channel/template version/status/attempts/last error/time. **2.8:** source/ref/channel/status/provider/time/template. **2.9:** create revision/test/submit; safe retry/resend; no override consent. **2.10:** classification/consent, dedup, masking, audit. **2.11:** queued/sent/delivered/failed/suppressed/delayed/provider unavailable. **2.12:** test/retry impact confirm. **2.13:** template vs operations perms. **2.14:** Notification Ops → source/Audit. **2.15:** delivery timeline distinguishes acceptance vs delivered.

#### 3. Reference / UX Layout Direction

Tabbed notification administration and delivery observability workspace.

#### Stitch AI Design Brief

> [STITCH AI DESIGN BRIEF] Thiết kế **Template, Route & Delivery Operations**, Web Admin. Template versions/locales/preview/test/approval; route/fallback configuration; delivery table source/recipient masked/channel/version/status/attempts/error; attempt/provider timeline. States queued/sent/delivered/failed/suppressed/delayed/duplicate/provider unavailable/expired endpoint. Retry/resend only by permission and consent; audit actions. [/STITCH AI DESIGN BRIEF]

#### Requirement Sources

EPIC 28 / US-NOTI-01~04; FR-30; UC-NOTI-01,02.

## 7. End-to-End Screen Flows

### FLOW-D2C-01 — Purchase Flow

`D2C-001 Home` → `D2C-002 Listing/Search` → `D2C-004 Product Detail` → `D2C-005 Cart` → `D2C-006 Checkout` → `D2C-007 Payment` → `D2C-008 Confirmation` → `D2C-015 Order Detail/Tracking` → `D2C-021 Review` hoặc `D2C-017 Return Request`.

Nhánh: AI discovery dùng `D2C-003`; COD bỏ qua bước thực hiện QR nhưng vẫn đi qua trạng thái giải thích ở `D2C-007/008`; Guest theo dõi qua `D2C-016`.

### FLOW-D2C-02 — Authentication & Account

`D2C-009 Login` ↔ `D2C-010 Register` / `D2C-011 Recovery` → `D2C-012 Profile` → `D2C-013 Addresses` / `D2C-014 Orders` / `D2C-023 Loyalty` / `D2C-029 Notifications`.

Guest Checkout: `D2C-005` → `D2C-006` mà không bắt buộc qua Login. Link Guest Order: `D2C-008/016` → `D2C-009/010` → verified link → `D2C-015`.

### FLOW-D2C-03 — Promotion, Loyalty & Gifting

`D2C-022 Promotion/Combo` → `D2C-004/005` → `D2C-006 Checkout (coupon/points/gift recipient/message/hide price)` → `D2C-008` → `D2C-015`.

Reorder: `D2C-015` → preview current sellability/price → `D2C-005`.

### FLOW-D2C-04 — Support, Return & Refund

`D2C-015 Order Detail` → `D2C-017 Return Request` → `D2C-018 Case Detail` → notification/refund result. Hỗ trợ: `D2C-015/018` → `D2C-019 Support Form` → `D2C-020 Ticket Detail`.

### FLOW-D2C-05 — B2B

`D2C-024 Quote Request` → Sales xử lý tại `ADM-034` → `D2C-025 Quote Detail/Acceptance` → revalidation → `D2C-007 Payment` hoặc `D2C-015 Order Detail` theo điều khoản chưa chốt.

### FLOW-D2C-06 — OCOP & Content

QR/PDP → `D2C-026 Traceability`; `D2C-001` → `D2C-027 Content Hub` → `D2C-028 Article` → `D2C-004 Product` / `D2C-026 Traceability`.

### FLOW-ADM-01 — Order to Fulfillment

`ADM-005 Order Operations` → `ADM-006 Order Detail` → `ADM-007 Payment Reconciliation` khi cần → `ADM-013 Packing Queue` → `ADM-014 Packing Task` → `ADM-016 Shipment Operations` → `ADM-017 Delivery Workspace` hoặc 3PL sync → `ADM-006 Completion`.

### FLOW-ADM-02 — Inventory & Procurement

MVP: `ADM-008 Inventory` → `ADM-010 Stock Receipt` / `ADM-011 Issue & Adjustment` → `ADM-009 Batch` → `ADM-012 Expiry/FEFO`.

GĐ3: `ADM-051 Supply Analytics` → `ADM-054 Supply Plan & PO` → `ADM-055 PO Receiving & Variance` → `ADM-010 Stock Receipt` → `ADM-009 Batch`.

### FLOW-ADM-03 — Return to Refund

`ADM-026 Return Queue/Review` → `ADM-015 Evidence` khi có → `ADM-027 Returned Goods Inspection` theo policy → `ADM-028 Refund` → `ADM-048 Finance` → `ADM-045 Audit`.

### FLOW-ADM-04 — Marketplace

`ADM-019 Marketplace Order Center` → `ADM-020 Listing Mapping` nếu cần → `ADM-006 Order Detail` → fulfillment chung → `ADM-021 Settlement` → `ADM-048 Finance`.

### FLOW-ADM-05 — Offline Sales

`ADM-022 Offline Receiving` → `ADM-023 POS Sale` / `ADM-024 Stock & Return` → `ADM-025 Shift Report & Reconciliation` → `ADM-011 Inventory Adjustment` / `ADM-048 Finance` khi được duyệt.

### FLOW-ADM-06 — Content & Marketing

`ADM-037 Content Workspace` → `ADM-038 Media/Approval/Publication` → `D2C-027/028`; `ADM-039 Campaign Workspace` → `ADM-040 Approval/Calendar` → `ADM-041 Performance/Integration` → `ADM-047 Finance` / `ADM-052 Insights`.

### FLOW-ADM-07 — Governance

`ADM-004 Employee Profile` → `ADM-042 Employee Account` → `ADM-043 Role & Permission` → `ADM-044 Access Review` → `ADM-045 Audit Explorer` → `ADM-046 Integrity/Export`.

### FLOW-ADM-08 — Analytics to Action

`ADM-049 Executive Dashboard` → `ADM-047/050/051/052` → human review → tạo **draft context** tại `ADM-033 Combo`, `ADM-039 Campaign` hoặc `ADM-054 Supply Plan`; không có bước tự động thay đổi business data.

## 8. Global UI Requirements

1. **Authentication/ownership:** Guest Checkout được phép; screen account yêu cầu đăng nhập. Guest Order/Ticket cần bằng chứng sở hữu ngoài mã tham chiếu. Social provider chỉ hiển thị khi enabled. [EPIC 01, BR-AUTH-01~04]
2. **Authorization:** Mọi Admin screen/action kiểm tra Role/Permission/scope ở thời điểm thực hiện; deep link kiểm tra lại quyền. View không mặc nhiên bao gồm create/update/approve/refund/export. [EPIC 22, NFR-05]
3. **Privacy:** Mask contact, address, payment, Supplier và HR fields theo vai trò. Packing Video và media nội bộ không dùng public/raw storage URL. [NFR-04,08,09]
4. **Currency/tax:** Mọi breakdown phân biệt merchandise, discount, shipping, tax/fee và final total khi có; currency, rounding, tax inclusion vẫn là open question nên UI không hard-code rule. [EPIC 04,06,17,18,24]
5. **Availability:** Sellable/available là dữ liệu theo SKU và Inventory; reserved/non-sellable/expired không được trình bày như có thể mua. Cart không giữ tồn; Checkout mới reserve. [BR-PROD-02, BR-BATCH-05, BR-ORDER-02]
6. **Status semantics:** Order, Payment, Fulfillment/Shipment, Return Case và Refund có nhãn/timeline riêng. Không dùng một trạng thái tổng hợp gây hiểu nhầm. [EPIC 07,08,11,14]
7. **Snapshots:** Order, Quote acceptance, campaign approval và public content/traceability đều gắn với exact revision/snapshot. Thay đổi nguồn sau đó không sửa hồi tố lịch sử. [EPIC 06,08,18~21]
8. **Idempotency feedback:** Nút mutation có processing/disabled state, nhưng UI không được coi đây là cơ chế chống trùng duy nhất. Khi retry, hiển thị kết quả nhất quán của request đã xử lý. [EPIC 06,07,09,11~14]
9. **Notifications:** Lỗi gửi thông báo không rollback Order/Shipment/Refund. Transactional/security/operational/marketing phải được phân loại; marketing opt-out không chặn thông báo bắt buộc. [EPIC 28]
10. **Audit-facing UX:** Action nhạy cảm yêu cầu reason/confirmation khi rule quy định; hiển thị phạm vi ảnh hưởng. Audit Event không có nút sửa/xóa thủ công. [BR-AUDIT, EPIC 23]
11. **Loading/error/performance:** Các thao tác phổ biến phải có phản hồi phù hợp; mọi screen dài hoặc tích hợp ngoài cần loading, timeout/error, retry an toàn và partial/stale state. Không hiển thị dữ liệu thiếu như giá trị 0. [NFR-01,03,10,11]
12. **Responsive/accessibility:** D2C mobile-first; các bảng Admin rộng hỗ trợ horizontal overflow/column priority; state/priority không chỉ truyền đạt bằng màu; media Content cần alternative text. [NFR-02, EPIC 20]
13. **AI:** AI output có nhãn, nguồn/độ tin cậy/thiếu dữ liệu khi relevant; người dùng phải xác nhận trước khi đưa vào Draft. AI không tự đổi giá, tồn, Order, Refund hay gửi phản hồi. [US-AI-01~04,06]
14. **External integrations:** Provider unavailable, timeout, duplicate/out-of-order callback và status chưa map có state riêng; không tự suy ra success. [NFR-11, EPIC 07,11,12,21,24,28]

## 9. Requirement → UI Traceability Matrix

| Source | Requirement/User Story | Screen ID | UI manifestation |
|---|---|---|---|
| EPIC 01 | US-AUTH-01 Guest Checkout | D2C-006,009 | Checkout không ép đăng nhập; Login có lối tiếp tục Guest |
| EPIC 01 | US-AUTH-02 Registration | D2C-010 | Registration/verification states |
| EPIC 01 | US-AUTH-03 Login | D2C-009 | Login và safe return destination |
| EPIC 01 | US-AUTH-04 Social Login | D2C-009 | Provider buttons conditional |
| EPIC 01 | US-AUTH-05 Password Recovery | D2C-011 | Recovery/reset flow |
| EPIC 01 | US-AUTH-06 Link Guest Order | D2C-008,010,016,015 | Verified linking path |
| EPIC 02 | US-USER-01 Customer Profile | D2C-012 | Own-profile form |
| EPIC 02 | US-USER-02 Customer Address | D2C-013,006 | Address book and checkout selector |
| EPIC 02 | US-USER-03 Employee Profile | ADM-004 | HR profile workspace |
| EPIC 02 | US-USER-04 Employee Status | ADM-004 | Employment status in profile; access effect remains open |
| EPIC 02 | US-USER-05 Customer Account Status | ADM-003 | Customer status action/history |
| EPIC 03 | US-DISC-01 Category Browse | D2C-001,002 | Category entry and public product listing |
| EPIC 03 | US-DISC-02 Keyword Search | D2C-001,002 | Search and results/no-results |
| EPIC 03 | US-DISC-03 Product Filters | D2C-002 | Applied filters and empty state |
| EPIC 03 | US-DISC-04 Bestseller/Recommendation | D2C-001,004 | Featured/recommendation sections with fallback |
| EPIC 03 | US-DISC-05 Brand/Huế Story | D2C-001,027,028 | Published editorial content |
| EPIC 03 | US-AI-01 Search Assistant | D2C-003 | Conversational discovery |
| EPIC 03 | US-AI-04 Product Recommendation | D2C-001,003,004 | Explained recommendation/fallback |
| EPIC 04 | US-PROD-01 Create/Update Product | ADM-001 | Product editor |
| EPIC 04 | US-PROD-02 Variant/SKU | ADM-001 | SKU table/editor |
| EPIC 04 | US-PROD-03 Select SKU | D2C-004 | Required SKU selector and availability |
| EPIC 04 | US-PROD-04 Food Information | D2C-004,ADM-001 | Public info and source editor |
| EPIC 04 | US-PROD-05 SKU Price | ADM-002,D2C-004 | Price administration/current display |
| EPIC 04 | US-PROD-06 Product Status | ADM-001,D2C-002,D2C-004,D2C-005 | Lifecycle action and unavailable states |
| EPIC 04 | US-PROD-07 Omnichannel Pricing | ADM-002 | Channel/effectivity matrix |
| EPIC 05 | US-CART-01 Add to Cart | D2C-004,005 | Add action and line creation |
| EPIC 05 | US-CART-02 Change Quantity | D2C-005 | Quantity validation/current stock |
| EPIC 05 | US-CART-03 Remove Item | D2C-005 | Remove/empty state |
| EPIC 05 | US-CART-04 Subtotal | D2C-005 | Line totals/subtotal disclosure |
| EPIC 06 | US-CHK-01 Shipping Information | D2C-006,013 | Guest form/saved address |
| EPIC 06 | US-CHK-02 Shipping Fee | D2C-006 | Quote, recalculation, unavailable states |
| EPIC 06 | US-CHK-03 Stock Reservation | D2C-006,007 | Processing/failed/expired reservation feedback |
| EPIC 06 | US-CHK-04 Review & Confirm | D2C-006 | Snapshot/breakdown/idempotent confirmation |
| EPIC 06 | US-CHK-05 Promotion Code | D2C-006 | Coupon apply/remove/revalidation |
| EPIC 07 | US-PAY-01 Transfer/QR | D2C-007 | QR amount/reference/status |
| EPIC 07 | US-PAY-02 COD | D2C-006~008,ADM-017 | COD selection/not-paid and delivery collection context |
| EPIC 07 | US-PAY-03 View Payment Status | D2C-007,008,015,016 | Customer/Guest status views |
| EPIC 07 | US-PAY-04 Staff Check Payment | ADM-006,007 | Payment beside Order, no manual Paid shortcut |
| EPIC 07 | US-PAY-05 Reconcile Transaction | ADM-007,048 | Exception comparison/resolution |
| EPIC 08 | US-ORD-01 Order History | D2C-014,015 | Own Order list/detail |
| EPIC 08 | US-ORD-02 Current Order Status | D2C-015,016 | Separate current states/timeline |
| EPIC 08 | US-ORD-03 Status Notification | D2C-029,015 | Feed/deep link and source status |
| EPIC 08 | US-ORD-04 Cancel When Eligible | D2C-015,D2C-017 | Direct cancel or Case route |
| EPIC 08 | US-ORD-05 Admin Order List | ADM-005,006 | Operations table/detail/actions |
| EPIC 08 | US-ORD-06 SLA Priority | ADM-005,013,056 | SLA queue/alert |
| EPIC 09 | US-INV-01 Stock by SKU | ADM-008 | Balance components and Batch drill-down |
| EPIC 09 | US-INV-02 Stock Receipt | ADM-010 | Receipt worksheet |
| EPIC 09 | US-INV-03 Stock Issue | ADM-011 | Controlled issue entry |
| EPIC 09 | US-INV-04 Batch/Lot | ADM-009 | Batch dossier/history |
| EPIC 09 | US-INV-05 NSX/HSD | ADM-009,012 | Date validation/expiry state |
| EPIC 09 | US-INV-06 Expiry Alert | ADM-012,056 | Risk list/internal alert |
| EPIC 09 | US-INV-07 Consumption Speed | ADM-012 | Consumption view |
| EPIC 09 | US-INV-08 FEFO | ADM-012,014 | Recommendation/actual pick context |
| EPIC 09 | US-INV-09 Damage/Loss/Adjustment | ADM-011 | Adjustment with reason/evidence/approval state |
| EPIC 09 | US-INV-10 Sales Expiry Alert | ADM-012,031,056 | Sales/CS coordination |
| EPIC 10 | US-PACK-01 Packing Queue | ADM-013 | Ready-task queue |
| EPIC 10 | US-PACK-02 Packing Detail | ADM-014 | Order snapshot/pick list |
| EPIC 10 | US-PACK-03 Checklist | ADM-014 | Required check states |
| EPIC 10 | US-PACK-04 Packing Video | ADM-014 | Capture/upload stored status |
| EPIC 10 | US-PACK-05 Link Video to Order | ADM-014,015 | Order/Task/Package evidence relation |
| EPIC 10 | US-PACK-06 Search Packing Video | ADM-015 | Secure evidence viewer |
| EPIC 10 | US-PACK-07 Deadline Priority | ADM-013 | SLA ordering/blockers |
| EPIC 10 | US-PACK-08 Complete Packing | ADM-014 | Completion eligibility/idempotency |
| EPIC 11 | US-SHIP-01 Create Shipment | ADM-016 | Shipment creation/label/handoff |
| EPIC 11 | US-SHIP-02 Customer Tracking | D2C-015,016 | Package timeline |
| EPIC 11 | US-SHIP-03 Assigned Delivery List | ADM-017 | Assignment-scoped task list |
| EPIC 11 | US-SHIP-04 Delivery Status Update | ADM-017 | Valid next steps/failure reason |
| EPIC 11 | US-SHIP-05 Delivery Rate | ADM-018 | KPI/sample/trend |
| EPIC 11 | US-SHIP-06 3PL Status Sync | ADM-016 | Sync timeline/exception state; background callback has no separate UI |
| EPIC 12 | US-MKT-01 Marketplace Orders | ADM-019 | Platform/shop/source/internal list |
| EPIC 12 | US-MKT-02 Internal Fulfillment | ADM-019,006 | Validation then shared fulfillment |
| EPIC 12 | US-MKT-03 Sync Status | ADM-019,020 | Sync/error/retry visibility |
| EPIC 12 | US-MKT-04 Reconcile Order/Revenue/Fee | ADM-021 | Component settlement table |
| EPIC 12 | US-MKT-05 Marketplace Revenue | ADM-021,047 | Gross/net/reconciled view |
| EPIC 12 | US-MKT-06 Source ID/Dedup | ADM-019 | Reference/duplicate state; dedup processing is background |
| EPIC 13 | US-OFF-01 Receive Stock | ADM-022 | Expected/actual handover |
| EPIC 13 | US-OFF-02 Record Sale | ADM-023 | POS transaction/receipt |
| EPIC 13 | US-OFF-03 Stock/Damage/Return | ADM-024 | Count/disposition flows |
| EPIC 13 | US-OFF-04 Periodic Report | ADM-025 | Shift/period submission |
| EPIC 13 | US-OFF-05 Revenue/Stock by Staff/Point | ADM-025 | Performance/drilldown tab |
| EPIC 13 | US-OFF-06 Reconciliation | ADM-025,048 | Goods/cash differences |
| EPIC 14 | US-RET-01 Cancel Request | D2C-015,017,ADM-026 | Direct cancel vs Cancellation Case |
| EPIC 14 | US-RET-02 Return Request | D2C-017,018 | Line/quantity/reason/case |
| EPIC 14 | US-RET-03 Complaint Evidence | D2C-017,018 | Media upload/result |
| EPIC 14 | US-RET-04 CS Views Order/Evidence | ADM-026,015 | Case context/evidence viewer |
| EPIC 14 | US-RET-05 Approve/Reject | ADM-026 | Decision panel/history |
| EPIC 14 | US-RET-06 Refund | ADM-028,D2C-018 | Execution/customer result |
| EPIC 14 | US-RET-07 Packing Video Investigation | ADM-015,026 | Secure Case-linked evidence |
| EPIC 15 | US-REV-01 Review Purchased Product | D2C-021 | Eligible Order Item review |
| EPIC 15 | US-REV-02 Review Images | D2C-021 | Image upload/progress/moderation state |
| EPIC 15 | US-REV-03 Verified Review | D2C-004,D2C-021 | System-assigned badge and SKU context |
| EPIC 15 | US-REV-04 Negative Review Queue | ADM-029,030 | Review queue/Ticket handoff |
| EPIC 15 | US-REV-05 Policy Moderation | ADM-029 | Hide/restore/media/reply history |
| EPIC 15 | US-REV-06 Review Analytics | ADM-029 | Distribution/trend/sample |
| EPIC 16 | US-CS-01 Submit Support Request | D2C-019,020 | Ticket intake/customer portal |
| EPIC 16 | US-CS-02 Priority Queue | ADM-030 | SLA queue/claim/transfer |
| EPIC 16 | US-CS-03 Conversation History | ADM-030,D2C-020 | Multichannel/internal vs public timeline |
| EPIC 16 | US-CS-04 Transaction Context | ADM-030 | Read-only Order/Payment/Shipment/Return context |
| EPIC 16 | US-CS-05 Unhandled Alert | ADM-030,056 | SLA alert/escalation |
| EPIC 16 | US-CS-06 Expiry Coordination | ADM-031 | Read-only Inventory source/action log |
| EPIC 16 | US-AI-02 Support Assistant | ADM-030 | Cited summary/draft; human sends |
| EPIC 16 | US-AI-06 Knowledge Assistant | ADM-030 | Permission-aware knowledge answer |
| EPIC 17 | US-PROMO-01 Use Coupon | D2C-006,ADM-032 | Checkout validation/coupon source |
| EPIC 17 | US-PROMO-02 Create Coupon | ADM-032 | Rule builder/lifecycle |
| EPIC 17 | US-PROMO-03 Create Combo | ADM-033,D2C-022 | Component builder/public offer |
| EPIC 17 | US-PROMO-04 View Promotion | D2C-001,004,022 | Active terms/status |
| EPIC 17 | US-LOY-01 Earn Points | D2C-023 | Balance/ledger; accrual itself background |
| EPIC 17 | US-LOY-02 Redeem Points | D2C-006,023 | Checkout reservation and wallet result |
| EPIC 17 | US-LOY-03 Reorder | D2C-015,005 | Current-data preview then Cart |
| EPIC 18 | US-B2B-01 Quote Request | D2C-024,ADM-034 | Multi-line request/Sales queue |
| EPIC 18 | US-B2B-02 Logo Upload | D2C-024,ADM-034 | Versioned file upload/view |
| EPIC 18 | US-B2B-03 Manage B2B Request | ADM-034 | SLA, versioned builder/issue |
| EPIC 18 | US-B2B-04 Accept Quote to Order | D2C-025,ADM-034 | Exact-version acceptance/conversion |
| EPIC 18 | US-B2B-05 Company Invoice Data | D2C-025,ADM-048 | Snapshot/provider result |
| EPIC 19 | US-OCOP-01 Scan QR | D2C-026 | Code resolution/invalid states |
| EPIC 19 | US-OCOP-02 Origin/Producer/Process | D2C-026,ADM-035 | Public evidence/source editor |
| EPIC 19 | US-OCOP-03 Certificates | D2C-026,ADM-035,036 | Scope/validity/verification |
| EPIC 19 | US-OCOP-04 Manage Product/Lot Origin | ADM-035 | Scope/revision workspace |
| EPIC 19 | US-OCOP-05 Supplier/Material Source | ADM-035,053 | Effective source links |
| EPIC 19 | US-OCOP-06 Verify Before Public | ADM-036 | Diff/provider/public preview/decision |
| EPIC 20 | US-CONTENT-01 Create Article | ADM-037,D2C-027,028 | Draft editor/public manifestation |
| EPIC 20 | US-CONTENT-02 Edit/Publish/Hide | ADM-037,038,D2C-028 | Revision/publication lifecycle |
| EPIC 20 | US-CONTENT-03 Article Media | ADM-038,D2C-028 | Library/usage/public media |
| EPIC 20 | US-CONTENT-04 Article SEO | ADM-037,D2C-028 | Metadata/preview/public result |
| EPIC 20 | US-CONTENT-05 Product SEO | ADM-037,D2C-004 | Product SEO panel/public PDP |
| EPIC 20 | US-CONTENT-06 URL/SEO | ADM-037,038,D2C-028 | Slug history/canonical/redirect state |
| EPIC 20 | US-CONTENT-07 Content Approval | ADM-038 | Revision diff/decision |
| EPIC 20 | US-AI-03 Content Assistant | ADM-037 | Selective AI suggestions |
| EPIC 21 | US-MKTG-01 Campaign Plan | ADM-039 | Campaign revision builder |
| EPIC 21 | US-MKTG-02 Content/Program for Campaign | ADM-039 | Linked brief/article/promotion/media readiness |
| EPIC 21 | US-MKTG-03 Submit Plan | ADM-039,040 | Exact revision approval request |
| EPIC 21 | US-MKTG-04 Approve/Reject Plan | ADM-040 | Authority/diff/decision |
| EPIC 21 | US-MKTG-05 Track KPI | ADM-041 | Target/actual/variance/freshness |
| EPIC 21 | US-MKTG-06 Import Channel Results | ADM-041 | Integration/mapping/retry states |
| EPIC 21 | US-MKTG-07 Marketing Cost vs Revenue | ADM-041,047 | ROI view with attribution warning |
| EPIC 22 | US-ADM-01 Create Employee Account | ADM-042 | Profile-linked creation/invite |
| EPIC 22 | US-ADM-02 Update Employee Account | ADM-042 | Account-only update/conflict |
| EPIC 22 | US-ADM-03 Lock Employee Account | ADM-042 | Scheduled/immediate lock impact |
| EPIC 22 | US-ADM-04 Role Management | ADM-043 | Role lifecycle |
| EPIC 22 | US-ADM-05 Permission Management | ADM-043 | Permission matrix |
| EPIC 22 | US-ADM-06 Assign Role | ADM-042,043,044 | Assignment/effective preview |
| EPIC 22 | US-ADM-07 View Employee Rights | ADM-044 | Effective permission provenance |
| EPIC 23 | US-AUDIT-01 Search History | ADM-045 | Search/table/scope |
| EPIC 23 | US-AUDIT-02 Before/After | ADM-045 | Field diff/masking |
| EPIC 23 | US-AUDIT-03 Source/Technical Location | ADM-045 | Source metadata/correlation |
| EPIC 23 | US-AUDIT-04 Business Reason | ADM-045 | Reason/annotation separation |
| EPIC 23 | US-AUDIT-05 Integrity Detection | ADM-046,056 | Verification/gap/alert |
| EPIC 24 | US-FIN-01 Revenue by Time | ADM-047 | Period KPI/trend |
| EPIC 24 | US-FIN-02 Revenue by Channel | ADM-047 | Channel breakdown/drill |
| EPIC 24 | US-FIN-03 Sales/Operating Cost | ADM-047 | Cost taxonomy/missing state |
| EPIC 24 | US-FIN-04 Profit by Product | ADM-047 | Product/SKU profitability |
| EPIC 24 | US-FIN-05 Revenue by Region | ADM-047 | Privacy-aware region analysis |
| EPIC 24 | US-FIN-06 Reconciliation/Tax | ADM-048 | Exception/tax period |
| EPIC 24 | US-FIN-07 Reports/Invoice | ADM-048 | Invoice/export job |
| EPIC 25 | US-DSS-01 Overview Dashboard | ADM-049 | KPI/freshness/drill |
| EPIC 25 | US-DSS-02 RFM | ADM-050 | Segment analytics |
| EPIC 25 | US-DSS-03 Churn Risk | ADM-050 | Risk/explanation/handoff |
| EPIC 25 | US-DSS-04 Product Performance | ADM-051 | Ranking/trend |
| EPIC 25 | US-DSS-05 SKU Velocity | ADM-051 | Velocity windows |
| EPIC 25 | US-DSS-06 Demand Forecast | ADM-051,054 | Forecast then planning context |
| EPIC 25 | US-DSS-07 Expiry Risk | ADM-051,012 | Analytic risk plus operational Batch data |
| EPIC 25 | US-DSS-08 Product Association | ADM-052,033 | Evidence then Combo draft |
| EPIC 25 | US-DSS-09 Seasonality | ADM-052,039 | Seasonal insight then Campaign draft |
| EPIC 25 | US-DSS-10 Strategic Recommendation | ADM-052 | Human decision/outcome tracking |
| EPIC 26 | US-GIFT-01 Different Recipient | D2C-006,015,ADM-014 | Recipient checkout/snapshot/packing |
| EPIC 26 | US-GIFT-02 Gift Message | D2C-006,015,ADM-014 | Message/preview/artifact |
| EPIC 26 | US-GIFT-03 Hide Price | D2C-006,015,ADM-014 | Preference and recipient artifact; finance unchanged |
| EPIC 27 | US-PROC-01 Supplier Profile | ADM-053 | Supplier directory/profile |
| EPIC 27 | US-PROC-02 Purchase Order | ADM-054 | PO draft/approval/issue |
| EPIC 27 | US-PROC-03 Supply Planning | ADM-054 | Forecast/scenario/net need |
| EPIC 27 | US-PROC-04 PO Receiving | ADM-055,010 | Expected/actual then stock receipt |
| EPIC 28 | US-NOTI-01 Email | D2C-008,011,ADM-057 | User feedback/delivery operations; sending is background |
| EPIC 28 | US-NOTI-02 SMS/Zalo | D2C-029,ADM-057 | Preference/status; sending is background |
| EPIC 28 | US-NOTI-03 In-app Push | D2C-029,ADM-057 | Feed/deep link/token/error visibility |
| EPIC 28 | US-NOTI-04 Internal Alert | ADM-056,057 | Alert center/operations |
| NFR / legacy | US-25 Observability | — | No direct product UI — external Grafana/Jaeger operations tooling |
| NFR / legacy | US-26 CI/CD | — | No direct UI — engineering pipeline/background requirement |

## 10. UI Coverage Audit

| Metric | Result |
|---|---:|
| Requirement files scanned | 77 |
| Content sources read/parsed | 52 |
| Epic files analyzed | 28/28 |
| User Stories detected in Epic files | 175 |
| Additional legacy User Stories in NFR | 2 |
| **Total User Stories detected** | **177** |
| User Stories mapped to at least one product screen | 175 |
| User Stories with no direct product UI | 2 |
| User Stories currently unmapped | 0 |
| Web D2C screens | 29 |
| Web Admin screens | 57 |
| Total screens | 86 |
| Open Questions / Requirement Gaps | 30 |
| Explicit cross-document conflicts among those questions | 8 |

Coverage notes:

- `US-25` Observability và `US-26` CI/CD nằm trong NFR dưới nhãn migrated legacy, không thuộc 28 Epic và không cần Web D2C/Admin screen. Chúng được ghi rõ là no-direct-UI thay vì tạo screen giả.
- Các story nền như webhook 3PL, Payment callback, notification sending, reservation/concurrency và dedup có UI manifestation ở loading/error/pending/duplicate-safe states của screen liên quan, nhưng xử lý nền không được dựng thành screen.
- `US-USER-04` chỉ có heading nhưng không có block chi tiết riêng trong EPIC 02; screen ADM-004 phản ánh Employee status ở mức hồ sơ và phần ảnh hưởng đến access vẫn là Open Question.
- Không phát hiện duplicate Screen ID. Overlap giữa Order, Ticket, Return, Payment, Packing và Shipping được giải quyết bằng nguồn dữ liệu + deep link, không cho từng workspace tự sửa entity nguồn ngoài quyền.

## 11. Open Questions & Requirement Gaps

### CRITICAL

#### OQ-01 — Reservation/Payment timeout và late payment **[CONFLICT]**

**Question:** TTL có cố định 15 phút cho mọi phương thức hay cấu hình theo method; xử lý payment đến sau `EXPIRED/CANCELLED` thế nào? **Why it matters:** quyết định countdown, error recovery, giữ/nhả tồn và Order state. **Affected Screen:** D2C-006~008, D2C-015, ADM-006~008. **Source:** UC-ORD-01, UC-PAY-02, Sprint risk/Draw.io ghi 15 phút; EPIC 06/08/09 để thời hạn và late-payment mở. **Suggested assumption:** hiển thị timeout từ dữ liệu runtime, không hard-code; late payment vào reconciliation, không tự Paid.

#### OQ-02 — State machine COD **[CONFLICT]**

**Question:** COD đi qua các trạng thái Order/Payment nào và khi nào coi đã thu tiền? **Why:** backlog state machine tuyến tính qua `PENDING_PAYMENT → PAID`, trong khi COD phải fulfillment trước khi thu. **Affected:** D2C-006~008,015; ADM-005~007,016,017. **Source:** EPIC 07/08 open decisions; Product Backlog state machine. **Suggested assumption:** tách Payment `COD_PENDING_COLLECTION` ở mức hiển thị khái niệm, nhưng không tạo status code chính thức trước PO duyệt.

#### OQ-03 — Order/Shipment/Return state taxonomy **[CONFLICT]**

**Question:** `RETURNED` của giao thất bại và `RETURNED` của hàng đổi trả có phải hai entity/state khác nhau; ma trận transition đầy đủ là gì? **Why:** action/labels/timeline và permission có thể sai. **Affected:** D2C-015,018; ADM-005,006,016,017,026~028. **Source:** Backlog state machine; EPIC 08/11/14 open decisions. **Suggested assumption:** UI luôn ghi rõ “Shipment returned” và “Returned goods received” theo entity, không hợp nhất.

#### OQ-04 — Return/exchange/refund policy

**Question:** cửa sổ, lý do, SKU/kênh, điều kiện hàng, phí return, exchange có trong MVP không, và refund xảy ra trước hay sau inspection? **Why:** không thể chốt eligibility, CTA và timeline. **Affected:** D2C-017,018; ADM-026~028. **Source:** EPIC 14 Mục 8; UC-RET-02/03, UC-PAY-04. **Suggested assumption:** hiển thị eligibility từ policy service/data; không hứa refund/exchange trước decision.

#### OQ-05 — Pricing/promotion/loyalty precedence

**Question:** thứ tự và khả năng stack giữa base/channel price, Combo, coupon, Loyalty, B2B và Marketplace promotion? **Why:** ảnh hưởng tổng tiền và giải thích checkout. **Affected:** D2C-004~007,022,023; ADM-002,032~034. **Source:** EPIC 04/05/06/17/18 open decisions. **Suggested assumption:** breakdown render kết quả engine có nhãn nguồn; không tự quyết precedence trong UI.

#### OQ-06 — Inventory balance/event milestones

**Question:** công thức on-hand/reserved/allocated/quarantined/damaged/available và mốc trừ tồn vật lý là pick, pack hay handoff? **Why:** các số dư và completion impact chưa thể diễn giải chính xác. **Affected:** ADM-008~014,016,019,022~025. **Source:** EPIC 09/10 open decisions. **Suggested assumption:** UI hiển thị các bucket backend cung cấp kèm definition tooltip cấu hình; không tính phía client.

#### OQ-07 — Split package/multi-shipment

**Question:** một Order có thể tách package/shipment, giao một phần và đổi provider không? **Why:** quyết định cấu trúc Packing Task, tracking và delivery rate. **Affected:** D2C-015; ADM-014~018. **Source:** EPIC 10/11 open decisions; US-SHIP-02 đã có scenario nhiều Shipment. **Suggested assumption:** mô hình UI hỗ trợ collection package/shipment dù MVP có thể chỉ một.

#### OQ-08 — Actor duyệt OCOP/Content **[CONFLICT]**

**Question:** “Manager” là ACT-12, ACT-15 hay role mới; creator/approver có bắt buộc tách? **Why:** permission và navigation approval không thể chốt. **Affected:** ADM-035,036,037,038. **Source:** US-OCOP-04~06; US-CONTENT-07; Epic Mục 8; Actor Registry không có generic Manager. **Suggested assumption:** dùng permission-based label “authorized approver”, không gán Actor ID giả.

#### OQ-09 — HR Manager identity **[CONFLICT]**

**Question:** bổ sung HR Manager Actor ID nào và quyền cụ thể đối với Employee Profile/Account/lock? **Why:** Actor được dùng ở Epic nhưng không có trong Registry 19 actor. **Affected:** ADM-004,042. **Source:** EPIC 02,22; Product Backlog Actor Registry. **Suggested assumption:** System Admin không mặc nhiên xem HR sensitive fields; hiển thị chức năng theo permission.

#### OQ-10 — Payment methods/providers **[CONFLICT]**

**Question:** MVP bật VietQR trực tiếp, VNPay, MoMo, ZaloPay hay tổ hợp nào; nguồn xác nhận có thẩm quyền và COD condition? **Why:** phương thức, logo, copy, failure/retry khác nhau. **Affected:** D2C-006,007; ADM-007,028. **Source:** EPIC 07 để mở; Use Case mặc định VietQR; Product Backlog liệt kê nhiều provider “có thể”. **Suggested assumption:** screen render danh sách configured methods; không hiển thị provider chưa bật.

### IMPORTANT

#### OQ-11 — Customer registration/recovery fields **[PARTIALLY RESOLVED 2026-09-24]**

**Resolved for Auth MVP:** email là identifier duy nhất; registration yêu cầu displayName/email/password/terms; xác minh và recovery dùng email magic link; password 15–128 ký tự và bắt buộc chữ hoa, chữ thường, số, ký tự đặc biệt; resend cooldown 60 giây và rate limit theo `SCOPE_TRACEABILITY.md`. **Still open outside Auth slice:** quy trình đổi email/contact và re-verification trong Profile/Notification settings. **Affected:** phần Auth của D2C-009~011 đã chốt; D2C-012,029 vẫn cần quyết định EPIC 02/28.

#### OQ-12 — Address and shipping data

**Question:** bộ trường hành chính, nguồn địa giới, default/delete policy, quote validity và fallback provider? **Why:** checkout/address form và re-quote behavior. **Affected:** D2C-006,013; ADM-016. **Source:** EPIC 02/06/11. **Suggested assumption:** một địa chỉ/Order ở MVP; preserve snapshot.

#### OQ-13 — Catalog/Product publication

**Question:** required Product/SKU fields, category taxonomy, SKU uniqueness scope và approval/publication lifecycle? **Why:** Admin validation và public visibility. **Affected:** D2C-001,002,004; ADM-001,037,038. **Source:** EPIC 03/04/20. **Suggested assumption:** chỉ show records flagged public/sellable; workflow labels chờ PO.

#### OQ-14 — NSX/HSD public semantics **[CONFLICT]**

**Question:** PDP hiển thị date theo Product/SKU hay Batch thực tế? **Why:** Sprint 02 nói hiển thị NSX/HSD ở PDP, nhưng Epic 04 nói Batch thực tế thuộc Inventory và để mở cách tránh gây hiểu nhầm. **Affected:** D2C-004,026; ADM-001,009. **Source:** Sprint 02; EPIC 04 US-PROD-04/open decisions. **Suggested assumption:** PDP chỉ hiển thị policy/shelf-life info; exact Batch dates ở QR/traceability khi biết Batch.

#### OQ-15 — Search/filter semantics

**Question:** search fields, Vietnamese diacritics/synonyms, Product-vs-SKU filter semantics và rating filter trước GĐ2? **Why:** listing query model và empty state. **Affected:** D2C-002. **Source:** EPIC 03; Sprint 02 yêu cầu filter rating ở MVP nhưng EPIC 15 Review là GĐ2. **Suggested assumption:** ẩn rating filter đến khi dữ liệu Review tồn tại.

#### OQ-16 — Cart merge/persistence

**Question:** Guest cart TTL, merge after login, multi-device sync và conflict quantity? **Why:** hành vi Login/Checkout. **Affected:** D2C-005,006,009. **Source:** EPIC 05 Mục 8; UC-CART-02. **Suggested assumption:** giữ cart hiện tại và hiển thị review merge result; không tự cộng vượt stock.

#### OQ-17 — Packing assignment/checklist/video policy

**Question:** claim vs manager assignment, checklist theo loại nào, Order nào bắt buộc video, retention/download/reopen rule? **Why:** Task actions, completion và evidence permissions. **Affected:** ADM-013~015. **Source:** EPIC 10 Mục 8. **Suggested assumption:** cấu hình checklist/evidence requirement theo Task; required media phải stored trước complete.

#### OQ-18 — Shipping service/status/COD evidence

**Question:** providers/services bật, mapping state, shipped milestone, retry/failure attempts và COD proof? **Why:** Shipment creation/tracking/actions. **Affected:** D2C-006,015; ADM-016~018. **Source:** EPIC 11 Mục 8. **Suggested assumption:** provider-normalized label + raw provider detail for operators.

#### OQ-19 — Marketplace scope **[CONFLICT]**

**Question:** shop/platform, fulfillment model, Product/Customer/price sync có chính thức không? **Why:** External Requirement yêu cầu Product/Customer sync nhưng không có User Story; Epic chủ yếu Order/tồn/settlement. **Affected:** ADM-019~021. **Source:** Product Backlog 36.2; EPIC 12 Mục 8. **Suggested assumption:** scope UI chỉ Order/mapping/sync/settlement đã có story; ghi Product/Customer sync là gap.

#### OQ-20 — Offline operation model

**Question:** loại điểm bán/ca, payment methods, offline-first, Batch granularity, POS device/print/scan và correction after close? **Why:** POS/shift UI và conflict recovery. **Affected:** ADM-022~025. **Source:** EPIC 13 Mục 8; FR-14. **Suggested assumption:** online operation only until offline-first story is added.

#### OQ-21 — Review eligibility/moderation

**Question:** eligible milestone, unique key, edit duration, image/video limits, pre/post moderation, negative threshold và reply policy? **Why:** Review form/queue/analytics. **Affected:** D2C-004,021; ADM-029. **Source:** EPIC 15 Mục 8; BR-REVIEW-06 chỉ nêu “ví dụ 30 ngày”. **Suggested assumption:** UI nhận eligibility/window từ server; không hard-code 30 ngày.

#### OQ-22 — Ticket taxonomy/SLA/channels

**Question:** Guest verification, topic/priority/status, working calendar, reopen/merge, supported two-way channels? **Why:** intake, queue và alerts. **Affected:** D2C-019,020; ADM-030,056. **Source:** EPIC 16 Mục 8. **Suggested assumption:** một web channel công khai cho MVP; display SLA only after policy configured.

#### OQ-23 — B2B commercial terms/priority **[CONFLICT]**

**Question:** Priority chính thức, business verification, MOQ, quote authority, payment/deposit/credit, multi-delivery và invoice fields? **Why:** request/quote/order conversion. **Affected:** D2C-024,025; ADM-034,048. **Source:** EPIC 18 ghi Priority “Chưa xác định”, Sprint xếp GĐ2 Should. **Suggested assumption:** dùng GĐ2 từ Release Plan nhưng giữ Priority conflict; acceptance không tự tạo Paid.

#### OQ-24 — Content/SEO lifecycle

**Question:** Article taxonomy, official states, slug namespace/redirect, SEO fields/limits, preview sharing và media constraints? **Why:** editor/public/error states. **Affected:** D2C-027,028; ADM-037,038. **Source:** EPIC 20 Mục 8. **Suggested assumption:** Draft → Review → Approved → Scheduled/Published → Hidden ở mức UX concept, không coi là code status.

#### OQ-25 — Finance/DSS definitions and release **[CONFLICT]**

**Question:** revenue/cost/profit formulas, accounting period, attribution, KPI dictionary và `US-DSS-01` GĐ2 hay toàn EPIC GĐ3? **Why:** dashboard values/filters và consistency. **Affected:** ADM-041,047~052. **Source:** EPIC 24/25; Sprint xếp Dashboard GĐ2, Epic 25 ghi cần PO xác nhận. **Suggested assumption:** mọi metric hiển thị definition/version/freshness; giữ ADM-049 priority “GĐ2?”.

### NICE TO CLARIFY

#### OQ-26 — Notification taxonomy/preferences

**Question:** event catalog, classification, mandatory vs optional, quiet hours/frequency/fallback/retry và template approval? **Why:** preference/admin fields. **Affected:** D2C-029; ADM-056,057. **Source:** EPIC 28 Mục 8. **Suggested assumption:** Customer chỉ chỉnh marketing/optional; transactional/security được giải thích nhưng không tắt nếu bắt buộc.

#### OQ-27 — Gifting boundaries

**Question:** một/nhiều recipient, COD payer, edit cutoff, anonymous sender, recipient consent/tracking và artifact templates? **Why:** checkout/Order/Packing states. **Affected:** D2C-006,015; ADM-014. **Source:** EPIC 26 Mục 8. **Suggested assumption:** một recipient/Order; fields lock at Packing start until policy confirms.

#### OQ-28 — Audit retention/export

**Question:** event catalog, sensitive-field masking, retention, legal hold, export format/approval/expiry và verification schedule? **Why:** Explorer/Integrity controls. **Affected:** ADM-045,046. **Source:** EPIC 23; BR-AUDIT-05 uses non-binding example “1 năm hoặc vĩnh viễn”. **Suggested assumption:** no UI delete; retention/export config read-only until governance approved.

#### OQ-29 — Procurement item/approval model

**Question:** item taxonomy, Supplier portal/channel, PO approval thresholds, currency/tax/MOQ/tolerance/quality and Accounts Payable scope? **Why:** plan/PO/receiving forms. **Affected:** ADM-053~055,048. **Source:** EPIC 27 Mục 8. **Suggested assumption:** PO can be issued through an unspecified delivery channel; do not create Supplier Portal screen.

#### OQ-30 — Localization/accessibility/SEO NFR depth

**Question:** supported locales, formal accessibility target, performance threshold và public traceability/content cache/index rules? **Why:** acceptance criteria và responsive variants. **Affected:** all public screens, especially D2C-026~028. **Source:** NFR only says mobile-first/performance generally; EPIC 19/20 asks for specifics. **Suggested assumption:** apply existing design-system accessibility conventions and mark quantitative targets pending NFR update.
