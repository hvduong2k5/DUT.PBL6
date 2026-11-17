# Kế hoạch UI khách hàng Gói Huế

## Kiến trúc và giới hạn

DUT.PBL6 dùng frontend Next.js hiện hữu, backend thông qua API adapter. Nhận diện dùng tài nguyên Gói Huế, font Việt phục vụ local. Điều hướng catalogue → trang chi tiết → giỏ → checkout giữ các hợp đồng và quyền đang có.

## Các mốc

1. Khảo sát dev và thư viện tham chiếu — đã làm.
2. Home và điều hướng theo mẫu — đã có checkpoint e5b8612. Người dùng cho phép thay cấu trúc để gọn hơn.
3. Trùng tu UI khách hàng và làm rõ recommendation — triển khai trong STOREFRONT_RENOVATION.md.
4. UI trùng tu đã được duyệt. Bổ sung dữ liệu mẫu SQL và chuyển mock/real/auto — đã triển khai, chờ duyệt dữ liệu.
5. Sau duyệt dữ liệu, tiếp tục UI khác và kiểm thử backend thật.

## Quyết định

Loại bỏ các nút mock khỏi bản mua sắm. Ưu tiên sản phẩm hơn thẻ danh mục trên Home. Tái sử dụng hệ font, palette và khung gọn, giới hạn họa tiết vào các vùng có nội dung văn hóa. Giữ API, quyền, consent và các luồng thanh toán hiện hữu. Admin thuộc phạm vi riêng.

Trạng thái tiếp tục ở docs/CURRENT_STATE.md.
