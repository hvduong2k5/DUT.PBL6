# MS-14 · Quyết định bàn giao lát 1 và triển khai tiếp theo

## 1. Bàn giao backend lát 1

Mở PR Marketing vào dev với workflow Campaign Draft, revision, Content Brief, Product evidence, gửi duyệt, quyết định duyệt và audit. Phạm vi gồm Admin API draft, PostgreSQL, route Kong, Identity BFF và danh mục quyền IAM V14. Khi review, đối chiếu permission theo DEPARTMENT, tính bất biến của snapshot, transaction audit và retry cùng Idempotency-Key.

Smoke test trên stack Docker cô lập đã đi qua Keycloak, Kong, Identity, Catalog, Marketing và PostgreSQL. Hai staff đăng nhập qua OAuth bằng HTTP client có cookie jar. Submit lưu đúng Catalog revision, reviewer dùng phiên riêng để duyệt và audit ghi đúng actor. Kiểm thử Chrome/Edge thuộc nghiệm thu Web Admin. Trước khi mở pilot dùng mã DEPARTMENT và role assignment thật của môi trường.

## 2. Web Admin cho lát 1

- Nối danh sách Campaign, form Draft và màn hình review vào route phiên nhân viên /api/v1/admin/marketing-api/campaigns qua Identity BFF.
- Danh sách hiển thị trạng thái, department và phân trang. Form tải version hiện hành, giữ thay đổi người dùng khi gặp 409 và cho đối chiếu với bản mới. Khi thử lại create hoặc action sau lỗi kết nối, giữ nguyên body và Idempotency-Key.
- Màn hình duyệt hiển thị snapshot đã gửi, ngân sách, Content Brief theo kênh và Product evidence. Chỉ hiển thị thao tác phù hợp quyền và trạng thái. Backend thực thi scope, thẩm quyền và quy tắc cấm tự duyệt.
- Nghiệm thu trên trình duyệt thật cho đăng nhập, phiên hết hạn, sai phòng ban, tự duyệt, dữ liệu cũ, Product không đủ điều kiện và một lượt tạo → gửi → duyệt với hai phiên nhân viên riêng. Dùng contract draft và state matrix làm đầu vào handoff.

## 3. Lát 2 · Lập lịch và triển khai thủ công

- Cho phép lên lịch revision APPROVED sau một lệnh riêng. Dùng thời điểm UTC kèm timezone của kế hoạch. Kiểm hiệu lực approval, lịch và readiness của từng kênh.
- Tạo execution theo revision, kênh và lịch. Pilot ghi nhận thao tác thủ công với actor, thời điểm, trạng thái và tham chiếu bằng chứng. Lưu lịch sử pause/cancel và kết quả từng kênh.
- Chốt state machine, permission, ràng buộc chống tạo trùng và Admin API draft trước khi code. Kiểm thử lên lịch lặp, lịch ngoài kỳ, asset đổi version, dừng một phần và bảo toàn lịch sử.
- Tích hợp gửi tự động sau khi chốt kênh pilot, chủ sở hữu dịch vụ gửi, consent/suppression, idempotency và cơ chế đối soát khi timeout. Các adapter tự động và KPI/Finance nằm ngoài PR lát 1.

[Kiến trúc MS-14](README.md) · [Thiết kế lát 1 và kết quả test](01_planning_and_approval.md) · [Admin OpenAPI draft](../../../03_api_specs/admin-draft-planned.openapi.yaml) · [Admin state matrix](../../../03_api_specs/admin-api-state-matrix.md)

## 4. Checklist phần còn thiếu sau PR lát 1

### Để nghiệm thu pilot lập kế hoạch và duyệt

- [ ] Nối Web Admin cho danh sách, form Draft và màn hình review. Kiểm Chrome/Edge với phiên thật, quyền sai phòng ban, phiên hết hạn và xung đột version. Smoke test hiện có dùng HTTP client với cookie jar.
- [ ] Cấu hình mã DEPARTMENT, role assignments và CATALOG_VIEW cho staff phù hợp trên môi trường pilot. Triển khai Marketing cùng route Kong và migration IAM V14, rồi chạy lại luồng có Product trên môi trường đó.
- [ ] Hoàn tất ví dụ và contract tests cho các trạng thái, sau đó xác nhận contract Admin trước khi chuyển các operation đủ điều kiện sang stable.

### Các phần còn thiếu để bao phủ EPIC 21

- [ ] **Audience và tài sản (US-MKTG-01–02).** Liên kết segment theo ID/version, kiểm quyền sử dụng và consent. Tích hợp Article/Publication, Promotion và Media/fileId có version và readiness. Lát 1 đã có Content Brief và bằng chứng Product.
- [ ] **Lập lịch và execution (US-MKTG-03–04, FR-MKTG-12/26).** Lịch, execution từng kênh, readiness trước khi chạy, pause/cancel/revoke, bằng chứng chạy thủ công. Sau đó bổ sung worker, adapter gửi, retry/idempotency và đối soát kết quả theo kênh.
- [ ] **KPI và đồng bộ (US-MKTG-05–06).** Từ điển KPI, mapping campaign ngoài, sync theo kênh, quarantine, correction/backfill, coverage và dashboard target–actual.
- [ ] **Attribution và Finance (US-MKTG-07).** Chốt bằng chứng nối Order với Campaign và contract Finance, tính doanh thu quy gán, chi phí đối soát, refund và báo cáo có công thức, phiên bản và scope tài chính.
- [ ] **Vận hành.** Bổ sung metrics, cảnh báo và runbook cho job/kênh khi triển khai worker. Nối audit tập trung sau khi xác nhận consumer và contract. Lát 1 đã ghi audit nội bộ trong transaction.

Ưu tiên nghiệm thu pilot và Web Admin, sau đó triển khai lập lịch và execution thủ công. Các tích hợp kênh, KPI và Finance theo thứ tự phụ thuộc nguồn.
