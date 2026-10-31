# B2B Customer Data/API Matrix

**Cập nhật:** 2026-10-01  
**Backlog:** `EPIC_18_B2B.md`  
**Mock mở rộng:** `mocks/mockoon/customer_extensions.json`

## Phạm vi đã triển khai

| User Story | Trải nghiệm Customer | BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| `US-B2B-01` | Xem doanh nghiệp đã xác minh, chọn nhiều SKU/số lượng, mục đích và ngày giao; gửi yêu cầu; xem yêu cầu gần đây | `GET /api/b2b/context`, `POST /api/b2b/quote-requests` | `GET /b2b/context`, `POST /b2b/quote-requests` | Đã tích hợp |
| `US-B2B-02` | Chọn logo/Brand Guidelines và gửi metadata cùng yêu cầu tùy biến | cùng request B2B | cùng request B2B | Đã mô phỏng metadata; chưa upload binary/scan |
| `US-B2B-04` | Xem Quote Version được phát hành, tổng tiền/đặt cọc/điều khoản và chấp thuận | `GET /api/b2b/quotes/:id`, `POST /api/b2b/quotes/:id/accept` | cùng path không có `/api` | Đã tích hợp |
| `US-B2B-05` | Hiển thị thông tin pháp lý/hóa đơn của doanh nghiệp đã xác minh và gửi cờ yêu cầu hóa đơn | `GET /api/b2b/context` | `GET /b2b/context` | Đã tích hợp vào request snapshot |

`US-B2B-03` thuộc Sales Manager và không triển khai trong `web-user`. Đây là lát cắt tiếp theo ở Web Admin: hàng đợi yêu cầu, yêu cầu bổ sung, tạo/phát hành Quote Version, từ chối và chuyển đổi Order theo quyền.

## Phân quyền

- B2B dùng chung phiên đăng nhập Customer; không có login riêng.
- Projection phải có `actor = B2B` và capability phù hợp: `B2B_COMPANY_VIEW`, `B2B_QUOTE_CREATE`, `B2B_QUOTE_VIEW`, `B2B_QUOTE_ACCEPT`.
- BFF kiểm tra cả actor và capability trước khi gọi upstream; ẩn menu không thay thế authorization phía server.
- Quyền phải gắn với quan hệ đại diện doanh nghiệp. Upstream thật phải kiểm tra ownership của request/quote và thẩm quyền chấp thuận; không tin `organizationId` hoặc `quoteId` từ browser.
- `CUSTOMER_MOCK_ACTOR` chỉ là công tắc local để Mockoon mô phỏng tài khoản đã được xác minh.

## Ranh giới dữ liệu

- Browser chỉ nhận Quote Version `SENT` hoặc trạng thái công khai; không nhận draft, biên lợi nhuận, giá vốn hay ghi chú nội bộ.
- Chấp thuận lưu đúng `quoteId`, version, actor/đại diện, doanh nghiệp và thời điểm; mock trả Order tham chiếu nhưng không đánh dấu `PAID`.
- Tệp hiện chỉ là metadata tối đa 3 tệp, 25 MiB/tệp. Production cần upload session, malware scan, trạng thái tệp và signed access.
- Danh mục địa chỉ B2B theo cấu trúc hành chính hai cấp Tỉnh/Thành phố → Phường/Xã; lát cắt hiện chỉ cần Tỉnh/Thành phố để lập yêu cầu sơ bộ.

## Việc còn lại

1. Hoàn thiện phần còn lại của Web Admin `ADM-034`: withdraw, chuyển Order, history/message/SLA và tệp thật. Hàng đợi, chi tiết và các hành động chính đã có tại `apps/web-admin`; xem `B2B_ADMIN_DATA_API_MATRIX.md`.
2. API upload/scan tài liệu thật và quản lý phiên bản logo.
3. Luồng `NEEDS_INFO` để Customer bổ sung dữ liệu có lịch sử trao đổi.
4. Định nghĩa đăng ký/xác minh pháp nhân, nhiều đại diện và một tài khoản đại diện nhiều doanh nghiệp.
5. Nối Order B2B thật với Payment, Inventory, Packing, Shipping và Finance.

## Scenario local

| `X-Mock-Scenario` | Kết quả |
| --- | --- |
| `b2b-context-error` | `503`, không tải được hồ sơ doanh nghiệp |
| `b2b-request-validation` | `422`, upstream từ chối MOQ/dữ liệu yêu cầu |
| `b2b-quote-not-found` | `404`, không tiết lộ quote không thuộc quyền hoặc chưa phát hành |
| `b2b-quote-conflict` | `409`, quote hết hạn/bị thay thế/không còn chấp thuận được |
