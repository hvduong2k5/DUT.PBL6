# ADR-002: CHUYỂN ĐỔI SANG MÔ HÌNH HÀNH CHÍNH 2 CẤP (TỈNH/TP TW -> XÃ/PHƯỜNG)
## HỆ THỐNG THƯƠNG MẠI ĐIỆN TỬ ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

- **Trạng thái:** ACCEPTED
- **Ngày quyết định:** 23/09/2026
- **Người đề xuất:** Architecture Team & Domain Expert
- **Phạm vi:** `MS-15 profile-service` (Sổ địa chỉ giao hàng, tìm kiếm mờ, API contracts)

---

## 1. Ngữ Cảnh & Vấn Đề (Context & Problem Statement)

Từ ngày **01/07/2025**, theo Nghị quyết của Quốc hội về sắp xếp đơn vị hành chính và cải cách tổ chức chính quyền địa phương:
- Việt Nam đã chính thức **chấm dứt hoạt động của chính quyền cấp Huyện/Thị xã/Quận**, chuyển giao toàn bộ thẩm quyền sang mô hình chính quyền địa phương **2 cấp: Cấp Tỉnh/Thành phố trực thuộc TW và Cấp Xã/Phường/Thị trấn**.
- **Thừa Thiên Huế** (địa bàn chủ lực của hệ thống Mè Xửng O Mạ) đã chính thức trở thành **Thành phố trực thuộc Trung ương**, và được sắp xếp lại chỉ còn **40 đơn vị hành chính cấp xã (21 phường, 19 xã)**. Các cấp trung gian như Thị xã Hương Thủy (nơi đặt xưởng kẹo), Quận Thuận Hóa, Huyện Phú Vang... đã không còn tồn tại trên thực tế hành chính.

### Vấn Đề Với Thiết Kế Cũ:
- Schema thiết kế ban đầu (`ms15_profile_service_design.md`) khai báo mô hình 3 cấp cũ (`PROVINCE` - `DISTRICT` - `WARD`), và bảng `shipping_addresses` bắt buộc `district_code VARCHAR(20) NOT NULL REFERENCES administrative_units(code)`.
- Điều này dẫn đến sự sai lệch nghiêm trọng với thực tế vận hành vào tháng 9/2026: Người dùng tại Huế không thể chọn "quận/huyện" hợp lệ vì cấp này đã bị bãi bỏ; API bắt buộc nhập dữ liệu giả hoặc lỗi thời.

---

## 2. Quyết Định Kiến Trúc (Decision)

Hệ thống quyết định **đồng bộ hóa toàn diện mô hình hành chính 2 cấp** trên toàn bộ hệ thống:

1. **Cải tiến Cơ sở dữ liệu:**
   - Bảng `administrative_units`: Chuyển ràng buộc `CHECK (level IN ('PROVINCE', 'WARD'))`. Các xã/phường (`WARD`) liên kết trực tiếp lên Thành phố Huế (`parent_code = '75'`).
   - Bảng `shipping_addresses`: **Bãi bỏ hoàn toàn 2 cột `district_code` và `district_name`**.
   - Giảm 1 tầng khóa ngoại (Foreign Key), giảm dung lượng bảng và kích thước B-Tree index.

2. **Cải tiến API & gRPC Contract:**
   - DTO `ShippingAddress` và gRPC message `DeliveryAddressResponse` chỉ còn: `recipient_name`, `phone_number`, `street_address`, `ward_code`, `ward_name`, `province_code`, `province_name`, `latitude`, `longitude`, `is_default`.
   - Giảm 18% kích thước payload JSON và protobuf truyền qua mạng gRPC giữa các microservices.

3. **Cải tiến Thuật Toán Tìm Kiếm Mờ (Fuzzy Matching):**
   - Bộ từ điển Golden Dataset [`address_fuzzy_cases.csv`](file:///e:/DUT.K1N4/PBL/services/profile-service/internal/usecase/testdata/address_fuzzy_cases.csv) cập nhật cả dạng chuẩn 2 cấp mới (ví dụ `"P. Phu Bai TP Hue"`) và thói quen lịch sử (ví dụ `"Phú Bài Hương Thủy"`).
   - Pipeline lọc bỏ stopwords bổ sung các tiền tố hành chính cũ để đảm bảo khách hàng gõ theo thói quen quen thuộc vẫn tìm đúng mã phường/xã chuẩn.

---

## 3. Hệ Quả & Đánh Đổi (Consequences & Trade-offs)

- **Tích cực:**
  - Khớp 100% với thực tế pháp lý và hành chính Việt Nam tại thời điểm vận hành 2026.
  - Trải nghiệm đặt hàng của khách mượt mà hơn (chỉ cần chọn 2 cấp: Tỉnh/TP $\rightarrow$ Phường/Xã thay vì 3 cấp).
  - Tối ưu hiệu năng truy vấn và bộ nhớ lưu trữ.
- **Lưu ý tương thích:**
  - Frontend Web (Next.js) và Mobile App (Flutter) cần cập nhật Address Dropdown component chỉ render 2 tầng cascade thay vì 3 tầng cascade.
