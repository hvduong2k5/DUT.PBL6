# Traceability — review readiness và nội dung PR dự kiến

Ngày 2026-10-17. Branch `feat/traceability-service`, base dự kiến `dev`.
Working tree chưa commit/push; chưa tạo PR. Đây là gói backend R1 sẵn sàng review,
không phải xác nhận tích hợp R2 hoặc hoàn tất toàn EPIC19.

## 1. Đối chiếu các mốc triển khai

| Mục | Bằng chứng hiện tại | Giới hạn còn phải ghi nhận |
| --- | --- | --- |
| P0 Thiết kế/contract | Service/database design, OpenAPI full validation, test matrix, typed event schema | Final audit đã đối chiếu; contract peer vẫn chưa chốt |
| P1 Runtime | Go module, config fail-closed, health, Docker UID10001, Compose cô lập, runtime LOGIN, SIGTERM exit0 | Production bị từ chối |
| P2 Persistence | Versioned migration, frozen revisions, pointer FK, CAS, dedup, audit/outbox atomicity, runtime grants | Schema thay đổi cần migration review; không reset tự động |
| P3 Draft | Signed sandbox JWT, explicit source fixture, list/create/get/edit/clone, permission/evidence masking, pagination | Identity/source thật chưa dùng |
| P4 Review | Submit snapshot, independent approve/reject, reason, contributor inheritance, concurrency/rollback | Không có ADMIN bypass hay auto-publish |
| P5 Public | Source recheck/hash/CAS, atomic publish, stable QR/qr_url, nullable public DTO, read-time expiry, suspend/archive | Web status UI và Inventory QR ownership còn gate R2 |
| P6 Reliability | Real broker ack/retry/crash-window/order, lease fencing, expiry worker dedup, metrics/trace | At-least-once; FAILED head cần operator xử lý |
| P7 Review package | Reusable init/tokens/smoke/Test/Race scripts; runbook; 79,0% coverage; full contract validation; CI draft | Final audit/image check hoàn tất; CI GitHub chưa chạy; PR chưa tạo |
| P8 Real peers | Các điểm X đã ghi trong file quyết định | Chưa có adapter/roundtrip để tuyên bố R2 |

Các test chạy thật, lệnh và hạn chế nằm trong [verification report](./01_verification.md).
Chạy lại theo [reviewer runbook](./02_reviewer_runbook.md).

## 2. Các invariant trọng tâm đã có bằng chứng

- Binding Product/SKU/source namespace/Batch và QR không đổi; không tạo root mới
  cho cùng Batch để vượt suspension. Provider fixture xác minh quan hệ, không tự
  chuyển sang fake khi source khác/unavailable.
- Một working revision và một published pointer. Sau submit, content/snapshot/
  contributor không bị sửa tại chỗ; các revision đã lưu không bị xóa.
- Clone giữ contributor của cả source và working bị thu hồi. Local append-only
  audit ghi rõ source_revision/withdrawn_revisions cùng version và reason, không
  chỉ dựa vào timestamp để suy ra lineage.
- Mọi command ghi có CAS và dedup cùng transaction; cùng key/body replay kết quả,
  body khác conflict; authorization và evidence masking được áp dụng lại.
- Approve/reject độc lập với người biên soạn, lý do bắt buộc; approve chưa public.
  Source lookup ngoài transaction, kết quả về sau vẫn phải vượt CAS/freshness check.
- Publish pointer, audit và outbox cùng commit. Audit/outbox lỗi không để hồ sơ
  public nửa chừng. Public chỉ đọc snapshot local, không gọi provider mỗi scan.
- Anonymous trước publish404; sau suspend/archive/eligibility cutoff410, không
  trả dữ liệu cũ như hợp lệ. Manual verification và sandbox luôn có metadata rõ.
- Runtime không dùng migrator, không được DDL/TRUNCATE/DELETE history. Token/key/
  DSN local không nằm trong build context hoặc file đưa vào review.
- Broker ack mới Published; owner lease cũ không Ack được; crash có thể duplicate
  cùng event ID/payload. Worker expiry không tự Suspended root và không ghi vô hạn.
- HTTP/worker/resource cleanup có overall deadline; timeout được báo lỗi. Metrics
  dùng route template, không dùng raw QR/profile ID/token/evidence làm label.

## 3. Đối chiếu cuối R1

- Đã đối chiếu P0..P7/test matrix với code/evidence và kiểm tra image/runtime cuối.
  Kết quả và giới hạn từng nhóm nằm ở [final audit](./04_final_r1_audit.md).
- Ref peer kiểm tra lại không đổi; chưa có bằng chứng để đóng contract P8.
  FR toàn Epic vẫn PARTIAL/DEFERRED ngoài phạm vi Batch MVP.
- Giữ sự khác biệt giữa local checks PASS và GitHub CI NOT_RUN; không tự tạo/push PR.

## 4. Nội dung PR dự kiến

**Title**: `feat(traceability): add batch revision workflow and public QR backend`

**Body**:

MS03 chưa có backend thực hiện hồ sơ truy xuất theo Batch. Thay đổi này bổ sung
Go service với PostgreSQL: tạo/sửa Draft, revision và contributor history, gửi duyệt,
duyệt độc lập, publish có kiểm tra lại nguồn, public QR ổn định và đình chỉ/archive.
Public trả snapshot đã duyệt, che bằng chứng nội bộ và áp dụng hạn chứng nhận/Batch
ngay lúc đọc. Audit/outbox commit nguyên tử; relay có lease/retry và expiry worker
ghi observation idempotent, không tự đình chỉ root.

Phạm vi review là sandbox backend: token Ed25519 và nguồn tổng hợp có nhãn,
PostgreSQL/broker thật. Mode integration/production không được giả lập bằng fixture.
Catalog/Inventory/Identity/Gateway/Web/consumer/Audit integration còn các contract
X-01..X-06/X-09; không thay code peer hoặc tuyên bố peer đã tương thích.

Validation local: `scripts/dev.ps1 -Action Test` PASS (coverage79,0%), `-Action Race`
PASS trên Go1.25 Linux với PostgreSQL/broker; Docker smoke full flow và rerun an toàn
PASS; SIGTERM exit0 trong0,56s; OpenAPI/draft event schema/runtime admin sample PASS.
Workflow CI đã chuẩn bị nhưng chưa có kết quả GitHub. Hướng dẫn tái hiện và giới hạn
ở `docs/04_testing/traceability-service/02_reviewer_runbook.md`.

**Checklist reviewer**:

- [ ] Rà domain/state/permissions và cách thu hồi working revision.
- [ ] Rà migration/constraints/runtime privileges và transaction boundaries.
- [ ] Rà public allowlist/date/nullability/manual/sandbox semantics.
- [ ] Rà lease fencing/retry/FAILED ordering/duplicate delivery và expiry dedup.
- [ ] Chạy quickstart, full tests/race và contract validator theo runbook.
- [ ] Đồng thuận các contract ngoài trước khi bật adapter integration thật.

Đây chỉ là nội dung chuẩn bị trong workspace; chưa có PR URL hoặc remote checks.
