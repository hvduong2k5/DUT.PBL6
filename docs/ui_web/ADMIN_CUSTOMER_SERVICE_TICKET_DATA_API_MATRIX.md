# Admin Customer Service Ticket — Data/API Matrix

**Cập nhật:** 2026-10-02  
**Nguồn:** `docs/01_requirements/epics/EPIC_16_Customer_Service.md`  
**Cấp phát hành:** Giai đoạn 2 (`US-CS-02~05`), không thuộc MVP v1.0  
**Màn hình:** `/support/tickets` trong `apps/web-admin`

## Phạm vi đã triển khai

| Khả năng | Nguồn Epic | BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Hàng đợi Ticket, trạng thái, ưu tiên và hai mốc SLA | `US-CS-02`, `FR-CS-05`, `FR-CS-15` | `GET /api/admin/support/tickets` | `GET /admin/support/tickets` | Read-only |
| KPI Ticket mở, chưa phân công, SLA cần chú ý và chờ khách | `US-CS-02`, `US-CS-05` | list projection | cùng list route | Đã có |
| Lọc trạng thái, ưu tiên, kênh và SLA | `US-CS-02` | frontend trên projection | cùng list route | Đã có |
| Hội thoại đa kênh và Internal Note tách biệt | `US-CS-03`, `FR-CS-07~09` | `GET /api/admin/support/tickets/:id` | cùng path không có `/api` | Popup read-only |
| Ngữ cảnh Order, Payment, Shipment, Return Case | `US-CS-04`, `FR-CS-11~13` | detail projection | detail route | Chỉ tham chiếu, không sửa nguồn |

## Permission

- `TICKET_QUEUE_VIEW`: xem hàng đợi và chi tiết cơ bản.
- `TICKET_CONVERSATION_VIEW`: xem timeline hội thoại.
- `TICKET_INTERNAL_NOTE_VIEW`: xem Internal Note; thiếu quyền này thì note bị loại tại BFF.
- `TICKET_CONTEXT_VIEW`: xem snapshot giao dịch đã được che dữ liệu nhạy cảm.
- Fixture `CUSTOMER_SERVICE` có bốn quyền với scope `SUPPORT_TICKET: ASSIGNED_ONLY`.
- Fixture `SALES_MANAGER` chỉ có `TICKET_QUEUE_VIEW`, scope `ASSIGNED_ONLY`; hội thoại, Internal Note và context vẫn bị che.
- `SYSTEM_ADMIN` và các Role khác không tự động có quyền Ticket.

Backend thật phải lọc record theo scope. Mock chỉ mô phỏng session; dữ liệu liên hệ trong detail đã được che.

## Ranh giới và phần tạm gác

- Ticket sở hữu hội thoại và SLA, không sở hữu trạng thái/quyết định của Order, Payment, Shipment hoặc Return Case.
- `RESOLVED` không có nghĩa nghiệp vụ liên quan đã hoàn tất.
- Internal Note không được gửi cho khách; trạng thái gửi của message tách khỏi việc lưu nội dung.
- Cảnh báo SLA và Ticket là hai thực thể riêng; đóng cảnh báo không đóng Ticket.
- Chưa triển khai nhận/chuyển Ticket, phản hồi, ghi chú, đóng/mở lại hoặc mutation SLA vì taxonomy, state machine, SLA calendar, routing/escalation và reopen policy chưa chốt.
- Không triển khai Packing Video (`BR-PACK-05` yêu cầu quyền riêng) hoặc AI (`US-AI-02/06` là đề xuất sau Giai đoạn 2).
- `US-CS-06` về phối hợp Batch sắp hết hạn cần một scope riêng từ Epic 09, không ghép vào Ticket queue.

## Chạy local

```powershell
npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
$env:ADMIN_MOCK_ROLE="CUSTOMER_SERVICE"
npm run dev
```

Mở `http://localhost:3001/support/tickets`. Dùng `SYSTEM_ADMIN` để kiểm tra BFF trả `403`.
