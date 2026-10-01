# B2B Admin Data/API Matrix

**Cập nhật:** 2026-10-02  
**Backlog:** `EPIC_18_B2B.md`  
**Màn hình:** `ADM-034`  
**Ứng dụng:** `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

**Nguồn phạm vi:** `US-B2B-03`, đặc biệt các scenario “Hai nhân viên cập nhật đồng thời”, “Sửa Quote đã phát hành” và `FR-B2B-07~12`. Các nội dung SLA escalation/message thread ở Mục 8 của Epic vẫn là điểm cần chốt nên không được tự mở rộng từ UI specification.

## Phạm vi đã triển khai

| Khả năng | UI/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- |
| Phiên nhân viên và projection quyền | `GET /api/admin/session` | `GET /admin/session` | Đã tích hợp |
| Hàng đợi yêu cầu B2B | `GET /api/admin/b2b/quote-requests` | `GET /admin/b2b/quote-requests` | Đã tích hợp |
| Chi tiết yêu cầu | `GET /api/admin/b2b/quote-requests/:requestId` | cùng path không có `/api` | Đã tích hợp trong modal |
| Lịch sử Quote Version và immutable snapshot | `GET /api/admin/b2b/quote-requests/:requestId/versions` | cùng path không có `/api` | Đã tích hợp trong modal |
| Phân công, yêu cầu bổ sung, lập nháp, phát hành, thu hồi, từ chối, chuyển Order | `POST /api/admin/b2b/quote-requests/:requestId/actions` | cùng path không có `/api` | Đã tích hợp trong modal |
| Chống ghi đè đồng thời | Mọi mutation gửi `expectedRevision`; BFF trả `409 B2B_CONCURRENCY_CONFLICT` khi revision cũ | scenario `admin-b2b-concurrency-conflict` trên action route | Đã tích hợp, buộc tải lại/đối chiếu |

Màn `/b2b/quotes` có status tabs, tìm theo mã/doanh nghiệp/mã số thuế, lọc owner/SLA, bảng tóm tắt và menu `•••`. Hành động được lọc theo permission của phiên và trạng thái yêu cầu.

## Role, permission và scope

- `SALES_MANAGER`: đầy đủ quyền xem, phân công, yêu cầu bổ sung, lập/phát hành/thu hồi/từ chối báo giá và chuyển Order.
- `CUSTOMER_SERVICE`: xem, yêu cầu bổ sung và xem tệp; không có quyền lập hoặc phát hành báo giá.
- `B2B_VIEWER`: chỉ xem.
- Đây là fixture phục vụ UI, không phải quy tắc role cố định của production. Production trả effective permissions sau khi tổng hợp Role và assignment thực tế.
- BFF kiểm tra `B2B_REQUEST_VIEW` cho list/detail và permission riêng cho từng mutation. Upstream production vẫn phải kiểm tra lại permission, scope và transition nghiệp vụ.

## Quy tắc trạng thái trên UI

| Trạng thái | Hành động có thể xuất hiện nếu có permission |
| --- | --- |
| `REQUESTED` | xem, phân công, yêu cầu bổ sung, lập nháp, từ chối |
| `NEEDS_INFO` | xem, phân công, lập nháp, từ chối |
| `DRAFT` | xem, phân công, yêu cầu bổ sung, sửa/lập nháp, phát hành, từ chối |
| `SENT` | xem, phân công, lập phiên bản mới, thu hồi |
| `ACCEPTED` | xem, chuyển thành Order |
| `REJECTED`, `CONVERTED` | xem |
| `EXPIRED`, `WITHDRAWN` | xem, lập phiên bản nháp mới |

## Ranh giới mock hiện tại

- Mutation trả acknowledgement và UI cập nhật projection trong phiên; Mockoon chưa lưu state bền qua lần tải lại.
- Quote Builder đang mô phỏng chiết khấu, phí tùy biến, vận chuyển, VAT, ngày hết hạn và điều khoản. Tổng tiền phía UI chỉ là preview; backend thật phải tính và xác nhận lại.
- Tệp mới chỉ có metadata/scan status; chưa có upload binary, signed URL hoặc malware scan thật.
- Thu hồi và conversion sang Order đã có UI/BFF/mock acknowledgement; upstream thật vẫn phải kiểm tra transition, availability, chống trùng và tạo Order thực tế.
- Version history đã hiển thị preview Customer-compatible và giữ snapshot chỉ đọc cho phiên bản đã phát hành; mock dùng một bộ fixture V1–V3 và BFF giới hạn theo version hiện tại của dòng được chọn.
- Concurrency dùng optimistic revision: UI gửi revision đã tải, mutation thành công tăng revision; conflict khóa nút gửi lại cho đến khi tải dữ liệu mới.
- Không triển khai message thread hai chiều hoặc SLA escalation khi Epic chưa chốt ranh giới với Ticket/Notification và chính sách SLA.
- Audit explorer thuộc phạm vi Epic/Audit tương ứng, chưa được suy diễn thành chức năng của ADM-034.
- Mock fixture dùng `ADMIN_MOCK_ROLE`; production không được tin header role do browser gửi.

## Chạy local

```powershell
npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
npm run dev
```

Mặc định Web Admin chạy ở `http://localhost:3001`. Chọn fixture bằng `ADMIN_MOCK_ROLE=SALES_MANAGER`, `CUSTOMER_SERVICE` hoặc `VIEW_ONLY` trong `.env.local`.

## Việc còn lại của ADM-034

1. Lưu state/action history thật và đồng bộ status counts sau mutation.
2. Kết nối `WITHDRAW` và conversion với backend Order/Inventory thật, gồm kiểm tra transition và idempotency.
3. Nối Quote Version history/snapshot hiện có với persistence và concurrency version thật.
4. Chờ Product Owner chốt ranh giới message/Ticket/Notification và chính sách SLA trước khi mở rộng UI.
5. Upload/download tệp có scan và quyền truy cập theo `US-B2B-02`; Audit UI chỉ làm khi Epic sở hữu màn được đưa vào scope.
