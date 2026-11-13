# Ma trận kiểm thử và bằng chứng Traceability

Trạng thái hiện hành: guard R1 đã có bằng chứng theo
[final audit và symbol mapping](../../../04_testing/traceability-service/04_final_r1_audit.md).
Bảng dưới giữ định nghĩa scenario; các checkpoint phía sau là lịch sử triển khai,
không thay trạng thái cuối. PEER-01/02 NOT_RUN. FR toàn Epic vẫn PARTIAL nếu chỉ
hoàn thành Batch MVP hoặc manual path.

| ID | Yêu cầu/guard | Kiểm thử cần thực hiện | Tầng | Gate |
| --- | --- | --- | --- | --- |
| CFG-01 | Mode isolation | Invalid/production mode; integration không dùng fixture; DB mode mismatch | Config + PG | P1/P2 |
| CFG-02 | Sandbox exposure | Host loopback; container chỉ explicit isolated mode | Config | P1 |
| OPS-01 | Readiness | DB/schema chưa đủ →503; liveness không bị suy ra từ dependency | HTTP + PG | P1/P7 |
| DB-01 | FR-11..14 identity | UUID Batch uniqueness, QR collision, binding immutable | PG | P2 |
| DB-02 | Revision integrity | One working/Published, deferred pointer, direct SQL update frozen content | PG | P2 |
| DB-03 | Atomicity | Outbox insert failure rollback state/audit/dedup | PG | P2/P5 |
| IDEM-01 | Dedup | Same key/body replay; mismatch409; principal/action isolation; concurrency | PG + HTTP | P2/P3 |
| AUTH-01 | NFR04/05 | Wrong signature/issuer/audience/expired; forged header; missing verifier | Unit + HTTP | P3 |
| AUTH-02 | FR-22 | Permission riêng; evidence redaction; ADMIN không bypass | HTTP | P3/P4 |
| DRAFT-01 | FR-11..14 | Incomplete Draft allowed; malformed content refused; stale version409 | Unit + HTTP | P3 |
| DEP-01 | FR-12 | Product/SKU/Batch mismatch, unavailable503, no fake fallback | Port + HTTP | P3/P4 |
| REVIEW-01 | FR-18..20 | Submit freeze, missing certificate/evidence, unknown source not valid | Unit + PG | P4 |
| REVIEW-02 | FR-19/22 | All contributors cannot approve, clone preserves contributors | Unit + HTTP | P4 |
| REVIEW-03 | FR-20 | Concurrent approve/reject; stale decision; replay idempotent | PG | P4 |
| PUB-01 | FR-01/13/19 | Approve not public; publish pointer; replace revision preserves old snapshot | HTTP + PG | P5 |
| PUB-02 | FR-20 | Source hash change between submit/publish →409; verify outside TX then CAS | Port + PG | P5 |
| QR-01 | FR-01..03/14 | Stable opaque code; never rebind; anonymous404 before first publish | Unit + HTTP | P5 |
| QR-02 | FR-03/21 | Suspend/archive410, cannot bypass via new revision/profile for same Batch | PG + HTTP | P5 |
| CERT-01 | FR-08..10/17 | OCOP scope/evidence; manual != external; no fabricated stars | Unit | P4/P5 |
| CERT-02 | FR-09/17 | Inclusive valid_until at Hue midnight, review_due instant, future issue, revoke | Unit + HTTP | P5 |
| BATCH-01 | FR-03/12 | Expired/restricted410; depleted still traceable; semantics preserved | Unit + HTTP | P5 |
| PRIV-01 | FR-07/22 | Public no internal evidence/PII/decision actor; headers no-store | HTTP | P5 |
| HIST-01 | FR-06/16 | Current Supplier/Producer change cannot alter historical revision | PG + HTTP | P5 |
| EVT-01 | Outbox durability | Broker down leaves pending; same event ID/payload across retries | PG + Kafka | P6 |
| EVT-02 | Delivery semantics | Crash publish-before-mark replays, no false sent, same-profile order/version | PG + Kafka | P6 |
| EXP-01 | FR-09/17/21 | Read guard without worker; worker dedup status transition and no automatic root suspend | Unit + PG | P6 |
| OPS-02 | NFR11 | Bounded shutdown, backlog/error metrics, redacted logs | Runtime | P6/P7 |
| E2E-01 | Batch MVP | Two signed actors: create→submit→approve→publish→QR→new revision→suspend | HTTP + PG | R1 |
| PEER-01 | X-01..04 | Provider proto compile + actual Catalog/Batch/Identity/Gateway roundtrip | Actual peers | R2 |
| PEER-02 | X-05..09 | Web status UI, Catalog/Content projection, Audit ingestion | Actual peers | R2 |

FR-10 external verification, FR-15 genealogy, multi-scope QR and advanced source
management: DEFERRED/PARTIAL, không suy ra pass từ fixture/manual tests. Các test
Kafka/peer chưa chạy phải báo NOT_RUN. Coverage mục tiêu >=75% theo DoD cùng
invariant evidence; không dùng coverage để thay proof transaction/concurrency.

Report sẽ lưu tại `docs/04_testing/traceability-service/01_verification.md`, ghi
toolchain, command, mode/ref, infrastructure và kết quả/skip/failure thực tế.

Checkpoint 2026-10-17: đã có [báo cáo thực tế](../../../04_testing/traceability-service/01_verification.md).
CFG-01/02 có config tests và PG mode check; DB-01/02/03 có bằng chứng schema
một phần từ `migrations/migrate_test.go`. Các mục này vẫn PARTIAL khi chưa có
application/HTTP tương ứng; không đổi toàn bộ hàng thành PASS. IDEM-01 chưa có
kiểm chứng toàn workflow. Transaction runner hiện đã có PG tests cho replay,
body conflict, namespace và 8 concurrent retries (`TestCommands*`); IDEM-01 vẫn
PARTIAL trên toàn workflow; create/edit HTTP đã có authorization/replay/body mismatch
tests (`TestSignedDraftHTTPAndReplayPermissions`, `TestDraftHTTPValidationAndSourceGuards`).

Domain tests đã kiểm tra một phần CERT-01/02, DEP-01 (snapshot validation),
PRIV-01 (redaction copy), DRAFT-01. Draft repository tests bổ sung CAS, contributor,
QR collision và rollback audit/dedup. Chưa suy ra PUB/REVIEW/E2E PASS từ các helper này.

Checkpoint P3: AUTH-01/02 có sandbox verifier + protected Draft HTTP evidence;
CFG-01/OPS-01 bổ sung runtime LOGIN, schema/mode/checksum/privilege validation.
Integration Identity chưa có; AUTH không được suy rộng sang X-04. Chưa có signed
HTTP review/publish ở checkpoint đó; đã bổ sung bằng chứng ở checkpoint P5 bên dưới.

Checkpoint P5: REVIEW-01/02/03, PUB-01/02, QR-01/02, PRIV-01, HIST-01 và E2E-01
có core evidence từ `workflow_test.go`/`publication_test.go` ở PG và HTTP (xem report).
DB-03 bổ sung failure injection thực tế vào INSERT outbox lúc publish.
CERT-02 có PG read-time expiry không worker; Batch/revoked/depleted có domain tests.
Chưa đánh dấu toàn FR PASS: no-expiry/BATCH HTTP cases và full matrix audit còn
thiếu. EVT-01/02 chưa có Kafka delivery; EXP-01 chưa có worker dedup.

Checkpoint P6 thay thế giới hạn ở dòng trên: EVT-01/02 đã có 3 tests broker thật
(`TestKafka*`), lease/backoff/max-attempts/crash-window/order đã có PG evidence.
EXP-01 đã có worker concurrent dedup/rollback, root không tự Suspended; no-expiry
và Batch expiry có HTTP cutoff tests. OPS-02 có metrics/log redaction; signal
shutdown/race vẫn NOT_RUN. Broker là stack test riêng, không phải peer integration.

Checkpoint P7: OPS-02 có deadline/drain/cleanup tests và Docker SIGTERM exit0;
Linux Go1.25 Race PASS với DB/broker thật. E2E-01 bổ sung reusable seed/smoke chạy
trên container runtime LOGIN và rerun terminal verification. CERT-02/BATCH-01 có
HTTP cutoff tests; dependency timeout503/no-write, video allowlist/qr_url và strict
non-nullable decoding đã test. Coverage79,0%. Full OpenAPI + sample/schema checks
PASS. Các giới hạn NOT_RUN của P6 ở trên đã được thay thế bằng evidence P7;
PEER-01/02 và GitHub CI vẫn NOT_RUN. Final requirement audit đã hoàn tất trong
phạm vi R1; bảng kết quả hiện hành ở file final audit liên kết đầu tài liệu.
