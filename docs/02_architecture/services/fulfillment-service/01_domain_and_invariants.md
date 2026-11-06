# 01 — Domain, ownership và invariants

[Index](README.md). Candidate 1.0. Nguồn: EPIC10, Order3.2 §2–5, wire baseline `dev` trong manifest.

## 1. Ubiquitous language

| Thuật ngữ | Ý nghĩa |
| --- | --- |
| Admission / barrier | Quyết định service đã nhận nghĩa vụ đóng gói cho một generation hay đã đóng cửa nhận |
| Ready observation | Thông báo Order cho biết có thể xin permit; không phải task được phép chạy |
| Generation | Epoch readiness do Order cấp; không phải task version, không tự tăng ở Fulfillment |
| Order guard | Bản ghi khóa theo Order, lưu high-watermark và task từng ACCEPTED |
| Task | Aggregate công việc đóng gói có người, checklist, scans và policy |
| Items digest | SHA-256 SKU/quantity canonical của Order; không bao warehouse |
| Allocation | Phân bổ batch đã COMMITTED do Inventory sở hữu; không phải gợi ý FEFO hiện tại |
| Checklist policy | Tập bước bắt buộc được pin version tại admission |
| Block | Một lý do chặn thao tác local, không phải remote-stop proof |
| Package | Một kiện vật lý có identity, measurements và seal; khác shipment |
| Evidence | Object/version đã verify và liên kết đúng task/package |
| Revision | Snapshot source immutable tại một resource_version, dùng replay/proof |
| UNKNOWN | Kết quả quan sát chưa biết của caller/worker; không phải provider resource state |

## 2. Data ownership

Fulfillment sở hữu admission/guard/registry; task/assignment/checklist/scans; parcel/seal; upload/media/revisions; inbox/outbox/jobs/audit local. Order IDs, SKU names, policy, reservation/batch IDs chỉ là reference/snapshot có provenance. Không có FK qua DB service khác.

| Dữ liệu | Owner | Fulfillment được làm |
| --- | --- | --- |
| Payment, hold eligibility, order version | Order | Query permit; không set PAID/PACKED trực tiếp |
| SKU/packaging snapshot đã mua | Order, bắt nguồn Catalog | Lưu immutable packing subset, không refresh Catalog để thay hàng |
| Stock, FEFO, allocation, quarantine | Inventory | Kiểm allocation và gửi yêu cầu xử lý sai lệch; không điều chỉnh stock |
| Receiver/address/fee/COD | Order | MVP không lưu PII; Shipping query Order trực tiếp |
| Task/checklist/actual parcel/seal | Fulfillment | Mutate theo guards/version và phát facts |
| Shipment/tracking/cancel carrier | Shipping | Đọc theo capability; không ghi shared shipment table |
| Employee status/warehouse entitlements | Identity/Profile authority | Kiểm quyền qua verified context, không đọc HR DB |
| Case/refund approval | Care/Order theo scope | Lưu case reference, không tự approve/refund |

## 3. Aggregate boundaries

`OrderGuard` + `AdmissionDecision(order_id,generation)` cùng local transaction với tạo task. `PackingTask` là root cho assignment, picks, checklist, blocks và parcel drafts; một transaction mutate một Order/task. `EvidenceFile` có lifecycle upload riêng; attach/seal cần lock media rows trong transaction task. `TaskRevision` immutable, version theo task; admission resource_version riêng theo `(order,generation)`.

`Package` có `package_id` riêng ngay khi draft được tạo. MVP một row/task; khi mở multi-package phải migration bỏ unique task_id có điều kiện, bổ sung line allocation/package và shipping authority từng kiện. Không dùng `task_id=package_id` để tiết kiệm bảng.

## 4. Invariants và enforcement

| ID | Invariant | Enforcement chính |
| --- | --- | --- |
| FUL-I01 | Không accept chỉ từ paid/ready hoặc boolean caller | Online permit + trusted service ACL |
| FUL-I02 | Một Order tối đa một task từng ACCEPTED | Order guard lock + unique task.order_id, không xóa ledger |
| FUL-I03 | Cancel/accept serialize cả khi absent | Insert guard ON CONFLICT; lock guard trước generation |
| FUL-I04 | CANCELLED generation không reopen; old generation không accept | Tombstones + high-watermark + permit generation |
| FUL-I05 | ACCEPTED không trả CANCELLED bằng CancelTask | Immutable admission decision; REJECTED original command evidence |
| FUL-I06 | Same scope/key/hash có original outcome; đổi payload conflict | Durable command registry, caller/action/order/key |
| FUL-I07 | Digest và warehouse khớp Order/Inventory | Canonical recompute, allocation bindings, no cross warehouse |
| FUL-I08 | Quantity nguyên dương, không scan vượt/phân bổ sai SKU | Per-line/batch quantities, transaction sums/checks, overflow guard |
| FUL-I09 | Không dùng batch expired/quarantined ở checkpoint seal | Inventory validation + local DB deadline check; recall sau checkpoint xử lý riêng |
| FUL-I10 | Chỉ assignee đủ quyền mới sửa checklist/seal | JWT/service context + warehouse scopes + assignment under lock |
| FUL-I11 | Required checklist/policy bất biến theo version | Full required item set pin; boolean client không đủ |
| FUL-I12 | Required media VERIFIED đúng task/package/object version | Media row locks, storage proof, foreign bindings |
| FUL-I13 | Seal một lần, globally unique normalized seal | Unique seal + task lock + command registry |
| FUL-I14 | Seal + package + exact revision + audit + outbox atomic | Một PostgreSQL transaction |
| FUL-I15 | Packing không trừ kho lần nữa, không tự tạo shipment | ACL/ports không expose write stock/carrier |
| FUL-I16 | Task COMPLETED không có nghĩa Order PACKED/shipment created | Independent integration projections, authoritative peer query |
| FUL-I17 | Exact proof không lấy latest thay lịch sử | Append-only revision, lookup exact task/version/kind |
| FUL-I18 | Media không public, xem theo purpose/warehouse/case | Private bucket + ABAC + short URL + audit |
| FUL-I19 | UNKNOWN không giải phóng identities/tombstones | Reconcile original key, no destructive cleanup |
| FUL-I20 | Recovery/extension không đổi cohort của nghĩa vụ đang chạy | Pinned protocol/policy/cohort; rollback chỉ đóng admission mới |

FUL-I09 là kiểm tra tại thời điểm cấp/áp validation, không phải tuyên bố distributed atomicity giữa quarantine và seal. Nếu nghiệp vụ đòi cấm tuyệt đối mọi seal sau quarantine tức thời, cần protocol stop/fencing được Inventory và Fulfillment thống nhất trước bật; G08 theo dõi giới hạn này.

## 5. Snapshot và value objects

Packing snapshot v1 gồm `order_id, snapshot_version, ready_generation, items_digest, warehouse_id, policy_version, packing_due_at, channel, lines[]`. Mỗi line có `order_line_id, sku_code, product_name, packaging_text, quantity, fragile`. Không có address/phone/price trừ khi sau này có use case được duyệt. Policy gồm required checklist definitions, evidence mode, permitted media types/limits, source deadline và version. Clone JSON typed, validate và hash để phát hiện đổi hồi tố.

Canonical digest dùng đúng Order3.2: consolidate SKU, sum có kiểm overflow int32; sort ASCII; UTF-8 JSON compact `[{"quantity":2,"sku_code":"MX-GION-500G"}]`; SHA-256 lowercase. Cấm float/locale/thay Unicode normalization SKU. Warehouse kiểm riêng. Hai line Order cùng SKU vẫn giữ line IDs để pick; digest tính tổng SKU, không gộp mất traceability line.

`ParcelMeasurements`: integer `weight_grams,length_cm,width_cm,height_cm >0`, `fragile` boolean, upper limits từ policy và carrier profile được snapshot. Không tự làm tròn số thực từ UI; UI chuyển đơn vị rõ trước submit. Physical measured parcel có thể khác estimate checkout; vượt policy/fee tolerance → BLOCKED, Order/Shipping chốt điều chỉnh; không âm thầm sửa checkout fee.

`SealCode`: trim ngoài, uppercase ASCII, định dạng đề xuất `[A-Z0-9][A-Z0-9-]{5,79}`, unique toàn service kể cả package cũ; không tái dùng seal đã VOID. Cấm lấy seal làm authorization secret. `Version`: bigint tăng đúng một trên mỗi task mutation; admission version riêng; không so số version giữa các service.

## 6. Phạm vi mở rộng có chủ ý

Multi-warehouse không đơn thuần thêm warehouse_id vào query: cần Order fulfillment groups, inventory allocation per group, partial handoff và returns. Multi-package cần line-to-package quantity coverage, per-package seal, shipping snapshot từng kiện và total completion guards. Marketplace/B2B/POS đi qua Order boundary chung; source channel chỉ chọn policy đã pin, không tạo logic bypass payment/stock.

Rework sau seal là workflow mới có proof stop/revoke ở Order/Shipping, lineage package/seal/evidence mới và giữ revision cũ. Trong MVP endpoint reopen trả feature-disabled; không biến COMPLETED về READY rồi phát lại package cũ. Checklist thêm mới áp dụng task mới, task đang chạy giữ policy cũ hoặc migration có manager approval/audit.
