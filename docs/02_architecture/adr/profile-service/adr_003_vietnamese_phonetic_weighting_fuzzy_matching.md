# ADR 003: Vietnamese Phonetic Weighting in Levenshtein Distance for Two-Tier Administrative Matching

## Status
**Accepted & Implemented** (24/09/2026)

## Context
Trong hệ thống bán hàng đặc sản Mè Xửng O Mạ OCOP Huế, sau khi Nghị quyết của Quốc hội có hiệu lực từ ngày 01/07/2025, Việt Nam đã chuyển đổi sang mô hình hành chính 2 cấp (Cấp Tỉnh/Thành phố trực thuộc Trung ương $\rightarrow$ Cấp Xã/Phường/Thị trấn), giải thể hoàn toàn cấp Huyện. Toàn bộ Thành phố Huế trực thuộc Trung ương quản lý 40 đơn vị hành chính cấp xã (21 phường, 19 xã).

Khách hàng khi nhập địa chỉ giao hàng thường gõ theo thói quen phát âm tiếng Việt địa phương miền Trung hoặc lỗi bộ gõ Telex:
- Hoán đổi nguyên âm đồng âm: `'i'` ngắn và `'y'` dài (Ví dụ: *"Vi Dạ"* thay vì *"Vỹ Dạ"*, *"Thúy Biều"* thay vì *"Thủy Biều"*).
- Bỏ dấu thanh, viết tắt tiền tố (*"P."*, *"X."*, *"TP Huế"*).

Khi áp dụng thuật toán so khớp Levenshtein Distance truyền thống (mọi phép thay thế substitution cost đều bằng 1), chuỗi *"vi da"* so với *"vy da"* (độ dài 5 ký tự) cho ra khoảng cách $dist = 1$, khiến độ tương đồng bị kéo xuống còn:
$$Score = 1.0 - \frac{1.0}{5.0} = 0.80$$
Mức điểm này thấp hơn ngưỡng chấp nhận chuẩn hóa ($0.85$), dẫn đến việc gợi ý địa chỉ giao hàng bị từ chối hoặc yêu cầu khách hàng phải chọn thủ công, làm tăng tỷ lệ rơi rụng (Drop-off rate) trong giỏ hàng.

## Decision
Chúng tôi quyết định nâng cấp thuật toán so khớp Levenshtein Distance trong `internal/usecase/address_fuzzy_matcher.go` với các nguyên tắc sau:
1. **Âm vị học tương đương (Phonetic Equivalence)**: Cặp ký tự hoán đổi giữa `'i'` và `'y'` được gán trọng số chi phí phạt $Cost = 0.5$ (thay vì $1.0$).
2. **Khoảng cách dạng số thực (`float64`)**: Thay vì ma trận số nguyên, ma trận quy hoạch động tính toán khoảng cách sử dụng `float64` để phản ánh chính xác các mức độ tương đồng vi mô.
3. **Bảo tồn Boundary Critical Path**: Thuật toán Fuzzy Matching có trọng số này **chỉ được gọi bất đồng bộ** tại endpoint validate địa chỉ của UI khách hàng (`POST /api/v1/profile/addresses/validate`). Trên Critical Checkout Path (`GetDeliveryAddress`), service truy vấn trực tiếp bằng UUID và cache 2 lớp (B-Tree + Memory), đảm bảo độ trễ $P99 \le 5\text{ms}$ (thực tế đo lường $421.9\text{ns}$).

## Consequences
- **Ưu điểm**:
  - Tỉ lệ khớp chính xác trên bộ dữ liệu kiểm thử vàng (Golden Dataset 50 ca kiểm thử thực tế địa bàn Huế) đạt **100% (50/50 test cases PASS)**.
  - Xử lý mượt mà thói quen người dùng địa phương Thừa Thiên Huế mà không cần phụ thuộc vào các dịch vụ Geocoding bên thứ ba đắt đỏ hoặc cồng kềnh.
- **Hạn chế**:
  - Phép tính số thực `float64` tốn thêm một lượng CPU không đáng kể so với `int`, nhưng do chỉ chạy trên chuỗi ngắn (tên phường xã $\le 30$ ký tự) và tách biệt khỏi checkout path nên hoàn toàn không ảnh hưởng tới tải hệ thống.
