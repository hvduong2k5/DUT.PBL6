# KẾ HOẠCH PHÁT TRIỂN & ĐẶC TẢ USER STORIES TOÀN DIỆN (12 TUẦN)
## ĐỀ TÀI: HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ ĐA KÊNH & HỆ THỐNG PHÂN TÍCH HỖ TRỢ QUYẾT ĐỊNH CHIẾN LƯỢC CHO NÔNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)
### TÀI LIỆU ĐẶC TẢ CHI TIẾT USER STORIES THEO CHUẨN AGILE/SCRUM & GHERKIN (BDD)

---

## MỤC LỤC TÀI LIỆU
1. [I. TỔNG QUAN HỆ THỐNG & MA TRẬN TÁC NHÂN (ACTORS MATRIX)](#i-tổng-quan-hệ-thống--ma-trận-tác-nhân-actors-matrix)
2. [II. BẢN ĐỒ USER STORIES (USER STORY MAP & EPICS OVERVIEW)](#ii-bản-đồ-user-stories-user-story-map--epics-overview)
3. [III. ĐẶC TẢ CHI TIẾT TỪNG USER STORY THEO CHUẨN GHERKIN](#iii-đặc-tả-chi-tiết-từng-user-story-theo-chuẩn-gherkin)
   - [EPIC 01: Quản Trị Danh Tính, Xác Thực & Phân Quyền (Auth & Identity)](#epic-01-quản-trị-danh-tính-xác-thực--phân-quyền-auth--identity)
   - [EPIC 02: Danh Mục Sản Phẩm OCOP, Câu Chuyện Di Sản & Tối Ưu SEO (Catalog & SEO)](#epic-02-danh-mục-sản-phẩm-ocop-câu-chuyện-di-sản--tối-ưu-seo-catalog--seo)
   - [EPIC 03: Giỏ Hàng, Khuyến Mãi & Quà Tặng B2B (Cart, Promotion & B2B)](#epic-03-giỏ-hàng-khuyến-mãi--quà-tặng-b2b-cart-promotion--b2b)
   - [EPIC 04: Xử Lý Đơn Hàng & Giao Dịch Phân Tán SAGA (Order & SAGA)](#epic-04-xử-lý-đơn-hàng--giao-dịch-phân-tán-saga-order--saga)
   - [EPIC 05: Thanh Toán VietQR Động, Webhook & Đối Soát (Payment & Billing)](#epic-05-thanh-toán-vietqr-động-webhook--đối-soát-payment--billing)
   - [EPIC 06: Quản Trị Kho Khóa Phân Tán & Giao Vận 3PL (Inventory & Logistics)](#epic-06-quản-trị-kho-khóa-phân-tán--giao-vận-3pl-inventory--logistics)
   - [EPIC 07: Trải Nghiệm Khách Hàng Đa Kênh & Realtime (Web, App & WebSocket)](#epic-07-trải-nghiệm-khách-hàng-đa-kênh--realtime-web-app--websocket)
   - [EPIC 08: Thu Thập Clickstream NoSQL & Gợi Ý Cá Nhân Hóa (Behavior & Recommendation)](#epic-08-thu-thập-clickstream-nosql--gợi-ý-cá-nhân-hóa-behavior--recommendation)
   - [EPIC 09: Phân Hệ BI Thống Kê & Động Cơ Ra Quyết Định Chiến Lược (BI & DSS Engine)](#epic-09-phân-hệ-bi-thống-kê--động-cơ-ra-quyết-định-chiến-lược-bi--dss-engine)
   - [EPIC 10: DevOps, Giám Sát Observability & Quản Trị Hệ Thống (DevOps & Admin)](#epic-10-devops-giám-sát-observability--quản-trị-hệ-thống-devops--admin)
4. [IV. LỘ TRÌNH 12 TUẦN (SPRINT PLANNING & SPRINT BACKLOG)](#iv-lộ-trình-12-tuần-sprint-planning--sprint-backlog)
5. [V. ĐỊNH NGHĨA HOÀN THÀNH (DEFINITION OF DONE - DOD) & MA TRẬN RỦI RO](#v-định-nghĩa-hoàn-thành-definition-of-done---dod--ma-trận-rủi-ro)

---

## I. TỔNG QUAN HỆ THỐNG & MA TRẬN TÁC NHÂN (ACTORS MATRIX)

### 1.1. Ma Trận Tác Nhân Hệ Thống (Actors)

| Mã Actor | Tên Tác Nhân | Mô Tả Vai Trò & Phạm Vi Hoạt Động |
| :---: | :--- | :--- |
| **ACT-01** | **Khách Hàng Vãng Lai (Guest User)** | Người dùng chưa đăng nhập; xem danh mục, đọc câu chuyện di sản làng nghề, quét mã QR Story, tìm kiếm sản phẩm và thêm hàng vào giỏ tạm thời. |
| **ACT-02** | **Khách Hàng Thành Viên (Registered Customer)** | Người dùng đã đăng ký tài khoản trên Web/App; quản lý thông tin giao hàng, đặt hàng thanh toán VietQR, tích điểm thành viên, tra cứu hành trình vận chuyển và nhận Push Notifications. |
| **ACT-03** | **Khách Hàng Doanh Nghiệp (Corporate B2B Client)** | Đại diện doanh nghiệp/tổ chức có nhu cầu mua sỉ số lượng lớn quà tặng mè xửng Huế (Set quà gỗ, in khắc logo công ty, xuất hóa đơn VAT điện tử). |
| **ACT-04** | **Thủ Kho & Nhân Viên Đóng Gói (Warehouse Staff)** | Quản lý số lượng tồn kho theo SKU, cập nhật xuất/nhập kho nguyên liệu mè đường mạch nha, in phiếu đóng gói và bàn giao cho đơn vị vận chuyển 3PL. |
| **ACT-05** | **Quản Lý Bán Hàng & CSKH (Store Manager)** | Quản lý danh mục sản phẩm, duyệt đơn hàng, xử lý khiếu nại/hoàn hàng, thiết lập mã ưu đãi (Coupon) và theo dõi thông báo chuông đơn mới qua WebSocket. |
| **ACT-06** | **Ban Giám Đốc / Nhà Quản Trị Chiến Lược (Executive Admin)** | Xem toàn bộ báo cáo phân tích BI đa chiều, phân tích phân khúc khách hàng RFM, theo dõi dự báo tồn kho và nhận đề xuất chiến lược kinh doanh thông minh (DSS Engine). |
| **ACT-07** | **Quản Trị Viên Kỹ Thuật (Super Admin / DevOps)** | Quản lý ma trận phân quyền RBAC, theo dõi hạ tầng máy chủ, cấu hình hệ thống giám sát Prometheus/Grafana/Jaeger và quản lý sao lưu dữ liệu. |
| **EXT-01** | **Cổng Thanh Toán Số (Payment Gateway API)** | Hệ thống ngoài (VietQR Napas / VNPay Sandbox) xử lý giao dịch chuyển khoản và gửi Webhook/IPN xác thực thanh toán. |
| **EXT-02** | **Đơn Vị Giao Vận 3PL (Logistics API)** | Hệ thống ngoài (GHN / GHTK Sandbox) tiếp nhận vận đơn, tính phí ship tự động và đẩy webhook trạng thái lộ trình shipper theo thời gian thực. |
| **EXT-03** | **Dịch Vụ Thông Báo Ngoại Vi (Notification Hub)** | Firebase Cloud Messaging (FCM), Zalo Cloud API (ZNS), SendGrid Email Service. |
| **EXT-04** | **Trình Cào Dữ Liệu Tìm Kiếm (Search Engine Bot)** | Googlebot / Bingbot thu thập dữ liệu SSR và Schema.org JSON-LD phục vụ xếp hạng SEO. |

---

## II. BẢN ĐỒ USER STORIES (USER STORY MAP & EPICS OVERVIEW)

```
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       USER STORY MAP - HỆ SINH THÁI MÈ XỬNG O MẠ                                       │
├─────────────────┬──────────────────┬─────────────────┬─────────────────┬──────────────────┬────────────────────────────┤
│ 1. DUYỆT & TÌM  │ 2. CHỌN HÀNG     │ 3. THANH TOÁN   │ 4. VẬN HÀNH &   │ 5. HÀNH VI &     │ 6. BI & ĐỊNH HƯỚNG         │
│    KIẾM D2C     │    & ƯU ĐÃI      │    & VẬN ĐƠN    │    KHO BÃI      │    GỢI Ý NoSQL   │    CHIẾN LƯỢC (DSS)        │
├─────────────────┼──────────────────┼─────────────────┼─────────────────┼──────────────────┼────────────────────────────┤
│ • Xem di sản    │ • Giỏ hàng Redis │ • VietQR động   │ • SAGA trừ kho  │ • Ingest Log     │ • Báo cáo doanh thu đa kênh│
│ • Lọc OCOP      │ • Mã giảm giá    │ • Webhook IPN   │ • Khóa Redlock  │ • Chuỗi click    │ • Phân khúc khách hàng RFM │
│ • SEO Schema    │ • Báo giá B2B    │ • Tra cứu 3PL   │ • Chuông Socket │ • Gợi ý NextClick│ • Dự báo cạn kho (Velocity)│
│ • Quét QR Story │ • Đồng bộ App    │ • Hóa đơn VAT   │ • Đổi trả hoàn  │ • Cá nhân hóa    │ • Gợi ý Combo & Mùa vụ     │
└─────────────────┴──────────────────┴─────────────────┴─────────────────┴──────────────────┴────────────────────────────┘
```

---

## III. ĐẶC TẢ CHI TIẾT TỪNG USER STORY THEO CHUẨN GHERKIN

---

### EPIC 01: QUẢN TRỊ DANH TÍNH, XÁC THỰC & PHÂN QUYỀN (AUTH & IDENTITY)
*Microservice phụ trách: `auth-service` (Port 4001) | Database: `om_identity_db` (PostgreSQL) + Redis*

#### US-01: Đăng Ký Tài Khoản Khách Hàng Với Xác Thực An Toàn
- **Mô tả**: *Là một Khách hàng mới (ACT-01), Tôi muốn đăng ký tài khoản bằng email/số điện thoại và mật khẩu an toàn, Để tôi có thể quản lý lịch sử đơn hàng và tham gia chương trình tích điểm.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 3 SP
- **Preconditions**: Email/Số điện thoại chưa từng được đăng ký trong hệ thống.
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Đăng ký tài khoản thành công
    Given Người dùng ở trang đăng ký "/dang-ky"
    When Nhập email "nguyenvana@gmail.com", mật khẩu "HueHeritage@2026", họ tên "Nguyễn Văn A"
    And Mật khẩu thỏa mãn tối thiểu 8 ký tự, gồm chữ hoa, chữ thường, số và ký tự đặc biệt
    And Nhấn nút "Tạo tài khoản"
    Then Hệ thống mã hóa mật khẩu bằng thuật toán Argon2id
    And Lưu thông tin vào CSDL "om_identity_db.users" với role mặc định là "CUSTOMER"
    And Gửi email kích hoạt tài khoản kèm mã xác thực OTP 6 số (hạn 10 phút)
    And Trả về mã HTTP 201 Created

  Scenario: Đăng ký thất bại do email đã tồn tại
    Given Người dùng nhập email "daco@gmail.com" đã tồn tại trong CSDL
    When Nhấn nút "Tạo tài khoản"
    Then Hệ thống không tạo bản ghi mới
    And Trả về mã HTTP 409 Conflict kèm thông báo "Email này đã được sử dụng. Vui lòng đăng nhập hoặc chọn Quên mật khẩu."
  ```
- **Kỹ thuật & API**: `POST /api/v1/auth/register` | Hash: `Argon2id (timeCost=3, memoryCost=65536)`.

---

#### US-02: Đăng Nhập Hệ Thống & Cấp Phát Cặp JWT Token Phân Tán
- **Mô tả**: *Là một Người dùng (ACT-02/04/05/06/07), Tôi muốn đăng nhập vào hệ thống để nhận phiên làm việc an toàn, Để tôi thực hiện các quyền hạn tương ứng của mình.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Đăng nhập thành công cấp Access Token và Refresh Token
    Given Người dùng gửi email và mật khẩu chính xác tới "/api/v1/auth/login"
    When Hệ thống xác thực thông tin thành công
    Then Cấp Access Token (JWT hạn 15 phút, chứa userId, email, role, permissions)
    And Cấp Refresh Token (hạn 7 ngày, chuỗi ngẫu nhiên lưu tại Redis "refresh_token:{userId}")
    And Thiết lập Refresh Token trong HttpOnly Secure Cookie (chống tấn công XSS)
    And Trả về HTTP 200 OK kèm thông tin cơ bản người dùng

  Scenario: Tự động gia hạn Access Token qua Refresh Token (Silent Refresh)
    Given Access Token của người dùng đã hết hạn (sau 15 phút)
    When Client gửi request tới "/api/v1/auth/refresh-token" kèm Refresh Token Cookie
    And Refresh Token tồn tại và hợp lệ trong Redis
    Then Hệ thống cấp Access Token mới có hạn 15 phút
    And Trả về HTTP 200 OK mà không bắt người dùng phải đăng nhập lại
  ```
- **Kỹ thuật & API**: `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh-token`, `POST /api/v1/auth/logout`.

---

#### US-03: Kiểm Soát Phân Quyền Dựa Trên Vai Trò (RBAC Guard)
- **Mô tả**: *Là một Quản trị viên Kỹ thuật (ACT-07), Tôi muốn hệ thống thực thi ma trận phân quyền chi tiết trên từng API endpoint, Để ngăn chặn hành vi truy cập trái phép và bảo vệ an toàn dữ liệu doanh nghiệp.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Chặn người dùng thường truy cập tài nguyên của Quản trị viên
    Given Người dùng đăng nhập với vai trò "CUSTOMER"
    When Gửi request xem báo cáo doanh thu "GET /api/v1/analytics/revenue"
    Then API Gateway kiểm tra JWT Claims thấy không có quyền "ANALYTICS:VIEW"
    And Chặn request ngay tại tầng Gateway, không forward vào microservice nội bộ
    And Ghi nhật ký cảnh báo an ninh (Security Audit Log)
    And Trả về mã lỗi HTTP 403 Forbidden kèm thông báo "Bạn không có quyền truy cập chức năng này."

  Scenario: Cho phép Giám Đốc truy cập phân hệ Báo cáo Chiến lược
    Given Người dùng đăng nhập với vai trò "EXECUTIVE_ADMIN"
    When Gửi request "GET /api/v1/analytics/strategic-insights"
    Then API Gateway xác thực thành công vai trò và quyền hạn
    And Cho phép forward request vào Analytics Service và trả về HTTP 200 OK
  ```
- **Kỹ thuật**: Middleware `RBACAuthGuard` trên API Gateway & NestJS Decorators `@Roles('ADMIN', 'MANAGER')`.

---

### EPIC 02: DANH MỤC SẢN PHẨM OCOP, CÂU CHUYỆN DI SẢN & TỐI ƯU SEO (CATALOG & SEO)
*Microservice phụ trách: `catalog-service` (Port 4002) | Database: `om_catalog_db` (PostgreSQL) + MinIO S3 + Redis*

#### US-04: Quản Trị Danh Mục & Sản Phẩm Biến Thể Mè Xửng OCOP
- **Mô tả**: *Là một Quản lý Bán hàng (ACT-05), Tôi muốn quản lý danh mục sản phẩm mè xửng đa dạng (Mè xửng dẻo, giòn, mè đen, kẹo gương, hạt sen) kèm các biến thể quy cách đóng gói và huy hiệu chứng nhận OCOP, Để thông tin hiển thị chính xác và hấp dẫn trên cửa hàng số.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Tạo sản phẩm mới kèm các biến thể SKU thành công
    Given Quản lý đăng nhập với quyền "CATALOG:MANAGE"
    When Nhập tên sản phẩm "Mè Xửng Dẻo Thượng Hạng O Mạ", danh mục "Mè Xửng Dẻo"
    And Đính kèm chứng nhận OCOP "4 Sao", hạn sử dụng "6 tháng"
    And Thêm 2 biến thể: SKU "MXD-250G" (Hộp 250g, Giá 35.000đ) và SKU "MXD-500G" (Hộp 500g, Giá 65.000đ)
    And Tải lên 3 ảnh chất lượng cao lên MinIO S3
    Then Lưu thông tin vào CSDL "om_catalog_db.products" và "om_catalog_db.product_variants"
    And Xóa cache Redis "catalog:categories:all" để làm mới dữ liệu
    And Trả về mã HTTP 201 Created
  ```
- **Kỹ thuật & API**: `POST /api/v1/catalog/products`, `PUT /api/v1/catalog/products/{id}`, `DELETE /api/v1/catalog/products/{id}`.

---

#### US-05: Tối Ưu SEO Kỹ Thuật (SSR Next.js 14, Schema.org JSON-LD & Dynamic Sitemap)
- **Mô tả**: *Là một Khách hàng tiềm năng (ACT-01) hoặc Bot tìm kiếm (EXT-04), Tôi muốn website có đường dẫn thân thiện, tải nhanh và có cấu trúc dữ liệu Schema chuẩn, Để tôi dễ dàng tìm thấy Mè Xửng O Mạ trên kết quả tìm kiếm Google.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Render trang chi tiết sản phẩm chuẩn SEO Server-Side Rendering (SSR)
    Given Googlebot truy cập URL "/san-pham/me-xung-deo-thuong-hang-500g"
    When Next.js Server Components tiếp nhận request
    Then Server render HTML hoàn chỉnh kèm đầy đủ thẻ "<title>", "<meta name='description'>"
    And Chèn cấu trúc Schema.org JSON-LD loại "Product", "BreadcrumbList", "Organization"
    And Hình ảnh có đầy đủ thuộc tính "alt" chứa từ khóa di sản Huế
    And Điểm kiểm tra Google Lighthouse SEO đạt tối thiểu 95/100, Performance đạt > 90/100

  Scenario: Tự động cập nhật sitemap.xml khi có sản phẩm mới
    Given Quản trị viên vừa xuất bản một sản phẩm OCOP mới
    When Người dùng hoặc Bot truy cập "/sitemap.xml"
    Then Hệ thống sinh danh sách URL động kèm ngày cập nhật "lastmod" mới nhất trong vòng < 50ms
  ```
- **Kỹ thuật**: Next.js 14 App Router SSR, Schema.org JSON-LD Generator, Dynamic `sitemap.ts`, `robots.ts`.

---

#### US-06: Truy Xuất Nguồn Gốc Làng Nghề Qua Mã QR Story
- **Mô tả**: *Là một Khách hàng (ACT-01/02), Tôi muốn quét mã QR in trên vỏ hộp bánh để xem video quy trình nấu kẹo thủ công, nguồn gốc nguyên liệu và chứng chỉ vệ sinh ATTP, Để tôi hoàn toàn tin tưởng vào chất lượng sản phẩm OCOP chính gốc.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 3 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Quét mã QR Story trên bao bì
    Given Khách hàng quét mã QR trên nắp hộp bánh Mè Xửng O Mạ
    When Trình duyệt mở trang đích "/qr-story/lo-san-xuat-2026-08-01"
    Then Trang hiển thị:
      | Thông tin hiển thị              | Chi tiết minh bạch                              |
      | Nghệ nhân phụ trách             | Nghệ nhân Trần Thị Lan (40 năm tuổi nghề)       |
      | Nguồn gốc nguyên liệu           | Mè vàng Quảng Điền, Đậu phộng Phù Mỹ, Mạch nha nếp |
      | Ngày sản xuất & Hạn dùng        | NSX: 01/08/2026 - HSD: 01/02/2027              |
      | Chứng nhận kiểm định ATTP       | Giấy chứng nhận OCOP 4 Sao số 128/QĐ-UBND       |
      | Video clip ngắn 30 giây         | Cận cảnh công đoạn ngào đường kéo chỉ truyền thống |
  ```
- **Kỹ thuật & API**: `GET /api/v1/catalog/qr-story/{batchCode}`.

---

### EPIC 03: GIỎ HÀNG, KHUYẾN MÃI & QUÀ TẶNG B2B (CART, PROMOTION & B2B)
*Microservice phụ trách: `order-service` & `catalog-service` | Database: Redis + PostgreSQL*

#### US-07: Quản Lý Giỏ Hàng Thời Gian Thực Đa Nền Tảng (Redis Caching)
- **Mô tả**: *Là một Khách hàng (ACT-01/02), Tôi muốn thêm/sửa/xóa sản phẩm trong giỏ hàng và dữ liệu được đồng bộ ngay lập tức giữa Web và App di động, Để tôi có thể mua sắm tiện lợi mà không bị mất giỏ hàng.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 3 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Đồng bộ giỏ hàng từ khách vãng lai sang tài khoản thành viên
    Given Khách vãng lai đã thêm 2 hộp Mè Xửng Giòn vào giỏ (lưu Redis key "cart:guest:{sessionId}")
    When Khách hàng tiến hành Đăng nhập tài khoản thành viên
    Then Hệ thống tự động gộp (Merge) giỏ hàng tạm vào giỏ hàng chính thức "cart:user:{userId}"
    And Giỏ hàng hiển thị chính xác tổng số lượng sản phẩm và giá tiền đã cập nhật
  ```
- **Kỹ thuật & API**: `GET /api/v1/cart`, `POST /api/v1/cart/items`, `PUT /api/v1/cart/items/{variantId}`, `DELETE /api/v1/cart/items/{variantId}`.

---

#### US-08: Áp Dụng Mã Ưu Đãi (Coupon / Voucher Engine)
- **Mô tả**: *Là một Khách hàng (ACT-02), Tôi muốn nhập mã giảm giá khi thanh toán, Để tôi được giảm trực tiếp trên tổng giá trị đơn hàng hoặc được miễn phí vận chuyển.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 3 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Áp mã giảm giá hợp lệ
    Given Đơn hàng có giá trị 300.000 VNĐ
    When Khách hàng nhập mã voucher "FESTIVALHUE10" (Giảm 10% cho đơn từ 200k, tối đa 50k)
    Then Hệ thống kiểm tra điều kiện hợp lệ (còn lượt dùng, trong thời hạn áp dụng)
    And Giảm trực tiếp 30.000 VNĐ vào tổng tiền thanh toán
    And Hiển thị số tiền phải trả còn lại là 270.000 VNĐ

  Scenario: Nhập mã voucher đã hết lượt sử dụng
    Given Voucher "CHAOXUAN" đã đạt giới hạn 500 lượt sử dụng
    When Khách hàng bấm "Áp dụng"
    Then Hệ thống từ chối và hiển thị thông báo "Mã giảm giá đã hết lượt sử dụng."
  ```
- **Kỹ thuật & API**: `POST /api/v1/orders/apply-coupon`.

---

#### US-09: Yêu Cầu Báo Giá & Đặt Hàng Quà Tặng Doanh Nghiệp B2B
- **Mô tả**: *Là một Khách hàng Doanh nghiệp (ACT-03), Tôi muốn gửi yêu cầu đặt set quà tặng mè xửng số lượng lớn (> 50 hộp), đính kèm file logo công ty và nhận bảng dự toán chiết khấu sỉ tự động, Để tôi nhanh chóng trình duyệt ngân sách công ty.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Gửi yêu cầu báo giá B2B thành công
    Given Người dùng truy cập trang "/qua-tang-doanh-nghiep"
    When Chọn loại set quà "Hộp Gỗ Di Sản Hoàng Cung", số lượng "150 hộp"
    And Tải lên file logo công ty dạng vector ".AI / .PNG"
    And Nhập thông tin doanh nghiệp, mã số thuế và yêu cầu xuất hóa đơn VAT
    And Bấm "Gửi yêu cầu báo giá"
    Then Hệ thống tự động tính toán mức chiết khấu sỉ 18%
    And Gửi email báo giá sơ bộ kèm file PDF dự toán cho khách hàng
    And Tạo 1 Ticket chăm sóc B2B ưu tiên trên Web Admin
  ```
- **Kỹ thuật & API**: `POST /api/v1/catalog/b2b-quotes`.

---

### EPIC 04: XỬ LÝ ĐƠN HÀNG & GIAO DỊCH PHÂN TÁN SAGA (ORDER & SAGA)
*Microservice phụ trách: `order-service` (Port 4003) | Database: `om_order_db` (PostgreSQL) + RabbitMQ*

#### US-10: Khởi Tạo Đơn Hàng & Điều Phối Giao Dịch Phân Tán (SAGA Orchestrator)
- **Mô tả**: *Là một Khách hàng (ACT-02), Tôi muốn đơn hàng của tôi được khởi tạo và kiểm tra tồn kho một cách an toàn và tự động, Để đảm bảo hàng được giữ cho tôi trong lúc tôi thanh toán.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Luồng khởi tạo đơn hàng thành công (Happy Path)
    Given Khách hàng bấm "Đặt hàng ngay" từ giỏ hàng
    When "Order Service" tiếp nhận request và kích hoạt tiến trình SAGA:
      1. Tạo bản ghi đơn hàng với trạng thái "PENDING_INVENTORY" trong "om_order_db"
      2. Gửi lệnh gRPC "ReserveStock" sang "Inventory Service" để tạm giữ hàng 15 phút
      3. Nhận phản hồi giữ kho thành công -> Chuyển trạng thái đơn sang "PENDING_PAYMENT"
      4. Gửi lệnh sang "Payment Service" để sinh dữ liệu VietQR động
    Then Trả về mã HTTP 201 Created kèm mã đơn hàng và mã QR thanh toán cho Client

  Scenario: Luồng bù trừ SAGA khi hết hàng trong kho (Compensating Flow)
    Given Khách hàng bấm "Đặt hàng ngay"
    When "Inventory Service" phát hiện sản phẩm đã hết tồn kho
    Then "Inventory Service" trả về lỗi "OUT_OF_STOCK"
    And "Order Service" kích hoạt đền bù: Chuyển trạng thái đơn sang "REJECTED_STOCK_UNAVAILABLE"
    And Trả về mã lỗi HTTP 400 Bad Request kèm thông báo "Rất tiếc, sản phẩm vừa hết hàng."
  ```
- **Kỹ thuật & API**: `POST /api/v1/orders`, SAGA State Machine Pattern, gRPC Client.

---

#### US-11: Tự Động Hủy Đơn Hàng Quá Hạn Thanh Toán (Order Timeout Handler)
- **Mô tả**: *Là một Thủ kho (ACT-04), Tôi muốn các đơn hàng không thanh toán sau 15 phút sẽ tự động bị hủy và nhả lại số lượng tồn kho, Để hàng hóa không bị giữ ảo và người khác có thể mua được.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Đơn hàng quá 15 phút chưa thanh toán
    Given Đơn hàng ở trạng thái "PENDING_PAYMENT" đã quá thời gian hết hạn TTL 15 phút
    When Tiến trình Scheduled Worker kiểm tra thấy chưa nhận được Webhook thanh toán
    Then Chuyển trạng thái đơn hàng thành "EXPIRED"
    And Phát sự kiện "OrderExpiredEvent" lên RabbitMQ
    And "Inventory Service" nhận sự kiện và nhả lại số lượng tồn kho đã tạm giữ
    And Gửi thông báo đến khách hàng "Đơn hàng của bạn đã hết hạn thanh toán."
  ```
- **Kỹ thuật**: RabbitMQ Dead Letter Queue (DLQ) / Redis Keyspace Notification.

---

### EPIC 05: THANH TOÁN VIETQR ĐỘNG, WEBHOOK & ĐỐI SOÁT (PAYMENT & BILLING)
*Microservice phụ trách: `payment-service` (Port 4004) | Database: `om_payment_db` (PostgreSQL) + Redis*

#### US-12: Sinh Mã VietQR Động Chuẩn NAPAS Tích Hợp Số Tiền
- **Mô tả**: *Là một Khách hàng (ACT-02), Tôi muốn màn hình hiển thị mã VietQR chứa đầy đủ số tài khoản, số tiền và nội dung chuyển khoản duy nhất, Để tôi chỉ cần quét mã trên App ngân hàng mà không cần nhập tay.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Sinh mã VietQR động
    Given Đơn hàng "OM-2026-8899" có tổng tiền thanh toán là 185.000 VNĐ
    When Khách hàng chọn phương thức "Thanh toán QR Napas"
    Then "Payment Service" sinh chuỗi mã VietQR chuẩn EMVCo chứa:
      | Trường dữ liệu          | Giá trị đặc tả                               |
      | Ngân hàng thụ hưởng     | Vietcombank (Chi nhánh Huế)                  |
      | Số tài khoản            | 0123456789 (Tài khoản Mè Xửng O Mạ)          |
      | Số tiền                 | 185000                                       |
      | Nội dung chuyển khoản   | OM20268899                                   |
    And Hiển thị hình ảnh mã QR và đồng hồ đếm ngược 15:00 trên màn hình
  ```
- **Kỹ thuật & API**: `POST /api/v1/payments/vietqr/generate`.

---

#### US-13: Tiếp Nhận Webhook Thanh Toán & Xử Lý Chống Trùng Lặp (Idempotent Consumer)
- **Mô tả**: *Là một Quản trị viên Doanh nghiệp (ACT-06), Tôi muốn hệ thống tự động xác nhận khi khách đã chuyển tiền qua Webhook ngân hàng và tuyệt đối không cộng tiền trùng lặp, Để tài chính luôn chuẩn xác 100%.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Nhận Webhook thanh toán thành công từ Ngân hàng
    Given Khách hàng đã quét mã QR và chuyển khoản thành công
    When Cổng thanh toán (EXT-01) gửi Webhook HTTP POST tới "/api/v1/payments/webhook"
    And Header chứa chữ ký số HMAC-SHA256 hợp lệ
    Then "Payment Service" kiểm tra Idempotency Key "payment:trans:{transId}" trong Redis:
      1. Khóa chưa tồn tại -> Lưu khóa vào Redis với TTL 24h
      2. Ghi nhận giao dịch thành công vào "om_payment_db.payments"
      3. Phát sự kiện "PaymentSuccessEvent" lên RabbitMQ
      4. "Order Service" chuyển đơn hàng sang trạng thái "PAID"
      5. Trả về HTTP 200 OK cho Ngân hàng trong vòng < 500ms

  Scenario: Ngân hàng gửi lại Webhook trùng lặp (Duplicate Webhook)
    Given Webhook của giao dịch "TRANS_12345" được gửi lại lần 2
    When "Payment Service" kiểm tra thấy Idempotency Key đã tồn tại trong Redis
    Then Lập tức phản hồi HTTP 200 OK
    And Bỏ qua bước xử lý cộng tiền lần hai, ngăn chặn thất thoát
  ```
- **Kỹ thuật & API**: `POST /api/v1/payments/webhook`, HMAC-SHA256 Signature Verification, Redis Distributed Lock.

---

### EPIC 06: QUẢN TRỊ KHO KHÓA PHÂN TÁN & GIAO VẬN 3PL (INVENTORY & LOGISTICS)
*Microservice phụ trách: `inventory-service` (Port 4005) & `shipping-service` (Port 4006)*

#### US-14: Quản Lý Tồn Kho Theo Lô & Cảnh Báo Ngưỡng Tồn Tối Thiểu
- **Mô tả**: *Là một Thủ kho (ACT-04), Tôi muốn theo dõi số lượng tồn kho theo từng lô sản xuất và nhận cảnh báo khi số lượng chạm ngưỡng tối thiểu, Để tôi kịp thời báo xưởng ngào thêm mẻ mè xửng mới.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Kích hoạt cảnh báo khi tồn kho chạm ngưỡng tối thiểu
    Given Sản phẩm "Mè Xửng Mè Đen Hộp 250g" có cấu hình ngưỡng tối thiểu là 50 hộp
    When Số lượng tồn kho sau đơn hàng giảm xuống còn 48 hộp
    Then "Inventory Service" phát sự kiện "LowStockWarningEvent" lên RabbitMQ
    And Gửi thông báo đỏ lên Dashboard Web Admin của Thủ kho
    And Ghi log sự kiện để phân hệ BI phân tích nhu cầu nhập nguyên liệu
  ```
- **Kỹ thuật & API**: `GET /api/v1/inventory/stocks`, `POST /api/v1/inventory/adjustments`.

---

#### US-15: Tự Động Tạo Vận Đơn 3PL & Nhận Cập Nhật Lộ Trình Shipper Realtime
- **Mô tả**: *Là một Nhân viên Đóng gói (ACT-04), Tôi muốn bấm 1 nút để hệ thống tự động đẩy đơn sang nhà vận chuyển (GHN/GHTK) và in mã vận đơn, Để rút ngắn thời gian bàn giao cho shipper.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Đẩy vận đơn sang đơn vị 3PL thành công
    Given Đơn hàng đã ở trạng thái "PAID" và đã được đóng gói hút chân không xong
    When Nhân viên kho bấm "Tạo vận đơn giao hàng"
    Then "Shipping Service" gọi API GHN/GHTK Sandbox gửi thông tin người nhận và trọng lượng
    And Nhận về mã vận đơn "GHN_HUE_998822"
    And In phiếu giao hàng khổ A6 có mã vạch
    And Chuyển trạng thái đơn sang "SHIPPED"
  ```
- **Kỹ thuật & API**: `POST /api/v1/shipping/create-shipment`, `POST /api/v1/shipping/webhook-3pl`.

---

### EPIC 07: TRẢI NGHIỆM KHÁCH HÀNG ĐA KÊNH & REALTIME (WEB, APP & WEBSOCKET)
*Microservice phụ trách: `notification-service` (Port 4007) | Apps: `web-user`, `web-admin`, `mobile-app`*

#### US-16: Trải Nghiệm Mua Sắm D2C Cố Đô Trên Website Next.js 14
- **Mô tả**: *Là một Khách hàng (ACT-01/02), Tôi muốn duyệt website với giao diện văn hóa Huế tinh tế, tốc độ phản hồi nhanh và thanh toán tiện lợi trong 3 bước, Để tôi có trải nghiệm mua sắm nông đặc sản cao cấp.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Khách hàng hoàn tất mua hàng 1-Click trên Web D2C
    Given Khách hàng duyệt trang chủ "/san-pham"
    When Bấm chọn "Mè Xửng Dẻo Hộp Quà Gỗ 500g" và nhấn "Mua Ngay"
    And Điền thông tin giao hàng gồm Tỉnh/Thành, Quận/Huyện, Số điện thoại
    And Quét mã VietQR thanh toán
    Then Màn hình tự động chuyển sang trang "/dat-hang-thanh-cong" trong vòng < 2 giây qua WebSocket
    And Khách hàng nhận được tin nhắn Zalo ZNS xác nhận đơn hàng
  ```

---

#### US-17: Ứng Dụng Di Động Khách Hàng Flutter & Push Notification FCM
- **Mô tả**: *Là một Khách hàng thân thiết (ACT-02), Tôi muốn sử dụng App Mè Xửng O Mạ trên điện thoại để tra cứu tiến trình đơn hàng và nhận thông báo khi shipper chuẩn bị giao bánh, Để tôi không bị lỡ đơn.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Nhận Push Notification khi đơn hàng đang trên đường giao
    Given Khách hàng đã cài đặt App Mè Xửng O Mạ trên Android/iOS
    When Đơn vị vận chuyển 3PL cập nhật trạng thái "Đang giao hàng" (Out for Delivery)
    Then Firebase Cloud Messaging (FCM) đẩy thông báo đến điện thoại:
      "Bánh Mè Xửng O Mạ thơm ngon đang trên đường giao đến bạn! Vui lòng chú ý điện thoại."
    And Khách hàng bấm vào thông báo để mở trực tiếp trang Tra cứu lộ trình trên App
  ```

---

#### US-18: Cụm WebSocket Server Bắn Thông Báo Chuông Đơn Mới Đến Web Admin
- **Mô tả**: *Là một Quản lý Bán hàng (ACT-05), Tôi muốn Web Admin phát âm thanh chuông báo và hiển thị pop-up ngay khi có khách thanh toán thành công, Để tôi kịp thời điều phối đóng gói mà không cần nhấn F5 reload trang.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Web Admin nhận thông báo thời gian thực qua WebSocket
    Given Quản lý đang mở Web Admin Portal trên trình duyệt
    When Có khách hàng thanh toán thành công trên Web hoặc App di động
    Then Cụm WebSocket Socket.io Gateway nhận sự kiện từ RabbitMQ
    And Đẩy sự kiện "NEW_ORDER_PAID" đến toàn bộ Admin đang kết nối trong vòng < 100ms
    And Màn hình Admin hiện badge đỏ, pop-up đơn mới và phát âm thanh chuông báo hiệu
  ```
- **Kỹ thuật**: Socket.io Gateway kết nối Redis Pub/Sub Adapter (hỗ trợ scale đa node).

---

### EPIC 08: THU THẬP CLICKSTREAM NoSQL & GỢI Ý CÁ NHÂN HÓA (BEHAVIOR & RECOMMENDATION)
*Microservice phụ trách: `analytics-service` (Port 4008) | Database: `om_behavior_db` (MongoDB) + Redis*

#### US-19: Thu Thập & Ghi Nhận Nhật Ký Hành Vi Người Dùng (Clickstream Ingestion)
- **Mô tả**: *Là một Kỹ sư Dữ liệu (ACT-07), Tôi muốn hệ thống thu thập toàn bộ các sự kiện tương tác của người dùng (Tìm kiếm từ khóa, Xem chi tiết sản phẩm, Nhấp vào mục gợi ý, Thời gian dừng) vào NoSQL MongoDB, Để phục vụ thuật toán phân tích hành vi và gợi ý thông minh.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Ghi nhận sự kiện điều hướng xem sản phẩm liên quan
    Given Khách hàng đang xem trang "Mè Xửng Dẻo 250g"
    When Khách hàng lướt xuống cuối trang và click vào sản phẩm "Trà Cung Đình Huế Túi Lọc" tại mục "Có thể bạn thích"
    Then Frontend gửi sự kiện Beacon không đồng bộ tới "POST /api/v1/analytics/events"
    And "Analytics Service" ghi nhận bản ghi Document vào MongoDB "om_behavior_db.user_events" dạng JSON:
      """
      {
        "sessionId": "sess_889922",
        "eventType": "NAVIGATION_PATH",
        "sourceProductId": "prod_me_xung_deo_250g",
        "targetProductId": "prod_tra_cung_dinh",
        "section": "RECOMMENDED_BOTTOM",
        "dwellTimeSeconds": 42,
        "timestamp": "2026-08-29T14:30:00Z"
      }
      """
    And Tốc độ phản hồi ghi log < 10ms, hoàn toàn không làm chậm giao diện người dùng
  ```

---

#### US-20: Động Cơ Gợi Ý Sản Phẩm Cá Nhân Hóa (Next-Click Prediction Engine)
- **Mô tả**: *Là một Khách hàng (ACT-01/02), Tôi muốn nhìn thấy danh sách các sản phẩm liên quan phù hợp với sở thích của tôi ngay tại trang chi tiết, Để tôi dễ dàng chọn mua thêm đặc sản Huế mà tôi thích.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Hiển thị sản phẩm gợi ý dựa trên thói quen người dùng
    Given Khách hàng đang xem sản phẩm "Mè Xửng Giòn"
    When Hệ thống truy vấn thuật toán gợi ý (tổng hợp từ dữ liệu clickstream MongoDB và đơn hàng SQL)
    Then Trả về danh mục 3 sản phẩm có điểm tương đồng cao nhất (Ví dụ: Kẹo Gương Huế, Kẹo Đậu Phộng)
    And Dữ liệu được cache trên Redis "recom:product:{id}" với TTL 1 giờ để phản hồi tức thì < 20ms
  ```

---

### EPIC 09: PHÂN HỆ BI THỐNG KÊ & ĐỘNG CƠ RA QUYẾT ĐỊNH CHIẾN LƯỢC (BI & DSS ENGINE)
*Microservice phụ trách: `analytics-service` (Port 4008) | Database: `om_analytics_db` (Data Mart PostgreSQL)*

#### US-21: Báo Cáo Thống Kê Doanh Thu Đa Chiều & Trực Quan Hóa Tương Tác
- **Mô tả**: *Là một Giám đốc Kinh doanh (ACT-06), Tôi muốn xem báo cáo thống kê doanh thu đa chiều theo khoảng thời gian tùy chọn (ngày, tuần, tháng, quý), theo kênh bán (Website D2C, Mobile App, Điểm bán sỉ B2B) và theo dòng sản phẩm, Để tôi đánh giá chính xác hiệu quả kinh doanh của doanh nghiệp.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Xem báo cáo doanh thu đa chiều và Drill-down chi tiết
    Given Giám đốc truy cập phân hệ "/admin/bi-analytics/revenue"
    When Chọn bộ lọc: Thời gian "Quý 3/2026", Kênh bán "Tất cả kênh", Tiêu chí "Dòng sản phẩm"
    Then Dashboard hiển thị biểu đồ tương tác ECharts:
      | Chỉ số hiển thị                 | Ý nghĩa quản trị                                |
      | Tổng doanh thu & Tăng trưởng    | Doanh số đạt được so với cùng kỳ quý trước      |
      | Giá trị đơn trung bình (AOV)    | Mức chi tiêu trung bình trên 1 đơn hàng         |
      | Tỷ lệ hoàn đơn (Return Rate)    | Tỷ lệ đơn bị hoàn/hủy để kiểm soát chất lượng   |
      | Cơ cấu doanh thu theo dòng bánh | Tỷ trọng đóng góp của Mè dẻo, Mè giòn, Hộp quà  |
    And Cho phép nhấp vào từng cột biểu đồ để Drill-down xem chi tiết danh sách đơn hàng
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/reports/revenue-multidimensional`.

---

#### US-22: Tự Động Phân Khúc Khách Hàng Theo Ma Trận RFM
- **Mô tả**: *Là một Giám đốc Marketing (ACT-06), Tôi muốn hệ thống tự động phân loại toàn bộ khách hàng thành 4 nhóm (Champions VIP, Loyal, Potential, At Risk) dựa trên mô hình RFM, Để tôi triển khai các chiến dịch chăm sóc khách hàng cá nhân hóa.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Chạy tiến trình phân loại khách hàng RFM định kỳ
    Given Dữ liệu mua sắm của 5.000 khách hàng trong Data Mart
    When Hệ thống chạy Cronjob tính toán điểm Recency, Frequency, Monetary lúc 00:00 hàng ngày
    Then Toàn bộ khách hàng được gán nhãn phân khúc:
      - Nhóm Champions (VIP): Điểm R(4-5), F(4-5), M(4-5)
      - Nhóm Loyal (Thân thiết): Điểm R(3-5), F(3-4), M(3-4)
      - Nhóm Potential (Tiềm năng): Điểm R(4-5), F(1-2), M(1-3)
      - Nhóm At Risk (Nguy cơ rời bỏ): Điểm R(1-2), F(3-5), M(3-5)
    And Dashboard hiển thị biểu đồ phân bổ khách hàng hình phễu và bản đồ nhiệt Heatmap
    And Cho phép xuất danh sách email/SĐT của nhóm "At Risk" để tạo chiến dịch kích cầu
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/rfm-segments`, `POST /api/v1/analytics/rfm/export`.

---

#### US-23: Phân Tích Tốc Độ Bán & Dự Báo Cạn Kho (Sales Velocity & Lead Time Forecast)
- **Mô tả**: *Là một Quản lý Sản xuất Làng nghề (ACT-04/06), Tôi muốn biết tốc độ tiêu thụ hàng ngày và số ngày dự kiến cạn kho của từng dòng mè xửng, Để chủ động lên kế hoạch nhập nguyên liệu mè vừng, đậu phộng, đường mạch nha trước các mùa cao điểm.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Dự báo thời điểm cạn kho dựa trên tốc độ tiêu thụ
    Given Dữ liệu bán hàng 14 ngày qua của SKU "Mè Xửng Dẻo Hộp 500g"
    When Tốc độ bán đạt 65 hộp/ngày và số lượng tồn kho còn lại là 260 hộp
    Then Hệ thống tính toán "Số ngày tồn kho còn lại" = 260 / 65 = 4 ngày
    And Thời gian cần để nhập nguyên liệu và ngào mẻ bánh mới (Lead Time) là 5 ngày
    And Hệ thống bật cảnh báo khẩn cấp màu đỏ:
      "CẢNH BÁO CẠN KHO: Mè Xửng Dẻo 500g sẽ hết hàng sau 4 ngày nữa. Cần nhập 300kg đường mạch nha và 150kg mè vàng trước ngày 02/09."
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/inventory-forecast`.

---

#### US-24: Động Cơ Đề Xuất Chiến Lược Kinh Doanh Thông Minh (Strategic Insights Engine)
- **Mô tả**: *Là một Nhà Quản trị Doanh nghiệp (ACT-06), Tôi muốn hệ thống tự động phân tích quy luật và đưa ra các khuyến nghị chiến lược hành động (Gợi ý gói Combo tăng AOV, Chiến dịch Marketing mùa vụ Cố Đô/Lễ Tết, Tái kích hoạt khách hàng cũ), Để doanh nghiệp tối đa hóa doanh thu và nâng cao sức cạnh tranh.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Hệ thống tự động sinh 3 khuyến nghị chiến lược hành động
    Given Dữ liệu tổng hợp từ giao dịch SQL và nhật ký NoSQL MongoDB
    When Quản trị viên mở trang "/admin/strategic-insights"
    Then Hệ thống hiển thị 3 thẻ khuyến nghị thông minh:
      1. Khuyến nghị Combo (Market Basket): "Phát hiện 68% khách mua Mè Xửng Giòn mua kèm Trà Cung Đình -> Gợi ý tạo Bundle 'Thưởng Trà Xứ Huế' chiết khấu 10% để tăng AOV thêm 20%."
      2. Khuyến nghị Mùa vụ: "Còn 30 ngày đến Lễ Hội Festival Huế -> Đề xuất tăng 40% ngân sách quảng cáo cho từ khóa 'Đặc sản Huế làm quà' và chuẩn bị 2.000 hộp quà gỗ."
      3. Khuyến nghị Tái kích hoạt: "Có 52 khách hàng VIP chưa mua lại sau 60 ngày -> Đề xuất gửi Voucher giảm 15% kèm tin nhắn Zalo ZNS cá nhân hóa."
    And Cung cấp nút "Kích hoạt chiến dịch ngay" và nút "Xuất Báo cáo Chiến lược PDF / Excel"
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/strategic-insights/recommendations`, `POST /api/v1/analytics/reports/export-pdf`.

---

### EPIC 10: DEVOPS, GIÁM SÁT OBSERVABILITY & QUẢN TRỊ HỆ THỐNG (DEVOPS & ADMIN)
*Công cụ: Docker Compose, Nginx, Prometheus, Grafana, Jaeger, Loki, GitHub Actions*

#### US-25: Giám Sát Hiệu Năng Tập Trung (Prometheus, Grafana & Jaeger Tracing)
- **Mô tả**: *Là một Kỹ sư Vận hành (ACT-07), Tôi muốn theo dõi các chỉ số tải CPU, RAM, RPS, Response Time P95 và truy vết Trace-ID xuyên suốt các Microservices trên Grafana, Để phát hiện và khắc phục điểm nghẽn hệ thống tức thì.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Truy vết lỗi phân tán bằng Trace-ID trên Jaeger
    Given Khách hàng gặp lỗi khi thanh toán đơn hàng
    When Kỹ sư tra cứu "Trace-ID: tr_8899aabb" trên Dashboard Jaeger
    Then Hệ thống hiển thị toàn bộ cây tiến trình (Span Waterfall) qua các service:
      Gateway (20ms) -> Order Service (45ms) -> Payment Service (120ms - ERROR 500 Bank Timeout)
    And Giúp kỹ sư xác định chính xác nguyên nhân lỗi do cổng ngân hàng phản hồi chậm
  ```

---

#### US-26: Tự Động Hóa Pipeline CI/CD Đa Dịch Vụ Với GitHub Actions
- **Mô tả**: *Là một Lập trình viên trong nhóm (ACT-07), Tôi muốn mỗi lần tạo Pull Request vào nhánh `develop` thì hệ thống sẽ tự động chạy kiểm thử Lint, Unit Test, Contract Test và Build Docker Image, Để đảm bảo mã nguồn luôn đạt chuẩn chất lượng trước khi bàn giao.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Pipeline CI/CD chạy tự động khi có Pull Request
    Given Lập trình viên push mã nguồn và tạo PR vào nhánh "develop"
    When GitHub Actions kích hoạt workflow ".github/workflows/ci.yml"
    Then Tự động chạy song song:
      1. Kiểm tra Linting & TypeScript compilation
      2. Chạy bộ Unit Test (Jest/PyTest) yêu cầu Code Coverage >= 75%
      3. Chạy Pact Contract Test kiểm tra hợp đồng API giữa Gateway và các Service
      4. Đóng gói Docker Images đa tầng (Multi-stage build)
    And Nếu tất cả Passed (Xanh) -> Cho phép Merge PR
    And Nếu có bất kỳ bước nào Failed (Đỏ) -> Khóa nút Merge và gửi cảnh báo lỗi vào Discord/Telegram nhóm
  ```

---

## IV. LỘ TRÌNH 12 TUẦN (SPRINT PLANNING & SPRINT BACKLOG)

```
┌────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                              BẢNG PHÂN BỔ SPRINT BACKLOG CHI TIẾT (12 TUẦN)                            │
├─────────┬──────────────────────┬─────────────────────────────────────────────────┬─────────────────────┤
│ Sprint  │ Thời Gian            │ Danh Sách User Stories Triển Khai               │ Tổng Story Points   │
├─────────┼──────────────────────┼─────────────────────────────────────────────────┼─────────────────────┤
│ **SP-1**│ Tuần 1 – Tuần 2      │ Phân tích Bounded Contexts, DFD, SRS, US-01     │ **15 SP**           │
├─────────┼──────────────────────┼─────────────────────────────────────────────────┼─────────────────────┤
│ **SP-2**│ Tuần 3 – Tuần 4      │ Thiết kế CSDL 3NF, API Specs, US-02, US-03      │ **20 SP**           │
├─────────┼──────────────────────┼─────────────────────────────────────────────────┼─────────────────────┤
│ **SP-3**│ Tuần 5 – Tuần 6      │ US-04 (Catalog), US-05 (Redlock), US-06 (SEO)   │ **25 SP**           │
├─────────┼──────────────────────┼─────────────────────────────────────────────────┼─────────────────────┤
│ **SP-4**│ Tuần 7 – Tuần 8      │ US-07 (SAGA), US-08 (VietQR), US-10 (Web/App)   │ **28 SP**           │
├─────────┼──────────────────────┼─────────────────────────────────────────────────┼─────────────────────┤
│ **SP-5**│ Tuần 9 – Tuần 10     │ US-18 (WebSocket), US-19 (NoSQL), US-21, 22, 24 │ **30 SP**           │
├─────────┼──────────────────────┼─────────────────────────────────────────────────┼─────────────────────┤
│ **SP-6**│ Tuần 11 – Tuần 12    │ US-25 (Observability), US-26 (CI/CD), Test, Doc │ **20 SP**           │
├─────────┴──────────────────────┴─────────────────────────────────────────────────┴─────────────────────┤
│ **TỔNG CỘNG 12 TUẦN (6 SPRINTS):**                                               │ **138 STORY POINTS**│
└────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## V. ĐỊNH NGHĨA HOÀN THÀNH (DEFINITION OF DONE - DOD) & MA TRẬN RỦI RO

### 5.1. Tiêu Chuẩn Nghiệm Thu Hoàn Thành Cho Mỗi User Story (Definition of Done)
Một User Story chỉ được chuyển sang cột `Done` trên bảng Jira khi thỏa mãn 7 tiêu chí bắt buộc:
1. **Mã Nguồn (Code Quality)**: Viết đúng Coding Convention, không còn cảnh báo Lint/TypeScript, đã được format qua Prettier/ESLint.
2. **Kiểm Thử Đơn Vị (Unit Testing)**: Đã viết Unit Test cho toàn bộ logic nghiệp vụ cốt lõi, độ bao phủ (Coverage) đạt `>= 75%`.
3. **Kiểm Thử Hợp Đồng & API (API Contract)**: Đã cập nhật file Swagger OpenAPI và vượt qua bài kiểm thử tích hợp (Integration Test).
4. **Bảo Mật (Security & Sanitization)**: Đã kiểm tra chống SQL Injection, XSS, kiểm tra quyền hạn RBAC và mã hóa dữ liệu nhạy cảm.
5. **Đánh Giá Mã Nguồn (Peer Review)**: Có ít nhất 1 thành viên trong nhóm Review và Approve Pull Request trên GitHub.
6. **Môi Trường Hoạt Động (Deployment)**: Chạy thông suốt trên môi trường Docker cục bộ không phát sinh lỗi ngoại lệ.
7. **Tài Liệu Kỹ Thuật (Documentation)**: Đã cập nhật tài liệu kiến trúc, sơ đồ cơ sở dữ liệu và tài liệu hướng dẫn sử dụng liên quan.

### 5.2. Ma Trận Quản Trị Rủi Ro & Kế Hoạch Dự Phòng

| Mã Rủi Ro | Nguy Cơ Tiềm Ẩn | Mức Độ | Kế Hoạch Phòng Ngừa & Ứng Phó |
| :---: | :--- | :---: | :--- |
| **RSK-01** | Bất đồng bộ dữ liệu giao dịch giữa các Microservices khi mạng chập chờn. | **Cao** | Triển khai **Transactional Outbox Pattern** kết hợp RabbitMQ đảm bảo tin nhắn gửi đi ít nhất 1 lần (At-Least-Once Delivery); áp dụng SAGA Compensating Transactions tự động nhả kho khi lỗi. |
| **RSK-02** | Xung đột và bán vượt tồn kho (Over-selling) khi có nhiều khách cùng đặt hàng. | **Cao** | Áp dụng khóa phân tán **Redis Redlock** với thời gian TTL 15 phút cho mỗi phiên giao dịch đặt hàng. |
| **RSK-03** | Khó khăn khi kiểm thử API Ngân hàng hoặc Giao vận thực tế. | **Trung bình** | Xây dựng bộ **Mock Server & Webhook Simulator** chuẩn OpenAPI giả lập đầy đủ các phản hồi thành công/thất bại của VietQR và GHN/GHTK. |
| **RSK-04** | Trùng lặp thanh toán khi nhận nhiều Webhook từ ngân hàng. | **Trung bình** | Triển khai cơ chế **Idempotent Consumer** với Redis Cache: Lưu mã giao dịch duy nhất `Transaction_ID`; nếu mã đã tồn tại lập tức trả về HTTP 200 OK mà không xử lý cộng tiền lần hai. |

---
*Tài liệu này là bản đặc tả User Stories và Kế hoạch phần mềm chính thức, được lưu trữ tại `docs/01_requirements/backlog_software.md` phục vụ công tác phát triển, kiểm thử và nghiệm thu đồ án chuyên ngành CNTT.*
