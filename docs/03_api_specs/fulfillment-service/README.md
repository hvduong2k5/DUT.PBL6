# Fulfillment contract artifacts — candidate 1.0

Ngày30/10/2026. [Design](../../02_architecture/services/fulfillment-service/README.md) · [REST OpenAPI](../fulfillment-service.candidate.openapi.yaml).

| File | Authority/status |
| --- | --- |
| `../fulfillment-service.candidate.openapi.yaml` | Proposed staff HTTP interface, examples/response/error/auth/versioning |
| `fulfillment-dependencies.candidate.proto` | Proposed Order packing inputs, Inventory allocation/validation, exact package proof; NOT shared deployed wire |
| `examples.json` | Synthetic linked examples for spec/schema/semantic verification; not live evidence |
| `canonical_hash_vectors.json` | Golden canonical UTF-8/hash đề xuất cho items, AcceptTask, CancelTask và REST seal; cần peer interop sign-off |
| `task_accepted.v1.candidate.schema.json` | Đề xuất discovery fact cho Order adapter; topic mới fulfillment.task.events.v1, chưa là shared runtime contract |
| `packing_completed.v2.baseline.schema.json` | Byte-preserving snapshot of dev schema, pinned source in manifest; do not edit as new authority |

Canonical shared Fulfillment baseline trên [dev@7f22226](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/packages/proto/fulfillment/v1/fulfillment.proto). Current checkout có thể chưa có `GetTaskRevision` từ dev. Proto candidate trong folder này dùng namespace riêng để phân biệt proposal; implementation phải promote additive methods/fields vào đúng owning proto sau review và regenerate, không deploy candidate package như stable API.

Examples dùng UUID ổn định, checksum synthetic và `.invalid` URL; không được coi là media/permit/command proof thật. Schema snapshot có required video fields, Shipping consumer còn kiểm nonempty video key và subject/source/traceparent exact; verifier kiểm cả semantic constraints bổ sung đó. Không tạo v2 event media-optional bằng checksum giả.

REST access URL endpoint không cache/replay signed URL theo idempotency: mỗi authorized access có access audit mới; URL issuance là read capability với TTL. Business task/upload mutations dùng Idempotency-Key; task mutations có If-Match. Không có API công khai nhận role, approval, stock status hoặc arbitrary command evidence để bypass backend authority.

Để đóng provider gaps đọc [G01–13](../../02_architecture/services/fulfillment-service/09_decisions_and_contract_gaps.md); để kiểm artifacts chạy [verifier](../../04_testing/fulfillment-service/verify_design.py). Validation tĩnh không đổi shared registry acceptance.
