# Chuẩn bị Order vào dev — 30/10/2026

Đồng bộ từ `dev` tại `7f22226` (PR #34) vào nhánh Order, giữ implementation đã
nghiệm thu ở PR #35. Các conflict thuộc tài liệu tổng quan, contract và codegen;
không có conflict trong source nghiệp vụ Order.

Quyết định giải quyết:

- Tài liệu tổng quan dùng bản chung trên dev; thiết kế riêng Order vẫn ở thư mục
  service và các ADR, không dùng tài liệu cũ để ghi đè bộ tích hợp MVP.
- Shared proto/events/contracts dùng release PR #34: Shipping v1 giữ wire provider,
  quote binding ở v2; giữ GetShippingSnapshot, revision proof và codegen Buf mới.
- Sinh lại stub Common/Order từ proto đã hợp nhất bằng toolchain service đã pin.
- Giữ sandbox regression contract 3.1 và gate các capability chưa có provider;
  generated GetShippingSnapshot không đồng nghĩa RPC đã có implementation.
- Profile lấy bản mới nhất trên nhánh Profile sau khi đồng bộ dev, gồm consumer
  Identity/provisioning; tránh mang phiên bản Profile cũ vào dev qua lịch sử Order.
- Thống nhất ignore rules với Identity/Profile để giữ dữ liệu private và build output
  ngoài Git, tránh conflict phụ khi merge theo thứ tự.

PR #32 là PR Order vào dev hiện có. CI phải chạy trên HEAD sau đồng bộ, vì kết quả
xanh trước đó chưa kiểm protobuf chung mới. Contract validator kiểm wire/schema,
không thay thế kiểm thử provider xuyên service. Không bật guest/COD/completion hoặc
provider capability chỉ vì đã merge implementation vào nhánh tích hợp.

Thứ tự đề xuất: Identity → Profile #18 → Order #32. Chỉ merge sau khi xem kết quả
CI và phạm vi PR; lần chuẩn bị này không trực tiếp merge vào dev.
