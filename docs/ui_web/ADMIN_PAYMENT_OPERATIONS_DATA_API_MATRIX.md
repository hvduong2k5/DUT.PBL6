# Admin Payment Operations — Data/API Matrix

**Cập nhật:** 2026-10-02  
**Nguồn phạm vi:** `docs/01_requirements/epics/EPIC_07_Payment.md`  
**Màn hình:** `/payments` trong `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

## Phạm vi đã triển khai

| Khả năng | Nguồn Epic | Browser/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Danh sách Payment cần theo dõi | `US-PAY-04`, `FR-PAY-12` | `GET /api/admin/payments/cases` | `GET /admin/payments/cases` | Read-only |
| KPI tổng hồ sơ, đã khớp, chờ bằng chứng và có chênh lệch | `US-PAY-04~05`, `FR-PAY-13~14` | list projection | cùng list route | Đã có |
| Lọc theo kết quả, phương thức, trạng thái Payment và mức chú ý | `US-PAY-04~05` | Lọc trên projection đã tải | cùng list route | Đã có |
| Đối chiếu số tiền, tiền tệ, tham chiếu và bằng chứng trạng thái | `US-PAY-05`, `FR-PAY-14` | `GET /api/admin/payments/cases/:caseId` | cùng path không có `/api` | Popup read-only |
| Phân biệt Payment Attempt, Transaction và Order | Business Rules Epic 07 | detail projection | detail route | Đã có |
| Lịch sử và bảo toàn bằng chứng gốc | `FR-PAY-15` | detail projection | detail route | Chỉ đọc |

## Permission và dữ liệu nhạy cảm

- `PAYMENT_VIEW`: xem danh sách Payment và trạng thái tổng hợp.
- `PAYMENT_RECONCILIATION_VIEW`: mở hồ sơ đối soát chi tiết.
- `PAYMENT_SENSITIVE_VIEW`: xem đầy đủ tham chiếu provider/ngân hàng; nếu thiếu, BFF chỉ trả bốn ký tự cuối.
- Fixture `SALES_MANAGER` có hai quyền xem, scope `PAYMENT: ALL`, nhưng không có quyền xem tham chiếu đầy đủ.
- Fixture `ACCOUNTANT` có cả ba quyền, scope `PAYMENT: ALL` để kiểm thử nhiệm vụ Accountant/Finance Staff trong Epic 07.
- `SYSTEM_ADMIN` không tự động có quyền Payment. Role không thay thế kiểm tra Permission tại BFF.

## Ranh giới nghiệp vụ

- Payment status và Order status được hiển thị riêng; giao dịch nhận được không tự làm Order thành `PAID`.
- Payment Attempt không phải Payment Transaction. Popup trình bày thành hai bảng độc lập.
- Chỉ giao dịch có bằng chứng đã xác minh mới được tính là số tiền xác nhận trong projection upstream.
- Bằng chứng gốc là bất biến; sửa sai trong tương lai phải tạo dấu vết mới, không ghi đè giao dịch cũ.
- Các phương thức và provider trong mock chỉ là fixture trung lập, không phải quyết định chọn nhà cung cấp production.
- KPI lấy từ projection đã tải và chỉ mô tả tập dữ liệu hiện tại; backend thật chịu trách nhiệm aggregate và scope.

## Mutation tạm gác

Chưa triển khai đánh dấu Paid, liên kết giao dịch, điều chỉnh, đóng hồ sơ hoặc xử lý chênh lệch vì Epic còn yêu cầu chốt:

1. Provider/phương thức được bật và nguồn xác nhận có thẩm quyền.
2. Vòng đời chuẩn cho Attempt, Transaction và Reconciliation Case.
3. Xử lý thanh toán đến muộn sau khi Order/giữ tồn kho hết hạn.
4. Quy tắc COD và nguồn xác nhận đã thu tiền.
5. Underpayment, overpayment, duplicate, unmatched và nhiều giao dịch thành công.
6. Quyền điều chỉnh/liên kết thủ công, phê duyệt và Audit bắt buộc.
7. Chu kỳ, input, SLA đối soát và ranh giới Sales với Accountant.
8. Retention, masking và encryption cho dữ liệu tài chính.

## Chạy local

```powershell
npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
$env:ADMIN_MOCK_ROLE="ACCOUNTANT"
npm run dev
```

Mở `http://localhost:3001/payments`. Dùng `SALES_MANAGER` để kiểm tra tham chiếu bị che và `SYSTEM_ADMIN` để kiểm tra từ chối truy cập.
