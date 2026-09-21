# BỘ SƠ ĐỒ BỐI CẢNH HỆ THỐNG (SYSTEM CONTEXT DIAGRAMS - C4 LEVEL 1)

## 1. Vị Trí & Vai Trò Trong Tài Liệu HLD
- **Cấp độ kiến trúc:** Toàn cảnh (System Context - Level 1 trong mô hình C4).
- **Vị trí trong HLD:** Đặt tại **Chương 1 (Mục Mở đầu)** — Xác định vị trí của Hệ sinh thái Mè Xửng O Mạ trong thế giới thực, trả lời câu hỏi: *Ai tương tác với hệ thống, qua kênh nào, và hệ thống phụ thuộc vào những đối tác ngoại vi nào?*
- **Chiến lược phân rã (Modular De-cluttering):** Để loại bỏ hoàn toàn tình trạng "All-In-One" khiến mũi tên chồng chéo, rối mắt và đè chữ chú thích, tài liệu phân rã System Context thành **3 sơ đồ chuyên biệt theo từng góc nhìn nghiệp vụ rõ ràng**:
  - **Sơ đồ 1.1 (Toàn Cảnh Cốt Lõi):** Bức tranh tối giản chỉ gồm Hệ thống trung tâm, 2 nhóm tác nhân lớn và 2 nhóm đối tác chính.
  - **Sơ đồ 1.2 (Góc Nhìn Khách Hàng & Bán Hàng Đa Kênh):** Chuyên sâu về luồng mua sắm, sàn TMĐT và cổng thanh toán.
  - **Sơ đồ 1.3 (Góc Nhìn Chuỗi Cung Ứng & Đối Tác Hậu Cần):** Chuyên sâu về xưởng đóng gói, kho vận, HTX nông sản, 3PL và kiểm toán.

---

## 2. Quy Chuẩn Ký Hiệu Áp Dụng (Tuân Thủ `regulation.md`)
- Sử dụng engine `C4Context` của Mermaid.
- Sử dụng macro `Person(...)` định danh người dùng.
- Sử dụng macro `Enterprise_Boundary(...)` và `System(...)` đóng khung nền tảng O Mạ Core.
- Sử dụng macro `System_Ext(...)` cho đối tác bên thứ ba theo đúng Dòng 8 của `regulation.md`.
- Ghi rõ giao thức và kênh truyền thông trong tham số thứ 4 của `Rel(...)`.

---

## 3. Sơ Đồ 1.1: System Context Tổng Quan (High-Level Core Context)
*Mục đích: Cung cấp cái nhìn nhanh trong 30 giây cho Ban Giám Đốc và các bên liên quan, đường nét thông thoáng, không chồng chéo.*

```mermaid
C4Context
    title Sơ Đồ 1.1: Bối Cảnh Cốt Lõi Hệ Thống Mè Xửng O Mạ (High-Level Context)

    Person(users_consumer, "Nhóm Khách Hàng", "Người tiêu dùng cá nhân D2C, Khách sỉ B2B, Khách du lịch tại xưởng")
    Person(users_staff, "Nhóm Vận Hành Nội Bộ", "Thợ đóng gói xưởng kẹo, Thủ kho, Kế toán, Nhân viên CSKH")

    Enterprise_Boundary(b_omamx, "HỆ SINH THÁI MÈ XỬNG O MẠ") {
        System(omamx_platform, "Nền Tảng O Mạ Core Platform", "Hệ thống 18 Microservices: Thương mại điện tử đa kênh, Quản lý kho FEFO, Đóng gói có video seal, Truy xuất OCOP")
    }

    System_Ext(ext_channels_pay, "Kênh Đối Tác & Thanh Toán", "Shopee, TikTok Shop, Cổng VietQR / Napas liên ngân hàng")
    System_Ext(ext_supply_cloud, "Chuỗi Cung Ứng & Hạ Tầng", "Hợp tác xã Huế, Logistics 3PL (GHN/ViettelPost), AWS S3 Cloud Storage")

    Rel(users_consumer, omamx_platform, "Tìm kiếm kẹo, đặt mua, thanh toán VietQR, theo dõi đơn", "HTTPS / Web Next.js & Mobile Flutter")
    Rel(users_staff, omamx_platform, "Đóng gói seal kẹo, kiểm kê kho, đối soát hóa đơn VAT, xem DSS", "HTTPS / Web Admin & Tablet")

    Rel(ext_channels_pay, omamx_platform, "Bắn Webhook đơn hàng sàn và Webhook xác nhận tiền về", "HTTPS REST / HMAC Webhook")
    Rel(omamx_platforVm, ext_channels_pay, "Đồng bộ tồn kho thời gian thực 2 chiều lên Shopee/TikTok", "Partner Open API")

    Rel(omamx_platform, ext_supply_cloud, "Gọi API tạo vận đơn 3PL, lưu trữ video seal và tài liệu kiểm toán", "REST API & AWS SDK")
    Rel(ext_supply_cloud, omamx_platform, "Cung ứng nguyên liệu mè/đậu OCOP, cập nhật tọa độ giao hàng", "Biên bản nhập kho & Webhook 3PL")
```

---

## 4. Sơ Đồ 1.2: Phân Rã Góc Nhìn Khách Hàng & Bán Hàng Đa Kênh (Inbound Omnichannel Context)
*Mục đích: Làm rõ mọi kênh bán hàng và luồng tiền vào hệ thống mà không bị phân tâm bởi thợ xưởng hay xe tải giao hàng.*

```mermaid
C4Context
    title Sơ Đồ 1.2: Góc Nhìn Khách Hàng & Đa Kênh (Inbound Omnichannel Context)

    Person(d2c_shopper, "Khách Hàng D2C", "Mua kẹo trên Web omamx.vn hoặc App điện thoại")
    Person(b2b_buyer, "Khách Doanh Nghiệp (B2B)", "Đặt mua quà Tết số lượng lớn (500+ hộp), in logo riêng")
    Person(pos_guest, "Khách Vãng Lai / Du Lịch", "Mua kẹo ăn thử trực tiếp tại Quầy Xưởng Huế")

    Enterprise_Boundary(b_omamx_sales, "NỀN TẢNG O MẠ - MIỀN THƯƠNG MẠI") {
        System(sales_engine, "Phân Hệ Bán Hàng & Đơn Hàng", "Xử lý giỏ hàng, bảng giá OCOP, khuyến mại, điều phối Saga thanh toán VietQR")
    }

    System_Ext(marketplace_platforms, "Sàn TMĐT (Shopee / TikTok Shop)", "Kênh đối tác bán hàng ngoại vi. Bắn webhook khi có đơn phát sinh")
    System_Ext(vietqr_gateway, "Cổng Thanh Toán VietQR / Napas", "Hạ tầng đối soát thanh toán chuyển khoản ngân hàng tự động")
    System_Ext(zns_gateway, "Cổng Zalo ZNS / SMS Brandname", "Gửi tin nhắn xác nhận đơn tức thời cho khách hàng")

    Rel(d2c_shopper, sales_engine, "Duyệt kẹo OCOP, giỏ hàng, đặt mua, áp mã voucher", "HTTPS / Web Next.js & Mobile App")
    Rel(b2b_buyer, sales_engine, "Gửi form yêu cầu báo giá sỉ, tải file vector logo công ty", "HTTPS / B2B Web Portal")
    Rel(pos_guest, sales_engine, "Mua kẹo tại quầy xưởng, thanh toán tiền mặt/quẹt thẻ, in bill", "POS App Tablet (Offline-First)")

    Rel(marketplace_platforms, sales_engine, "Bắn webhook đơn hàng phát sinh trên Shopee Mall/TikTok Shop", "HTTPS REST Webhook")
    Rel(sales_engine, marketplace_platforms, "Đồng bộ số lượng tồn kho khả dụng thời gian thực (chống bán lố)", "Partner Open API")

    Rel(sales_engine, vietqr_gateway, "Tạo mã VietQR động định danh kèm số tiền chuẩn", "gRPC / HTTPS API")
    Rel(vietqr_gateway, sales_engine, "Bắn webhook HMAC-SHA256 báo tiền đã về tài khoản xưởng O Mạ", "HTTPS HMAC Webhook")

    Rel(sales_engine, zns_gateway, "Yêu cầu gửi tin nhắn Zalo ZNS / SMS cảm ơn và xác nhận đơn", "REST API HTTPS")
```

---

## 5. Sơ Đồ 1.3: Phân Rã Góc Nhìn Chuỗi Cung Ứng & Hậu Cần (Outbound Supply Chain & Logistics Context)
*Mục đích: Làm rõ hậu kỳ xưởng kẹo Huế, quy trình dán seal niêm phong, lưu trữ video AWS S3, vận chuyển 3PL và vùng nguyên liệu.*

```mermaid
C4Context
    title Sơ Đồ 1.3: Góc Nhìn Chuỗi Cung Ứng & Vận Tải (Supply Chain & Fulfillment Context)

    Person(staff_packing, "Thợ Đóng Gói Xưởng Kẹo", "Thao tác trên tablet tại bàn đóng gói xưởng Hương Thủy")
    Person(staff_warehouse, "Thủ Kho & Thu Mua", "Kiểm soát kho xưởng Huế, kho hub, lập PO mua nguyên liệu")
    Person(staff_cs, "Nhân Viên CSKH", "Tra cứu video đóng gói có seal để giải quyết khiếu nại kẹo vỡ")

    Enterprise_Boundary(b_omamx_sc, "NỀN TẢNG O MẠ - MIỀN CHUỖI CUNG ỨNG & KHO VẬN") {
        System(sc_engine, "Phân Hệ Chuỗi Cung Ứng & Kho Vận", "Quản lý tồn kho đa kho FEFO, điều phối nhặt hàng, ghi nhận video seal, tạo vận đơn")
    }

    System_Ext(coop_hue, "Hợp Tác Xã Nông Nghiệp Huế", "Vùng trồng mè đen, đậu phụng, đường phèn OCOP đạt chuẩn VietGAP")
    System_Ext(logistics_3pl, "Hãng Vận Chuyển 3PL (GHN / ViettelPost)", "Tiếp nhận thùng kẹo tại xưởng Huế, vận chuyển liên tỉnh chặng cuối")
    System_Ext(aws_s3_storage, "AWS S3 Object Storage", "Lưu trữ video đóng gói có seal (WORM), cấp Pre-signed URL TTL 15m")
    System_Ext(audit_worm, "AWS S3 WORM / Object Lock", "Lưu trữ checkpoint Merkle Tree kiểm toán pháp lý định kỳ")

    Rel(staff_packing, sc_engine, "Nhận picking task theo lô FEFO, quét mã vạch, quay video có seal", "HTTPS / Tablet Web App")
    Rel(staff_warehouse, sc_engine, "Kiểm kê tồn kho, tạo phiếu nhập mè/đậu, xuất kho cho 3PL", "HTTPS / Web Admin CMS")
    Rel(staff_cs, sc_engine, "Tra cứu video đóng gói tương ứng với mã seal trên kiện kẹo", "HTTPS / Web Admin CMS")

    Rel(coop_hue, sc_engine, "Giao mè/đậu nông sản kèm chứng từ nguồn gốc OCOP", "Biên bản nhập kho")
    Rel(sc_engine, logistics_3pl, "Tự động gọi Open API tạo vận đơn, truyền cân nặng/kích thước", "REST API HTTPS")
    Rel(logistics_3pl, sc_engine, "Cập nhật tọa độ bưu tá và trạng thái giao hàng thành công/thất bại", "HTTPS Webhook")

    Rel(sc_engine, aws_s3_storage, "Upload video đóng gói có tem seal niêm phong (băm SHA-256)", "AWS S3 SDK (TLS 1.3)")
    Rel(sc_engine, audit_worm, "Ghi định kỳ Merkle Root Hash của chuỗi kiểm toán pháp lý", "AWS S3 Object Lock")
```

---

## 6. Tổng Kết So Sánh: Trước & Sau Phân Rã
- **Trước phân rã (All-In-One):** 1 sơ đồ chứa hơn 16 đối tượng và gần 30 đường kết nối chéo nhau, chữ chú thích đè lên đường vẽ, không thể in ấn hoặc trình chiếu trên slide.
- **Sau phân rã (Modular 3 Views):**
  - Sơ đồ 1.1 cho bức tranh tổng quan ngắn gọn, chuẩn xác.
  - Sơ đồ 1.2 tách bạch 100% luồng bán hàng & khách hàng.
  - Sơ đồ 1.3 tách bạch 100% chuỗi cung ứng, kho vận xưởng kẹo và hạ tầng lưu trữ AWS S3.
  - Mọi chú thích mũi tên đều nằm gọn gàng, rõ nét, không một mũi tên nào bị cắt ngang hoặc che chữ.
