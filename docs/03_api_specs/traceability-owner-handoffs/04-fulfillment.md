# PBL6 — Bàn giao Fulfillment: FUL-TRC-01

Version nghiệp vụ/contract: **0.2.2-draft** · Snapshot tài liệu: **`89e15ca19432749b606c6d0a61700e98fdb7fd67`**.

File này là bản `.md` độc lập để gửi qua Zalo. Đề bài, yêu cầu đầu ra, test và contract liên quan được nhúng bên dưới; không cần đọc lịch sử chat hoặc chờ phản hồi ACCEPT để bắt đầu code. Draft không đồng nghĩa chức năng đã chạy hoặc đã deploy.

## Nguồn tài liệu và cách lấy về

- Repository: [https://github.com/hvduong2k5/DUT.PBL6](https://github.com/hvduong2k5/DUT.PBL6). Nhánh tài liệu: `codex/traceability-order-flow`.
- **Đọc theo commit đã ghim**, không tự thay bằng HEAD của dev hoặc nhánh tài liệu đang tiếp tục thay đổi. Đây là nguồn tài liệu; owner giữ nhánh làm việc riêng cho service, không phải checkout đè công việc của mình.
- Có thể đọc toàn bộ đặc tả qua các link dưới. Người truy cập repository riêng tư cần quyền hiện có của nhóm. Contract nhúng trong file này vẫn đọc được khi chưa mở được link.

| Tài liệu liên quan trong repository | Link bất biến |
| --- | --- |
| `docs/03_api_specs/traceability-workflow-README.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-README.md) |
| `docs/03_api_specs/traceability-workflow-peer-handoff.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-peer-handoff.md) |
| `docs/03_api_specs/traceability-workflow-acceptance.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-acceptance.md) |
| `docs/03_api_specs/traceability-workflow-service-spec.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-service-spec.md) |
| `docs/03_api_specs/traceability-workflow-review-2026-11-25.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-review-2026-11-25.md) |
| `docs/03_api_specs/traceability-workflow-candidates/packing.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-candidates/packing.proto) |

Trong repository đã clone, các lệnh đọc sau không checkout/reset hay ghi đè source đang làm:

```powershell
git fetch origin codex/traceability-order-flow
git show 89e15ca19432749b606c6d0a61700e98fdb7fd67:docs/03_api_specs/traceability-workflow-README.md
git show 89e15ca19432749b606c6d0a61700e98fdb7fd67:docs/03_api_specs/traceability-workflow-peer-handoff.md
```

## Bắt đầu thực hiện

**Phân biệt ba lớp:** (1) đầu ra/ngữ nghĩa bắt buộc để nghiệp vụ đúng; (2) wire contract đích trong REST/proto đính kèm; (3) cách implement nội bộ do owner service quyết định. Tên candidate/draft nghĩa chưa được promote stable và kiểm thử runtime, **không phải yêu cầu chờ duyệt để bắt đầu**. Nhận định baseline chỉ phản ánh SHA đã rà; owner tự kiểm toàn bộ code/config hiện có trước khi bổ sung, không cần báo cáo rà soát trước.

Thực hiện theo thứ tự, không có bước xin xác nhận:

1. Đọc section task/OUT của mình, contract đích và test IDs. Xem AGENTS.md/rule của service trước khi sửa.
2. Đối chiếu khả năng hiện có và chọn cách đáp ứng theo bảng dưới; tự thực hiện mapping kỹ thuật và phần sửa trong service mình phụ trách.
3. Sửa contract canonical/proto/codegen/migration mới nếu deliverable yêu cầu, giữ tương thích; không sửa migration cũ hoặc generated artifact sai quy trình.
4. Chạy test theo OUT/test IDs; cập nhật tài liệu đi kèm và nộp kết quả/PR theo quy trình nhóm. Không cần gửi phương án rồi đợi owner nghiệp vụ trả lời. Review PR/release sau thực hiện vẫn giữ quy trình bình thường.

| Tình trạng hiện có | Owner tự làm ngay |
| --- | --- |
| Đã đáp ứng đúng contract đích | Tái sử dụng, kiểm thử và ghi evidence theo SHA vào kết quả cuối; không tạo PR rỗng |
| Có khả năng tương đương nhưng khác wire | Tái sử dụng logic nội bộ, tự bổ sung adapter/serialization hoặc route tương thích trong phần mình phụ trách để **consumer vẫn nhận đúng contract đích**; ghi mapping trong tài liệu/PR, không chờ duyệt mapping trước |
| Còn thiếu | Bổ sung đúng gap, tự chọn cấu trúc nội bộ; giữ đầu ra, semantics, backward compatibility và test đích |

Không yêu cầu thay framework, tên hàm hoặc thêm bảng nếu đã đạt đầu ra. Không tự đổi path/field/status/permission/event để đẩy việc sửa sang consumer khác. Nếu API khác wire đã có sẵn, mặc định là làm lớp tương thích, không đề nghị consumer đổi contract rồi chờ phản hồi.

Chỉ báo **BLOCKED** khi không thể đáp ứng contract mà không vi phạm dữ liệu authoritative, phá contract stable, thiếu dependency/credential không thể cung cấp trong phần phụ trách hoặc cần thay đổi nghiệp vụ ngoài scope. Ghi yêu cầu OUT/test bị chặn, bằng chứng và phần đã làm; tiếp tục phần độc lập. Không coi chưa có phản hồi của owner khác là blocker để dừng code: dùng fixture đúng contract có nhãn để phát triển/test, ghi runtime integration còn chờ khi provider chưa sẵn sàng. Không tự bịa dữ liệu hoặc báo test thật đã PASS bằng mock.

- Wire/case/ID và permission là contract đích, không đổi tên cho giống convention riêng. Không fake provider verification bằng mock, không sửa stable/generated examples hoặc migration cũ.
- HTTP error chuẩn mới `{code,message,requestId}`; mutation mới có Idempotency-Key. Legacy route giữ contract riêng. Internal RPC/call dùng danh tính service và transport tin cậy, không tin X-User-ID/role do browser tự gửi.
- Không kích hoạt/publish QR khi nhận packing hoặc GoodsReceived. Không thêm trừ kho, tạo profile, cấp mã hoặc payment side effect trong read flow.

## Phần việc của Fulfillment

**Mục đích:** trả lô đã đóng gói của đúng dòng đơn từ bản niêm phong bất biến, không từ phân bổ kho hiện tại.

**Đầu ra bắt buộc:** FUL-OUT-01: mỗi cặp dòng đơn/lô có UUID, SKU, mã lô và số lượng thực đóng; FUL-OUT-02: dữ liệu gắn đúng order/package/sealed-version/generation/digest/kho, truy vấn lại cùng version không đổi; FUL-OUT-03: phân biệt thiếu proof lịch sử với đơn không có hàng. Nếu đã có read API/proof đáp ứng đủ, tự tái sử dụng logic và map ra field14 bên dưới để consumer giữ contract đích; không cần hỏi lại trước khi làm adapter. Không bắt làm lại cách lưu hoặc truy vấn nội bộ.

**Provider:** [packing candidate](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-candidates/packing.proto), RPC GetPackageRevision hiện có. Request giữ package_id+sealed_task_version và xác thực Order service. Giữ nguyên fields1–13.

Field14 `repeated PackedBatchLine packed_lines`; mỗi row:

| Field | Kiểu | Nguồn |
| --- | --- | --- |
| order_line_id(1) | string UUID | Task frozen Order inputs |
| sku_code(2) | string | Task allocation SKU |
| batch_id(3) | string UUID | Task allocation Batch UUID |
| batch_code(4) | string | Batch code lưu tại sealed revision |
| packed_quantity(5) | int32 >0 | Counted quantity của lô tại seal |

Group các allocation cùng line+batch thành một row, tổng counted quantity; kiểm counted=allocated trước seal như baseline. Baseline Order/Inventory hiện một SKU/một line, không hỗ trợ cùng batch ở hai line SKU trùng. Giữ grouping theo line+batch để không mất identity; không mở rộng duplicate-SKU manifest trong scope này, đầu vào/proof không hợp lệ phải bị chặn. Tổng row từng line phải bằng frozen quantity. Warehouse nằm trong PackageRevision.warehouse_id. Full proof giữ order/task/package/version/generation/items_digest/parcel/media/time theo baseline; chỉ bổ sung rows, không bỏ kiểm chứng cũ.

Đọc exact sealed snapshot đã lưu. Cùng request trả cùng rows/số lượng/mã lô dù tồn/nhãn hiện tại đổi. Snapshot cũ thực sự có đầy đủ allocation có thể được provider serialize lại deterministic; nếu thiếu không tái dựng bằng live Inventory. Empty packed_lines ở provider cũ là thiếu capability, không phải đơn0sản phẩm.

Không đổi event packing v2, không thêm required fields event cũ, không gọi Traceability để publish/activate. Notify Order giữ event ID/key/version baseline. Consumer Order nâng cấp trước provider field14; Shipping cũ bỏ qua unknown field vẫn chạy.

**Nộp:** bảng đáp ứng FUL-OUT-01…03, contract/version/SHA, request/response đã lọc và test exact snapshot/aggregation. Nếu có thay wire thì thêm codegen/copies+compatibility proof; nếu đã đủ thì không cần PR mới. Nghiệm thu FUL-01…04, EVT-01/02.

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| FUL-01 | One line two batches3+2, hai SKU/dòng riêng; duplicate-SKU invalid | packed_lines đúng pair/quantity; không ép provider hỗ trợ duplicate SKU ngoài current manifest; invalid proof không gắn nhầm lô |
| FUL-02 | Query exact sealed version sau khi live allocation/mã hiển thị đổi | Proof lịch sử bất biến, không dùng latest |
| FUL-03 | Old proof thiếu allocation hoặc wrong version/order | Không fake rows; trả lỗi/capability thiếu theo target; không sửa dữ liệu cũ bằng tồn hiện tại |
| FUL-04 | Contract Fulfillment đích, old Shipping consumer | Field14/codegen giữ wire cũ; logic/API được tái sử dụng phía sau adapter có test. Cả hai không phá Shipping/lifecycle cũ |
| EVT-01 | Duplicate event/restart, notification before peer ready | Inbox dedup, retry read proof, không double quantity/QR publish/stock mutation |
| EVT-02 | Out-of-order source version/generation | Không ghi lùi projection; event đúngorder/version, không bỏ kiểm proof |
| ORD-07 | Base proof hợp lệ nhưng packed_lines thiếu/sai; sau đó thử base proof sai thật | Chỉ extension lỗi → traceUNAVAILABLE, PACKING_SEALED/shipping baseline vẫn tiến; base proof sai vẫn bị từ chối, không nới guard cũ |

Nộp kết quả cuối/PR: OUT đã đáp ứng, SHA, phần tái sử dụng/bổ sung, mapping adapter nếu có, PASS/FAIL/SKIPPED từng test và bằng chứng đã lọc. Chưa có peer chạy thật vẫn code/test bằng fixture đúng contract, ghi `IMPLEMENTED_NOT_INTEGRATED`; không báo full E2E bằng mock. Chỉ báo blocker vượt scope, không gửi phương án rồi chờ duyệt trước sửa code.

Đọc AGENTS.md và rule service trong repository trước khi sửa. Commit mới dùng tiếng Anh, giữ một type/scope; contract thay trước hoặc cùng code, không sửa migration cũ/generated rows sai quy trình. Không tự push dev/release hoặc deploy production; tạo PR theo quy trình nhóm.

## Proto target nhúng: `docs/03_api_specs/traceability-workflow-candidates/packing.proto`

Không compile cạnh proto canonical cùng package/message vì đây là candidate full definition. Khi implement cập nhật canonical/copies/codegen theo contract, giữ field/method cũ; không hand-edit generated code.

```proto
// Implementation target 0.2.2-draft, 2026-11-25. NOT canonical/runtime code.
// Compile this directory separately; do not compile beside the canonical files.
syntax = "proto3";

// Opt-in internal MVP packing ports. Existing candidate wire names are retained
// for compatibility; public consumers must not call these authenticated services.
package omamx.fulfillment.integration.candidate.v1;

option go_package = "github.com/omamx/proto/packing/v1;packingv1";

import "google/protobuf/timestamp.proto";

// Owned by Order. Transport identity Fulfillment; response excludes customer PII.
service OrderPackingInputsCandidate {
  rpc GetPackingSnapshot(GetPackingSnapshotRequest) returns (PackingSnapshot);
}
// Owned by Inventory. These reads never reserve/finalize/reverse or change stock.
service InventoryPackingInputsCandidate {
  rpc GetCommittedPackingAllocation(GetCommittedPackingAllocationRequest) returns (CommittedPackingAllocation);
  rpc ValidatePackingAllocation(ValidatePackingAllocationRequest) returns (PackingValidation);
}
// Owned by Fulfillment. Exact sealed version, not current/latest parcel lookup.
service FulfillmentPackageProofCandidate {
  rpc GetPackageRevision(GetPackageRevisionRequest) returns (PackageRevision);
}

message GetPackingSnapshotRequest {
  string order_id = 1;
  int64 ready_generation = 2;
  string contract_version = 3; // order-3.2
}
message PackingLine {
  string order_line_id = 1;
  string sku_code = 2;
  string product_name = 3;
  string packaging_text = 4;
  int32 quantity = 5;
  bool fragile = 6;
}
message ChecklistDefinition {
  string item_code = 1;
  string definition = 2;
  bool required = 3;
}
message PackingPolicy {
  string policy_version = 1;
  string evidence_mode = 2; // OPTIONAL or VIDEO_REQUIRED; integrated-v2 requires VIDEO_REQUIRED
  repeated ChecklistDefinition checklist = 3;
  int32 maximum_weight_grams = 4;
  int32 maximum_dimension_cm = 5;
  int64 maximum_video_bytes = 6;
  int32 maximum_video_seconds = 7;
}
message PackingSnapshot {
  string schema_version = 1; // packing-inputs.v1
  string contract_version = 2;
  string snapshot_id = 3;
  int64 snapshot_version = 4;
  string order_id = 5;
  int64 ready_generation = 6;
  string items_digest = 7;
  string warehouse_id = 8;
  string reservation_id = 9;
  string channel = 10;
  google.protobuf.Timestamp packing_due_at = 11;
  int64 deadline_revision = 12;
  repeated PackingLine lines = 13;
  PackingPolicy policy = 14;
}
message GetCommittedPackingAllocationRequest {
  string order_id = 1;
  string reservation_id = 2;
  int64 ready_generation = 3;
  string items_digest = 4;
  string warehouse_id = 5;
  string contract_version = 6;
  // Order-owned line IDs joined to committed Inventory SKU/quantity allocations.
  repeated PackingLine lines = 7;
}
message BatchSlot {
  string allocation_id = 1;
  string order_line_id = 2;
  string sku_code = 3;
  string batch_id = 4;
  string batch_code = 5;
  int32 allocated_quantity = 6;
  google.protobuf.Timestamp batch_valid_until = 7;
  int64 eligibility_revision = 8;
  bool currently_eligible = 9;
  string location_reference = 10;
}
message CommittedPackingAllocation {
  string contract_version = 1;
  string order_id = 2;
  string reservation_id = 3;
  string state = 4; // exactly COMMITTED; absence/legacy SUCCESS is insufficient
  int64 allocation_version = 5;
  string items_digest = 6;
  string warehouse_id = 7;
  google.protobuf.Timestamp observed_at = 8;
  repeated BatchSlot slots = 9;
  string original_commit_decision_id = 10;
  int64 ready_generation = 11;
}
message PickedSlot {
  string allocation_id = 1;
  string order_line_id = 2;
  string batch_id = 3;
  int32 counted_quantity = 4;
}
message ValidatePackingAllocationRequest {
  string order_id = 1;
  string task_id = 2;
  string reservation_id = 3;
  int64 allocation_version = 4;
  int64 ready_generation = 5;
  string items_digest = 6;
  string warehouse_id = 7;
  repeated PickedSlot picks = 8;
  string contract_version = 9;
}
message PackingValidation {
  bool allowed = 1;
  string reason_code = 2;
  string validation_id = 3;
  google.protobuf.Timestamp observed_at = 4;
  google.protobuf.Timestamp valid_until = 5;
  string order_id = 6;
  string task_id = 7;
  string reservation_id = 8;
  int64 allocation_version = 9;
  int64 ready_generation = 10;
  string items_digest = 11;
  string warehouse_id = 12;
  string contract_version = 13;
  repeated BatchSlot slots = 14;
  // Checkpoint authority only. A later recall requires a separate stop/quality
  // protocol; valid_until is NOT a distributed lock against quarantine.
}
message GetPackageRevisionRequest {
  string package_id = 1;
  int64 sealed_task_version = 2;
}
message ParcelMeasurements {
  int32 weight_grams = 1;
  int32 length_cm = 2;
  int32 width_cm = 3;
  int32 height_cm = 4;
  bool fragile = 5;
}
message PackageRevision {
  string order_id = 1;
  string task_id = 2;
  string package_id = 3;
  int64 sealed_task_version = 4;
  int64 ready_generation = 5;
  string items_digest = 6;
  string warehouse_id = 7;
  string seal_code = 8;
  ParcelMeasurements parcel = 9;
  google.protobuf.Timestamp packed_at = 10;
  string immutable_video_object_key = 11;
  string video_checksum_sha256 = 12;
  string contract_version = 13;
  // Exact sealed snapshot only. Empty from an older provider is not an empty order.
  repeated PackedBatchLine packed_lines = 14;
}

// One sealed order-line/batch pair; warehouse is PackageRevision.warehouse_id.
message PackedBatchLine {
  string order_line_id = 1;
  string sku_code = 2;
  string batch_id = 3;
  string batch_code = 4;
  int32 packed_quantity = 5;
}
```

## Tính toàn vẹn bản gửi

Nguồn commit: `89e15ca19432749b606c6d0a61700e98fdb7fd67`. SHA-256 tập source: `345b103c088d8ec8a98eca0f8720f57eef83d9aed26f2558049f17008a4dc994`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
