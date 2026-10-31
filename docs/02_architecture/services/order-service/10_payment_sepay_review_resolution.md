# 10 — Phản hồi review Payment SePay + VietQR

**Ngày:** 16/10/2026 · **Thiết kế:** `payment-sepay-1.1-review` · **Phạm vi:** sửa tài liệu/contract, chưa sửa runtime.

[Thiết kế chi tiết](09_payment_sepay_vietqr_design.md) · [Customer/Guest/Webhook OpenAPI](../../../03_api_specs/order-payment-sepay.candidate.openapi.yaml) · [Staff/Refund OpenAPI](../../../03_api_specs/order-payment-staff.candidate.openapi.yaml)

## 1. Kết quả xử lý từng ý kiến

| Review | Quyết định | Thay đổi cụ thể | Phần còn phải nghiệm thu |
| --- | --- | --- | --- |
| P1-1 API v2 trên critical path | Giữ một canonical authority trong MVP; đây là đánh đổi có chủ đích | Mục 5.4 thêm ví dụ trả trước hạn/xác minh sau hạn, Product sign-off và phép đo latency; không backdate receipt decision | Product chưa phê duyệt; chưa có số đo với receiver thật |
| P1-2 Checkpoint | Chốt window scan + overlap làm mặc định; since_id=false | Mục 9.1.1 định nghĩa run fields, 3 coverage clocks, CAS epoch, restart từ page1, crash matrix và cursor gate | Multi-page/concurrent insert/restore suite với API thực |
| P1-3 Refund chưa đủ production | Automatic refund vẫn tắt; manual là release gate bắt buộc | Mục 10.4 và OpenAPI Staff khai báo approval ingestion, queue, claim/start, evidence, verify, retry/cancel, quyền/MFA/audit, separation-of-duties | Implement UI/API, xác minh Care/Identity, staff training và diễn tập trước nhận tiền thật |
| P1-4 Exact-only | Giữ; không gộp hoặc tách tiền tự động | Mục 11.2 thêm nội dung UI under/over/multiple, support reference, case SLA và notification dedup | Product review thông điệp/SLA; Care và Notification integration |
| P2-5 Guest | Bổ sung order-bound signed grant, không dùng mã đơn làm credential | Mục 11.1 và OpenAPI thêm mint/revoke, claims, TTL10m/session24h, authoritative revocation, ownership recheck, fail closed | Identity/session authority và registry chưa implement; recovery vẫn cần verified flow |
| P2-6 VERIFYING | Chờ có hạn ở UI, không tạo payment success giả | due_at=min(first observation+60s, payment expiry), display VERIFICATION_DELAYED, support/timer contract và polling dừng spinner | Timer/retry/expiry race tests và UI behavior |

## 2. Các thay đổi wire đáng review

- Customer payment response thêm verification_started_at/due_at, support_reference/reason, next_poll_after_seconds và display_state VERIFICATION_DELAYED. Các field là required nullable khi chưa có observation; consumer phải nâng contract cùng phiên bản.
- Guest mint/revoke dùng Guest checkout credential; grant mới chỉ có quyền đọc payment của một Order. Token Customer và token Guest có issuer/scope riêng.
- Staff mutation dùng X-Idempotency-Key, expected_version, reason, permission và scope. Replay không tạo execution attempt mới; sensitive instruction/evidence URL luôn kiểm lại quyền hiện tại.
- Care approval ingestion qua endpoint mới hỗ trợ receipt không có Order; không mở public refund request cho phép tự nhập amount/account.
- Staff verify cần nguồn ngân hàng độc lập. Proof chưa chắc chắn giữ UNKNOWN và budget; chỉ không nhìn thấy giao dịch chưa đủ kết luận NOT_SENT.

## 3. Điểm không được hiểu là đã hoàn tất

Viết contract không đóng G08/R04 hoặc biến simulator thành production integration. Các giá trị 60 giây verification, 10 phút grant, 24 giờ session, 5 phút claim và SLA staff là policy đề xuất để review, không phải hạn mức SePay. Chưa có deployment, secret, live receiver, chuyển tiền, refund hoặc thông báo ngoài hệ thống trong lượt này.

Bank Webhooks Test mode và SePay API v2 Sandbox là môi trường đúng cho adapter đã chọn. Hosted Payment Gateway Sandbox là sản phẩm khác; không dùng PASS ở đó để nghiệm thu HMAC bank webhook, numeric-ID→UUID mapping hoặc refund capability.

## 4. Cách review tiếp

Đọc mục 5.4 để quyết định late-payment policy; mục 9.1.1 để kiểm checkpoint; mục 10.4 và Staff OpenAPI để review quy trình vận hành tiền; mục 11.1–11.2 để review Guest/UI. Duyệt thiết kế có thể diễn ra trước triển khai; go-live chỉ sau khi các gates tương ứng có bằng chứng.

## 5. Kiểm tra static của bản 1.1

- Hai OpenAPI qua `openapi_spec_validator`: Customer/Guest/Webhook có 4 paths/5 operations; Staff/Refund có 16 paths/16 operations.
- 7 examples hợp lệ theo schema; verification schema từ chối SENT thiếu bank evidence và NOT_SENT mang bộ field của SENT; INCONCLUSIVE hợp lệ vẫn giữ quy tắc budget ở application.
- Staff commands đều khai báo permission và idempotency header; kiểm này không chứng minh runtime đã enforce quyền.
- 18 liên kết local trong chương09/10 hợp lệ; Markdown code fences cân bằng. `git diff --check` không có lỗi whitespace.
- Kiểm rộng hơn phát hiện các liên kết lịch sử tới chương09–12 đang thiếu vì các file đó đã bị xóa trong workspace trong lúc review; không khôi phục hoặc thay đổi nội dung các file đã xóa trong lượt này.
- Không chạy runtime, không gọi API có credential và không thực hiện giao dịch ngân hàng.
