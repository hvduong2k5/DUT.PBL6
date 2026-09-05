# 02. SPRINT PLANNING & RELEASE PLAN
## ĐỀ TÀI: HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ ĐA KÊNH & HỆ THỐNG PHÂN TÍCH HỖ TRỢ QUYẾT ĐỊNH (MÈ XỬNG O MẠ)

---

# I. PHÂN KỲ PHÁT TRIỂN (RELEASE PLAN)

Toàn bộ hệ sinh thái được chia thành **03 giai đoạn phát triển liên tiếp**. **Giai đoạn 1** gồm 03 bản MVP tăng dần; **Giai đoạn 2** và **Giai đoạn 3** mở rộng sản phẩm sau khi luồng bán hàng D2C đã ổn định. Mỗi bản MVP là một phiên bản **chạy được (Runnable)**, tự tạo ra giá trị nghiệp vụ và có thể đưa vào vận hành.

Việc phân kỳ tuân theo thứ tự ưu tiên trong Product Backlog: `Must Have — 1st` → `Must Have — 2nd` → `Must Have — 3rd` → `Should Have` → `Could Have`. Mỗi MVP và mỗi giai đoạn sau đều kế thừa dữ liệu, quy trình của phần trước; không có giai đoạn nào là một hệ thống tách rời.

## Giai đoạn 1 — Chuỗi MVP vận hành D2C

### MVP v1.0 — Nền tảng E-commerce (Must Have — 1st)
*Mục tiêu: Khách hàng có thể tìm kiếm sản phẩm, đặt hàng và thanh toán.*

| Nhóm | Tính năng (User Stories) |
| :--- | :--- |
| **Auth** | Đăng ký, Đăng nhập, Quên MK, Guest Checkout (US-AUTH-01~03, 05) |
| **Customer** | Hồ sơ KH, Địa chỉ giao hàng (US-USER-01~02) |
| **Catalog** | Danh mục SP, Tìm kiếm, Bộ lọc (US-DISC-01~03) |
| **Product** | Tạo SP, Variant/SKU, Giá, Trạng thái, Chi tiết HSD (US-PROD-01~06) |
| **Cart** | Thêm/sửa/xóa giỏ, Tổng tạm tính (US-CART-01~04) |
| **Checkout** | Địa chỉ, Phí ship, Tạm giữ tồn kho, Review đơn (US-CHK-01~04) |
| **Payment** | QR/Chuyển khoản, COD, Trạng thái TT (US-PAY-01~03) |
| **Order** | Lịch sử đơn, Trạng thái đơn, Hủy đơn (US-ORD-01~02, 04) |

### MVP v1.1 — Vận hành & Fulfillment (Must Have — 2nd)
*Mục tiêu: Đội ngũ vận hành xử lý được đơn hàng, quản lý tồn kho và giao hàng.*

| Nhóm | Tính năng (User Stories) |
| :--- | :--- |
| **Inventory** | Xem/Nhập/Xuất kho, Batch/Lot, HSD, Cảnh báo HSD, Điều chỉnh (US-INV-01~06, 09) |
| **Packing** | Danh sách, Chi tiết, Checklist, Ưu tiên đơn gấp, Xác nhận (US-PACK-01~03, 07~08) |
| **Shipping** | Tạo vận đơn, Tracking, Webhook 3PL (US-SHIP-01~02, 06) |
| **Order Ops** | Admin quản lý đơn, SLA ưu tiên (US-ORD-05~06) |
| **Payment Ops** | NV kiểm tra thanh toán, Đối soát (US-PAY-04~05) |
| **Admin** | Employee Account, Role, Permission, Assign Role, Access Review (US-ADM-01~07) |
| **Notification** | Email (OTP, xác nhận đơn) (US-NOTI-01) |

### MVP v1.2 — Niềm tin & Tuân thủ (Must Have — 3rd)
*Mục tiêu: Xây dựng niềm tin thương hiệu OCOP, xử lý hậu mãi và kiểm toán.*

| Nhóm | Tính năng (User Stories) |
| :--- | :--- |
| **OCOP** | QR truy xuất, Nguồn gốc, Chứng nhận OCOP, Quản lý nguồn gốc (US-OCOP-01~04) |
| **Return/Refund** | Hủy đơn, Đổi/trả, Bằng chứng, CSKH xem đơn, Duyệt, Hoàn tiền (US-RET-01~06) |
| **Customer Service** | Kênh liên hệ cơ bản (US-CS-01) |
| **Audit** | Tra cứu lịch sử, Giá trị trước/sau (US-AUDIT-01~02) |

> **Điểm hoàn thành của chuỗi MVP:** Sau MVP v1.2, Mè Xửng O Mạ đã là một kênh D2C vận hành đầu-cuối: khách có thể tìm, đặt và thanh toán sản phẩm; nhân viên có thể quản lý tồn kho theo lô/HSD, đóng gói và giao hàng; khách có thể truy xuất OCOP, gửi yêu cầu hỗ trợ/đổi trả; quản lý có dữ liệu truy vết cho các thao tác quan trọng.

## Giai đoạn 2 — Mở rộng doanh thu đa kênh & tăng trưởng (Should Have)
*Mục tiêu: Mở rộng điểm chạm bán hàng, tăng khả năng mua lại và cung cấp công cụ marketing/vận hành nâng cao. Đây là các bản phát hành sau MVP, không phải MVP mới.*

| Nhóm | Tính năng (User Stories) |
| :--- | :--- |
| **Marketplace** | Shopee/TikTok Order, Đồng bộ, Đối soát (US-MKT-01~06) |
| **Offline** | Nhận hàng, Ghi nhận bán, Báo cáo, Đối soát (US-OFF-01~06) |
| **B2B** | Báo giá, Bao bì, Xác nhận, Hóa đơn DN (US-B2B-01~05) |
| **Promotion** | Coupon, Combo, KM đang áp dụng, Áp coupon checkout (US-PROMO-01~04, US-CHK-05) |
| **Loyalty** | Tích điểm, Đổi ưu đãi, Reorder (US-LOY-01~03) |
| **Gifting** | Gửi tặng, Lời chúc, Ẩn giá (US-GIFT-01~03) |
| **Review** | Đánh giá, Ảnh, Verified, Quản lý review (US-REV-01~06) |
| **Content & SEO** | Blog, SEO sản phẩm, URL thân thiện, Duyệt nội dung (US-CONTENT-01~07) |
| **Marketing** | Chiến dịch, KPI, Duyệt KH, ROI (US-MKTG-01~07) |
| **CSKH nâng cao** | Ticket system, Lịch sử, Cảnh báo (US-CS-02~06) |
| **Auth nâng cao** | Social Login, Link Guest Order (US-AUTH-04, 06) |
| **Packing Video** | Quay, Liên kết, Tra cứu video (US-PACK-04~06) |
| **Delivery nâng cao** | DS cho Delivery Staff, Cập nhật, Tỷ lệ (US-SHIP-03~05) |
| **Notification** | SMS/Zalo, Push, Internal Alert (US-NOTI-02~04) |
| **Audit nâng cao** | Nguồn truy cập, Lý do, Tamper-evident (US-AUDIT-03~05) |
| **Inventory nâng cao** | Tốc độ tiêu thụ, FEFO, Cảnh báo Sales (US-INV-07~08, 10) |
| **Product nâng cao** | Giá đa kênh, SP nổi bật, Câu chuyện Huế (US-PROD-07, US-DISC-04~05) |
| **Profile nâng cao** | HR Profile, Quản lý trạng thái KH, OCOP NCC/Duyệt (US-USER-03, 05, US-OCOP-05~06) |
| **Dashboard** | Dashboard tổng quan (US-DSS-01) |
| **Order** | Thông báo trạng thái đơn (US-ORD-03) |
| **Return nâng cao** | Tra video đóng gói (US-RET-07) |

## Giai đoạn 3 — Phân tích, AI & chuỗi cung ứng (Could Have)
*Mục tiêu: Dùng dữ liệu giao dịch đa kênh để hỗ trợ quyết định, tối ưu sản xuất/cung ứng và tăng hiệu suất đội ngũ.*

| Nhóm | Tính năng (User Stories) |
| :--- | :--- |
| **Finance BI** | Doanh thu, Chi phí, Lợi nhuận, Khu vực, Thuế, Báo cáo (US-FIN-01~07) |
| **Customer Analytics** | RFM, Churn, Product Performance, Seasonal (US-DSS-02~04, 09) |
| **Supply Analytics** | Sales Velocity, Expiry Risk, Demand Forecast (US-DSS-05~07) |
| **Strategic DSS** | Product Association, Strategic Recommendation (US-DSS-08, 10) |
| **AI** | AI Search, AI Recommend, AI CSKH, AI Content, AI Knowledge (US-AI-01~04, 06) |
| **Procurement** | Supplier Profile, PO, Supply Planning, PO Receiving (US-PROC-01~04) |


## SẢN PHẨM SAU CHUỖI 03 MVP SẼ PHÁT TRIỂN THÀNH GÌ?

Sau các MVP, sản phẩm không chỉ là website bán mè xửng. Đây sẽ là một **nền tảng thương mại điện tử đa kênh tập trung vào sản phẩm OCOP thực phẩm**, nơi cùng một dữ liệu sản phẩm, SKU, tồn kho và đơn hàng được dùng xuyên suốt các kênh bán.

### 1. Bán hàng đa kênh từ một nguồn dữ liệu chung

- Website D2C tiếp tục là nơi khách lẻ khám phá, đặt hàng và theo dõi đơn.
- Đơn hàng Shopee/TikTok Shop được đồng bộ vào quy trình nội bộ, có mã đơn nguồn để tránh trùng lặp.
- Khi có đơn từ sàn, tồn kho khả dụng được cập nhật để hạn chế bán vượt ở Website, sàn và điểm bán.
- Nhân viên bán hàng offline nhận hàng từ kho, ghi nhận hàng bán/tồn/hỏng/trả và đối soát cuối kỳ theo từng điểm bán.
- Khách doanh nghiệp gửi yêu cầu báo giá số lượng lớn, cung cấp thông tin xuất hóa đơn, duyệt báo giá rồi chuyển thành đơn mua sỉ.
- Quản lý xem được doanh thu, tồn kho và hiệu quả theo từng kênh bán thay vì tổng hợp thủ công.

### 2. Trải nghiệm mua hàng và giữ chân khách hàng

- Khách dùng coupon, mua combo và thấy chương trình khuyến mãi phù hợp ngay trong quá trình mua.
- Hệ thống tích điểm sau mua, đổi ưu đãi và cho phép đặt lại đơn cũ nhanh chóng.
- Khách có thể mua quà: nhập người nhận khác, thêm lời chúc và ẩn giá trên phiếu quà tặng.
- Chỉ người đã mua mới được tạo đánh giá xác thực; khách có thể thêm ảnh thực tế và nhân viên xử lý các review tiêu cực/vi phạm.
- Khách nhận thông báo qua Email, SMS/Zalo, push trên web/app về đơn hàng, giao hàng và ưu đãi.
- Khách đăng nhập thuận tiện qua mạng xã hội, đồng thời có thể liên kết các đơn Guest đã mua với tài khoản mới.

### 3. Vận hành kho, đóng gói và giao hàng chặt chẽ hơn

- Kho ưu tiên xuất các Batch gần hết hạn theo chính sách FEFO, đồng thời cảnh báo Sales/CSKH để có phương án xử lý hàng cận date.
- Quy trình đóng gói có thể quay video, gắn video với đúng đơn hàng và chỉ người có quyền mới được tra cứu.
- Nhân viên giao hàng nội bộ có danh sách đơn được phân công, cập nhật trạng thái giao; quản lý theo dõi tỷ lệ giao thành công/thất bại.
- Ticket CSKH tập hợp lịch sử trao đổi, thông tin đơn/thanh toán/vận chuyển và cảnh báo ticket sắp quá SLA.
- Audit log được mở rộng với nguồn truy cập, lý do thay đổi nhạy cảm và cơ chế kiểm tra tính toàn vẹn của nhật ký.

### 4. Nội dung thương hiệu và marketing có kiểm soát

- Content Manager quản lý bài viết, hình ảnh/video, câu chuyện văn hóa Huế và thông tin thương hiệu.
- Nội dung và SEO sản phẩm có quy trình soạn, duyệt, xuất bản/ẩn; quản lý được URL thân thiện, metadata và từ khóa.
- Marketing lập kế hoạch chiến dịch, tạo nội dung, gửi duyệt và theo dõi KPI/ROI theo từng chiến dịch.
- Dashboard cơ bản cho phép lãnh đạo nhìn nhanh doanh thu, đơn hàng, khách hàng và tồn kho.

### 5. Quản trị bằng dữ liệu, không chỉ bằng cảm tính

- Finance BI tổng hợp doanh thu, chi phí, lợi nhuận, khu vực và dữ liệu đối soát/thuế theo các kênh.
- Phân tích khách hàng phân nhóm RFM, nhận diện nguy cơ rời bỏ và đánh giá sản phẩm hoặc mùa vụ bán hàng.
- Phân tích cung ứng hiển thị tốc độ bán, rủi ro hết hạn và dự báo nhu cầu để chuẩn bị sản xuất/nhập hàng.
- DSS đề xuất sản phẩm thường mua cùng nhau, combo tiềm năng và khuyến nghị về giá, tồn kho, marketing.
- Supply Manager quản lý nhà cung cấp/hộ nông dân/HTX, Purchase Order, nhận hàng theo PO và lập kế hoạch cung ứng dựa trên dự báo.

### 6. AI là lớp hỗ trợ, không thay thế quyết định nghiệp vụ

- AI hỗ trợ khách tìm sản phẩm và gợi ý sản phẩm dựa trên nhu cầu/hành vi.
- AI hỗ trợ CSKH tra cứu thông tin và soạn nháp phản hồi; nhân viên vẫn là người duyệt và gửi phản hồi.
- AI hỗ trợ Content tạo tiêu đề, mô tả và dàn ý marketing.
- AI hỗ trợ nhân viên tìm kiếm tri thức nội bộ và tổng hợp dữ liệu cho quyết định.
- AI không tự thay đổi giá, tồn kho, đơn hàng, hoàn tiền hoặc dữ liệu nghiệp vụ quan trọng nếu không có bước phê duyệt phù hợp.

### Tóm tắt hình dung sản phẩm hoàn chỉnh

```text
Khách lẻ / Khách B2B / Khách sàn / Điểm bán offline
                         │
                         ▼
      Website D2C + Marketplace + B2B + Offline Sales
                         │
                         ▼
       Nền tảng đơn hàng dùng chung: Product / SKU / Giá / Tồn kho
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
  Kho - Batch - HSD   Đóng gói - Giao   CSKH - Đổi trả
          │              │              │
          └──────────────┼──────────────┘
                         ▼
  OCOP Traceability + Marketing + Finance BI + DSS/AI + Procurement
```


---

# II. BẢN ĐỒ USER STORIES (USER STORY MAP & EPICS OVERVIEW)

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

# III. LỘ TRÌNH 12 TUẦN (SPRINT PLANNING & SPRINT BACKLOG)

Lộ trình dùng **12 Sprint, mỗi Sprint kéo dài 01 tuần**. Công việc vẫn được triển khai theo **Vertical Slices (lát cắt dọc)**: mỗi tuần tạo ra một phần chức năng có thể kiểm thử từ giao diện, API đến dữ liệu, thay vì chỉ hoàn thành một lớp kỹ thuật đơn lẻ.

## KẾ HOẠCH SPRINT THEO TỪNG TUẦN

> **Lưu ý xuyên suốt 12 tuần:** Unit Test, Integration/Contract Test, kiểm tra phân quyền, bảo mật đầu vào, review mã nguồn và cập nhật tài liệu là điều kiện hoàn thành của mọi Sprint; không để dồn vào Sprint 12.


### Sprint 01 — Tuần 1: Nền tảng tài khoản và dữ liệu sản phẩm

- Thiết lập môi trường dự án, cơ sở dữ liệu ban đầu, CI/CD cơ bản, logging và khung kiểm thử.
- Hoàn thành đăng ký, đăng nhập, quên mật khẩu; tạo được hồ sơ và địa chỉ giao hàng của khách hàng (`US-AUTH-02, 03, 05`, `US-USER-01, 02`).
- Hoàn thành trang quản trị tối thiểu để tạo sản phẩm, SKU/Variant, giá và trạng thái bán (`US-PROD-01, 02, 05, 06`).
- **Đầu ra tuần:** có thể tạo tài khoản, đăng nhập và tạo dữ liệu sản phẩm thật trên môi trường kiểm thử.

### Sprint 02 — Tuần 2: Khám phá và chọn sản phẩm

- Hiển thị danh mục, danh sách sản phẩm và trang chi tiết sản phẩm cho khách (`US-DISC-01`, `US-PROD-03, 04`).
- Bổ sung tìm kiếm theo từ khóa và lọc theo giá, khối lượng, loại sản phẩm, đánh giá và tình trạng còn hàng (`US-DISC-02, 03`).
- Kiểm thử luồng hiển thị đúng SKU, giá, thành phần, khối lượng, ngày sản xuất và hạn sử dụng.
- **Đầu ra tuần:** khách có thể tìm được đúng loại mè xửng/SKU và biết rõ thông tin trước khi mua.

### Sprint 03 — Tuần 3: Giỏ hàng và chuẩn bị checkout

- Hoàn thành thêm, sửa số lượng, xóa sản phẩm và tính tổng tạm tính trong giỏ (`US-CART-01~04`).
- Hoàn thành Guest Checkout để khách chưa có tài khoản vẫn đi tiếp được đến bước đặt hàng (`US-AUTH-01`).
- Tạo màn hình checkout hiển thị địa chỉ nhận hàng, phí vận chuyển dự kiến, chi tiết đơn và tổng tiền (`US-CHK-01, 02, 04`).
- **Đầu ra tuần:** khách đăng nhập hoặc khách vãng lai đều tạo được một đơn nháp đầy đủ thông tin.

### Sprint 04 — Tuần 4: Thanh toán, đơn hàng và phát hành MVP v1.0

- Tạm giữ tồn khi checkout để ngăn hai khách cùng mua vượt số lượng khả dụng (`US-CHK-03`).
- Hỗ trợ thanh toán QR/chuyển khoản và COD; ghi nhận, hiển thị đúng trạng thái thanh toán (`US-PAY-01~03`).
- Hoàn thành lịch sử đơn, theo dõi trạng thái và tự hủy đơn khi còn đủ điều kiện (`US-ORD-01, 02, 04`).
- Kiểm thử E2E luồng: tìm sản phẩm → giỏ hàng → checkout → thanh toán → xem/hủy đơn.

### Sprint 05 — Tuần 5: Quản trị nhân viên và xử lý đơn nội bộ

- Tạo/cập nhật/khóa tài khoản nhân viên; cấu hình role, permission, gán role và tra cứu quyền (`US-ADM-01~07`).
- Tạo màn hình cho Sales Manager xem, xử lý danh sách đơn; giúp nhân viên nhận biết đơn cần xử lý trước theo SLA (`US-ORD-05, 06`).
- Bổ sung chức năng để nhân viên kiểm tra trạng thái thanh toán và Manager đối soát giao dịch với đơn (`US-PAY-04, 05`).
- **Đầu ra tuần:** mỗi vai trò nội bộ chỉ thấy đúng chức năng được cấp và đơn đã thanh toán được chuyển cho vận hành xử lý.

### Sprint 06 — Tuần 6: Kho và tính khả dụng theo SKU

- Xem tồn kho theo SKU; ghi nhận nhập kho, xuất kho và điều chỉnh hàng hỏng/thất thoát (`US-INV-01~03, 09`).
- Liên kết thay đổi tồn với các trạng thái đơn để không xác nhận đơn vượt tồn khả dụng.
- Kiểm thử đồng thời: khi chỉ còn một đơn vị, chỉ một trong các yêu cầu đặt hàng cạnh tranh được xác nhận.
- **Đầu ra tuần:** số liệu tồn kho phản ánh các giao dịch thực tế và được dùng làm điều kiện đặt hàng.

### Sprint 07 — Tuần 7: Batch/Lot, hạn sử dụng và cảnh báo

- Quản lý Batch/Lot, ngày sản xuất và hạn sử dụng của hàng thực phẩm (`US-INV-04, 05`).
- Thiết lập cảnh báo Batch/Lot sắp hết hạn cho nhân viên kho (`US-INV-06`).
- Kiểm thử quy tắc không cho phép bán Batch đã hết hạn và đảm bảo dữ liệu Batch có thể truy xuất theo SKU.
- **Đầu ra tuần:** kho biết chính xác hàng đang còn thuộc lô nào, còn hạn bao lâu và lô nào cần xử lý.

### Sprint 08 — Tuần 8: Đóng gói, giao hàng và phát hành MVP v1.1

- Hiển thị danh sách đơn cần đóng gói, chi tiết SKU/số lượng, checklist kiểm tra, thứ tự ưu tiên và xác nhận đã đóng gói (`US-PACK-01~03, 07, 08`).
- Tạo vận đơn, hiển thị tracking và nhận cập nhật trạng thái từ đơn vị giao vận (`US-SHIP-01, 02, 06`).
- Gửi Email xác nhận đơn; kiểm tra toàn bộ chuỗi đơn hàng từ Paid đến Packed, Shipped và Delivered (`US-NOTI-01`).

### Sprint 09 — Tuần 9: Truy xuất nguồn gốc OCOP

- Quản lý dữ liệu nguồn gốc của sản phẩm/lô hàng: vùng nguyên liệu, nhà sản xuất và chứng nhận (`US-OCOP-04`).
- Xây dựng trang quét QR/tra cứu để khách xem thông tin nguồn gốc, quy trình và chứng nhận OCOP (`US-OCOP-01~03`).
- Kiểm thử phân quyền sửa dữ liệu nguồn gốc và kiểm tra nội dung công khai khớp với Batch/SKU.
- **Đầu ra tuần:** một sản phẩm OCOP có QR để khách tự kiểm tra thông tin minh bạch.

### Sprint 10 — Tuần 10: Đổi trả, khiếu nại và hoàn tiền

- Khách gửi yêu cầu hủy/đổi trả và đính kèm ảnh hoặc video bằng chứng (`US-RET-01~03`).
- CSKH xem đơn, thanh toán, bằng chứng; Manager duyệt/từ chối yêu cầu và thực hiện hoàn tiền (`US-RET-04~06`).
- Kiểm thử state machine đổi trả: `RETURN_REQUESTED → REVIEWING → APPROVED/REJECTED → RETURNED → REFUNDED`.
- **Đầu ra tuần:** yêu cầu hậu mãi được xử lý theo trạng thái rõ ràng, có người chịu trách nhiệm duyệt.

### Sprint 11 — Tuần 11: Kênh hỗ trợ cơ bản và Audit Log

- Cung cấp kênh để khách gửi câu hỏi/yêu cầu hỗ trợ (`US-CS-01`).
- Ghi nhận Audit Log cho thao tác quan trọng: ai thực hiện, lúc nào, trên đối tượng nào, giá trị trước và sau (`US-AUDIT-01, 02`).
- Kiểm thử các thao tác nhạy cảm như đổi giá sản phẩm, đổi trạng thái đơn, duyệt hoàn tiền và điều chỉnh tồn kho.
- **Đầu ra tuần:** khách có nơi liên hệ hỗ trợ và quản lý có thể truy vết nguyên nhân thay đổi dữ liệu quan trọng.

### Sprint 12 — Tuần 12: Kiểm thử tích hợp, UAT và phát hành MVP v1.2

- Kiểm thử E2E các luồng mua hàng, kho, giao hàng, truy xuất OCOP, đổi trả/hoàn tiền và Audit.
- Thực hiện UAT với người dùng đại diện: khách hàng, Sales Manager, Warehouse Staff, Packing Staff và CSKH.
- Khắc phục lỗi ưu tiên cao; hoàn thiện tài liệu hướng dẫn vận hành, dữ liệu khởi tạo và kế hoạch rollback phát hành.


---

# IV. ĐỊNH NGHĨA HOÀN THÀNH (DOD) & QUẢN TRỊ RỦI RO

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
*Tài liệu này là Kế hoạch phần mềm chính thức, phục vụ công tác quản trị dự án, kiểm thử và nghiệm thu đồ án chuyên ngành CNTT.*
