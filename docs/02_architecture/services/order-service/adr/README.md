# ADR của đợt giao đầu Order

ADR là bản ghi quyết định thiết kế: bối cảnh, lựa chọn, lý do và đánh đổi. Ngày accepted không đồng nghĩa ngày tích hợp production thành công.

| ADR | Quyết định |
| --- | --- |
| [001 — Phạm vi giao hàng](ADR-001-mvp-delivery-profile.md) | `mvp1` là đợt đầu của Order, còn thiếu story của MVP 1.0 sản phẩm |
| [002 — SePay và payment attempt](ADR-002-sepay-payment-attempt.md) | Gateway sandbox, một invoice, form có thứ tự, IPN và query |
| [003 — Hoàn tiền manual](ADR-003-manual-refund.md) | Ngân sách, nhận việc độc quyền, bằng chứng và người kiểm tra độc lập |
| [004 — Review và sự kiện](ADR-004-local-review-and-manual-settlement-event.md) | Duyệt hủy tại Order; fact manual riêng, không giả provider proof |
| [005 — Kiểm chứng sandbox local](ADR-005-local-sepay-sandbox-validation.md) | Mô tả đơn được ký; database riêng; IPN qua tunnel giới hạn đường dẫn |
| [006 — Tra cứu theo invoice](ADR-006-sepay-query-verified-identifiers.md) | Định danh detail đã thử thật, timeout có giới hạn và giữ xác thực IPN |
| [007 — Ngoại lệ phục hồi thanh toán](ADR-007-payment-recovery-exception-tests.md) | Worker IPN độc lập; query thiếu transaction vào review; không chặn IPN hợp lệ đến sau |

[Báo cáo implementation và bằng chứng](../14_mvp_implementation_and_adr_report.md) · [Phạm vi thiết kế](../13_mvp_delivery_decisions.md)
