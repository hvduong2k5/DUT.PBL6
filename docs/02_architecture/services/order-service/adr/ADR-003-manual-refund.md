# ADR-003 — Hoàn tiền thủ công có ngân sách và bằng chứng

Ngày: 15/10/2026. Quyết định: accepted cho đợt giao đầu.

## Quyết định

Tạo ledger manual_refunds riêng, không đưa công việc này vào worker REFUND tự động. Một khoản thu chỉ được hoàn toàn bộ trong đợt đầu. Duyệt dưới khóa receipt; tính cả nghĩa vụ tự động cũ và manual chưa kết thúc để ngăn vượt ngân sách.

Khoản đã allocate cho Order chỉ được hoàn khi đơn đã hủy và việc nhả/đảo kho đã có kết quả terminal; đang finalizing/không rõ thì phải làm rõ trước. Khoản dư/chưa khớp đơn có thể hoàn riêng sau xác minh người nhận; không làm thay đổi khoản allocation hợp lệ.

Một người được nhận việc bằng cập nhật nguyên tử. SUBMITTED/UNKNOWN giữ người thực hiện và ngân sách; không tự chuyển việc cho người khác. Người thực hiện ghi mã giao dịch và thông tin đối chiếu. Người kiểm tra khác người thực hiện xác nhận SUCCEEDED. Đây là xác nhận vận hành có audit, không phải truy vấn ngân hàng tự động.

Không nhận ảnh chụp đơn thuần như xác nhận cuối; lưu dữ liệu đối chiếu giao dịch ngân hàng, số tiền, thời điểm, nguồn chứng từ. Bằng chứng cùng ngân hàng/tài khoản nguồn + mã giao dịch chỉ được dùng cho một nghĩa vụ. Retry cùng nội dung trả kết quả cũ, nội dung khác trả conflict.

## Trade-off

Cần ít nhất hai nhân viên có quyền để xác nhận cuối và cần trực vận hành. Không có API nào trong thiết kế này tự chuyển tiền. Không hỗ trợ hoàn một phần đồng nghĩa chưa đáp ứng đầy đủ RET-06 của MVP sản phẩm. Tính năng này phải được mở rộng trước nghiệm thu toàn bộ epic.
