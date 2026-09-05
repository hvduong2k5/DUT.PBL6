# USER STORIES & BUSINESS REQUIREMENTS

# HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ ĐA KÊNH & HỖ TRỢ QUYẾT ĐỊNH CHO OCOP HUẾ

## DỰ ÁN: MÈ XỬNG O MẠ

**Phiên bản:** 3.0  
**Loại tài liệu:** Product Backlog / Epic Map / User Stories / Business Rules  
**Phạm vi:** Website D2C + Mobile App + Web Admin + Marketplace + Offline Sales + Analytics/DSS  

---

# 1. MỤC ĐÍCH TÀI LIỆU

Tài liệu này đóng vai trò là **xương sống (Backbone)** của hệ thống thương mại điện tử đa kênh **Mè Xửng O Mạ**. Mục đích của tài liệu là thiết lập ranh giới nghiệp vụ (Scope) và làm cơ sở cho mọi tài liệu chi tiết khác.

Tài liệu này định nghĩa:
*   **Các Nhóm người dùng (Actor):** Những ai tương tác với hệ thống.
*   **Quy tắc nghiệp vụ cốt lõi (Business Rules):** Các ràng buộc không thể vi phạm.
*   **Bản đồ tính năng (Epic Map):** Bức tranh toàn cảnh về các nhóm chức năng lớn.
*   **Danh sách User Stories:** Toàn bộ ~170 User Stories cấu thành hệ thống kèm **Mức độ ưu tiên (Priority)**.

Hệ thống được định hướng không chỉ là một website bán mè xửng mà là một **hệ sinh thái quản lý bán hàng đa kênh**, bao gồm:
* Bán hàng trực tiếp D2C trên Website và Mobile App (iOS/Android).
* Bán hàng B2B (Báo giá, Mua sỉ) và bán hàng Offline tại cửa hàng.
* Bán hàng qua Marketplace (Shopee/TikTok Shop).
* Tìm kiếm, bộ lọc nâng cao và AI Assistant.
* Quản lý kho, đóng gói, giao hàng, lô/Batch, HSD thực phẩm và lưu bằng chứng video.
* Quản lý khách hàng, CSKH đa kênh, khiếu nại và quy trình đổi trả/hoàn tiền nghiêm ngặt.
* Truy xuất nguồn gốc minh bạch chứng nhận OCOP.
* Phân tích tài chính, hành vi khách hàng (RFM, Churn) và DSS hỗ trợ ra quyết định.

---

# 2. NGUYÊN TẮC TRUY XUẤT NGUỒN GỐC (TRACEABILITY)

Hệ thống tài liệu tuân thủ nguyên tắc phân tách và liên kết chặt chẽ theo luồng:

```text
BUSINESS GOAL 
    ↓
ACTOR (Nằm tại file này)
    ↓
USER STORY & PRIORITY (Nằm tại file này)
    ↓
EPIC & ACCEPTANCE CRITERIA (Nằm tại docs/01_requirements/epics/)
    ↓
USE CASE (Nằm tại use_case_specification.md)
    ↓
FUNCTIONAL REQUIREMENT (Nằm tại 04_Functional_Requirements.md)
    ↓
TECHNICAL DESIGN / ARCHITECTURE (Nằm tại docs/02_architecture/)
    ↓
IMPLEMENTATION```

User Story trong tài liệu này được viết theo chuẩn, chỉ tập trung giải quyết:
* Ai? (As a...)
* Muốn làm gì? (I want to...)
* Để đạt được giá trị gì? (So that...)

**Tuyệt đối không đưa trực tiếp các công nghệ** (như Redis, Kafka, PostgreSQL, S3, JWT, WebSocket...) vào nội dung User Story. Các quyết định về mặt kỹ thuật chỉ được xác định ở giai đoạn Technical Design.

**Ví dụ một chuỗi truy vết đầy đủ:**
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
| ACT-12 | Sales Manager                | Quản lý bán hàng, giá, đơn hàng, chương trình bán hàng, xem báo cáo bán hàng và xem tồn kho |
| ACT-13 | Content Manager              | Quản lý bài viết, nội dung Website, SEO và xem thống kê hiệu quả nội dung/SEO |
| ACT-14 | Marketing Staff              | Lập kế hoạch nội dung, chiến dịch Marketing và theo dõi KPI |
| ACT-15 | Inventory/Supply Manager     | Theo dõi cung ứng, hạn sử dụng, kế hoạch sản xuất           |
| ACT-16 | Executive / Business Manager | Xem Dashboard, phân tích doanh thu, lợi nhuận và ra quyết định chiến lược kinh doanh |
| ACT-17 | System Administrator         | Quản trị tài khoản, Role, Permission và hệ thống            |
| ACT-18 | Auditor / Security Operator  | Kiểm tra Audit Log và truy vết hoạt động                    |
| ACT-19 | Accountant / Finance Staff   | Đối soát thanh toán, xem thống kê doanh thu, chi phí, thuế và lập báo cáo tài chính |

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
| Sales Manager        | Điều hành hoạt động bán hàng và theo dõi doanh số |
| Content Manager      | Quản lý nội dung và theo dõi hiệu quả SEO |
| Marketing            | Xây dựng và đánh giá chiến dịch       |
| Supply Manager       | Quản lý Batch, HSD và cung ứng        |
| Executive            | Xem Dashboard và ra quyết định chiến lược |
| System Admin         | Tài khoản, Role, Permission           |
| Auditor              | Truy vết và kiểm tra Audit            |
| Accountant / Finance | Đối soát tài chính, kế toán và thuế   |

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

## BR-PRODUCT

### BR-PROD-01
Mỗi sản phẩm có thể có nhiều Variant/SKU.

### BR-PROD-02
Giá, khối lượng, quy cách đóng gói và tồn kho được quản lý theo SKU.

### BR-PROD-03
Sản phẩm thực phẩm phải có thông tin ngày sản xuất và hạn sử dụng phù hợp.

### BR-PROD-04
Thông tin sản phẩm hiển thị cho khách phải được quản lý và phê duyệt trước khi công khai.

---

## BR-BATCH & EXPIRY

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

## BR-ORDER

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

## BR-PACKING

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

## BR-MARKETPLACE

### BR-MKTPLACE-01
Đơn hàng từ Marketplace phải được xác định nguồn phát sinh.

### BR-MKTPLACE-02
Một đơn Marketplace phải có mã tham chiếu của nền tảng gốc.

### BR-MKTPLACE-03
Đơn Marketplace phải được đối soát với đơn nội bộ.

### BR-MKTPLACE-04
Không được tạo trùng đơn nội bộ khi đồng bộ lại cùng một đơn Marketplace.

### BR-MKTPLACE-05
Khi hệ thống ghi nhận đơn hàng mới từ Marketplace, tồn kho khả dụng nội bộ phải được trừ/tạm giữ ngay lập tức để tránh bán vượt mức trên các kênh khác.

### BR-MKTPLACE-06
Khi đơn hàng bị hủy từ phía Marketplace, hệ thống nội bộ phải tự động cập nhật trạng thái Hủy và hoàn trả lại tồn kho tương ứng.

### BR-MKTPLACE-07
Giá trị đơn hàng ghi nhận trên hệ thống nội bộ phải tuân theo giá thanh toán thực tế trên sàn (bao gồm các khoản trợ giá, phí sàn) để phục vụ đối soát chính xác.

---

## BR-OFFLINE SALES

### BR-OFFLINE-01
Nhân viên bán hàng Offline phải được xác định danh tính.

### BR-OFFLINE-02
Hàng lấy từ kho cho nhân viên bán Offline phải được ghi nhận.

### BR-OFFLINE-03
Nhân viên phải báo cáo số lượng bán, tồn, hàng hỏng và hàng trả theo kỳ.

### BR-OFFLINE-04
Tồn kho Offline phải được đối soát với tồn kho hệ thống.

### BR-OFFLINE-05
Mỗi giao dịch bán Offline phải được ghi nhận thành một đơn hàng (Order) trên hệ thống để đảm bảo đồng bộ dữ liệu thống kê khách hàng.

### BR-OFFLINE-06
Nhân viên Offline Sales chỉ được phép bán đúng giá hệ thống hoặc áp dụng mức chiết khấu/khuyến mãi nằm trong thẩm quyền được cấp.

### BR-OFFLINE-07
Doanh thu thực thu (Tiền mặt, Chuyển khoản) cuối ca phải khớp với tổng giá trị hàng hóa đã báo cáo bán ra.

---

## BR-REVIEW

### BR-REVIEW-01
Chỉ khách hàng đã mua sản phẩm mới được tạo Verified Review.

### BR-REVIEW-02
Review phải gắn với sản phẩm/Variant tương ứng.

### BR-REVIEW-03
Nội dung review vi phạm chính sách có thể bị ẩn hoặc xử lý bởi nhân viên có quyền.

### BR-REVIEW-04
Điểm đánh giá bắt buộc phải nằm trong thang đo chuẩn (từ 1 đến 5 sao).

### BR-REVIEW-05
Nhân viên được phân quyền (CSKH / Manager) có quyền trả lời (Reply) đánh giá của khách hàng, và phản hồi này sẽ được hiển thị công khai.

### BR-REVIEW-06
Khách hàng được phép chỉnh sửa đánh giá trong một khoảng thời gian nhất định (ví dụ: 30 ngày) kể từ lúc đăng.

---

## BR-REFUND

### BR-REFUND-01
Chỉ Sales Manager hoặc cấp cao hơn mới được quyền duyệt yêu cầu hoàn tiền.

### BR-REFUND-02
Hệ thống phải hỗ trợ hoàn tiền toàn phần hoặc hoàn tiền một phần (Partial Refund) tùy theo thỏa thuận với khách hàng.

### BR-REFUND-03
Tiền hoàn phải được trả về qua đúng phương thức thanh toán gốc (Payment Provider) hoặc quy đổi thành ví điểm nội bộ (Loyalty) nếu khách hàng đồng ý.

### BR-REFUND-04
Mọi thao tác duyệt/từ chối và thực hiện hoàn tiền bắt buộc phải được lưu trữ vào Audit Log.

---

## BR-AUDIT

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

### BR-AUDIT-04
Chỉ những tài khoản có đặc quyền (Auditor, System Administrator) mới được phép tra cứu và trích xuất Audit Log.

### BR-AUDIT-05
Audit Log phải được lưu trữ tối thiểu theo thời gian quy định (ví dụ: 1 năm hoặc vĩnh viễn) và không có chức năng xóa thủ công trên giao diện.

---

# 7. EPIC MAP

**Chú thích Effort:** [S] (Nhỏ) - [M] (Vừa) - [L] (Lớn) - [XL] (Cực lớn)

```text
[ĐỊNH DANH & HỒ SƠ]
EPIC 01  Authentication & Identity                 [Size: M]  [MVP]
EPIC 02  Customer & Employee Profile               [Size: S]  [MVP]

[CATALOG & KHÁM PHÁ]
EPIC 03  Product Discovery                         [Size: M]  [MVP]
EPIC 04  Product, Variant & SKU                    [Size: L]  [MVP]

[PHỄU MUA HÀNG TRỰC TUYẾN]
EPIC 05  Shopping Cart                             [Size: S]  [MVP]
EPIC 06  Checkout                                  [Size: M]  [MVP]
EPIC 07  Payment                                   [Size: M]  [MVP]
EPIC 08  Order Management                          [Size: M]  [MVP]
EPIC 26  Gifting Experience                        [Size: S]  [Phase 2]

[KHO, GIAO NHẬN & CUNG ỨNG]
EPIC 27  Procurement & Supplier Management         [Size: L]  [Phase 3]
EPIC 09  Inventory & Batch                         [Size: L]  [MVP]
EPIC 10  Packing & Fulfillment                     [Size: M]  [MVP]
EPIC 11  Shipping & Delivery                       [Size: S]  [MVP]

[KÊNH BÁN HÀNG]
EPIC 18  B2B Sales & Wholesale                     [Size: M]  [Phase 2]
EPIC 12  Marketplace (Shopee/TikTok)               [Size: XL] [Phase 2]
EPIC 13  Offline Sales                             [Size: XL] [Phase 2]

[HẬU MÃI & CSKH]
EPIC 14  Return / Refund / Complaint               [Size: L]  [MVP]
EPIC 15  Review & Rating                           [Size: M]  [Phase 2]
EPIC 16  Customer Service                          [Size: L]  [MVP/Phase 2]

[TĂNG TRƯỞNG & NỘI DUNG]
EPIC 17  Promotion & Loyalty                       [Size: L]  [Phase 2]
EPIC 20  Content & SEO                             [Size: M]  [MVP]
EPIC 21  Marketing                                 [Size: M]  [Phase 2]

[TUÂN THỦ, QUẢN TRỊ & NỀN TẢNG]
EPIC 19  OCOP Traceability                         [Size: M]  [MVP]
EPIC 22  User / Role / Permission Administration   [Size: M]  [MVP]
EPIC 23  Audit & Security                          [Size: M]  [MVP/Phase 2]
EPIC 28  Omnichannel Notification System           [Size: M]  [Phase 2]

[PHÂN TÍCH & HỖ TRỢ QUYẾT ĐỊNH]
EPIC 24  Finance & Business Analytics              [Size: L]  [Phase 3]
EPIC 25  Customer Analytics & DSS                  [Size: XL] [Phase 3]
```

---

# 8. EPIC 01 — AUTHENTICATION & IDENTITY

### US-AUTH-01 — Guest Checkout
**Actor:** Guest  
> Là một khách vãng lai, tôi muốn đặt hàng mà không cần tạo tài khoản để có thể mua sản phẩm nhanh chóng.  
**Priority:** Must Have — 1st

### US-AUTH-02 — Customer Registration
> Là một khách hàng, tôi muốn đăng ký tài khoản để lưu thông tin và theo dõi lịch sử mua hàng.  
**Priority:** Must Have — 1st

### US-AUTH-03 — Login
> Là một khách hàng, tôi muốn đăng nhập tài khoản để truy cập thông tin cá nhân và đơn hàng.  
**Priority:** Must Have — 1st

### US-AUTH-04 — Social Login
> Là một khách hàng, tôi muốn đăng nhập bằng tài khoản Google, Facebook hoặc Zalo khi được hỗ trợ để không phải tạo mật khẩu mới.  
**Priority:** Should Have

### US-AUTH-05 — Password Recovery
> Là một khách hàng, tôi muốn khôi phục quyền truy cập tài khoản khi quên mật khẩu.  
**Priority:** Must Have — 1st

### US-AUTH-06 — Link Guest Order
> Là một khách hàng đã từng mua với tư cách Guest, tôi muốn liên kết đơn hàng cũ với tài khoản mới để theo dõi lịch sử mua hàng.  
**Priority:** Should Have

---

# 9. EPIC 02 — CUSTOMER & EMPLOYEE PROFILE

### US-USER-01 — Customer Profile
> Là một khách hàng, tôi muốn quản lý họ tên, số điện thoại, email và thông tin cá nhân để sử dụng khi mua hàng.
**Priority:** Must Have — 1st

### US-USER-02 — Customer Address
> Là một khách hàng, tôi muốn lưu nhiều địa chỉ giao hàng để đặt hàng nhanh hơn.
**Priority:** Must Have — 1st

### US-USER-03 — Employee Profile
> Là HR Manager, tôi muốn quản lý hồ sơ nhân sự (họ tên, phòng ban, hợp đồng, chức vụ) để quản lý thông tin nhân viên (HR Profile).
**Priority:** Should Have

### US-USER-04 — Employee Status
> [Đã gộp vào US-ADM-03 thuộc EPIC 22 để tránh trùng lặp]

### US-USER-05 — Customer Account Status
> Là nhân viên có quyền quản lý khách hàng, tôi muốn xem và xử lý trạng thái tài khoản khách hàng khi cần thiết.
**Priority:** Should Have

---

# 10. EPIC 03 — PRODUCT DISCOVERY

### US-DISC-01
> Là khách hàng, tôi muốn xem danh mục sản phẩm để tìm sản phẩm phù hợp.
**Priority:** Must Have — 1st

### US-DISC-02
> Là khách hàng, tôi muốn tìm kiếm sản phẩm theo từ khóa để nhanh chóng tìm được sản phẩm mong muốn.
**Priority:** Must Have — 1st

### US-DISC-03
> Là khách hàng, tôi muốn lọc theo giá, khối lượng, loại sản phẩm, đánh giá và tình trạng còn hàng để thu hẹp lựa chọn.
**Priority:** Must Have — 1st

### US-DISC-04
> Là khách hàng, tôi muốn xem sản phẩm bán chạy hoặc được đề xuất để dễ lựa chọn.
**Priority:** Should Have

### US-DISC-05
> Là khách hàng, tôi muốn xem câu chuyện thương hiệu và văn hóa Huế để hiểu giá trị của sản phẩm.
**Priority:** Should Have

### US-AI-01 — Product Search Assistant (AI)
> Là khách hàng, tôi muốn được AI hỗ trợ tìm sản phẩm phù hợp dựa trên nhu cầu (chat/hỏi đáp) để dễ lựa chọn.
**Priority:** Could Have

### US-AI-04 — Product Recommendation (AI)
> Là khách hàng, tôi muốn được đề xuất sản phẩm phù hợp với lịch sử hoặc hành vi mua hàng.
**Priority:** Could Have

---

# 11. EPIC 04 — PRODUCT, VARIANT & SKU

### US-PROD-01
> Là Sales Manager, tôi muốn tạo và cập nhật sản phẩm để đưa sản phẩm lên hệ thống bán hàng.
**Priority:** Must Have — 1st

### US-PROD-02
> Là Sales Manager, tôi muốn quản lý Variant/SKU theo khối lượng, hương vị và quy cách đóng gói.
**Priority:** Must Have — 1st

### US-PROD-03
> Là khách hàng, tôi muốn chọn Variant/SKU trước khi mua để nhận đúng sản phẩm.
**Priority:** Must Have — 1st

### US-PROD-04
> Là khách hàng, tôi muốn xem thành phần, khối lượng, ngày sản xuất, hạn sử dụng và hướng dẫn bảo quản.
**Priority:** Must Have — 1st

### US-PROD-05
> Là Sales Manager, tôi muốn quản lý giá bán theo từng SKU để đảm bảo giá hiển thị chính xác.
**Priority:** Must Have — 1st

### US-PROD-06
> Là Sales Manager, tôi muốn quản lý trạng thái sản phẩm như đang bán, tạm ngừng bán hoặc ngừng kinh doanh.
**Priority:** Must Have — 1st

### US-PROD-07 — Omnichannel Pricing
> Là Sales Manager, tôi muốn quản lý chính sách giá khác nhau cho từng kênh (Website, Shopee, Offline, B2B) để tối ưu chiến lược bán hàng.
**Priority:** Should Have

---

# 12. EPIC 05 — SHOPPING CART

### US-CART-01
> Là khách hàng, tôi muốn thêm sản phẩm vào giỏ hàng để chuẩn bị mua.
**Priority:** Must Have — 1st

### US-CART-02
> Là khách hàng, tôi muốn thay đổi số lượng sản phẩm trong giỏ.
**Priority:** Must Have — 1st

### US-CART-03
> Là khách hàng, tôi muốn xóa sản phẩm khỏi giỏ.
**Priority:** Must Have — 1st

### US-CART-04
> Là khách hàng, tôi muốn xem tổng tiền tạm tính trước khi checkout.
**Priority:** Must Have — 1st

---

# 13. EPIC 06 — CHECKOUT

### US-CHK-01
> Là khách hàng, tôi muốn nhập thông tin nhận hàng để hoàn tất đơn.
**Priority:** Must Have — 1st

### US-CHK-02
> Là khách hàng, tôi muốn xem phí vận chuyển trước khi xác nhận đơn.
**Priority:** Must Have — 1st

### US-CHK-03
> Là khách hàng, tôi muốn được tạm giữ sản phẩm trong thời gian thanh toán để tránh sản phẩm bị người khác mua mất.
**Priority:** Must Have — 1st

### US-CHK-04
> Là khách hàng, tôi muốn kiểm tra lại sản phẩm, số lượng, địa chỉ, phí vận chuyển và tổng tiền trước khi đặt hàng.
**Priority:** Must Have — 1st

### US-CHK-05
> Là khách hàng, tôi muốn sử dụng mã ưu đãi hợp lệ trong quá trình checkout.
**Priority:** Should Have

---

# 14. EPIC 07 — PAYMENT

### US-PAY-01
> Là khách hàng, tôi muốn thanh toán bằng chuyển khoản/QR để hoàn tất đơn hàng thuận tiện.
**Priority:** Must Have — 1st

### US-PAY-02
> Là khách hàng, tôi muốn thanh toán COD nếu phương thức này được hỗ trợ.
**Priority:** Must Have — 1st

### US-PAY-03
> Là khách hàng, tôi muốn biết trạng thái thanh toán của đơn hàng.
**Priority:** Must Have — 1st

### US-PAY-04
> Là nhân viên bán hàng, tôi muốn kiểm tra trạng thái thanh toán để xử lý đơn chính xác.
**Priority:** Must Have — 2nd

### US-PAY-05
> Là quản lý, tôi muốn đối soát giao dịch thanh toán với đơn hàng để phát hiện sai lệch.
**Priority:** Must Have — 2nd

---

# 15. EPIC 08 — ORDER MANAGEMENT

### US-ORD-01
> Là khách hàng, tôi muốn xem lịch sử đơn hàng để biết các lần mua trước.
**Priority:** Must Have — 1st

### US-ORD-02
> Là khách hàng, tôi muốn xem trạng thái hiện tại của đơn hàng.
**Priority:** Must Have — 1st

### US-ORD-03
> Là khách hàng, tôi muốn nhận thông báo khi đơn hàng thay đổi trạng thái.
**Priority:** Should Have

### US-ORD-04
> Là khách hàng, tôi muốn hủy đơn khi đơn còn đủ điều kiện hủy.
**Priority:** Must Have — 1st

### US-ORD-05
> Là Sales Manager, tôi muốn xem và xử lý danh sách đơn hàng để đảm bảo đơn được thực hiện đúng.
**Priority:** Must Have — 2nd

### US-ORD-06
> Là nhân viên vận hành, tôi muốn biết đơn nào cần xử lý trước để đảm bảo SLA giao hàng.
**Priority:** Must Have — 2nd

---

# 16. EPIC 09 — INVENTORY & BATCH

### US-INV-01
> Là Warehouse Staff, tôi muốn xem tồn kho theo SKU để biết số lượng sản phẩm khả dụng.
**Priority:** Must Have — 2nd

### US-INV-02
> Là Warehouse Staff, tôi muốn ghi nhận nhập kho để cập nhật số lượng thực tế.
**Priority:** Must Have — 2nd

### US-INV-03
> Là Warehouse Staff, tôi muốn ghi nhận xuất kho để theo dõi lượng hàng đã sử dụng.
**Priority:** Must Have — 2nd

### US-INV-04
> Là Warehouse Staff, tôi muốn quản lý Batch/Lot để truy xuất từng lô sản phẩm.
**Priority:** Must Have — 2nd

### US-INV-05
> Là Warehouse Staff, tôi muốn quản lý ngày sản xuất và hạn sử dụng của từng Batch.
**Priority:** Must Have — 2nd

### US-INV-06
> Là Warehouse Staff, tôi muốn nhận cảnh báo khi Batch/Lot sắp hết hạn để chủ động xử lý.
**Priority:** Must Have — 2nd

### US-INV-07
> Là Supply Manager, tôi muốn xem tốc độ tiêu thụ và tồn kho để lập kế hoạch sản xuất.
**Priority:** Should Have

### US-INV-08
> Là Warehouse Staff, tôi muốn hệ thống ưu tiên xuất các Batch gần hết hạn phù hợp với chính sách để giảm lãng phí.
**Priority:** Should Have

### US-INV-09
> Là Warehouse Staff, tôi muốn ghi nhận hàng hỏng, thất thoát hoặc điều chỉnh tồn kho để số liệu phản ánh thực tế.
**Priority:** Must Have — 2nd

### US-INV-10
> Là Sales Manager, tôi muốn nhận cảnh báo sản phẩm sắp hết hạn để phối hợp với Sales/CSKH triển khai chương trình bán phù hợp.
**Priority:** Should Have

---

# 17. EPIC 10 — PACKING & FULFILLMENT

### US-PACK-01
> Là Packing Staff, tôi muốn xem danh sách đơn cần đóng gói để biết đơn nào phải xử lý.
**Priority:** Must Have — 2nd

### US-PACK-02
> Là Packing Staff, tôi muốn xem chi tiết sản phẩm và số lượng cần đóng gói để tránh lấy sai hàng.
**Priority:** Must Have — 2nd

### US-PACK-03
> Là Packing Staff, tôi muốn xác nhận từng bước kiểm tra đơn trước khi hoàn tất đóng gói.
**Priority:** Must Have — 2nd

### US-PACK-04
> Là Packing Staff, tôi muốn lưu video quá trình đóng gói để làm bằng chứng khi cần kiểm tra.
**Priority:** Should Have

### US-PACK-05
> Là Packing Staff, tôi muốn liên kết video đóng gói với Order để việc tra cứu sau này dễ dàng.
**Priority:** Should Have

### US-PACK-06
> Là Admin/Manager/CSKH được cấp quyền, tôi muốn tra cứu video đóng gói của một đơn hàng để kiểm tra khi xảy ra sai sót hoặc khiếu nại.
**Priority:** Should Have

### US-PACK-07
> Là Packing Staff, tôi muốn biết đơn nào có thời hạn giao gần nhất để ưu tiên đóng gói.
**Priority:** Must Have — 2nd

### US-PACK-08
> Là Packing Staff, tôi muốn xác nhận đơn đã đóng gói xong để chuyển sang công đoạn giao hàng.
**Priority:** Must Have — 2nd

---

# 18. EPIC 11 — SHIPPING & DELIVERY

### US-SHIP-01
> Là nhân viên vận hành, tôi muốn tạo vận đơn cho đơn hàng để bàn giao cho đơn vị giao hàng.
**Priority:** Must Have — 2nd

### US-SHIP-02
> Là khách hàng, tôi muốn theo dõi trạng thái giao hàng.
**Priority:** Must Have — 2nd

### US-SHIP-03
> Là Delivery Staff, tôi muốn xem danh sách đơn được giao cho mình để thực hiện giao hàng.
**Priority:** Should Have

### US-SHIP-04
> Là Delivery Staff, tôi muốn cập nhật trạng thái giao hàng để hệ thống phản ánh đúng tiến trình.
**Priority:** Should Have

### US-SHIP-05
> Là quản lý, tôi muốn xem tỷ lệ giao thành công/thất bại để đánh giá hiệu quả giao hàng.
**Priority:** Should Have

### US-SHIP-06
> Là hệ thống, tôi muốn nhận cập nhật trạng thái từ đơn vị giao vận để đồng bộ trạng thái đơn hàng.
**Priority:** Must Have — 2nd

---

# 19. EPIC 12 — MARKETPLACE

### US-MKT-01
> Là Marketplace Operator, tôi muốn xem các đơn hàng phát sinh trên Shopee/TikTok để xử lý.
**Priority:** Should Have

### US-MKT-02
> Là Marketplace Operator, tôi muốn đưa đơn hàng từ Marketplace vào quy trình xử lý nội bộ để nhân viên kho và đóng gói thực hiện.
**Priority:** Should Have

### US-MKT-03
> Là Marketplace Operator, tôi muốn theo dõi trạng thái đồng bộ giữa Marketplace và hệ thống nội bộ để phát hiện lỗi.
**Priority:** Should Have

### US-MKT-04
> Là Marketplace Operator, tôi muốn đối soát đơn hàng, doanh thu và phí từ Marketplace.
**Priority:** Should Have

### US-MKT-05
> Là Manager, tôi muốn xem doanh thu theo từng Marketplace để đánh giá hiệu quả từng kênh.
**Priority:** Should Have

### US-MKT-06
> Là hệ thống, tôi muốn lưu mã đơn hàng gốc của Marketplace để tránh tạo đơn trùng.
**Priority:** Should Have

---

# 20. EPIC 13 — OFFLINE SALES

### US-OFF-01
> Là Offline Sales Staff, tôi muốn nhận hàng từ kho để mang đi bán tại điểm bán.
**Priority:** Should Have

### US-OFF-02
> Là Offline Sales Staff, tôi muốn ghi nhận số lượng hàng đã bán để cập nhật tình hình kinh doanh.
**Priority:** Should Have

### US-OFF-03
> Là Offline Sales Staff, tôi muốn ghi nhận hàng còn tồn, hỏng hoặc trả lại.
**Priority:** Should Have

### US-OFF-04
> Là Offline Sales Staff, tôi muốn gửi báo cáo bán hàng theo ngày/tháng.
**Priority:** Should Have

### US-OFF-05
> Là Sales Manager, tôi muốn xem doanh thu và tồn kho theo từng nhân viên/điểm bán.
**Priority:** Should Have

### US-OFF-06
> Là Manager, tôi muốn đối soát lượng hàng đã cấp cho điểm bán với lượng hàng đã bán và còn tồn.
**Priority:** Should Have

---

# 21. EPIC 14 — RETURN, REFUND & COMPLAINT

> **Boundary Note:** EPIC này chỉ quản lý **State Machine và quy trình xử lý nghiệp vụ** (duyệt trả hàng, hoàn tiền, nhập kho). Việc tiếp nhận và giao tiếp với khách hàng ban đầu thuộc về EPIC 16 (Customer Service).

### US-RET-01
> Là khách hàng, tôi muốn yêu cầu hủy đơn khi đơn còn đủ điều kiện.
**Priority:** Must Have — 3rd

### US-RET-02
> Là khách hàng, tôi muốn gửi yêu cầu đổi/trả khi sản phẩm bị lỗi, hư hỏng hoặc không đúng đơn.
**Priority:** Must Have — 3rd

### US-RET-03
> Là khách hàng, tôi muốn đính kèm hình ảnh/video khiếu nại để chứng minh tình trạng sản phẩm.
**Priority:** Must Have — 3rd

### US-RET-04
> Là CSKH, tôi muốn xem thông tin đơn hàng và bằng chứng liên quan để xử lý khiếu nại.
**Priority:** Must Have — 3rd

### US-RET-05
> Là Sales Manager, tôi muốn duyệt hoặc từ chối yêu cầu đổi/trả theo chính sách.
**Priority:** Must Have — 3rd

### US-RET-06
> Là Manager, tôi muốn xử lý hoàn tiền cho khách hàng khi yêu cầu hợp lệ.
**Priority:** Must Have — 3rd

### US-RET-07
> Là CSKH, tôi muốn tra cứu video đóng gói để kiểm tra nguyên nhân khi khách phản ánh thiếu/sai/hỏng sản phẩm.
**Priority:** Should Have

---

# 22. EPIC 15 — REVIEW & RATING

### US-REV-01
> Là khách hàng đã mua hàng, tôi muốn đánh giá sản phẩm để chia sẻ trải nghiệm.
**Priority:** Should Have

### US-REV-02
> Là khách hàng, tôi muốn đăng hình ảnh thực tế cùng đánh giá.
**Priority:** Should Have

### US-REV-03
> Là khách hàng, tôi muốn biết đánh giá nào đến từ người đã mua hàng thực tế.
**Priority:** Should Have

### US-REV-04
> Là CSKH, tôi muốn xem các đánh giá tiêu cực để chủ động hỗ trợ khách hàng.
**Priority:** Should Have

### US-REV-05
> Là Sales Manager, tôi muốn quản lý và xử lý những đánh giá vi phạm chính sách.
**Priority:** Should Have

### US-REV-06
> Là Manager, tôi muốn phân tích đánh giá để phát hiện vấn đề về chất lượng sản phẩm hoặc dịch vụ.
**Priority:** Should Have

---

# 23. EPIC 16 — CUSTOMER SERVICE

> **Boundary Note:** EPIC này quản lý luồng **Giao tiếp & Tương tác** (Ticket, Chat đa kênh, SLA phản hồi). Khi có khiếu nại liên quan đến trả hàng/hoàn tiền, CSKH sẽ tạo Ticket ở đây và kích hoạt (trigger) quy trình nghiệp vụ bên EPIC 14.


### US-CS-01
> Là khách hàng, tôi muốn gửi câu hỏi hoặc yêu cầu hỗ trợ.
**Priority:** Must Have — 3rd

### US-CS-02
> Là CSKH, tôi muốn xem danh sách yêu cầu hỗ trợ để xử lý theo mức độ ưu tiên.
**Priority:** Should Have

### US-CS-03
> Là CSKH, tôi muốn xem lịch sử trao đổi với khách hàng để hỗ trợ chính xác.
**Priority:** Should Have

### US-CS-04
> Là CSKH, tôi muốn xem đơn hàng, thanh toán, vận chuyển và lịch sử mua hàng của khách để xử lý vấn đề.
**Priority:** Should Have

### US-CS-05
> Là CSKH, tôi muốn nhận cảnh báo các khách hàng có vấn đề hoặc khiếu nại chưa được xử lý.
**Priority:** Should Have

### US-CS-06
> Là CSKH/Sales, tôi muốn nhận danh sách sản phẩm sắp hết hạn để phối hợp triển khai chương trình bán phù hợp.
**Priority:** Should Have

### US-AI-02 — Customer Support Assistant (AI)
> Là CSKH, tôi muốn được hệ thống AI hỗ trợ tổng hợp thông tin liên quan đến câu hỏi của khách (Draft reply, tra cứu đơn) để trả lời nhanh hơn.
**Priority:** Could Have

### US-AI-06 — Internal Knowledge Assistant (AI)
> Là nhân viên nội bộ/CSKH, tôi muốn tìm kiếm thông tin nghiệp vụ/sản phẩm bằng AI trong kho tri thức của doanh nghiệp để giải quyết công việc.
**Priority:** Could Have

---

# 24. EPIC 17 — PROMOTION & LOYALTY

### US-PROMO-01
> Là khách hàng, tôi muốn sử dụng coupon hợp lệ để nhận ưu đãi.
**Priority:** Should Have

### US-PROMO-02
> Là Sales Manager, tôi muốn tạo coupon theo điều kiện để triển khai chương trình bán hàng.
**Priority:** Should Have

### US-PROMO-03
> Là Sales Manager, tôi muốn tạo Combo sản phẩm để tăng giá trị đơn hàng.
**Priority:** Should Have

### US-PROMO-04
> Là khách hàng, tôi muốn xem các chương trình khuyến mãi đang áp dụng.
**Priority:** Should Have

### US-LOY-01
> Là khách hàng, tôi muốn tích điểm sau mỗi lần mua hàng.
**Priority:** Should Have

### US-LOY-02
> Là khách hàng, tôi muốn đổi điểm thành ưu đãi.
**Priority:** Should Have

### US-LOY-03
> Là khách hàng, tôi muốn đặt lại sản phẩm từ đơn hàng cũ để tiết kiệm thời gian.
**Priority:** Should Have

---

# 25. EPIC 18 — B2B SALES & WHOLESALE

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

# 26. EPIC 19 — OCOP TRACEABILITY

### US-OCOP-01
> Là khách hàng, tôi muốn quét QR trên sản phẩm để kiểm tra nguồn gốc.
**Priority:** Must Have — 3rd

### US-OCOP-02
> Là khách hàng, tôi muốn xem thông tin vùng nguyên liệu, nhà sản xuất và quy trình sản xuất để tăng niềm tin.
**Priority:** Must Have — 3rd

### US-OCOP-03
> Là khách hàng, tôi muốn xem chứng nhận OCOP và các chứng nhận liên quan của sản phẩm.
**Priority:** Must Have — 3rd

### US-OCOP-04
> Là Manager, tôi muốn quản lý thông tin nguồn gốc của từng sản phẩm/lô hàng.
**Priority:** Must Have — 3rd

### US-OCOP-05
> Là Manager, tôi muốn cập nhật thông tin nhà cung cấp/nguyên liệu để đảm bảo dữ liệu truy xuất chính xác.
**Priority:** Should Have

### US-OCOP-06
> Là Manager, tôi muốn kiểm tra và xác nhận nguồn dữ liệu nguồn gốc trước khi công khai cho khách hàng.
**Priority:** Should Have

---

# 27. EPIC 20 — CONTENT & SEO

### US-CONTENT-01
> Là Content Manager, tôi muốn tạo bài viết để truyền tải câu chuyện văn hóa Huế và thương hiệu.
**Priority:** Should Have

### US-CONTENT-02
> Là Content Manager, tôi muốn chỉnh sửa, xuất bản hoặc ẩn bài viết.
**Priority:** Should Have

### US-CONTENT-03
> Là Content Manager, tôi muốn quản lý hình ảnh/video trong bài viết.
**Priority:** Should Have

### US-CONTENT-04
> Là Content Manager, tôi muốn tối ưu tiêu đề, mô tả, từ khóa và thông tin SEO cho bài viết.
**Priority:** Should Have

### US-CONTENT-05
> Là Content Manager, tôi muốn quản lý thông tin SEO cho từng sản phẩm để sản phẩm có khả năng được tìm thấy trên công cụ tìm kiếm.
**Priority:** Should Have

### US-CONTENT-06
> Là Content Manager, tôi muốn quản lý URL thân thiện, Meta Title, Meta Description và nội dung liên quan SEO.
**Priority:** Should Have

### US-CONTENT-07
> Là Manager, tôi muốn duyệt nội dung trước khi xuất bản để đảm bảo thông tin chính xác.
**Priority:** Should Have

### US-AI-03 — Content Assistant (AI)
> Là Content Manager, tôi muốn được AI hỗ trợ tạo tiêu đề, mô tả và outline bài viết Marketing để tăng hiệu suất sản xuất nội dung.
**Priority:** Could Have

---

# 28. EPIC 21 — MARKETING

### US-MKTG-01
> Là Marketing Staff, tôi muốn lập kế hoạch Marketing theo chiến dịch để tổ chức hoạt động quảng bá.
**Priority:** Should Have

### US-MKTG-02
> Là Marketing Staff, tôi muốn tạo nội dung video, bài viết và chương trình truyền thông cho từng chiến dịch.
**Priority:** Should Have

### US-MKTG-03
> Là Marketing Staff, tôi muốn gửi kế hoạch cho người có quyền duyệt trước khi triển khai.
**Priority:** Should Have

### US-MKTG-04
> Là Manager/Executive, tôi muốn duyệt hoặc từ chối kế hoạch Marketing.
**Priority:** Should Have

### US-MKTG-05
> Là Marketing Staff, tôi muốn theo dõi KPI của chiến dịch sau khi triển khai.
**Priority:** Should Have

### US-MKTG-06
> Là Marketing Staff, tôi muốn lấy dữ liệu hiệu quả từ các kênh Marketing để đánh giá chiến dịch.
**Priority:** Should Have

### US-MKTG-07
> Là Executive, tôi muốn so sánh chi phí Marketing với doanh thu tạo ra để đánh giá hiệu quả đầu tư.
**Priority:** Should Have

---

# 29. EPIC 22 — USER / ROLE / PERMISSION ADMINISTRATION

### US-ADM-01 — Employee Account
> Là System Administrator hoặc HR Manager, tôi muốn tạo tài khoản (System Access) cho nhân viên để cấp quyền sử dụng hệ thống.
**Priority:** Must Have — 2nd

### US-ADM-02 — Update Employee Account
> Là System Administrator, tôi muốn cập nhật thông tin tài khoản nhân viên để dữ liệu luôn chính xác.
**Priority:** Must Have — 2nd

### US-ADM-03 — Lock Employee Account
> Là System Administrator hoặc HR Manager, tôi muốn khóa tài khoản nhân viên khi nhân viên nghỉ việc hoặc không còn được phép truy cập.
**Priority:** Must Have — 2nd

### US-ADM-04 — Role Management
> Là System Administrator, tôi muốn tạo và quản lý Role để phân nhóm quyền truy cập.
**Priority:** Must Have — 2nd

### US-ADM-05 — Permission Management
> Là System Administrator, tôi muốn quản lý Permission để kiểm soát từng chức năng mà mỗi Role được phép sử dụng.
**Priority:** Must Have — 2nd

### US-ADM-06 — Assign Role
> Là System Administrator, tôi muốn gán Role cho nhân viên để nhân viên chỉ truy cập đúng phạm vi công việc.
**Priority:** Must Have — 2nd

### US-ADM-07 — Access Review
> Là System Administrator, tôi muốn xem quyền của từng nhân viên để kiểm tra việc phân quyền.
**Priority:** Must Have — 2nd

---

# 30. EPIC 23 — AUDIT & SECURITY

### US-AUDIT-01
> Là System Administrator/Auditor, tôi muốn tra cứu lịch sử thao tác để biết ai đã thực hiện hành động gì, khi nào và trên đối tượng nào.
**Priority:** Must Have — 3rd

### US-AUDIT-02
> Là Auditor, tôi muốn xem giá trị trước và sau khi dữ liệu quan trọng thay đổi để xác định nguyên nhân sai lệch.
**Priority:** Must Have — 3rd

### US-AUDIT-03
> Là Auditor, tôi muốn biết nguồn truy cập hoặc vị trí kỹ thuật của thao tác khi thông tin này khả dụng để phục vụ điều tra.
**Priority:** Should Have

### US-AUDIT-04
> Là Auditor, tôi muốn biết lý do nghiệp vụ của các thay đổi nhạy cảm khi thao tác đó yêu cầu người dùng cung cấp lý do.
**Priority:** Should Have

### US-AUDIT-05
> Là Security Operator, tôi muốn phát hiện Audit Log bị thay đổi hoặc mất tính liên tục để bảo vệ tính toàn vẹn của nhật ký.
**Priority:** Should Have

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

# 31. EPIC 24 — FINANCE & BUSINESS ANALYTICS

### US-FIN-01
> Là Giám đốc (ACT-16) hoặc Kế toán (ACT-19), tôi muốn xem doanh thu theo ngày/tháng/năm để đánh giá tình hình kinh doanh.
**Priority:** Could Have

### US-FIN-02
> Là Giám đốc (ACT-16) hoặc Kế toán (ACT-19), tôi muốn xem doanh thu theo từng kênh Website, App, Marketplace, B2B và Offline.
**Priority:** Could Have

### US-FIN-03
> Là Giám đốc (ACT-16) hoặc Kế toán (ACT-19), tôi muốn xem chi phí bán hàng và vận hành để đánh giá hiệu quả kinh doanh.
**Priority:** Could Have

### US-FIN-04
> Là Giám đốc (ACT-16) hoặc Kế toán (ACT-19), tôi muốn xem lợi nhuận theo sản phẩm để biết sản phẩm nào mang lại giá trị cao.
**Priority:** Could Have

### US-FIN-05
> Là Giám đốc (ACT-16) hoặc Kế toán (ACT-19), tôi muốn xem doanh thu theo khu vực để đánh giá thị trường.
**Priority:** Could Have

### US-FIN-06
> Là Kế toán (ACT-19), tôi muốn đối soát giao dịch và theo dõi nghĩa vụ thuế (kết nối với EXT-11 Tax Provider) để phục vụ quản lý tài chính.
**Priority:** Could Have

### US-FIN-07
> Là Giám đốc (ACT-16) hoặc Kế toán (ACT-19), tôi muốn xuất báo cáo tài chính/kinh doanh (kết nối với EXT-11 Invoice Provider) để phục vụ họp và ra quyết định.
**Priority:** Could Have

---

# 32. EPIC 25 — CUSTOMER ANALYTICS & DSS

### US-DSS-01 — Dashboard
> Là Executive, tôi muốn xem Dashboard tổng quan về doanh thu, đơn hàng, khách hàng, tồn kho và lợi nhuận để nắm tình hình doanh nghiệp.
**Priority:** Should Have

### US-DSS-02 — RFM
> Là Marketing/Executive, tôi muốn phân nhóm khách hàng theo hành vi mua hàng để xây dựng chính sách chăm sóc phù hợp.
**Priority:** Could Have

### US-DSS-03 — Customer Churn
> Là Marketing Staff, tôi muốn phát hiện khách hàng có nguy cơ không mua lại để triển khai chiến dịch giữ chân.
**Priority:** Could Have

### US-DSS-04 — Product Performance
> Là Executive, tôi muốn biết sản phẩm nào bán chạy, bán chậm hoặc đang giảm doanh số.
**Priority:** Could Have

### US-DSS-05 — Sales Velocity
> Là Supply Manager, tôi muốn biết tốc độ tiêu thụ từng SKU để lập kế hoạch sản xuất.
**Priority:** Could Have

### US-DSS-06 — Demand Forecast
> Là Executive/Supply Manager, tôi muốn dự báo nhu cầu sản phẩm trong tương lai để chuẩn bị nguyên liệu và sản xuất.
**Priority:** Could Have

### US-DSS-07 — Expiry Risk
> Là Supply Manager, tôi muốn biết lượng hàng có nguy cơ hết hạn trong tương lai để chủ động xử lý.
**Priority:** Could Have

### US-DSS-08 — Product Association
> Là Sales Manager, tôi muốn biết các sản phẩm thường được mua cùng nhau để xây dựng Combo.
**Priority:** Could Have

### US-DSS-09 — Seasonal Analysis
> Là Marketing/Executive, tôi muốn phân tích nhu cầu theo mùa, lễ hội và dịp Tết để lập kế hoạch kinh doanh.
**Priority:** Could Have

### US-DSS-10 — Strategic Recommendation
> Là Executive, tôi muốn nhận các khuyến nghị dựa trên dữ liệu để hỗ trợ quyết định về sản phẩm, giá, tồn kho và Marketing.
**Priority:** Could Have

---


---

# 33. EPIC 26 — GIFTING EXPERIENCE

### US-GIFT-01
> Là khách hàng, tôi muốn gửi sản phẩm đến một người nhận khác với người đặt hàng để mua quà.
**Priority:** Should Have

### US-GIFT-02
> Là khách hàng, tôi muốn thêm lời chúc vào đơn quà tặng.
**Priority:** Should Have

### US-GIFT-03
> Là khách hàng, tôi muốn ẩn giá sản phẩm trên phiếu quà tặng.
**Priority:** Should Have

---

# 34. EPIC 27 — PROCUREMENT & SUPPLIER MANAGEMENT

> Bù đắp khoảng trống quản lý thượng nguồn (cung ứng) cho sản phẩm OCOP và nhà cung cấp.

### US-PROC-01 — Supplier Profile
> Là Supply Manager, tôi muốn quản lý thông tin nhà cung cấp, hộ nông dân, hợp tác xã (Tên, địa chỉ, hợp đồng, chính sách giá).
**Priority:** Could Have

### US-PROC-02 — Purchase Order (PO)
> Là Supply Manager, tôi muốn tạo Đơn đặt hàng (PO) gửi nhà cung cấp để chuẩn bị nguồn hàng.
**Priority:** Could Have

### US-PROC-03 — Production & Supply Planning
> Là Supply Manager, tôi muốn lập kế hoạch sản xuất/nhập hàng dựa trên dự báo nhu cầu để đảm bảo không thiếu hụt hoặc tồn kho quá mức.
**Priority:** Could Have

### US-PROC-04 — PO Receiving
> Là Warehouse Staff, tôi muốn đối chiếu hàng thực nhận với PO khi nhập kho để phát hiện chênh lệch.
**Priority:** Could Have

---

# 35. EPIC 28 — OMNICHANNEL NOTIFICATION SYSTEM

> Hạ tầng thông báo cắt ngang toàn hệ thống.

### US-NOTI-01 — Email Notification
> Là khách hàng, tôi muốn nhận Email xác nhận đơn hàng, mã OTP, hóa đơn và các thông báo quan trọng để nắm bắt thông tin kịp thời.
**Priority:** Must Have — 2nd

### US-NOTI-02 — SMS / Zalo ZNS
> Là khách hàng, tôi muốn nhận tin nhắn SMS hoặc Zalo cập nhật trạng thái giao hàng để dễ dàng theo dõi hành trình đơn hàng.
**Priority:** Should Have

### US-NOTI-03 — In-app Push Notification
> Là khách hàng, tôi muốn nhận thông báo đẩy (Push) trên Mobile App/Web về trạng thái đơn hàng và khuyến mãi.
**Priority:** Should Have

### US-NOTI-04 — Internal Alert
> Là nhân viên nội bộ, tôi muốn nhận cảnh báo in-app khi có đơn khẩn, ticket quá hạn SLA hoặc hàng hóa sắp hết hạn.
**Priority:** Should Have

---
# 36. EXTERNAL SYSTEM REQUIREMENTS

## 36.1. Identity Provider
Hệ thống có thể hỗ trợ đăng nhập thông qua:
```text
Google
Facebook
Zalo
```
External Identity Provider chỉ chịu trách nhiệm xác thực danh tính bên ngoài; hồ sơ nghiệp vụ Customer/Employee vẫn thuộc hệ thống.

## 36.2. Marketplace
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

## 36.3. Analytics
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

## 36.4. AI Provider
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

## 36.5. Media Storage
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

## 36.6. Payment Gateway
Có thể tích hợp:
```text
VNPay
MoMo
ZaloPay
VietQR (Bank API)
```
Các nghiệp vụ cần hỗ trợ:
* Tạo phiên thanh toán (Checkout).
* Xử lý Webhook (IPN) để xác nhận giao dịch thành công.
* Xử lý hoàn tiền (Refund API).

## 36.7. Logistics Provider
Có thể tích hợp:
```text
GHN (Giao Hàng Nhanh)
GHTK (Giao Hàng Tiết Kiệm)
Viettel Post
```
Các nghiệp vụ cần hỗ trợ:
* Tính phí vận chuyển (Shipping fee) theo thời gian thực dựa trên khối lượng và địa chỉ.
* Đẩy đơn vận sang đối tác (Tạo mã vận đơn).
* Đồng bộ trạng thái giao hàng (Webhook Tracking).
* Đối soát cước phí vận chuyển.

## 36.8. Notification Provider
Có thể tích hợp:
```text
AWS SES / SendGrid (Email)
eSMS / Zalo ZNS API (Tin nhắn)
Firebase Cloud Messaging (Push)
```
Các nghiệp vụ cần hỗ trợ:
* Gửi mã xác thực (OTP).
* Gửi Email xác nhận đơn hàng, hóa đơn.
* Gửi Zalo ZNS / SMS trạng thái đơn hàng.
* Gửi thông báo Marketing.

## 36.9. Tax & E-Invoice Provider
Có thể tích hợp:
```text
MISA eInvoice
VNPT Invoice
Viettel Invoice
```
Các nghiệp vụ cần hỗ trợ:
* Tự động phát hành hóa đơn điện tử (VAT) cho đơn B2B.
* Gửi hóa đơn điện tử cho khách hàng cá nhân có yêu cầu xuất VAT.

## 36.10. Origin & Traceability Provider (OCOP)
Hệ thống có thể liên kết với:
```text
Cổng thông tin truy xuất nguồn gốc địa phương/quốc gia
```
Các nghiệp vụ cần hỗ trợ:
* Xác thực mã chứng nhận OCOP.
* Truy xuất và hiển thị dữ liệu minh bạch nguồn gốc lô hàng.


---

# 37. ACCEPTANCE CRITERIA — CÁC LUỒNG QUAN TRỌNG

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

# 38. ORDER STATE MACHINE

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

# 39. MEDIA ACCESS MODEL

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

