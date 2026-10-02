# Admin Return & Refund — Data/API Matrix

**Cập nhật:** 2026-10-02  
**Nguồn phạm vi:** `docs/01_requirements/epics/EPIC_14_Return_Refund.md`  
**Màn hình:** `/returns` trong `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

## Phạm vi đã triển khai

| Khả năng | Nguồn Epic | Browser/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Danh sách Return/Refund Case | `US-RET-04~06`, `FR-RET-07~10` | `GET /api/admin/returns/cases` | `GET /admin/returns/cases` | Read-only |
| KPI Case hoạt động, thẩm định, thiếu bằng chứng và Refund cần theo dõi | `US-RET-04~06` | list projection | cùng list route | Đã có |
| Lọc trạng thái Case, loại yêu cầu, bằng chứng và Refund | `US-RET-04~06` | Lọc trên projection đã tải | cùng list route | Đã có |
| Xem snapshot Order line, Payment, Shipment và lịch sử | `US-RET-04`, `FR-RET-07` | `GET /api/admin/returns/cases/:caseId` | cùng path không có `/api` | Popup read-only |
| Xem metadata bằng chứng khách gửi theo Permission | `US-RET-03~04`, `FR-RET-05~07` | detail projection | detail route | Không trả binary/URL lưu trữ |
| Phân biệt Case, hàng trả và Refund | `FR-RET-10~16`, quy tắc dữ liệu Epic 14 | list + detail | hai route trên | Đã có |

## Permission và phạm vi dữ liệu

- `RETURN_CASE_VIEW`: xem danh sách và chi tiết Case cơ bản.
- `RETURN_EVIDENCE_VIEW`: xem metadata media khách gửi. Không đồng nghĩa quyền xem Packing Video của Epic 10.
- `RETURN_FINANCIAL_VIEW`: xem số đã thu, số có thể hoàn, số Refund được duyệt/hoàn và phương thức.
- `CUSTOMER_SERVICE`: Case + evidence, scope `ASSIGNED_ONLY`; không mặc nhiên xem số liệu Refund.
- `WAREHOUSE_STAFF`: Case cơ bản, scope `ASSIGNED_ONLY`; evidence và tài chính bị che.
- `SALES_MANAGER`: cả ba Permission, scope `ALL`, phù hợp trách nhiệm review/decision trong Epic nhưng màn hiện chỉ đọc.
- `ACCOUNTANT`: Case + tài chính, scope `ALL`; không mặc nhiên xem media khách gửi hoặc phê duyệt Case.
- `SYSTEM_ADMIN` không tự động có quyền Return/Refund.

Backend thật phải áp dụng scope tại truy vấn. Mock chỉ mô phỏng projection quyền và BFF tiếp tục che evidence/tài chính trước khi trả browser.

## Ranh giới nghiệp vụ

- Return Case status, Returned Goods status, Order status, Shipment status, Payment status và Refund status là các vòng đời riêng.
- `APPROVED` không đồng nghĩa đã nhận hàng hoặc đã hoàn tiền; `RETURNED` không đồng nghĩa Refund thành công.
- Hàng trả về chưa tự trở thành tồn bán được; kết quả kiểm tra của Warehouse phải đi qua Epic 09.
- Bằng chứng khách gửi chỉ hiển thị metadata; binary thuộc Media/Object Storage và bản gốc phải được bảo toàn.
- Packing Video là nguồn khác của Epic 10 và không được đưa vào MVP này (`US-RET-07` thuộc giai đoạn 2).
- Cancellation Case chỉ dùng cho yêu cầu cần duyệt/xử lý hậu quả. Hủy tự phục vụ đơn giản vẫn thuộc Epic 08.

## Mutation tạm gác

Chưa triển khai yêu cầu bổ sung, chuyển review, duyệt/từ chối, nhận/kiểm hàng hoặc thực hiện Refund vì cần chốt:

1. Ma trận điều kiện theo Order, thời hạn, SKU, lý do, kênh, Payment và Shipment.
2. State machine riêng cho Cancellation Case, Return Case, Returned Goods, Exchange và Refund.
3. MVP có đổi hàng hay chỉ trả/hoàn; Order thay thế và chênh lệch giá nếu có.
4. Quy trình Shipment hoàn, chi phí và trường hợp không cần gửi hàng lại.
5. Tiêu chí Warehouse phân loại hàng và thời điểm Epic 09 cập nhật tồn.
6. Thời điểm được Refund và cách phân bổ giá, phí, coupon, loyalty, thuế.
7. Phân tách người duyệt/người thực hiện, ngưỡng hai cấp và quyền mở lại Case.
8. Media policy, retention/legal hold, SLA và dữ liệu được xem theo từng actor.

## Chạy local

```powershell
npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
$env:ADMIN_MOCK_ROLE="CUSTOMER_SERVICE"
npm run dev
```

Mở `http://localhost:3001/returns`. Dùng `WAREHOUSE_STAFF` để kiểm tra evidence/tài chính bị che, `ACCOUNTANT` để kiểm tra chỉ tài chính được mở và `SYSTEM_ADMIN` để kiểm tra từ chối truy cập.
