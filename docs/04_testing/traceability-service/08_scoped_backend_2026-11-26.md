# Traceability scoped backend — checkpoint26/11/2026

> Historical checkpoint0.2.2 before the gRPC amendment. Internal REST/env examples below are superseded by [report09](09_grpc_transport_2026-11-27.md) and [gRPC guide](../../03_api_specs/traceability-workflow-grpc.md). The188PASS result applies to that earlier transport snapshot.

RunID: `traceability-scoped-20261126-local`. Worktree `codex/traceability-order-flow`, nền tài liệu `f8e3baf9`; triển khai hiện là working-tree change để review, chưa có runtime commit/PR/deploy. Đã fetch `origin/dev@9ed59193` ngày26/11: runtime Traceability và BatchSource canonical không đổi so với nền đã dùng. Worktree chính có thay đổi khác của người dùng, không sửa hoặc reset.

## Phạm vi đã triển khai

- API nhân viên `/api/v2/admin/trace/...`: list/get/create/edit/revision/submit/approve/reject/publish/suspend/archive, context theo lô và lookup trước tạo. V2 dùng `Idempotency-Key`, lỗi `code/message/requestId`, no-store; v1 giữ header/envelope/GLOBAL-only và không thêm metadata kho vào root response.
- Permission/scope là tuple: intersection JWT và `/users/me`, hỗ trợ GLOBAL bao phủ WAREHOUSE theo từng permission. Không role bypass hoặc ghép quyền ở khoA với scope khoB. Scoped reads dùng metadata kho bất biến; list lọc SQL trước pagination; cursor HMAC gắn actor và digest quyền hiện hành.
- Create lấy warehouse authoritative; context đọc Inventory → kiểm scope → Catalog exact-SKU trong budget2s. Submit/publish kiểm kho và strict source hash; stock version đổi vẫn409 và cần review lại. Live adapter không tự dùng fixture khi thiếu provider capability.
- Namespace idempotency `v2.<logical_action>` tách v1. Replay kiểm lại resource/current grants. Edit fingerprint từ client request đã normalize trước hydrate reference; preserve reference trong transaction dưới lock/version, reference mới thiếu bị400.
- Resolver `POST /api/v1/internal/traceability/batch-links/resolve`: credential riêng,1–100 bindings, body≤256KiB, budget1.5s, một statement SQL theo snapshot, dedup query và reconstruct đúng thứ tự/duplicates. Chỉ đọc public revision, không gọi peer; các trạng thái AVAILABLE/NOT_AVAILABLE/NO_LONGER_PUBLIC/UNAVAILABLE tách biệt. DB lỗi trả503; malformed binding422/INVALID_BINDING.
- Migration000002 giữ migration000001 nguyên vẹn. Warehouse immutable, legacy null chỉ GLOBAL, guard writer được bật một chiều trước v2 traffic. CLI maintenance backfill đúng một profile bằng nguồn đã xác minh; tăng version/audit, giữ QR, snapshot/hash/public pointer. Runtime LOGIN không có quyền đổi binding/backfill.
- Compose sandbox chuyển tiếp bốn feature env; compose integration dùng `.env` hiện có. Client Go field9 được generate từ candidate đã chốt; canonical provider proto/copies của Inventory chưa bị sửa. Có script codegen/check để tái lập.

Web tạo ảnh QR/in tem, Identity cookie BFF, Gateway routes v2, Inventory/Catalog provider bổ sung và Order/Fulfillment là phần việc owner tương ứng. Backend không sinh ảnh QR hoặc tự publish từ event nhập hàng/packing.

## Môi trường và bằng chứng

Windows, Go1.27.0; PostgreSQL16.15 trong container riêng `pbl6-trace-scoped-20261125`, chỉ loopback15433. Lượt cuối26/11 chạy01:16:34–01:16:51 (UTC+7). Mỗi bài PostgreSQL dùng schema `trace_test_*` riêng; runtime tests tạo LOGIN tạm không sở hữu schema/bảng. Credential nằm trong sandbox bị gitignore, không đưa vào report.

Inventory/Catalog trong signed-HTTP workflow là **simulated providers**. Staff tests dùng JWT ký thật với khóa test và HTTP Identity giả lập; đây không phải xác nhận IAM/Keycloak/provider của nhóm. Assembly test chạy listener TCP loopback thật với source fixture explicit trong sandbox. Không chạy18service, không sửa các stack khác hoặc production.

| Kiểm tra | Kết quả/phạm vi |
| --- | --- |
| `go test -json -count=1 ./...` với PostgreSQL thật | PASS exit0;188test/subtest PASS,3test Kafka SKIP, không FAIL;11package PASS,7package không có test |
| `go vet ./...`, `go build ./...` | PASS exit0 |
| Response HTTP thực ↔ OpenAPI |10response đã PASS: create, publish, list,401, context, empty lookup, resolver available/gone/422 và public v1 |
| Codegen/check | PASS; generated client khớp generator từ candidate field9, provider canonical không đổi |
| OpenAPI lint | PASS exit0 cả6file: scoped admin, workflow integration, admin/customer stable và admin/customer draft-planned; còn warning baseline, không promote stable |
| `git diff --check` | PASS; không sửa migration000001, canonical provider/packing proto hoặc generated frontend snapshots |

Ba ca SKIP: `TestKafkaRelayUnavailableThenAck`, `TestKafkaCrashWindowReplaysSameEvent`, `TestKafkaSameProfileOrder`. Không có broker fixture riêng trong lượt này; không báo các ca này PASS. Log/số đếm ở `go-test.jsonl` và `summary.json` trong artifact RunID. Kiểm mã sinh tự động được chạy sau khi sinh lại, không giữ thay đổi format thủ công trong generated client.

Các response capture chỉ chứa fixture synthetic và body/header cho contract; không chứa JWT, DSN hoặc service credential. Artifact cuối đặt tại `services/traceability-service/.sandbox/traceability-scoped-20261126-local/`, không commit log riêng tư.

## Mapping tiêu chí nghiệm thu

| Test IDs | Bằng chứng tự động mới |
| --- | --- |
| T-SCOPE-01/02/05 | `TestScopedWarehouseReplayAndEvidence`, `TestScopedContextLookupAndCursorIsolation`: khoA/B, replay403 sau đổi grant, cursor theo actor/quyền, SQL filter trước page |
| T-SCOPE-03 | `TestStaffIntersectsGlobalAndWarehouseCoverage` và baseline signed Staff tests: token/me intersection, GLOBAL↔WAREHOUSE, disjoint/department/customer/offline; IAM thật vẫn pending |
| T-SCOPE-04, T-ROLL-01 | `TestScopedProviderRequiresWarehouseAndExactSKU`, `TestLegacyNullWarehouseIsVisibleOnlyToGlobal`, `TestScopedAssemblyRequiresWriterGuardAndServesLoopback`, DB warehouse tests: thiếu kho không tạo, kho đổi409, legacy/null và old writer guard |
| T-DATA-01/02, T-REPLAY-01 | `TestScopedConcurrentCreateKeepsOneBatchQR`, replay/evidence test, `TestLegacyWriterAndV2IdempotencyNamespaces`: một batch/QR, raw fingerprint trước hydrate, namespace v1/v2 |
| T-DATA-03 | `TestWarehouseBackfillPreservesHistoryAndRejectsRuntime`, `TestWarehouseBackfillKeepsLegacyPublicSnapshot`: metadata/audit/version, public hash/QR giữ nguyên, không rollback guard |
| T-DATA-04 | Same evidenceID + omitted/empty reference được preserve; retry sau content đổi vẫn replay; new evidence thiếu reference400 |
| T-WF-01/02/03/04 | `TestScopedWorkflowAndOrderResolver` cùng baseline domain/DB workflow: reviewer độc lập, source version/kho đổi, publish rollback khi outbox lỗi rồi retry cùng key |
| T-CTX-01 | Context kiểm scope trước Catalog; raw provider capability tests; timeout xử lý qua context bounded; peer thật chưa chạy |
| T-RES-01/02 | Workflow/resolver và corrupt-row/DB tests: duplicates, SKU mismatch, draft/gone/corrupt,101items/body cap/credential sai/DB503, không query source khi source offline |
| T-QR-01/02/03 | Chỉ backend public eligibility/QR continuity có regression tests; decode ảnh, browser, camera và bản in **NOT RUN** |

Fixture tests không đóng INV-*, CAT-*, IAM-*, FUL-*, EVT-* hoặc ORD-* integration của service khác. Không gọi đây là full E2E. Event schema/topic và canonical packing proto không thay đổi.

## Chạy lại kiểm thử

Trong thư mục service, cấu hình `TRACEABILITY_TEST_DATABASE_URL` tới PostgreSQL dùng riêng cho test và tên DB `om_traceability_db` để chạy cả smoke initializer baseline. User của DSN test cần quyền tạo schema/LOGIN tạm. Không dùng DB chung/production. `TRACEABILITY_TEST_ARTIFACT_DIR` đặt vào thư mục RunID trong `.sandbox`.

```powershell
go test -count=1 ./...
go vet ./...
go build ./...
python scripts/validate_scoped_runtime.py .sandbox/traceability-scoped-20261126-local
python scripts/generate_scoped_batch_client.py --check
```

Validator HTTP cần các dependency hiện có trong `scripts/spec-requirements.txt`. Codegen pin Python theo `scripts/codegen-requirements.txt` (`grpcio-tools==1.85.4`), `protoc-gen-go v1.36.6` và `protoc-gen-go-grpc v1.5.1`; không sửa tay generated code. Proto candidate được compile riêng, không trộn với canonical cùng package/message. Go tests thiếu DSN sẽ SKIP PostgreSQL; không coi exit0 của lượt đó là bằng chứng database.

## Thứ tự bật và rollback

1. Để hai feature flag false. Dùng migrator credential chạy `go run ./cmd/migrate`; nếu cần, thêm `-runtime-role` với LOGIN runtime đã tạo. Apply000002 trước khi chạy binary mới; readiness runtime yêu cầu đúng hai checksum/mode và quyền DB.
2. Nâng toàn bộ writer lên bản hiểu warehouse. V1 client vẫn gọi được writer mới; warehouse lấy từ provider rồi ghi server-side. Xác minh Inventory field9 + owning warehouse semantics, Catalog exact-SKU/verifier và IAM grant thật theo phiếu owner. Provider thiếu kho trả503 cho scoped source; không gán default.
3. Dùng migrator/table-owner session chạy `go run ./cmd/migrate -require-warehouse-writes`. Switch một chiều chặn writer cũ tạo null, không backfill toàn bộ lịch sử. API v2 khởi động thất bại nếu chưa bật switch.
4. Chỉ backfill legacy có nguồn xác minh: `go run ./cmd/warehouse-backfill -profile-id UUID` với migrator DSN và peer config đúng mode. Mỗi lệnh là một profile, lỗi giữ null; nhận report changed/version nguồn/time. Retry cùng binding không tăng version/audit lần hai. Maintenance actor `00000000-0000-4000-8000-000000000003` là system actor, không phải giả danh reviewer.
5. Runtime session dùng LOGIN hạn quyền, `TRACEABILITY_SCOPED_API_ENABLED=true` và `TRACEABILITY_CURSOR_HMAC_KEY` riêng tối thiểu32bytes. Bật sau peer acceptance; không đưa bearer/secret vào URL, browser hay log. API integration chỉ nhận namespace inventory; sandbox fixture tách DB/mode và không là provider thật.
6. Resolver bật độc lập bằng `TRACEABILITY_ORDER_RESOLVER_ENABLED=true` + `TRACEABILITY_ORDER_RESOLVER_TOKEN` riêng≥32bytes, không trùng token Catalog/Inventory. Giữ route chỉ internal/trusted transport; không thêm vào wildcard Kong public. Order kiểm ownership trước gọi. Deploy resolver trước bật customer feature.
7. Rollback tắt flag/UI/routes mới trước. Giữ000002/warehouse metadata/QR/public snapshots và guard. Binary cũ không tương thích schema/guard không được khôi phục làm writer. Không chạy down migration ngoài schema test, không xoá binding để ép downgrade.

Source fixture sandbox mới dùng Batch UUID và warehouse explicit. Sandbox directory cũ được giữ nguyên bởi initializer; không sửa fixture lịch sử rồi coi là dữ liệu authoritative. Dùng sandbox mới cho v2 hoặc giữ directory cũ cho v1 regression.

## Phần còn chờ tích hợp

Inventory field9/receipt/workbench, Catalog exact-SKU, Identity permission assignments/cookie proxy và Gateway v2 phải có bằng chứng thật. Order/Fulfillment bổ sung exact sealed packed lines và ownership/read adapter; Web triển khai receipt panel, scoped projection, nguồn gốc trong đơn và QR download/print. Các owner dùng [6 phiếu Markdown đã ghim](../../03_api_specs/traceability-owner-handoffs/README.md) và [acceptance chung](../../03_api_specs/traceability-workflow-acceptance.md).

Không bật production để kiểm tra bản local. Review PR vào dev, trưởng nhóm duyệt rồi đưa lên release đúng quy trình nhóm. Contract vẫn0.2.2-draft; stable promotion cần peer/runtime/UI evidence tương ứng.
