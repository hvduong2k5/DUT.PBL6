# Fulfillment — acceptance và verification plan

Ngày30/10/2026. [Design index](../../02_architecture/services/fulfillment-service/README.md). **Plan, không phải báo cáo runtime đã PASS.** Static report riêng [02_design_verification.md](02_design_verification.md).

## 1. Các mức bằng chứng

| Level | Environment | Chứng minh | Không chứng minh |
| --- | --- | --- | --- |
| D0 static | Markdown/OpenAPI/JSON/proto/SQL | Artifact parse, references/examples/bindings | Runtime concurrency/peer/provider safety |
| D1 domain | Pure domain + deterministic clock | FSM/invariants/canonicalization | Real DB locks/transactions |
| D2 real infrastructure | Isolated PostgreSQL/Kafka/private object store | Constraints/commit/race/replay/storage safety | Actual peer semantics nếu peer fake |
| D3 peer TEST E2E | Pinned Order/Inventory/Fulfillment/Shipping/Identity, fake carrier labeled | Full integration scope and negative paths | Production carrier/money/operations |
| D4 release drill | Deployment security/config/backup restore/operator | Operational readiness within enabled scope | Features còn disabled |

Không chạy destructive fault/restore trên DB/service team đang dùng. Test fixtures dùng UUIDs riêng, TEST bucket/prefix/topic, no real payment/customer PII. Mọi test có cleanup scoped exact resources; giữ evidence metadata, không commit tokens/URLs.

## 2. Test matrix và oracles

| ID | Scenario | Oracle bắt buộc | Level |
| --- | --- | --- | --- |
| FUL-T01 | Hai AcceptTask cùng Order/key đồng thời | Một task/package; same original decision, one accepted revision/outbox | D2 |
| FUL-T02 | Accept vs Cancel khi chưa có row | Một winner; guard khóa absence; loser reject; no orphan task | D2 |
| FUL-T03 | Cancel trước ready/late permit | CANCELLED tombstone persisted; late accept không task | D2/D3 |
| FUL-T04 | Generation thấp sau cancelled/new generation | Old rejected; no reopened tombstone; high-water stable | D2 |
| FUL-T05 | Ever-accepted Order dùng generation/key mới | Existing task hoặc conflict, không task thứ hai | D2 |
| FUL-T06 | Cùng key khác hash/body/expected version | Conflict, no history/version/effect thêm | D1/D2 |
| FUL-T07 | Replay successful accept sau permit expired/hold | Original successful outcome; authentication vẫn checked | D2/D3 |
| FUL-T08 | Query NOT_FOUND/Cancel ACCEPTED | No reversal proof from NOT_FOUND; Cancel REJECTED with task identity | D2/D3 |
| FUL-T09 | Multiple SKU/batch quantities | Coverage exactly requested; no wrong SKU/batch/warehouse | D1/D2 |
| FUL-T10 | Scan retry same operation; set total | Counted total không nhân đôi, version tăng một lần | D2 |
| FUL-T11 | Batch expired/equality/invalid quality | No seal, typed block; no extra stock mutation | D2/D3 |
| FUL-T12 | Live Catalog renamed/FEFO proposal changed | Purchased snapshot/allocation không bị rewrite | D3 |
| FUL-T13 | Checklist missing/FAIL/unknown item | Seal rejects, required set from pinned policy | D1/D2 |
| FUL-T14 | Staff A/B tranh assign hoặc stale seal | One assignment; wrong assignee denied; stale version412 | D2 |
| FUL-T15 | Nhiều active block, resolve một block | Still BLOCKED; no clear unrelated stock/financial source | D2 |
| FUL-T16 | Duplicate seal same/new key | Same package/seal original; no duplicate source/outbox/stock/shipment | D2/D3 |
| FUL-T17 | Seal code normalized/reused across tasks | Unique conflict, old seal registry retained | D2 |
| FUL-T18 | Fake MIME/SHA256/multipart ETag | No VERIFIED until actual proof/scan safe | D2 |
| FUL-T19 | Expired session/missing object/scan timeout | EXPIRED/VERIFYING/REJECTED honest state, no placeholder video | D2 |
| FUL-T20 | Link media from another task/package | Denied in Tx; original association unchanged | D2 |
| FUL-T21 | Reuse upload URL overwrite after verification | Sealed/downloaded bytes same immutable final object/hash | D2 |
| FUL-T22 | Cleanup vs attach/seal/scan race | Required linked verified media never deleted; stale scan ignored | D2 |
| FUL-T23 | Cross warehouse/actor/purpose/Care IDOR/admin role | Forbidden; no content/location leak; no default admin bypass | D2/D3 |
| FUL-T24 | Access URL TTL/revocation/replay | TTL≤900s, expired URL denied; revocation limit documented | D2 |
| FUL-T25 | Audit exporter down at video access/seal | Local audit+intent durable; no unaudited URL; business not wait sink | D2 |
| FUL-T26 | SLA due_at/source revision update | Deterministic priority, previous deadline retained, stale update ignored | D2 |
| FUL-T27 | Queue cursor/filter/warehouse/null deadline | No out-of-scope rows; bounded pages; blocked explicit; no fake SLA | D2 |
| FUL-T28 | Wrong digest/warehouse/reservation/unsupported contract | Fail closed before admission/seal; no legacy fallback | D1/D3 |
| FUL-T29 | Crash after seal resource write before commit | Rollback package/task/history/outbox/registry together | D2 |
| FUL-T30 | COMMIT ACK lost after accept/seal | UNKNOWN caller; same key returns one original effect | D2 |
| FUL-T31 | Kafka send ACK lost/retry/late old publisher | Same event bytes/id/time/key; consumers dedup; no exactly-once assumption | D2 |
| FUL-T32 | Inbox duplicate same ID/different hash/rebalance | Same→dedup; different→quarantine; no offset before durable disposition | D2 |
| FUL-T33 | Order adapter delayed; Shipping reads early | Deferred/retry; Order eventually PACKED and exactly one shipment/package | D3 |
| FUL-T34 | Exact old revision replay after task advanced | Original kind/time/generation/package proof; latest state not substituted | D2/D3 |
| FUL-T35 | Reopen before protocol accepted | FEATURE_DISABLED, no COMPLETED→READY, originals retained | D2 |
| FUL-T36 | Forged JWT/audience/service headers/event source | Denied/quarantined; no PII/secret logs; method ACL enforced | D2/D3 |
| FUL-T37 | DB restore older than accepted task/shipment | Admission/effects gated; reconcile remote original IDs before reopening | D4 |
| FUL-T38 | Worker lease lost/stale result/budget exhaustion | No stale local write; UNKNOWN obligations still queried/escalated | D2 |
| FUL-T39 | Audit row/revision UPDATE/DELETE attempted | Rejected by role/trigger; export/checkpoint integrity checked | D2 |
| FUL-T40 | Clock skew/permit lock wait reaches expiry | DB decision clock rejects expiry, no accepted stale permit | D2/D3 |
| FUL-T41 | Queue/API load with dataset declared | Measured P95/P99 at stated concurrency; bounded DB/worker resources | D2/D4 |
| FUL-T42 | Kafka/S3/peer outage recovery storm | Backpressure/bulkhead/backoff, API health honest, no duplicate effect | D2/D4 |

## 3. Integration sequence để nghiệm thu milestone C

1. Seed Catalog/Inventory bằng supported owner APIs; create legitimate customer/staff identities and warehouse scopes. Pin commits/contracts/config in evidence, no fabricated headers outside isolated negative tests.
2. Place/pay one TEST Order via supported Order flow; verify stock COMMITTED through authority. Inventory legacy paid deduction disabled for new cohort.
3. Consume ready; verify permit/snapshot/real allocation and one ACCEPTED task. Two parallel staff claims produce one assignee.
4. Scan actual allocated batch quantities, complete required checklist, upload actual test video and obtain VERIFIED immutable final object.
5. Seal twice/retry after dropped response; one COMPLETED task, one package/seal, one source revision/outbox identity.
6. Delay Order adapter deliberately, let Shipping retry snapshot; then apply exact accepted/sealed proof and verify Order PACKED + one fake-carrier TEST shipment. Verify address/fee from checkout snapshot, not current Profile.
7. Replay event and source revision, restart workers/consumer group, verify stock unchanged/no extra shipment.
8. Run independent accept-cancel race and invalid batch/media/auth tests; confirm false/no proof cannot reverse stock. Record disabled COD/reopen/returns scope.

## 4. D0 commands

```powershell
python docs/04_testing/fulfillment-service/verify_design.py
```

Core dependencies: existing Python `PyYAML`, `jsonschema`, `openapi-spec-validator`. Optional `grpcio-tools` compiles dependency proposal in temp directory; optional PostgreSQL DSN is never auto-used. SQL runtime validation, if performed, uses isolated disposable PostgreSQL explicitly and report names environment. Default static verifier does not connect external services/mutate DB.

[requirements-design.txt](requirements-design.txt) pin phiên bản tooling đã quan sát; cài trong virtualenv riêng nếu cần tái lập, không thêm vào dependencies service. Compiler/parser checks được đánh dấu SKIP nếu thiếu tool, không ghi PASS giả.

DDL smoke harness là kiểm tra riêng, chỉ chạy với flag rõ ràng và Docker image đã có:

```powershell
python docs/04_testing/fulfillment-service/verify_candidate_ddl.py --docker --report docs/04_testing/fulfillment-service/ddl-verification.json
```

Harness tự tạo PostgreSQL disposable `--network none`, không publish port/mount host volume, chạy candidate DDL và8 constraint checks, rồi dừng đúng container do nó tạo. Không đọc kết nối DB từ env hoặc đụng DB đang dùng. Đây là kiểm schema nhỏ, không thay full D2 race/service tests.

## 5. Evidence template cho implementation PR

```text
Date/client timezone:
Fulfillment commit / peers commit SHAs:
Contract/policy/cohort/config versions:
Environment and isolated fixture resource names:
Test IDs + exact commands:
Observed result and oracle query:
Skipped tests and why:
Remaining gates/known limitations:
Cleanup and recovery evidence locations:
Reviewer/sign-off:
```

Không tick PASS cho toàn matrix chỉ vì D0 hoặc unit xanh. Static sample event does not establish scanner safety, serialized accept/cancel or one-shipment E2E.
