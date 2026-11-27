# Review chéo tài liệu, code và rủi ro truy xuất theo lô

> Historical code/design review0.2.2 dated25/11. Business/security findings remain relevant. Internal REST target examples are superseded by [gRPC0.3.0](traceability-workflow-grpc.md); use current owner handoffs.


Ngày25/11/2026 · So sánh `origin/dev@7a80e3f4bc23a1824db91615bfddbd09d8cd3c9b`.

## Kết luận

Bộ0.2.1 còn khoảng trống thực tế, chưa nên giao như thể mọi đầu vào đã có. Bản0.2.2 sửa các điểm dưới đây trong tài liệu/contract. Không sửa runtime code trong lượt review; những khác biệt còn lại là deliverable triển khai có owner/test, không phải chức năng đã chạy. Chế độ triển khai trực tiếp, không hỏi ACCEPT trước code, giữ nguyên.

Worktree chưa merge/rebase lên dev mới để bảo toàn tài liệu đang sửa. Đã kiểm diff các source Traceability, Inventory, Fulfillment, Catalog và Identity proxy giữa base6f54a4ab và7a80e3f4: không đổi. Order có guest-session từ1564a7bc; phần mới sau mốc đó là validation địa chỉ/simulator, không thay proof/batch contract. Vì vậy các bằng chứng source dưới đây khớp dev mới ở phạm vi được review, không suy rộng sang service/nhánh khác.

## Phát hiện và cách xử lý

| ID / mức độ | Bằng chứng code | Vấn đề của bản trước | Sửa trong target0.2.2 / owner |
| --- | --- | --- | --- |
| R01 / cao | Inventory chỉ có [receipt HTTP](../../services/inventory-service/internal/presentation/httpapi/receipt.go); [Web handler](../../apps/web-user/src/features/employee/inventory/server/handler.ts) gọi SKU detail/workbench, nhưng chưa thấy provider các route read này trong Inventory | Form phụ thuộc Inventory SKU detail để lấy productId mà chưa có bằng chứng backend | Traceability có batch-context endpoint; Inventory vẫn phải bảo đảm workbench read hiện có trong admin contract để UI nhập kho chạy thật. Không coi Mockoon context là provider live |
| R02 / cao | [Catalog SKU public](../../services/catalog-service/src/main/java/com/hvduong/catalog/api/CatalogHttpController.java) trả variant/giá, không product_id; [internal verifier](../../services/catalog-service/src/main/java/com/hvduong/catalog/api/TraceabilitySourceController.java) lại yêu cầu productId đã biết | Không bootstrap được Product từ SKU, đặc biệt SKU ẩn/không giá | Thêm internal exact-SKU binding độc lập giá/saleability; Traceability gọi sau kiểm scope kho; có task Catalog, không còn kết luận Catalog chắc chắn không phải sửa |
| R03 / cao | [Inventory Web upstream](../../apps/web-user/src/features/employee/inventory/server/handler.ts) chuyển cookie; backend receipt/Traceability cần bearer. [Identity filter](../../services/identity-service/src/main/java/com/hvduong/identity/security/BffSessionAuthFilter.java) yêu cầu Origin khi cookie mutation; [SecurityConfig](../../services/identity-service/src/main/java/com/hvduong/identity/security/SecurityConfig.java) employee matcher chỉ v1 | Chỉ cấp permission không làm cookie flow chạy. Thiếu proxy, forward Origin hoặc guard v2 sẽ401/403 hoặc chỉ kiểm authenticated | Contract Identity BFF riêng cho Traceability và workbench/receipt. IAM owner thêm transport khi dùng cookie mode, guard ACTIVE EMPLOYEE/v2, giữ CSRF; direct OIDC không cần cookie proxy |
| R04 / cao | [Employee projection](../../apps/web-user/src/features/employee/auth/types.ts) whitelist permission, scope chỉ ALL/ASSIGNED_ONLY, có suy GLOBAL từ role | Quyền lowercase Traceability bị loại; scopeId kho mất hoặc UI suy quyền quá rộng | Web thêm projection Traceability từ permissionGrants giữ từng permission/scopeId, không tái sử dụng role-derived ALL làm authorization. Backend vẫn kiểm đủ quyền |
| R05 / cao | [Order customer projection](../../services/order-service/internal/app/customer_views.go) không gán ItemID từ canonical LineID; [UI](../../apps/web-user/src/lib/orders/mvp.ts) dùng orderId-index khi thiếu ID | Subresource có canonical line ID nhưng UI không biết gắn vào dòng detail nào | Thêm sku_code vào line-batch result. Vì current manifest unique SKU, Web dùng typed SKU exact làm bridge có kiểm chứng, giữ canonical ID từ endpoint; không parse label/index hoặc đoán Product |
| R06 / vừa | [HashSelection](../../services/order-service/internal/domain/core/identity.go) và [Inventory packing](../../services/inventory-service/internal/presentation/grpc/packing.go) chặn duplicate SKU | Test bắt cùng batch ở hai line của một đơn sẽ buộc mở rộng ngoài phạm vi | Nghiệm thu current flow một SKU/một line, một line nhiều lô. Duplicate-SKU input là negative test; thuật toán grouping không được mất line, nhưng không tuyên bố runtime hỗ trợ duplicated-SKU lines |
| R07 / cao | [Command registry](../../services/traceability-service/internal/infrastructure/postgres/commands.go) dùng actor/action/key, lưu response JSON; [Draft.Edit](../../services/traceability-service/internal/application/drafts.go) validate trước repository | V1/v2 dùng chung action có thể replay body sai version; hydrate reference trước fingerprint hoặc từ content mới nhất làm retry/key sai | Action namespace v2 riêng, fingerprint raw normalized request trước hydrate; reauthorize resource trước replay; hydrate theo evidence ID dưới row lock/expected version. V1 giữ namespace/wire cũ |
| R08 / cao | [Batch table](../../services/inventory-service/migrations/000001_init_inventory_schema.up.sql) FK SKU, kho ở inventory_items; [batch revision trigger](../../services/inventory-service/migrations/000003_packing_reads.up.sql) tăng mọi UPDATE | JOIN kho hiện tại không tự chứng minh ownership bất biến; version tăng cả khi biến động tồn | Provider phải bảo đảm ownership trong đời UUID hoặc fail closed khi không chứng minh được; không đoán kho cho legacy. Giữ strict source-hash baseline: stock update có thể gây publish conflict, không tự bỏ version để né review |
| R09 / vừa | Order ≤100line nhưng mỗi line có thể nhiều batch; resolver chỉ≤100bindings/call | Chunk tuần tự không giới hạn hoặc bỏ bớt rows làm timeout/response lớn, UI hiểu thiếu lô | Budget5s outer,≤1.5s mỗi call, bounded concurrency≤2; phần chưa resolve báo UNAVAILABLE. Cap1000line-batch rows, vượt cap/root bound báo UNAVAILABLE, không truncate giả READY |
| R10 / cao | V1 create hiện không lưu warehouse; backfill theo source và new v2 cần metadata kho | Chạy writer binary cũ cùng scoped writer tạo thêm row null hoặc bỏ kiểm mới | Nâng mọi writer v1/v2 sang application có warehouse trước bật scoped; new-row guard sau giai đoạn tương thích, không sửa lịch sử; rollback phải tắt scoped traffic trước và không cho old writer tạo null âm thầm |
| R11 / cao | Order [Package adapter](../../services/order-service/internal/fulfillmentprovider/adapter.go) tham gia trực tiếp xử lý notification PACKING_SEALED | Nếu nhét validation packed_lines vào lỗi base proof sẽ vô tình chặn lifecycle/shipping khi extension thiếu hoặc lỗi | Giữ base proof validation cũ; extension được kiểm riêng, sai chỉ làm traceability UNAVAILABLE. Không bỏ các guard base parcel/media/seal để chạy được test mới |

Các dòng cao là lỗi/thiếu của tài liệu0.2.1 hoặc rủi ro tích hợp đã có bằng chứng, không phải cáo buộc code baseline phải có sẵn feature mới. Source mới không có v2 là expected implementation gap; việc giả định một provider đã có trong khi không tìm thấy route mới là vấn đề cần sửa trong handoff.

## Rủi ro còn lại phải thể hiện khi nghiệm thu

| Rủi ro | Biện pháp bắt buộc | Giới hạn không được che |
| --- | --- | --- |
| Recall/quarantine sau publish | Người có quyền suspend và quy trình xử lý kho; test cùng QR chuyển410 | Không có event đồng bộ recall trong scope này; scan không kiểm live Inventory. Không quảng bá “xác minh thời gian thực” |
| Bản tem đã tải offline / đổi domain | URL public ổn định, HTTPS, duy trì domain/redirect an toàn; cùng QR dẫn trạng thái hiện hành | Không thể thu hồi ảnh/giấy đã in hoặc đảm bảo QR tồn tại nếu bỏ domain |
| Nguồn đổi sau network read | Budget/freshness, version/hash, expected_version và audit | Không có distributed transaction với Catalog/Inventory; snapshot là xác minh tại thời điểm đọc |
| Provider chưa deploy | Implement/test fixture đúng contract, feature/capability gate, runtime test riêng | Không coi fixture hoặc lint là bằng chứng integration |
| Lịch sử thiếu packed_lines/kho | Giữ nguyên snapshot, GLOBAL-only legacy metadata, báo UNAVAILABLE khi không đủ proof | Không dựng lại lô thực giao từ tồn kho hiện tại và không sửa JSON đã công bố |
| In/preview từ draft hoặc dữ liệu chứa markup | Dùng public revision, encode text khi render SVG/HTML, không arbitrary QR URL | Preview đẹp không chứng minh in thực tế scan được; cần decode/thiết bị thật |
| Native local port trùng | Inventory admin HTTP dùng INVENTORY_ADMIN_HTTP_ADDR; tách host ports với Traceability8003, giữ alias/container ports đúng config | Hai service khác container có thể dùng cùng internal port; không mặc định cả hai native cùng8003 |

## Điều kiện có thể giao triển khai

Owner nhận target0.2.2, OUT tương ứng và test bổ sung trong acceptance. Không còn việc hỏi “lấy productId ở đâu”, “map dòng đơn thế nào”, “cookie đi vào đâu” hoặc chờ ACCEPT. Những phần phải viết thêm được ghi đích danh: Catalog binding read, Traceability context, Identity cookie transports, Web scoped projection và provider workbench read nếu chưa có; không bắt owner xây lại module đã đáp ứng.

Chưa có kết quả runtime, chưa tuyên bố hết rủi ro. Provider tests, schema/examples, proxy/CSRF/scope và rollout guards là điều kiện nghiệm thu trước stable/deploy.

## Kết quả kiểm tra artifact sau sửa target0.2.2

| Kiểm tra | Kết quả |
| --- | --- |
| Redocly3contract target + admin/customer draft-planned + admin/customer stable | PASS, cả7file exit0. Integration còn1warning receipt delta chưa dùng trong operation; customer draft1 và stable15/22warning baseline, không có lỗi lint |
| Expand refs và validate request/response examples JSON Schema2020-12 | PASS305examples sau khi bổ sung proxy errors; kiểm schema không thay domain/runtime tests |
| Negative contract cases | PASS6cases: AVAILABLE/gone không được null URL; NOT_AVAILABLE không được có URL; READY không được rỗng; pending không được có rows; row thiếuSKU không hợp lệ |
| Proto candidate compile và so descriptor canonical | PASS2artifacts, giữ17message types/4service definitions cũ; không đổi field/method cũ, fields9/14 additive |
| Tham chiếu tài liệu và registry | PASS, file tồn tại/ID không trùng; không chứng minh service đã deploy |
| git diff --check | PASS |
| Runtime code/stable contracts/canonical proto/event schemas | Không sửa trong lượt review này |

Trạng thái runtime/peer implementation: **NOT RUN / NOT VERIFIED**. Những lỗi tài liệu đã được sửa không đồng nghĩa đã vá code production hoặc chạy được full flow. Owner vẫn được bắt đầu triển khai trực tiếp từ0.2.2 mà không cần phản hồi ACCEPT; chỉ nghiệm thu/deploy sau các test tương ứng.
