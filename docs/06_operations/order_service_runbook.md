# MS-04 — runbook sandbox 3.1

[Implementation](../02_architecture/services/order-service/08_implementation_and_acceptance.md) · [Cách chạy](../../services/order-service/README.md) · [Evidence](../04_testing/order-service/03_implementation_verification.md).

## 1. Health và nghĩa vụ

HTTP `/livez` kiểm process; `/readyz` kiểm khả năng durable DB write/read. Redis/Kafka/dependency down không xóa intent đã commit. Metrics `/metrics` gồm histogram acceptance/completion/payment-finalization và backlog count/oldest-age cho checkout, outbox, deferred inbox, refund UNKNOWN, reconciliation, manual review, SLA overdue.

Threshold sandbox đề xuất: oldest outbox >60s, refund UNKNOWN bất kỳ, checkout/manual review >60s, deferred inbox >5m, reconciliation mới. Chỉ gắn phase/outcome vào labels; tra cứu references trong DB/logs có quyền. Tracing exporter/HA/PITR chưa được kiểm chứng.

## 2. Triage

1. Lấy operation/order/job/refund reference, tránh raw token/PII.
2. Query PostgreSQL authoritative và status resource; kiểm stage, epoch, keys, provider cursor và remote terminal ledger.
3. UNKNOWN không được tự đổi FAILED. Query/retry đúng key; receipt thật phải được giữ cả khi Order cancelled/unmatched.
4. Chỉ Manager/Admin được resume recovery; giữ input/hash/planned ID/keys và ghi audit. Không sửa trạng thái hoặc tăng tồn bằng SQL.
5. Đóng incident khi local và remote nghĩa vụ terminal; nếu MANUAL_REVIEW thì giữ owner/deadline xử lý.

## 3. Các thao tác đã triển khai

| Trường hợp | API/protocol và hành động |
| --- | --- |
| Checkout MANUAL_REVIEW | `POST /internal/recovery/{operation_id}` với internal credential + Manager/Admin bearer; operation recovery yêu cầu ADMIN; restart cùng key/stage |
| Job MANUAL_REVIEW | cùng recovery endpoint với job ID; kiểm scoped Order; query terminal resource trước resume |
| Order hold | `POST /api/v1/admin/orders/{id}/hold`, If-Match, key; active=true |
| Unhold | cùng command active=false; chặn khi FINALIZE/CANCEL chưa terminal hoặc active Case; generation mới cho ready |
| Lost provider callback | statement reconciliation tự chạy; cursor advance sau receipt commit; không xóa cursor để “sửa” số tiền |
| Receipt mismatch/late/unmatched | giữ reconciliation case; Care/Manager phát approval, sau đó trusted `/internal/refunds` |
| Refund UNKNOWN | query `/provider/refunds/{operation_key}`; không tạo key mới; amount vẫn giữ budget |
| Cancellation paid | Customer tạo cancellation request; source Care approval qua `/internal/cancellations/approve`; refund approval độc lập |
| Event DEFERRED | giữ inbox payload/source/version; prerequisite đến rồi worker apply; kiểm resource association/signature trước replay |
| Outbox pending | khôi phục Kafka/topics/network rồi publisher retry stable ID; crash sau ACK có thể duplicate |

Simulator HTTP 18104 và Inventory RPC 19104 có query terminal outcome. Các control `/test/*` chỉ thuộc sandbox, không phải API admin production. Source event có signature và authority/resource/version checks. Replay cùng ID nhưng khác payload bị từ chối.

## 4. Restart và dữ liệu

```powershell
docker compose -f services/order-service/docker-compose.test.yml restart api
```

PostgreSQL và Kafka dùng volumes riêng; DB có migrations version lock. Stop stack bằng `docker compose ... down` giữ volumes. Không dùng `down -v` để giải quyết obligation UNKNOWN hoặc xóa evidence.

Forward migrations bảo toàn financial records/registry; down migration ban đầu chủ động từ chối destructive rollback. Snapshot/audit có immutable triggers. Audit HMAC chain local cần external anchoring để chứng minh privileged tail deletion.

## 5. Giới hạn nghiệm thu

Go tests chứng minh resume/replay/rollback/lease và simulator terminal protocol. Production provider, Identity, FEFO, OTP, MS-18 end-to-end và full PITR restore chưa có evidence. `APP_MODE=production` bị từ chối. Không áp tài khoản/secret sandbox vào môi trường nhận tiền thật.
