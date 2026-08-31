# USER STORIES & BUSINESS REQUIREMENTS

# HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ ĐA KÊNH & HỖ TRỢ QUYẾT ĐỊNH CHO OCOP HUẾ

## DỰ ÁN: MÈ XỬNG O MẠ

**Phiên bản:** 2.0  
**Loại tài liệu:** User Stories / Business Requirements / Functional Requirements  
**Phạm vi:** Website D2C + Mobile App + Web Admin + Marketplace + Offline Sales + Analytics/DSS  

---

# 1. MỤC ĐÍCH TÀI LIỆU

Tài liệu này mô tả nhu cầu nghiệp vụ, mục tiêu của các tác nhân, User Stories, Business Rules và Acceptance Criteria của hệ thống thương mại điện tử đa kênh **Mè Xửng O Mạ**.

Hệ thống được định hướng không chỉ là một website bán mè xửng mà là một **hệ sinh thái quản lý bán hàng đa kênh**, bao gồm:

* Bán hàng trực tiếp D2C trên Website.
* Bán hàng trên Mobile App.
* Bán hàng B2B.
* Bán hàng qua Shopee/TikTok Shop và các Marketplace khác.
* Bán hàng Offline tại cửa hàng, chợ, đại lý, tạp hóa.
* Quản lý sản phẩm, SKU, Batch/Lot và hạn sử dụng.
* Quản lý kho.
* Quản lý đóng gói và bằng chứng video đóng gói.
* Quản lý giao hàng.
* Quản lý khách hàng và chăm sóc khách hàng.
* Quản lý nội dung, SEO và Marketing.
* Truy xuất nguồn gốc sản phẩm OCOP.
* Phân tích doanh thu, chi phí, thuế và lợi nhuận.
* Phân tích hành vi khách hàng.
* Hỗ trợ dự báo và ra quyết định chiến lược.

---

# 2. NGUYÊN TẮC PHÂN TÁCH YÊU CẦU

Tài liệu tuân thủ nguyên tắc:

```text
USER STORY
    ↓
BUSINESS RULE
    ↓
ACCEPTANCE CRITERIA
    ↓
USE CASE
    ↓
FUNCTIONAL REQUIREMENT
    ↓
NON-FUNCTIONAL REQUIREMENT
    ↓
TECHNICAL DESIGN
    ↓
IMPLEMENTATION
```

User Story chỉ mô tả:

* Ai?
* Muốn làm gì?
* Để đạt được giá trị gì?

Không đưa trực tiếp các công nghệ như:

* Redis
* Kafka/RabbitMQ
* Microservices
* JWT
* PostgreSQL
* S3
* MinIO
* HMAC-SHA256
* WebSocket

vào nội dung User Story.

Các công nghệ chỉ được xác định ở tài liệu Architecture/Technical Design.

---

# 3. ACTORS

## 3.1. Customer & External Business Actors

| ID     | Actor                | Mô tả                                              |
| ------ | -------------------- | -------------------------------------------------- |
| ACT-01 | Guest Customer       | Khách chưa đăng nhập                               |
| ACT-02 | Registered Customer  | Khách hàng cá nhân đã đăng ký                      |
| ACT-03 | B2B Customer         | Đại diện doanh nghiệp mua số lượng lớn             |
| ACT-04 | Marketplace Customer | Khách mua hàng thông qua Shopee/TikTok/Marketplace |
| ACT-05 | Offline Customer     | Khách mua trực tiếp tại cửa hàng/điểm bán          |

---

## 3.2. Internal Actors

| ID     | Actor                        | Trách nhiệm                                                 |
| ------ | ---------------------------- | ----------------------------------------------------------- |
| ACT-06 | Warehouse Staff              | Quản lý tồn kho, Batch/Lot, nhập/xuất kho                   |
| ACT-07 | Packing Staff                | Lấy hàng, đóng gói, quay/lưu video đóng gói                 |
| ACT-08 | Delivery Staff               | Giao hàng nội bộ                                            |
| ACT-09 | Marketplace Operator         | Vận hành đơn hàng Shopee/TikTok                             |
| ACT-10 | Offline Sales Staff          | Bán hàng tại cửa hàng/chợ/tạp hóa/đại lý                    |
| ACT-11 | Customer Service             | Hỗ trợ khách hàng, khiếu nại, đổi trả                       |
| ACT-12 | Sales Manager                | Quản lý bán hàng, giá, đơn hàng, chương trình bán hàng      |
| ACT-13 | Content Manager              | Quản lý bài viết, nội dung Website và SEO                   |
| ACT-14 | Marketing Staff              | Lập kế hoạch nội dung, chiến dịch Marketing và theo dõi KPI |
| ACT-15 | Inventory/Supply Manager     | Theo dõi cung ứng, hạn sử dụng, kế hoạch sản xuất           |
| ACT-16 | Executive / Business Manager | Phân tích và ra quyết định kinh doanh                       |
| ACT-17 | System Administrator         | Quản trị tài khoản, Role, Permission và hệ thống            |
| ACT-18 | Auditor / Security Operator  | Kiểm tra Audit Log và truy vết hoạt động                    |

---

# 4. EXTERNAL SYSTEM ACTORS

| ID     | External Actor                 | Vai trò                                                 |
| ------ | ------------------------------ | ------------------------------------------------------- |
| EXT-01 | Payment Provider               | Xử lý thanh toán                                        |
| EXT-02 | Bank / Payment Confirmation    | Xác nhận giao dịch                                      |
| EXT-03 | Logistics Provider             | Giao vận và tracking                                    |
| EXT-04 | Marketplace Platform           | Shopee, TikTok Shop và các sàn khác                     |
| EXT-05 | Identity Provider              | Google, Facebook, Zalo hoặc nhà cung cấp đăng nhập khác |
| EXT-06 | Notification Provider          | Email, SMS, Push, Zalo                                  |
| EXT-07 | Analytics Platform             | Google Analytics 4 hoặc hệ thống phân tích bên ngoài    |
| EXT-08 | AI Provider                    | Dịch vụ AI bên ngoài                                    |
| EXT-09 | Media / Object Storage         | Hệ thống lưu trữ ảnh/video                              |
| EXT-10 | Origin / Traceability Provider | Nguồn dữ liệu xác minh nguồn gốc sản phẩm               |
| EXT-11 | Tax / Invoice Provider         | Hệ thống hóa đơn điện tử / dữ liệu thuế nếu tích hợp    |
| EXT-12 | Social / Marketing Platform    | Facebook, TikTok, Google và các nền tảng quảng cáo      |

---

# 5. ACTOR RESPONSIBILITY MATRIX

| Actor                | Mục tiêu chính                        |
| -------------------- | ------------------------------------- |
| Guest                | Tìm hiểu và mua sản phẩm nhanh        |
| Customer             | Mua hàng, theo dõi, đánh giá, mua lại |
| B2B                  | Đặt hàng số lượng lớn và nhận báo giá |
| Marketplace Customer | Mua qua sàn                           |
| Warehouse            | Đảm bảo tồn kho chính xác             |
| Packing              | Đóng gói đúng đơn và lưu bằng chứng   |
| Delivery             | Giao hàng                             |
| Marketplace Operator | Đồng bộ và xử lý đơn từ sàn           |
| Offline Sales        | Bán hàng trực tiếp và báo cáo tồn     |
| CSKH                 | Hỗ trợ khách hàng                     |
| Sales Manager        | Điều hành hoạt động bán hàng          |
| Content Manager      | Quản lý nội dung                      |
| Marketing            | Xây dựng và đánh giá chiến dịch       |
| Supply Manager       | Quản lý Batch, HSD và cung ứng        |
| Executive            | Phân tích và quyết định               |
| System Admin         | Tài khoản, Role, Permission           |
| Auditor              | Truy vết và kiểm tra Audit            |

---

# 6. BUSINESS RULES

## BR-AUTH

### BR-AUTH-01
Khách vãng lai được phép mua hàng mà không bắt buộc tạo tài khoản.

### BR-AUTH-02
Khách hàng có thể đăng ký tài khoản bằng thông tin cá nhân hoặc thông qua nhà cung cấp Identity Provider được hỗ trợ.

### BR-AUTH-03
Tài khoản nội bộ phải được gán Role phù hợp trước khi truy cập chức năng quản trị.

### BR-AUTH-04
Một nhân viên chỉ được thực hiện những thao tác thuộc quyền được cấp.

---

# 7. BUSINESS RULES — PRODUCT

### BR-PROD-01
Mỗi sản phẩm có thể có nhiều Variant/SKU.

### BR-PROD-02
Giá, khối lượng, quy cách đóng gói và tồn kho được quản lý theo SKU.

### BR-PROD-03
Sản phẩm thực phẩm phải có thông tin ngày sản xuất và hạn sử dụng phù hợp.

### BR-PROD-04
Thông tin sản phẩm hiển thị cho khách phải được quản lý và phê duyệt trước khi công khai.

---

# 8. BUSINESS RULES — BATCH & EXPIRY

### BR-BATCH-01
Tồn kho thực phẩm phải có thể truy xuất theo Batch/Lot.

### BR-BATCH-02
Mỗi Batch/Lot phải có:
* Mã lô
* Ngày sản xuất
* Hạn sử dụng
* Số lượng nhập
* Số lượng còn lại
* Nhà cung cấp/nguồn sản xuất nếu có

### BR-BATCH-03
Hệ thống phải cảnh báo trước khi Batch/Lot đến hạn sử dụng.

### BR-BATCH-04
Sản phẩm gần hết hạn phải được ưu tiên xử lý theo chính sách của doanh nghiệp.

### BR-BATCH-05
Sản phẩm hết hạn không được phép tiếp tục bán.

### BR-BATCH-06
Việc xuất kho nên ưu tiên Batch có hạn sử dụng gần hơn khi phù hợp với chính sách tồn kho.

---

# 9. BUSINESS RULES — ORDER

### BR-ORDER-01
Đơn hàng phải có trạng thái rõ ràng trong toàn bộ vòng đời.

### BR-ORDER-02
Không xác nhận đơn vượt quá tồn kho khả dụng.

### BR-ORDER-03
Đơn hàng chỉ chuyển sang Paid khi giao dịch được xác nhận hợp lệ.

### BR-ORDER-04
Khách hàng chỉ được tự hủy đơn trong thời gian được phép.

### BR-ORDER-05
Mọi thay đổi trạng thái quan trọng của đơn hàng phải được ghi nhận để truy vết.

---

# 10. BUSINESS RULES — PACKING

### BR-PACK-01
Chỉ nhân viên đóng gói được phân công mới được xác nhận hoàn thành công đoạn đóng gói.

### BR-PACK-02
Nhân viên phải kiểm tra SKU và số lượng trước khi đóng gói.

### BR-PACK-03
Đơn hàng có thể yêu cầu lưu bằng chứng hình ảnh/video đóng gói.

### BR-PACK-04
Video đóng gói phải được liên kết với Order/Shipment tương ứng.

### BR-PACK-05
Video đóng gói không được cho phép người không có quyền truy cập tùy ý xem hoặc tải xuống.

---

# 11. BUSINESS RULES — MARKETPLACE

### BR-MKTPLACE-01
Đơn hàng từ Marketplace phải được xác định nguồn phát sinh.

### BR-MKTPLACE-02
Một đơn Marketplace phải có mã tham chiếu của nền tảng gốc.

### BR-MKTPLACE-03
Đơn Marketplace phải được đối soát với đơn nội bộ.

### BR-MKTPLACE-04
Không được tạo trùng đơn nội bộ khi đồng bộ lại cùng một đơn Marketplace.

---

# 12. BUSINESS RULES — OFFLINE SALES

### BR-OFFLINE-01
Nhân viên bán hàng Offline phải được xác định danh tính.

### BR-OFFLINE-02
Hàng lấy từ kho cho nhân viên bán Offline phải được ghi nhận.

### BR-OFFLINE-03
Nhân viên phải báo cáo số lượng bán, tồn, hàng hỏng và hàng trả theo kỳ.

### BR-OFFLINE-04
Tồn kho Offline phải được đối soát với tồn kho hệ thống.

---

# 13. BUSINESS RULES — REVIEW

### BR-REVIEW-01
Chỉ khách hàng đã mua sản phẩm mới được tạo Verified Review.

### BR-REVIEW-02
Review phải gắn với sản phẩm/Variant tương ứng.

### BR-REVIEW-03
Nội dung review vi phạm chính sách có thể bị ẩn hoặc xử lý bởi nhân viên có quyền.

---

# 14. BUSINESS RULES — AUDIT

### BR-AUDIT-01
Các thao tác quan trọng phải được ghi Audit Log.

### BR-AUDIT-02
Audit phải xác định được:
* Who
* What
* When
* Where
* Why nếu có lý do nghiệp vụ
* Object/Resource
* Before Value
* After Value

### BR-AUDIT-03
Audit Log sau khi ghi phải có cơ chế bảo vệ chống sửa/xóa trái phép.

---

# 15. EPIC MAP

```text
EPIC 01  Authentication & Identity
EPIC 02  Customer & Employee Profile
EPIC 03  Product Discovery
EPIC 04  Product, Variant & SKU
EPIC 05  Shopping Cart
EPIC 06  Checkout
EPIC 07  Payment
EPIC 08  Order Management
EPIC 09  Inventory & Batch
EPIC 10  Packing & Fulfillment
EPIC 11  Shipping & Delivery
EPIC 12  Marketplace
EPIC 13  Offline Sales
EPIC 14  Return / Refund / Complaint
EPIC 15  Review & Rating
EPIC 16  Customer Service
EPIC 17  Promotion & Loyalty
EPIC 18  Gift & B2B
EPIC 19  OCOP Traceability
EPIC 20  Content & SEO
EPIC 21  Marketing
EPIC 22  User / Role / Permission Administration
EPIC 23  Audit & Security
EPIC 24  Finance & Business Analytics
EPIC 25  Customer Analytics & DSS
EPIC 26  AI / Intelligent Assistance
```

---

# 16. EPIC 01 — AUTHENTICATION & IDENTITY

### US-AUTH-01 — Guest Checkout
**Actor:** Guest  
> Là một khách vãng lai, tôi muốn đặt hàng mà không cần tạo tài khoản để có thể mua sản phẩm nhanh chóng.  
**Priority:** Must Have — MVP

### US-AUTH-02 — Customer Registration
> Là một khách hàng, tôi muốn đăng ký tài khoản để lưu thông tin và theo dõi lịch sử mua hàng.  
**Priority:** Must Have — MVP

### US-AUTH-03 — Login
> Là một khách hàng, tôi muốn đăng nhập tài khoản để truy cập thông tin cá nhân và đơn hàng.  
**Priority:** Must Have — MVP

### US-AUTH-04 — Social Login
> Là một khách hàng, tôi muốn đăng nhập bằng tài khoản Google, Facebook hoặc Zalo khi được hỗ trợ để không phải tạo mật khẩu mới.  
**Priority:** Should Have — Phase 2

### US-AUTH-05 — Password Recovery
> Là một khách hàng, tôi muốn khôi phục quyền truy cập tài khoản khi quên mật khẩu.  
**Priority:** Must Have — MVP

### US-AUTH-06 — Link Guest Order
> Là một khách hàng đã từng mua với tư cách Guest, tôi muốn liên kết đơn hàng cũ với tài khoản mới để theo dõi lịch sử mua hàng.  
**Priority:** Should Have — Phase 2

---

# 17. EPIC 02 — CUSTOMER & EMPLOYEE PROFILE

### US-USER-01 — Customer Profile
> Là một khách hàng, tôi muốn quản lý họ tên, số điện thoại, email và thông tin cá nhân để sử dụng khi mua hàng.

### US-USER-02 — Customer Address
> Là một khách hàng, tôi muốn lưu nhiều địa chỉ giao hàng để đặt hàng nhanh hơn.

### US-USER-03 — Employee Profile
> Là quản trị viên, tôi muốn quản lý hồ sơ nhân viên gồm mã nhân viên, họ tên, phòng ban, chức vụ và trạng thái làm việc để quản lý người dùng nội bộ.

### US-USER-04 — Employee Status
> Là quản trị viên, tôi muốn khóa tài khoản nhân viên khi nhân viên nghỉ việc hoặc không còn quyền truy cập.

### US-USER-05 — Customer Account Status
> Là nhân viên có quyền quản lý khách hàng, tôi muốn xem và xử lý trạng thái tài khoản khách hàng khi cần thiết.

---

# 18. EPIC 03 — PRODUCT DISCOVERY

### US-DISC-01
> Là khách hàng, tôi muốn xem danh mục sản phẩm để tìm sản phẩm phù hợp.

### US-DISC-02
> Là khách hàng, tôi muốn tìm kiếm sản phẩm theo từ khóa để nhanh chóng tìm được sản phẩm mong muốn.

### US-DISC-03
> Là khách hàng, tôi muốn lọc theo giá, khối lượng, loại sản phẩm, đánh giá và tình trạng còn hàng để thu hẹp lựa chọn.

### US-DISC-04
> Là khách hàng, tôi muốn xem sản phẩm bán chạy hoặc được đề xuất để dễ lựa chọn.

### US-DISC-05
> Là khách hàng, tôi muốn xem câu chuyện thương hiệu và văn hóa Huế để hiểu giá trị của sản phẩm.

---

# 19. EPIC 04 — PRODUCT, VARIANT & SKU

### US-PROD-01
> Là Sales Manager, tôi muốn tạo và cập nhật sản phẩm để đưa sản phẩm lên hệ thống bán hàng.

### US-PROD-02
> Là Sales Manager, tôi muốn quản lý Variant/SKU theo khối lượng, hương vị và quy cách đóng gói.

### US-PROD-03
> Là khách hàng, tôi muốn chọn Variant/SKU trước khi mua để nhận đúng sản phẩm.

### US-PROD-04
> Là khách hàng, tôi muốn xem thành phần, khối lượng, ngày sản xuất, hạn sử dụng và hướng dẫn bảo quản.

### US-PROD-05
> Là Sales Manager, tôi muốn quản lý giá bán theo từng SKU để đảm bảo giá hiển thị chính xác.

### US-PROD-06
> Là Sales Manager, tôi muốn quản lý trạng thái sản phẩm như đang bán, tạm ngừng bán hoặc ngừng kinh doanh.

---

# 20. EPIC 05 — SHOPPING CART

### US-CART-01
> Là khách hàng, tôi muốn thêm sản phẩm vào giỏ hàng để chuẩn bị mua.

### US-CART-02
> Là khách hàng, tôi muốn thay đổi số lượng sản phẩm trong giỏ.

### US-CART-03
> Là khách hàng, tôi muốn xóa sản phẩm khỏi giỏ.

### US-CART-04
> Là khách hàng, tôi muốn xem tổng tiền tạm tính trước khi checkout.

---

# 21. EPIC 06 — CHECKOUT

### US-CHK-01
> Là khách hàng, tôi muốn nhập thông tin nhận hàng để hoàn tất đơn.

### US-CHK-02
> Là khách hàng, tôi muốn xem phí vận chuyển trước khi xác nhận đơn.

### US-CHK-03
> Là khách hàng, tôi muốn được tạm giữ sản phẩm trong thời gian thanh toán để tránh sản phẩm bị người khác mua mất.

### US-CHK-04
> Là khách hàng, tôi muốn kiểm tra lại sản phẩm, số lượng, địa chỉ, phí vận chuyển và tổng tiền trước khi đặt hàng.

### US-CHK-05
> Là khách hàng, tôi muốn sử dụng mã ưu đãi hợp lệ trong quá trình checkout.

---

# 22. EPIC 07 — PAYMENT

### US-PAY-01
> Là khách hàng, tôi muốn thanh toán bằng chuyển khoản/QR để hoàn tất đơn hàng thuận tiện.

### US-PAY-02
> Là khách hàng, tôi muốn thanh toán COD nếu phương thức này được hỗ trợ.

### US-PAY-03
> Là khách hàng, tôi muốn biết trạng thái thanh toán của đơn hàng.

### US-PAY-04
> Là nhân viên bán hàng, tôi muốn kiểm tra trạng thái thanh toán để xử lý đơn chính xác.

### US-PAY-05
> Là quản lý, tôi muốn đối soát giao dịch thanh toán với đơn hàng để phát hiện sai lệch.

---

# 23. EPIC 08 — ORDER MANAGEMENT

### US-ORD-01
> Là khách hàng, tôi muốn xem lịch sử đơn hàng để biết các lần mua trước.

### US-ORD-02
> Là khách hàng, tôi muốn xem trạng thái hiện tại của đơn hàng.

### US-ORD-03
> Là khách hàng, tôi muốn nhận thông báo khi đơn hàng thay đổi trạng thái.

### US-ORD-04
> Là khách hàng, tôi muốn hủy đơn khi đơn còn đủ điều kiện hủy.

### US-ORD-05
> Là Sales Manager, tôi muốn xem và xử lý danh sách đơn hàng để đảm bảo đơn được thực hiện đúng.

### US-ORD-06
> Là nhân viên vận hành, tôi muốn biết đơn nào cần xử lý trước để đảm bảo SLA giao hàng.

---

# 24. EPIC 09 — INVENTORY & BATCH

### US-INV-01
> Là Warehouse Staff, tôi muốn xem tồn kho theo SKU để biết số lượng sản phẩm khả dụng.

### US-INV-02
> Là Warehouse Staff, tôi muốn ghi nhận nhập kho để cập nhật số lượng thực tế.

### US-INV-03
> Là Warehouse Staff, tôi muốn ghi nhận xuất kho để theo dõi lượng hàng đã sử dụng.

### US-INV-04
> Là Warehouse Staff, tôi muốn quản lý Batch/Lot để truy xuất từng lô sản phẩm.

### US-INV-05
> Là Warehouse Staff, tôi muốn quản lý ngày sản xuất và hạn sử dụng của từng Batch.

### US-INV-06
> Là Warehouse Staff, tôi muốn nhận cảnh báo khi Batch/Lot sắp hết hạn để chủ động xử lý.

### US-INV-07
> Là Supply Manager, tôi muốn xem tốc độ tiêu thụ và tồn kho để lập kế hoạch sản xuất.

### US-INV-08
> Là Warehouse Staff, tôi muốn hệ thống ưu tiên xuất các Batch gần hết hạn phù hợp với chính sách để giảm lãng phí.

### US-INV-09
> Là Warehouse Staff, tôi muốn ghi nhận hàng hỏng, thất thoát hoặc điều chỉnh tồn kho để số liệu phản ánh thực tế.

### US-INV-10
> Là Sales Manager, tôi muốn nhận cảnh báo sản phẩm sắp hết hạn để phối hợp với Sales/CSKH triển khai chương trình bán phù hợp.

---

# 25. EPIC 10 — PACKING & FULFILLMENT

### US-PACK-01
> Là Packing Staff, tôi muốn xem danh sách đơn cần đóng gói để biết đơn nào phải xử lý.

### US-PACK-02
> Là Packing Staff, tôi muốn xem chi tiết sản phẩm và số lượng cần đóng gói để tránh lấy sai hàng.

### US-PACK-03
> Là Packing Staff, tôi muốn xác nhận từng bước kiểm tra đơn trước khi hoàn tất đóng gói.

### US-PACK-04
> Là Packing Staff, tôi muốn lưu video quá trình đóng gói để làm bằng chứng khi cần kiểm tra.

### US-PACK-05
> Là Packing Staff, tôi muốn liên kết video đóng gói với Order để việc tra cứu sau này dễ dàng.

### US-PACK-06
> Là Admin/Manager/CSKH được cấp quyền, tôi muốn tra cứu video đóng gói của một đơn hàng để kiểm tra khi xảy ra sai sót hoặc khiếu nại.

### US-PACK-07
> Là Packing Staff, tôi muốn biết đơn nào có thời hạn giao gần nhất để ưu tiên đóng gói.

### US-PACK-08
> Là Packing Staff, tôi muốn xác nhận đơn đã đóng gói xong để chuyển sang công đoạn giao hàng.

---

# 26. EPIC 11 — SHIPPING & DELIVERY

### US-SHIP-01
> Là nhân viên vận hành, tôi muốn tạo vận đơn cho đơn hàng để bàn giao cho đơn vị giao hàng.

### US-SHIP-02
> Là khách hàng, tôi muốn theo dõi trạng thái giao hàng.

### US-SHIP-03
> Là Delivery Staff, tôi muốn xem danh sách đơn được giao cho mình để thực hiện giao hàng.

### US-SHIP-04
> Là Delivery Staff, tôi muốn cập nhật trạng thái giao hàng để hệ thống phản ánh đúng tiến trình.

### US-SHIP-05
> Là quản lý, tôi muốn xem tỷ lệ giao thành công/thất bại để đánh giá hiệu quả giao hàng.

### US-SHIP-06
> Là hệ thống, tôi muốn nhận cập nhật trạng thái từ đơn vị giao vận để đồng bộ trạng thái đơn hàng.

---

# 27. EPIC 12 — MARKETPLACE

### US-MKT-01
> Là Marketplace Operator, tôi muốn xem các đơn hàng phát sinh trên Shopee/TikTok để xử lý.

### US-MKT-02
> Là Marketplace Operator, tôi muốn đưa đơn hàng từ Marketplace vào quy trình xử lý nội bộ để nhân viên kho và đóng gói thực hiện.

### US-MKT-03
> Là Marketplace Operator, tôi muốn theo dõi trạng thái đồng bộ giữa Marketplace và hệ thống nội bộ để phát hiện lỗi.

### US-MKT-04
> Là Marketplace Operator, tôi muốn đối soát đơn hàng, doanh thu và phí từ Marketplace.

### US-MKT-05
> Là Manager, tôi muốn xem doanh thu theo từng Marketplace để đánh giá hiệu quả từng kênh.

### US-MKT-06
> Là hệ thống, tôi muốn lưu mã đơn hàng gốc của Marketplace để tránh tạo đơn trùng.

---

# 28. EPIC 13 — OFFLINE SALES

### US-OFF-01
> Là Offline Sales Staff, tôi muốn nhận hàng từ kho để mang đi bán tại điểm bán.

### US-OFF-02
> Là Offline Sales Staff, tôi muốn ghi nhận số lượng hàng đã bán để cập nhật tình hình kinh doanh.

### US-OFF-03
> Là Offline Sales Staff, tôi muốn ghi nhận hàng còn tồn, hỏng hoặc trả lại.

### US-OFF-04
> Là Offline Sales Staff, tôi muốn gửi báo cáo bán hàng theo ngày/tháng.

### US-OFF-05
> Là Sales Manager, tôi muốn xem doanh thu và tồn kho theo từng nhân viên/điểm bán.

### US-OFF-06
> Là Manager, tôi muốn đối soát lượng hàng đã cấp cho điểm bán với lượng hàng đã bán và còn tồn.

---

# 29. EPIC 14 — RETURN, REFUND & COMPLAINT

### US-RET-01
> Là khách hàng, tôi muốn yêu cầu hủy đơn khi đơn còn đủ điều kiện.

### US-RET-02
> Là khách hàng, tôi muốn gửi yêu cầu đổi/trả khi sản phẩm bị lỗi, hư hỏng hoặc không đúng đơn.

### US-RET-03
> Là khách hàng, tôi muốn đính kèm hình ảnh/video khiếu nại để chứng minh tình trạng sản phẩm.

### US-RET-04
> Là CSKH, tôi muốn xem thông tin đơn hàng và bằng chứng liên quan để xử lý khiếu nại.

### US-RET-05
> Là Sales Manager, tôi muốn duyệt hoặc từ chối yêu cầu đổi/trả theo chính sách.

### US-RET-06
> Là Manager, tôi muốn xử lý hoàn tiền cho khách hàng khi yêu cầu hợp lệ.

### US-RET-07
> Là CSKH, tôi muốn tra cứu video đóng gói để kiểm tra nguyên nhân khi khách phản ánh thiếu/sai/hỏng sản phẩm.

---

# 30. EPIC 15 — REVIEW & RATING

### US-REV-01
> Là khách hàng đã mua hàng, tôi muốn đánh giá sản phẩm để chia sẻ trải nghiệm.

### US-REV-02
> Là khách hàng, tôi muốn đăng hình ảnh thực tế cùng đánh giá.

### US-REV-03
> Là khách hàng, tôi muốn biết đánh giá nào đến từ người đã mua hàng thực tế.

### US-REV-04
> Là CSKH, tôi muốn xem các đánh giá tiêu cực để chủ động hỗ trợ khách hàng.

### US-REV-05
> Là Sales Manager, tôi muốn quản lý và xử lý những đánh giá vi phạm chính sách.

### US-REV-06
> Là Manager, tôi muốn phân tích đánh giá để phát hiện vấn đề về chất lượng sản phẩm hoặc dịch vụ.

---

# 31. EPIC 16 — CUSTOMER SERVICE

### US-CS-01
> Là khách hàng, tôi muốn gửi câu hỏi hoặc yêu cầu hỗ trợ.

### US-CS-02
> Là CSKH, tôi muốn xem danh sách yêu cầu hỗ trợ để xử lý theo mức độ ưu tiên.

### US-CS-03
> Là CSKH, tôi muốn xem lịch sử trao đổi với khách hàng để hỗ trợ chính xác.

### US-CS-04
> Là CSKH, tôi muốn xem đơn hàng, thanh toán, vận chuyển và lịch sử mua hàng của khách để xử lý vấn đề.

### US-CS-05
> Là CSKH, tôi muốn nhận cảnh báo các khách hàng có vấn đề hoặc khiếu nại chưa được xử lý.

### US-CS-06
> Là CSKH/Sales, tôi muốn nhận danh sách sản phẩm sắp hết hạn để phối hợp triển khai chương trình bán phù hợp.

---

# 32. EPIC 17 — PROMOTION & LOYALTY

### US-PROMO-01
> Là khách hàng, tôi muốn sử dụng coupon hợp lệ để nhận ưu đãi.

### US-PROMO-02
> Là Sales Manager, tôi muốn tạo coupon theo điều kiện để triển khai chương trình bán hàng.

### US-PROMO-03
> Là Sales Manager, tôi muốn tạo Combo sản phẩm để tăng giá trị đơn hàng.

### US-PROMO-04
> Là khách hàng, tôi muốn xem các chương trình khuyến mãi đang áp dụng.

### US-LOY-01
> Là khách hàng, tôi muốn tích điểm sau mỗi lần mua hàng.

### US-LOY-02
> Là khách hàng, tôi muốn đổi điểm thành ưu đãi.

### US-LOY-03
> Là khách hàng, tôi muốn đặt lại sản phẩm từ đơn hàng cũ để tiết kiệm thời gian.

---

# 33. EPIC 18 — GIFT & B2B

### US-GIFT-01
> Là khách hàng, tôi muốn gửi sản phẩm đến một người nhận khác với người đặt hàng để mua quà.

### US-GIFT-02
> Là khách hàng, tôi muốn thêm lời chúc vào đơn quà tặng.

### US-GIFT-03
> Là khách hàng, tôi muốn ẩn giá sản phẩm trên phiếu quà tặng.

### US-B2B-01
> Là khách hàng doanh nghiệp, tôi muốn gửi yêu cầu báo giá theo số lượng.

### US-B2B-02
> Là khách hàng doanh nghiệp, tôi muốn tải logo công ty để yêu cầu thiết kế bao bì.

### US-B2B-03
> Là Sales Manager, tôi muốn quản lý yêu cầu báo giá B2B để phản hồi khách hàng.

### US-B2B-04
> Là khách hàng doanh nghiệp, tôi muốn xác nhận báo giá để chuyển sang đặt hàng.

### US-B2B-05
> Là khách hàng doanh nghiệp, tôi muốn cung cấp thông tin doanh nghiệp để nhận hóa đơn phù hợp.

---

# 34. EPIC 19 — OCOP TRACEABILITY

### US-OCOP-01
> Là khách hàng, tôi muốn quét QR trên sản phẩm để kiểm tra nguồn gốc.

### US-OCOP-02
> Là khách hàng, tôi muốn xem thông tin vùng nguyên liệu, nhà sản xuất và quy trình sản xuất để tăng niềm tin.

### US-OCOP-03
> Là khách hàng, tôi muốn xem chứng nhận OCOP và các chứng nhận liên quan của sản phẩm.

### US-OCOP-04
> Là Manager, tôi muốn quản lý thông tin nguồn gốc của từng sản phẩm/lô hàng.

### US-OCOP-05
> Là Manager, tôi muốn cập nhật thông tin nhà cung cấp/nguyên liệu để đảm bảo dữ liệu truy xuất chính xác.

### US-OCOP-06
> Là Manager, tôi muốn kiểm tra và xác nhận nguồn dữ liệu nguồn gốc trước khi công khai cho khách hàng.

---

# 35. EPIC 20 — CONTENT & SEO

### US-CONTENT-01
> Là Content Manager, tôi muốn tạo bài viết để truyền tải câu chuyện văn hóa Huế và thương hiệu.

### US-CONTENT-02
> Là Content Manager, tôi muốn chỉnh sửa, xuất bản hoặc ẩn bài viết.

### US-CONTENT-03
> Là Content Manager, tôi muốn quản lý hình ảnh/video trong bài viết.

### US-CONTENT-04
> Là Content Manager, tôi muốn tối ưu tiêu đề, mô tả, từ khóa và thông tin SEO cho bài viết.

### US-CONTENT-05
> Là Content Manager, tôi muốn quản lý thông tin SEO cho từng sản phẩm để sản phẩm có khả năng được tìm thấy trên công cụ tìm kiếm.

### US-CONTENT-06
> Là Content Manager, tôi muốn quản lý URL thân thiện, Meta Title, Meta Description và nội dung liên quan SEO.

### US-CONTENT-07
> Là Manager, tôi muốn duyệt nội dung trước khi xuất bản để đảm bảo thông tin chính xác.

---

# 36. EPIC 21 — MARKETING

### US-MKTG-01
> Là Marketing Staff, tôi muốn lập kế hoạch Marketing theo chiến dịch để tổ chức hoạt động quảng bá.

### US-MKTG-02
> Là Marketing Staff, tôi muốn tạo nội dung video, bài viết và chương trình truyền thông cho từng chiến dịch.

### US-MKTG-03
> Là Marketing Staff, tôi muốn gửi kế hoạch cho người có quyền duyệt trước khi triển khai.

### US-MKTG-04
> Là Manager/Executive, tôi muốn duyệt hoặc từ chối kế hoạch Marketing.

### US-MKTG-05
> Là Marketing Staff, tôi muốn theo dõi KPI của chiến dịch sau khi triển khai.

### US-MKTG-06
> Là Marketing Staff, tôi muốn lấy dữ liệu hiệu quả từ các kênh Marketing để đánh giá chiến dịch.

### US-MKTG-07
> Là Executive, tôi muốn so sánh chi phí Marketing với doanh thu tạo ra để đánh giá hiệu quả đầu tư.

---

# 37. EPIC 22 — USER / ROLE / PERMISSION ADMINISTRATION

### US-ADM-01 — Employee Account
> Là System Administrator, tôi muốn tạo tài khoản cho nhân viên để cấp quyền sử dụng hệ thống.

### US-ADM-02 — Update Employee Account
> Là System Administrator, tôi muốn cập nhật thông tin tài khoản nhân viên để dữ liệu luôn chính xác.

### US-ADM-03 — Lock Employee Account
> Là System Administrator, tôi muốn khóa tài khoản nhân viên khi nhân viên không còn được phép truy cập.

### US-ADM-04 — Role Management
> Là System Administrator, tôi muốn tạo và quản lý Role để phân nhóm quyền truy cập.

### US-ADM-05 — Permission Management
> Là System Administrator, tôi muốn quản lý Permission để kiểm soát từng chức năng mà mỗi Role được phép sử dụng.

### US-ADM-06 — Assign Role
> Là System Administrator, tôi muốn gán Role cho nhân viên để nhân viên chỉ truy cập đúng phạm vi công việc.

### US-ADM-07 — Access Review
> Là System Administrator, tôi muốn xem quyền của từng nhân viên để kiểm tra việc phân quyền.

---

# 38. EPIC 23 — AUDIT & SECURITY

### US-AUDIT-01
> Là System Administrator/Auditor, tôi muốn tra cứu lịch sử thao tác để biết ai đã thực hiện hành động gì, khi nào và trên đối tượng nào.

### US-AUDIT-02
> Là Auditor, tôi muốn xem giá trị trước và sau khi dữ liệu quan trọng thay đổi để xác định nguyên nhân sai lệch.

### US-AUDIT-03
> Là Auditor, tôi muốn biết nguồn truy cập hoặc vị trí kỹ thuật của thao tác khi thông tin này khả dụng để phục vụ điều tra.

### US-AUDIT-04
> Là Auditor, tôi muốn biết lý do nghiệp vụ của các thay đổi nhạy cảm khi thao tác đó yêu cầu người dùng cung cấp lý do.

### US-AUDIT-05
> Là Security Operator, tôi muốn phát hiện Audit Log bị thay đổi hoặc mất tính liên tục để bảo vệ tính toàn vẹn của nhật ký.

### Technical Security Constraint
Audit Log phải được thiết kế theo cơ chế **tamper-evident**.

Ở tầng Technical Design có thể triển khai:

```text
Audit Record
    │
    ├── current_data
    ├── timestamp
    ├── actor
    ├── before_value
    ├── after_value
    │
    └── integrity
          ├── HMAC-SHA256
          └── prev_hash
```

Mỗi record tham chiếu hash của record trước để tạo chuỗi kiểm tra tính toàn vẹn.

---

# 39. EPIC 24 — FINANCE & BUSINESS ANALYTICS

### US-FIN-01
> Là Executive, tôi muốn xem doanh thu theo ngày/tháng/năm để đánh giá tình hình kinh doanh.

### US-FIN-02
> Là Executive, tôi muốn xem doanh thu theo từng kênh Website, App, Marketplace, B2B và Offline.

### US-FIN-03
> Là Executive, tôi muốn xem chi phí bán hàng và vận hành để đánh giá hiệu quả kinh doanh.

### US-FIN-04
> Là Executive, tôi muốn xem lợi nhuận theo sản phẩm để biết sản phẩm nào mang lại giá trị cao.

### US-FIN-05
> Là Executive, tôi muốn xem doanh thu theo khu vực để đánh giá thị trường.

### US-FIN-06
> Là Executive, tôi muốn theo dõi nghĩa vụ thuế và các khoản liên quan đến doanh thu để phục vụ quản lý tài chính.

### US-FIN-07
> Là Executive, tôi muốn xuất báo cáo tài chính/kinh doanh theo kỳ để phục vụ họp và ra quyết định.

---

# 40. EPIC 25 — CUSTOMER ANALYTICS & DSS

### US-DSS-01 — Dashboard
> Là Executive, tôi muốn xem Dashboard tổng quan về doanh thu, đơn hàng, khách hàng, tồn kho và lợi nhuận để nắm tình hình doanh nghiệp.

### US-DSS-02 — RFM
> Là Marketing/Executive, tôi muốn phân nhóm khách hàng theo hành vi mua hàng để xây dựng chính sách chăm sóc phù hợp.

### US-DSS-03 — Customer Churn
> Là Marketing Staff, tôi muốn phát hiện khách hàng có nguy cơ không mua lại để triển khai chiến dịch giữ chân.

### US-DSS-04 — Product Performance
> Là Executive, tôi muốn biết sản phẩm nào bán chạy, bán chậm hoặc đang giảm doanh số.

### US-DSS-05 — Sales Velocity
> Là Supply Manager, tôi muốn biết tốc độ tiêu thụ từng SKU để lập kế hoạch sản xuất.

### US-DSS-06 — Demand Forecast
> Là Executive/Supply Manager, tôi muốn dự báo nhu cầu sản phẩm trong tương lai để chuẩn bị nguyên liệu và sản xuất.

### US-DSS-07 — Expiry Risk
> Là Supply Manager, tôi muốn biết lượng hàng có nguy cơ hết hạn trong tương lai để chủ động xử lý.

### US-DSS-08 — Product Association
> Là Sales Manager, tôi muốn biết các sản phẩm thường được mua cùng nhau để xây dựng Combo.

### US-DSS-09 — Seasonal Analysis
> Là Marketing/Executive, tôi muốn phân tích nhu cầu theo mùa, lễ hội và dịp Tết để lập kế hoạch kinh doanh.

### US-DSS-10 — Strategic Recommendation
> Là Executive, tôi muốn nhận các khuyến nghị dựa trên dữ liệu để hỗ trợ quyết định về sản phẩm, giá, tồn kho và Marketing.

---

# 41. EPIC 26 — AI & INTELLIGENT ASSISTANCE

AI là **khả năng hỗ trợ**, không phải Actor nghiệp vụ bắt buộc.

### US-AI-01 — Customer Assistant
> Là khách hàng, tôi muốn được hỗ trợ tìm sản phẩm phù hợp dựa trên nhu cầu để dễ lựa chọn.

### US-AI-02 — Customer Support Assistant
> Là CSKH, tôi muốn được hệ thống hỗ trợ tổng hợp thông tin liên quan đến câu hỏi của khách để trả lời nhanh hơn.

### US-AI-03 — Content Assistant
> Là Content Manager, tôi muốn được hỗ trợ tạo tiêu đề, mô tả và nội dung Marketing để tăng hiệu quả sản xuất nội dung.

### US-AI-04 — Product Recommendation
> Là khách hàng, tôi muốn được đề xuất sản phẩm phù hợp với lịch sử hoặc hành vi mua hàng.

### US-AI-05 — Management Recommendation
> Là Executive, tôi muốn nhận đề xuất chiến lược dựa trên dữ liệu bán hàng, khách hàng và tồn kho để hỗ trợ quyết định.

### US-AI-06 — Internal Knowledge Assistant
> Là nhân viên nội bộ, tôi muốn tìm kiếm thông tin nghiệp vụ trong kho tri thức của doanh nghiệp để giải quyết công việc nhanh hơn.

---

# 42. EXTERNAL SYSTEM REQUIREMENTS

## 42.1. Identity Provider
Hệ thống có thể hỗ trợ đăng nhập thông qua:
```text
Google
Facebook
Zalo
```
External Identity Provider chỉ chịu trách nhiệm xác thực danh tính bên ngoài; hồ sơ nghiệp vụ Customer/Employee vẫn thuộc hệ thống.

## 42.2. Marketplace
Có thể tích hợp:
```text
Shopee
TikTok Shop
Các Marketplace khác
```
Các nghiệp vụ cần hỗ trợ:
* Đồng bộ sản phẩm.
* Đồng bộ đơn.
* Đồng bộ trạng thái.
* Đồng bộ khách hàng ở mức dữ liệu được phép.
* Đối soát doanh thu.
* Đối soát phí.
* Phát hiện lỗi đồng bộ.

## 42.3. Analytics
Có thể tích hợp:
```text
Google Analytics 4
```
Các dữ liệu có thể phục vụ:
* Product View
* Add To Cart
* Checkout
* Purchase
* Traffic Source
* Campaign
* Conversion

Dữ liệu Analytics bên ngoài phải được phân biệt với dữ liệu giao dịch nội bộ.

## 42.4. AI Provider
Hệ thống có thể sử dụng:
```text
External AI Provider
hoặc
Internal AI/RAG
```
AI có thể phục vụ:
* Customer Assistant
* Content Assistant
* Recommendation
* Internal Knowledge Search
* DSS Assistant

AI không được tự ý thay đổi dữ liệu kinh doanh quan trọng nếu không có cơ chế phê duyệt phù hợp.

## 42.5. Media Storage
Hệ thống cần lưu:
```text
Product Images
Content Images
Marketing Videos
Packing Videos
Customer Review Images/Videos
B2B Logo Files
```
Hạ tầng lưu trữ có thể sử dụng:
```text
Object Storage
Cloud Storage
Self-hosted Storage
```
Việc lựa chọn S3, MinIO, Cloudinary... thuộc Technical Architecture.

---

# 43. ACCEPTANCE CRITERIA — CÁC LUỒNG QUAN TRỌNG

## AC-01 — Inventory Concurrency
```gherkin
Given sản phẩm chỉ còn 1 đơn vị
When hai khách hàng cùng đặt sản phẩm
Then chỉ một đơn hàng được xác nhận tồn kho thành công
And không được tạo đơn vượt quá tồn kho thực tế
```

## AC-02 — Packing Video
```gherkin
Given đơn hàng OM-001 đang ở trạng thái cần đóng gói
When Packing Staff hoàn thành đóng gói
And lưu video đóng gói
Then video được liên kết với OM-001
And người không có quyền không thể truy cập video
```

## AC-03 — Packing Investigation
```gherkin
Given khách hàng khiếu nại thiếu sản phẩm
When CSKH tra cứu OM-001
Then hệ thống hiển thị thông tin đóng gói
And cho phép xem bằng chứng video nếu CSKH có quyền
```

## AC-04 — Expiry Warning
```gherkin
Given Batch B001 có hạn sử dụng sắp đến
When hệ thống kiểm tra tồn kho định kỳ
Then Warehouse Staff nhận được cảnh báo
And Supply/Sales Staff có thể xem danh sách sản phẩm cần xử lý
```

## AC-05 — Marketplace Order
```gherkin
Given có đơn hàng mới từ Marketplace
When hệ thống nhận thông tin đơn
Then đơn được ghi nhận với mã đơn Marketplace
And không tạo đơn nội bộ trùng lặp
And đơn được đưa vào quy trình fulfillment
```

## AC-06 — Offline Stock
```gherkin
Given Offline Sales Staff được cấp 100 hộp
When nhân viên báo cáo đã bán 60 hộp
Then hệ thống ghi nhận 40 hộp còn lại
And Manager có thể đối soát số lượng
```

## AC-07 — Audit
```gherkin
Given Manager thay đổi giá sản phẩm
When thay đổi được lưu
Then Audit Log ghi nhận actor
And thời gian
And đối tượng thay đổi
And giá trị trước
And giá trị sau
And lý do nếu nghiệp vụ yêu cầu
```

---

# 44. ORDER STATE MACHINE

```text
PENDING_PAYMENT
       │
       ├── PAID
       │     │
       │     └── CONFIRMED
       │            │
       │            └── PROCESSING
       │                   │
       │                   └── PACKED
       │                          │
       │                          └── SHIPPED
       │                                 │
       │                                 ├── DELIVERED
       │                                 │      │
       │                                 │      └── COMPLETED
       │                                 │
       │                                 └── DELIVERY_FAILED
       │                                        │
       │                                        └── RETURNED
       │
       ├── EXPIRED
       └── CANCELLED
```

Return flow:

```text
DELIVERED
    ↓
RETURN_REQUESTED
    ↓
REVIEWING
    ├── REJECTED
    └── APPROVED
             ↓
        RETURNED
             ↓
         REFUNDED
```

---

# 45. MEDIA ACCESS MODEL

Đối với ảnh/video, đặc biệt là Packing Video:

```text
Customer
    │
    └── Review Media

Content Manager
    │
    └── Content Media

Packing Staff
    │
    └── Packing Video

CSKH
    │
    └── Packing Video
       nếu được cấp quyền

Manager/Admin
    │
    └── Packing Video
       theo quyền
```

Không cho phép mọi nhân viên truy cập toàn bộ video.

---

# 46. USER STORY TRACEABILITY

Mỗi User Story phải có khả năng truy ngược:

```text
Business Goal
      ↓
Actor
      ↓
User Story
      ↓
Acceptance Criteria
      ↓
Use Case
      ↓
Functional Requirement
      ↓
API / UI / Database
      ↓
Test Case
```

Ví dụ:

```text
US-PACK-06
    ↓
UC-PACK-04
    ↓
FR-PACK-04
    ↓
Packing Video API
    ↓
Packing Video Storage
    ↓
Permission Check
    ↓
Test Case TC-PACK-04
```

---

# 47. FUNCTIONAL REQUIREMENTS

## FR-01 — Authentication
Hệ thống phải hỗ trợ đăng ký, đăng nhập, đăng xuất, khôi phục tài khoản và Guest Checkout.

## FR-02 — User Profile
Hệ thống phải quản lý riêng thông tin Customer và Employee.

## FR-03 — Role & Permission
Hệ thống phải quản lý Role và Permission, cho phép phân quyền theo chức năng.

## FR-04 — Product
Hệ thống phải quản lý Product, Category, Variant và SKU.

## FR-05 — Cart & Checkout
Hệ thống phải hỗ trợ giỏ hàng, checkout, địa chỉ, vận chuyển và tạm giữ tồn kho.

## FR-06 — Payment
Hệ thống phải hỗ trợ các phương thức thanh toán được doanh nghiệp cấu hình.

## FR-07 — Order
Hệ thống phải quản lý đầy đủ vòng đời đơn hàng.

## FR-08 — Inventory
Hệ thống phải quản lý tồn kho theo SKU và Batch/Lot.

## FR-09 — Expiry
Hệ thống phải quản lý ngày sản xuất, hạn sử dụng và cảnh báo hàng gần hết hạn.

## FR-10 — Packing
Hệ thống phải hỗ trợ quy trình đóng gói và bằng chứng đóng gói.

## FR-11 — Packing Video
Hệ thống phải cho phép lưu, liên kết, tra cứu và kiểm soát quyền truy cập Packing Video.

## FR-12 — Logistics
Hệ thống phải hỗ trợ vận đơn và tracking.

## FR-13 — Marketplace
Hệ thống phải hỗ trợ tiếp nhận và đối soát đơn từ Marketplace.

## FR-14 — Offline Sales
Hệ thống phải quản lý hàng cấp cho nhân viên/điểm bán Offline và báo cáo bán hàng.

## FR-15 — Customer Service
Hệ thống phải hỗ trợ Ticket/Complaint/Return.

## FR-16 — Review
Hệ thống phải hỗ trợ Verified Review.

## FR-17 — Promotion
Hệ thống phải hỗ trợ Coupon, Discount và Combo.

## FR-18 — Loyalty
Hệ thống phải hỗ trợ Loyalty Point và Reorder.

## FR-19 — B2B
Hệ thống phải hỗ trợ Quotation và Bulk Order.

## FR-20 — OCOP Traceability
Hệ thống phải hỗ trợ truy xuất nguồn gốc sản phẩm.

## FR-21 — Content
Hệ thống phải quản lý bài viết, hình ảnh, video và nội dung Website.

## FR-22 — SEO
Hệ thống phải hỗ trợ SEO cho cả Product và Content.

## FR-23 — Marketing
Hệ thống phải hỗ trợ lập kế hoạch, phê duyệt, thực thi và đánh giá chiến dịch Marketing.

## FR-24 — Finance
Hệ thống phải hỗ trợ báo cáo doanh thu, chi phí, lợi nhuận và các chỉ số tài chính cần thiết.

## FR-25 — Analytics
Hệ thống phải hỗ trợ phân tích hành vi, doanh thu, sản phẩm và khách hàng.

## FR-26 — DSS
Hệ thống phải hỗ trợ RFM, Forecast, Product Association, Seasonal Analysis và Strategic Recommendation.

## FR-27 — Audit
Hệ thống phải ghi nhận Audit Log đối với các thao tác quan trọng.

---

# 48. NON-FUNCTIONAL REQUIREMENTS

## NFR-01 — Performance
Các thao tác phổ biến như xem sản phẩm, tìm kiếm và thêm giỏ hàng phải có thời gian phản hồi phù hợp với trải nghiệm người dùng.

## NFR-02 — Mobile First
Website phải tối ưu cho thiết bị di động.

## NFR-03 — Availability
Hệ thống phải đảm bảo khả dụng cao trong các giai đoạn cao điểm như Tết và các mùa quà tặng.

## NFR-04 — Security
Dữ liệu Customer, Employee, Order, Payment và Audit phải được bảo vệ khỏi truy cập trái phép.

## NFR-05 — Authorization
Người dùng chỉ được truy cập dữ liệu và chức năng thuộc quyền được cấp.

## NFR-06 — Data Integrity
Thông tin đơn hàng, thanh toán và tồn kho phải đảm bảo tính nhất quán.

## NFR-07 — Audit Integrity
Audit Log phải có khả năng phát hiện việc sửa đổi hoặc xóa trái phép.

### Technical Security Constraint
Cơ chế Audit Integrity có thể sử dụng:
```text
HMAC-SHA256
+
prev_hash
+
hash chain
```
để phát hiện thay đổi trái phép trong chuỗi Audit.

## NFR-08 — Privacy
Hệ thống phải bảo vệ dữ liệu cá nhân và chỉ sử dụng dữ liệu theo mục đích được phép.

## NFR-09 — Media Security
Ảnh/video nội bộ như Packing Video phải được kiểm soát quyền truy cập.

## NFR-10 — Scalability
Hệ thống phải có khả năng mở rộng khi số lượng Customer, Order, Media và Marketplace tăng.

## NFR-11 — Observability
Hệ thống phải có khả năng theo dõi lỗi, hiệu năng và trạng thái xử lý nghiệp vụ.

---

# 49. MVP

## Customer
* Product Catalog
* Search
* Product Detail
* Variant/SKU
* Cart
* Checkout
* Guest Checkout
* Payment
* Order Tracking
* Customer Profile

## Operations
* Inventory
* Batch/Lot
* Expiry
* Order Processing
* Packing
* Packing Checklist
* Packing Video
* Basic Delivery

## Admin
* Employee Account
* Role
* Permission
* Product Management
* Order Management
* Inventory Management

## OCOP
* Product Story
* QR Traceability
* Basic Origin Information

## Content
* Product Content
* Basic Blog
* Basic SEO

---

# 50. PHASE 2

* Google/Facebook/Zalo Login
* Marketplace Integration
* Shopee/TikTok Order Management
* Offline Sales
* Internal Delivery
* Customer Service
* Review
* Return/Refund
* Coupon
* Combo
* Loyalty
* B2B
* Gift Order
* Advanced SEO
* Marketing Workflow
* Audit Dashboard
* Packing Video Investigation

---

# 51. PHASE 3

## Business Intelligence
* Revenue Analytics
* Cost Analytics
* Profit Analytics
* Tax Reporting
* Channel Analytics

## Customer Analytics
* RFM
* Churn Detection
* Customer Segmentation
* Customer Lifetime Value

## Supply Analytics
* Sales Velocity
* Expiry Risk
* Demand Forecast
* Inventory Forecast

## Strategic DSS
* Product Bundle Recommendation
* Seasonal Campaign Recommendation
* Customer Retention Recommendation
* Inventory/Supply Recommendation

## AI
* Customer Assistant
* CSKH Assistant
* Content Assistant
* Internal Knowledge Assistant
* AI Recommendation
* AI-supported DSS

---

# 52. OUT OF SCOPE CỦA USER STORY

Các nội dung sau **không được xem là User Story**, mà thuộc Architecture/Technical Design:

```text
Microservices
REST / GraphQL
Redis
Kafka / RabbitMQ
PostgreSQL
MongoDB
JWT
OAuth implementation
S3 / MinIO implementation
Cloudinary implementation
HMAC-SHA256 implementation details
Database schema
API endpoint
Caching strategy
Distributed Lock
Saga
Circuit Breaker
Deployment
Docker
Kubernetes
Cloud Architecture
```

Chúng chỉ được thiết kế **sau khi User Stories và Functional Requirements đã được chốt**.

---

# 53. REQUIREMENT TRACEABILITY PRINCIPLE

Mỗi tính năng quan trọng phải được trace theo:

```text
Actor
  ↓
User Story
  ↓
Acceptance Criteria
  ↓
Business Rule
  ↓
Use Case
  ↓
Functional Requirement
  ↓
Test Case
```

Ví dụ:

```text
Packing Staff
    ↓
US-PACK-04
    ↓
Save Packing Video
    ↓
BR-PACK-03
    ↓
UC-PACK-04
    ↓
FR-11
    ↓
TC-PACK-04
```

---

# 54. TỔNG QUAN HỆ SINH THÁI

```text
                         MÈ XỬNG O MẠ
                              │
       ┌──────────────────────┼──────────────────────┐
       │                      │                      │
      B2C                    B2B                  OFFLINE
       │                      │                      │
 Website / App          Quotation / Bulk        Shop / Chợ
       │                      │                      │
       └──────────────────────┼──────────────────────┘
                              │
                        ORDER PLATFORM
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
    INVENTORY              PACKING              DELIVERY
        │                     │                     │
   Batch / HSD           Video Evidence        Internal / 3PL
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                        MULTI-CHANNEL
                              │
             ┌────────────────┼────────────────┐
             │                │                │
          Website          Shopee          TikTok Shop
             │                │                │
             └────────────────┼────────────────┘
                              │
                       CUSTOMER DATA
                              │
             ┌────────────────┼────────────────┐
             │                │                │
           CRM              GA4              Review
             │                │                │
             └────────────────┼────────────────┘
                              │
                         ANALYTICS
                              │
                ┌─────────────┼─────────────┐
                │             │             │
               RFM         Forecast       DSS
                │             │             │
                └─────────────┼─────────────┘
                              │
                           EXECUTIVE
```

---

# 55. KẾT LUẬN

Hệ thống Mè Xửng O Mạ không được xác định đơn thuần là một Website bán hàng.

Mô hình nghiệp vụ mục tiêu là:

```text
DIGITAL COMMERCE
       +
OMNICHANNEL
       +
INVENTORY & FULFILLMENT
       +
CUSTOMER MANAGEMENT
       +
OCOP TRACEABILITY
       +
CONTENT & MARKETING
       +
BUSINESS ANALYTICS
       +
DECISION SUPPORT
```

User Stories trong tài liệu này là cơ sở để phát triển tiếp:

```text
User Stories
     ↓
Use Case Diagram
     ↓
Use Case Specification
     ↓
Activity Diagram
     ↓
Sequence Diagram
     ↓
Domain Model
     ↓
ERD
     ↓
API Specification
     ↓
Architecture
     ↓
Implementation
     ↓
Test Cases
```

**End of User Story Specification**
