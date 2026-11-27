# Chốt nghiệp vụ và thiết kế tích hợp truy xuất theo lô

Ngày: 21/11/2026 (Asia/Ho_Chi_Minh). Đối chiếu code `origin/dev@6f54a4ab`.

**Bổ sung 25/11/2026:** [Index bộ tài liệu](traceability-workflow-README.md), [đặc tả Traceability](traceability-workflow-service-spec.md), [phiếu bàn giao](traceability-workflow-peer-handoff.md) và [nghiệm thu](traceability-workflow-acceptance.md) làm rõ các lựa chọn kỹ thuật còn để mở bên dưới. V2 scoped admin dùng common error/header; v1 giữ tương thích. Handoff mặc định không hỗ trợ chuyển kho cùng UUID âm thầm, không backfill từ live allocations, dùng candidate field9/14. Theo chỉ đạo mới ngày25/11, owner triển khai trực tiếp theo target0.2.2-draft, không gửi ACCEPT hoặc chờ duyệt mapping. Gate còn lại là kiểm thử capability thật trước rollout/stable.

**Cách giao việc cho peer:** các bổ sung field/endpoint/projection dưới đây là phương án candidate để đạt đầu ra, không khẳng định owner phải viết mới. Owner tự đối chiếu OUT và triển khai: đã đủ thì tái sử dụng/test, thiếu thì bổ sung, API nội bộ khác wire thì tự map ra contract đích trong phần phụ trách. Không cần báo cáo trước hoặc chờ trả lời. Chỉ thay cách implement nội bộ, không thay ngầm semantics nguồn, scope, lịch sử hoặc wire consumer.

**Amendment27/11 — target0.3.0:** [gRPC guide](traceability-workflow-grpc.md) và ba shared proto mới thay business internal REST của0.2.2. Nghiệp vụ giữ nguyên; Web/BFF vẫn REST và OIDC vẫn HTTPS. Các checkpoint0.2.2 bên dưới là lịch sử, không dùng làm wire contract mới.

## 1. Hiệu lực và phạm vi

Chủ yêu cầu đã chốt QR theo lô, in tem sau công bố, truy xuất công khai, xem từ lịch sử/chi tiết đơn; ngày 21/11 chốt thêm nhân viên chỉ thao tác trong kho được giao và thiết kế toàn luồng dù triển khai theo đợt. Tích hợp bước lập hồ sơ ngay sau nhập kho trên cùng UI; hai nghiệp vụ vẫn có transaction và vòng đời riêng.

Đây là thiết kế đích, không phải biên bản kiểm thử hoặc xác nhận tích hợp của các owner. API bổ sung nằm ở [draft contract](traceability-workflow-draft.openapi.yaml), không thay thế stable. Các tài liệu P8 ngày 13/11 vẫn mô tả baseline; mục này thay phần thiết kế đích về scope kho, in tem và kết nối đơn hàng. Chưa sửa production, chưa nâng contract lên stable, chưa triển khai chức năng.

## 2. Quyết định nghiệp vụ

- Inventory sở hữu lô, kho, NSX/HSD và tồn. Traceability sở hữu hồ sơ nguồn gốc, mã/link QR và workflow duyệt/công bố. Nhập kho không tự chứng minh xuất xứ hoặc tự publish.
- QR chứa `qr_url` tuyệt đối do server tạo từ public web base URL; không chứa API URL, token, thông tin đơn/người mua. Không dùng URL từ input tùy ý. Một hồ sơ gắn đúng Product–SKU–namespace–Batch; một mã không chuyển sang lô khác.
- Nhiều đơn vị cùng lô dùng chung mã; in lại giữ nguyên mã. Nếu chuyển kho sinh Batch UUID mới thì không tự gộp với hồ sơ cũ; chuyển kho dùng lại UUID phải được provider xác định rõ trước rollout scope kho.
- Nhập kho thành công rồi mới có hành động “Lập/Xem hồ sơ nguồn gốc”. Không tự tạo nháp qua Kafka hoặc sau mọi lần nhập. Khi thiếu quyền, giữ kết quả nhập kho và hướng dẫn chuyển việc cho người có quyền, không hiện lỗi như nhập kho thất bại.
- UI lấy binding từ kết quả nhập và dữ liệu lô/Catalog thật, không ghép Product theo tên, không đoán ID từ SKU. `receiptResult` hiện chỉ có operation/movement/revision, chưa có Batch UUID; thiết kế bổ sung batchId, batchCode, skuCode, warehouseId vào result trong cùng transaction và response idempotency. Gọi scoped batch-context của Traceability để lấy productId đã xác minh sau scope check và Catalog exact-SKU binding; không dựa Inventory SKU detail chưa có provider. Không lookup mơ hồ theo tên/mã hiển thị.
- Thiếu chứng nhận vẫn ghi nhận tồn kho; hồ sơ nháp có thể hoàn thiện sau. Tạo nháp lỗi không rollback receipt. Gửi lại lệnh nhập dùng cùng idempotency key, không tạo lô/receipt thứ hai.
- Nhân viên được cấp `read_internal`, `create`, `edit`, và `submit` nếu được giao, theo kho. Không tự cấp approve/publish/evidence/suspend/archive. Người duyệt/công bố được cấp riêng; giữ nguyên quy tắc người đóng góp không tự duyệt.
- Tải/in là trình bày dữ liệu, không có nghiệp vụ cấp mã mới, không cần quyền in riêng trong phạm vi này: dùng quyền đọc nội bộ và scope đúng kho. Không yêu cầu Audit server cho số bản in vật lý; nếu cần quản lý từng tem/serial/chống giả thì mở phạm vi riêng.
- Chỉ cung cấp tem hợp lệ khi hồ sơ đang được công bố và còn đủ điều kiện công khai. Web sinh QR cục bộ từ link, không gửi sang dịch vụ QR bên thứ ba. Tem gồm tên sản phẩm, mã lô, QR, mã nhập tay và link dự phòng; hỗ trợ tải SVG/PNG và trang in. Định dạng giấy/kích thước vật lý cần kiểm chứng với máy in/bao bì thực tế, không ấn định chưa có bằng chứng.
- Trước mở trang in/tải, đọc lại trạng thái được phép và bản public. Khi chưa đọc xong hoặc upstream lỗi, chặn xuất tem. Không hứa chặn được bản đã tải/in offline sau đình chỉ; trang QR luôn kiểm tra eligibility khi khách truy cập.
- QR là công cụ truy cập hồ sơ, không phải chứng minh từng hộp là hàng thật. Không lấy QR của một lô đại diện thùng nhiều sản phẩm/lô. QR tổng hợp kiện và tự động genealogy Procurement ngoài phạm vi.

## 3. Ma trận kết nối: hiện có và phần thiếu

| Kết nối | Bằng chứng tại baseline | Thiết kế đích / owner |
| --- | --- | --- |
| Web → Inventory nhập kho | `features/employee/inventory/server/handler.ts`, `httpapi/receipt.go`: NEW/EXISTING, transaction, idempotency, movement; result thiếu Batch UUID | Giữ nguyên transaction; bổ sung binding lô vào receipt result; Inventory/Web |
| Identity → Keycloak → Traceability | V20 permission, V21 bootstrap ADMIN, scoped assignments và sync; Traceability `staff.go` chỉ nhận GLOBAL, Principal chỉ có flat permissions | Identity cấu hình role/assignment; Traceability giữ grant scope, kiểm tra từng resource; không cấp GLOBAL thay WAREHOUSE |
| Traceability → Catalog | Baseline REST verifier có version | Target0.3.0 gRPC CatalogTraceabilitySource.GetSkuBinding, version semantics cũ, hidden/no-price; không gọi mỗi scan |
| Traceability → Inventory | `BatchSource.GetTraceabilityBatch`, UUID/SKU/version/NSX/HSD, chưa có kho trong response | Bổ sung `warehouse_id` từ Inventory; Traceability lưu binding kho, xác minh server-side |
| Web → Traceability lập/xem hồ sơ theo lô | CRUD/workflow đã có; list chỉ page_size/cursor | GET lookup binding riêng; trả cùng hồ sơ khi nhập bổ sung; xử lý race bằng unique binding và đọc lại sau conflict |
| Web → Traceability in tem | Có `qr_url`, API public, chưa có UI in | Dùng read hiện có với scope bổ sung; không cần service tạo ảnh hoặc lệnh cấp QR mới |
| Web → public QR | `/trace/{code}` → BFF → Kong `/api/v2/trace/{code}` → service v1 | Tái sử dụng public contract, anonymous; thêm `/trace` làm trang nhập mã ở footer, bỏ mã mẫu cứng |
| Fulfillment → Inventory | `GetCommittedPackingAllocation` và `ValidatePackingAllocation` có line ID, Batch UUID/code, allocated/count | Tái sử dụng; allocation hiện tại không thay thế lịch sử đã niêm phong |
| Fulfillment → Order | `GetPackageRevision` đúng sealed version; snapshot chứa allocations nhưng response thiếu chúng | Bổ sung packed lines vào proof, Order persist projection; cùng contract gRPC, không query DB chéo |
| Order → Traceability | Chưa có resolver binding lô | gRPC BatchLinkResolver read-only, danh tính riêng `order-service`, bounded batch 100 |
| Web → Order xem nguồn gốc | Detail DTO hiện chưa có binding | GET subresource riêng, không sửa nghĩa commercial snapshot/stable Order detail |

## 4. Scope kho và mô hình dữ liệu

Identity đã hỗ trợ `permission#WAREHOUSE#warehouseId`; không thêm một hệ IAM mới. Không suy quyền từ role ADMIN/nhân viên kho. Dùng intersection của grant token và grant hiện hành qua gRPC Identity GetCurrentStaffGrants theo từng tuple permission/scope/ID; GLOBAL bao phủ resource, WAREHOUSE chỉ khớp chính kho. Mất Identity trả 503, không fallback sang quyền trong token cũ. Preserve scopeId trong adapter và domain thay vì chỉ flat permission.

Inventory đề xuất mở rộng `BatchSnapshot` bằng field mới `string warehouse_id = 9` (không đánh lại số cũ). Giá trị lấy từ quan hệ kho authoritative của lô, không do caller cung cấp. Mã SKU hoặc tổng tồn không đủ chứng minh quyền nếu SKU có nhiều kho; provider phải xác nhận mô hình ánh xạ. Chỉ mở rollout scope khi mỗi UUID có kho xác định được và có test chuyển kho. Thiếu/không xác minh được kho phải từ chối thao tác scoped, không coi là GLOBAL.

Traceability lưu `warehouse_id` server-derived trên profile và snapshot nguồn; kho không phải dữ liệu form có thể tùy sửa. Với dữ liệu cũ: backfill bằng provider có kiểm chứng, bản chưa backfill chỉ cho GLOBAL xử lý; WAREHOUSE không thấy dữ liệu ngoài phạm vi. Mọi list/get/create/edit/submit/review/publish/suspend/archive và replay đều kiểm quyền + scope trước trả dữ liệu. Không để list trả toàn bộ rồi UI lọc; cursor gắn phạm vi, đổi quyền không giữ quyền đọc cũ. Đọc mã hồ sơ không thuộc phạm vi trả 403, không trả nội dung; lookup theo binding cần kiểm tra kho trước cả kết quả rỗng.

Khi warehouse authoritative đổi, không tự mở quyền kho mới bằng cách sửa profile âm thầm. Trả conflict, giữ lịch sử, cần migration/chuyển giao rõ ràng do owner chốt; chưa hỗ trợ tự động trong đợt này. GLOBAL vẫn phải theo workflow, không bypass contributor policy.

## 5. Luồng nhập kho → hồ sơ → tem

1. Web gửi receipt có revision và idempotency tới Inventory. Inventory kiểm quyền kho, ghi lô/tồn/movement cùng transaction.
2. Thành công, UI giữ result. Nếu có quyền Traceability, lookup binding; 200 rỗng cho tạo mới, 200 có hồ sơ cho mở hồ sơ. Timeout có retry đọc; không gọi lại nhập kho.
3. Form điền sẵn binding, đọc NSX/HSD authoritative, nhân viên nhập nguồn gốc/chứng nhận. Create dùng idempotency key ổn định cho cùng lần gửi. Hai người tạo đồng thời: unique binding trả conflict cho một bên, UI đọc lại và mở hồ sơ tồn tại; không cấp mã thứ hai.
4. Workflow hiện có: draft → submit → approve → publish. Provider được đọc lại theo chính sách snapshot/version hiện có; thiếu quyền/nguồn thay đổi chặn trước commit.
5. Đọc lại điều kiện public và quyền đọc nội bộ; sinh preview, tải/in cùng qr_url. Loading/error không xuất tem hợp lệ. Nhân viên đối chiếu tên/mã lô trên tem với hàng trước dán.

## 6. Luồng đóng gói → xem từ đơn

1. Fulfillment lấy Order lines và committed Inventory allocations, ghi counted quantities, revalidate trước seal. Giữ snapshot sealed bất biến.
2. Contract đề xuất `PackageRevision` thêm `repeated PackedBatchLine packed_lines = 14`; phần tử gồm order_line_id, sku_code, batch_id, batch_code, packed_quantity. `warehouse_id` lấy ở proof cấp kiện; `source_namespace=inventory` do adapter đã xác minh gán. Không dùng batch_code đơn lẻ làm định danh.
3. Chỉ trả các lô từ revision `PACKING_SEALED` yêu cầu, counted bằng allocated, tổng từng dòng đúng lượng đóng; không đọc phân bổ live để dựng lịch sử. Kiểm tra order/package/task/version/generation/items_digest/warehouse. Rỗng ở producer cũ nghĩa capability thiếu, không suy là đơn không có lô.
4. Giữ event packing v2 làm notification. Order cung cấp projection lô theo order_line_id cùng dấu vết proof đúng revision; có thể tái sử dụng read model/proof reader đã đủ hoặc persist projection khi cần, không bắt thêm bảng mới. Consumer phải chấp nhận proof cũ thiếu packed_lines mà không làm hỏng chuyển trạng thái/Shipping; riêng nguồn gốc báo chưa sẵn sàng. Lỗi Traceability không chặn PAID/PACKED/shipping.
5. Khách vào lịch sử → chi tiết → GET subresource traceability. Order kiểm ownership/guest capability tương đương detail hiện có trước mọi dependency call; không mở guest access mới. Chuẩn hóa ID dòng từ snapshot packing chính thức, không dùng `orderId-index` do UI tạo làm khóa cross-service. Order giữ mapping với snapshot thương mại; mismatch thì không gắn nhầm hồ sơ.
6. Khi chưa có sealed proof, trả PACKING_PENDING hoặc NOT_APPLICABLE với items rỗng và total=0. Khi có, trả từng dòng/lô/số lượng. Snapshot Order hiện không có product_id; resolver dùng `(source_namespace, batch_id, sku_code)` từ sealed proof. CSDL Traceability đã unique `(source_namespace, batch_id)`, resolver đối chiếu SKU với hồ sơ trước trả link; Product đã được xác minh khi tạo/publish hồ sơ. Không cần thêm Catalog lookup vào mỗi lượt đọc đơn và không đoán product_id.
7. Order gọi resolver Traceability theo binding lô/SKU chính xác, tối đa 100 mỗi request, chia batch nếu nhiều hơn. Resolver chỉ trả link khi eligibility hiện hành đạt; draft/không tồn tại cùng NOT_AVAILABLE, không tiết lộ lịch sử nội bộ. Resolver timeout → UNAVAILABLE; không coi là không có hồ sơ và không thay bằng mã mẫu.
8. Hỏng phần nguồn gốc không làm mất chi tiết đơn: endpoint riêng, UI retry riêng. Link mở public page; trạng thái phải được kiểm lại khi mở, không dựa vào eligibility đã cache. Lô đã công bố nhưng nay ngừng công bố có thể trả NO_LONGER_PUBLIC cùng link cũ dẫn 410 an toàn; không tạo QR mới.

## 7. Kafka và nguyên tắc không kích hoạt QR qua đóng gói

| Event/topic | Sử dụng trong thiết kế này | Giới hạn |
| --- | --- | --- |
| `fulfillment.events.v2`, packing.completed.v2 | Order được thông báo để lấy proof đúng version | Không kích hoạt/publish QR; `scanned_batch_codes` không đủ join dòng đơn hoặc UUID |
| `procurement.events.v1`, goods.received.v1 | Ngoài luồng nối UI; có thể tham khảo bằng chứng sau này | Schema thiếu Batch UUID, NSX/HSD và mapping nguyên liệu → thành phẩm; không tự tạo hồ sơ hoặc cộng tồn lần hai |
| `traceability.events.v1` service-local | Outbox publish/status theo implementation hiện có | Consumer chung chưa được xác nhận; không coi Kafka catalog là bằng chứng đã tích hợp |
| Inventory stock events | Giữ cập nhật tồn theo flow hiện có | Không làm trigger tự công bố nguồn gốc |

Không thêm topic, consumer Traceability của packing, hoặc event in tem cho scope này. Giữ v1/v2 payload đã có; mở rộng gRPC proof theo kiểu additive, không thêm required fields vào event cũ. Nếu sau này đổi schema event thì version mới và consumer rollout trước producer.

Consumer Order dùng inbox dedup event ID, giữ order key, kiểm source/generation/version; lấy proof bằng đúng ID/version của event, không latest. Projection update nguyên tử, retry có backoff và giữ event chưa xử lý, không bỏ bằng chứng khi peer gián đoạn. Event tới muộn/đảo thứ tự không ghi lùi revision hoặc thay lô. Không retry generation QR vì không có generation backend trong flow này.

Không có push thu hồi lô tự động trong scope này: scan xét trạng thái public/snapshot hiện có, nguồn mới bị cách ly cần người có quyền đình chỉ theo quy trình hiện hành. Không mô tả hồ sơ là dữ liệu kho live từng giây.

## 8. Ownership, rollout và các gate bắt buộc

| Owner | Deliverable | Điều kiện kỹ thuật kiểm trước rollout, không phải duyệt trước code |
| --- | --- | --- |
| Web + Traceability | Batch context + lookup, scoped CRUD, form nối receipt, typed grant/line mapping, preview/download/print | Contract draft, quyền theo kho, deep-link không nhận URL tùy ý, lỗi nhập kho độc lập |
| Catalog | Exact-SKU internal binding không phụ thuộc giá/public | Dedicated credential, hidden/stopped SKU tests, không expose public |
| Inventory | BatchSnapshot.warehouse_id; receipt binding; live workbench read theo contract sẵn | Xác nhận authoritative warehouse, transfer semantics; không đổi transaction nhập |
| Identity | Role/assignment và cookie transports theo BFF contract | Active employee cho v2, Origin/CSRF, tests không lộ token; không sửa migration cũ |
| Fulfillment | packed_lines từ exact sealed proof, cập nhật proto và codegen các bản copy | Đúng field/mapping target và legacy compatibility tests |
| Order | Persist proof projection, map line ID, resolver client và customer subresource | Ownership tests; không chặn lifecycle nếu resolver lỗi; consumer rollout trước producer |
| Gateway | Route GET lookup, customer subresource theo owner/bff; internal resolver không public | Test route thật/strip forged identity, không expose resolver bằng wildcard |

Phạm vi thực tế toàn luồng là Traceability, Order, Fulfillment và Inventory (bổ sung kho); Identity cần cả transport cookie proxy khi dùng BFF ngoài cấu hình grants; Web thay đổi UI/BFF/scoped projection; Gateway bổ sung routing. Catalog cần exact-SKU binding để bootstrap Product. Xem review0.2.2; nhận định trước đây Catalog không cần sửa/Identity chỉ cấu hình không đủ cho full flow. Nhận định trước đây “Inventory không cần sửa” chỉ đúng cho phần phân bổ lô, không còn đủ khi chốt giới hạn theo kho.

Các quyết định sản phẩm đã chốt; owner thực hiện trực tiếp theo các lựa chọn kỹ thuật trong đặc tả service/handoff0.2.2: bảo toàn authoritative warehouse, canonical line mapping, field9/14 và rollout tương thích; lịch sử thiếu proof báo chưa sẵn sàng, không backfill từ live allocation. Không chờ ký nhận trước code. Thiếu khả năng provider thật chỉ giữ ca integration ở trạng thái chưa nghiệm thu; nếu không thể đáp ứng semantics trong scope thì ghi blocker và làm phần độc lập. Promotion stable vẫn cần contract tests và bằng chứng provider thật, không phải chỉ vì nghiệp vụ đã chốt.

### Delta contract đề xuất, chưa sửa wire hoặc stable

- Receipt result tại `POST /api/v1/admin/inventory/workbench/receipts` (provider) và `/api/admin/inventory/workbench/receipts` (Web BFF): thêm `batchId: UUID`, `batchCode: string`, `skuCode: string`, `warehouseId: UUID` cho APPLIED, cả lần đầu và idempotent replay. BFF giữ nguyên các trường này. Dữ liệu receipt lịch sử chưa có field phải hiển thị cần đọc lại lô, không điền UUID giả. Khi triển khai phải cập nhật operation tương ứng trong admin contract và contract tests trước khi promote.
- `BatchSnapshot`: giữ field 1–8; đề xuất `string warehouse_id = 9`, không thay đổi namespace/RPC. Baseline kho nằm ở `inventory_items.warehouse_id` và batch tham chiếu SKU; provider có thể JOIN theo quan hệ này ở mô hình hiện tại, nhưng không suy rộng thành hỗ trợ chuyển kho/đa kho. Phiên bản nguồn phải phản ánh thay đổi kho ảnh hưởng scope.
- `PackageRevision`: giữ field 1–13, đề xuất field 14 `repeated PackedBatchLine packed_lines`; message mới có `string order_line_id = 1`, `string sku_code = 2`, `string batch_id = 3`, `string batch_code = 4`, `int32 packed_quantity = 5`. Scope kho ở proof cấp kiện, không nhận warehouse do khách gửi.
- Update proto canonical và các bản copy/codegen trong từng service cùng provider/consumer tests theo target triển khai trực tiếp; tài liệu này không sửa generated code hoặc required event schema. Dùng compatibility gate trước rollout, không đổi `order-3.2` chỉ vì thêm field nếu semantics baseline giữ nguyên.

## 9. Kế hoạch kiểm chứng và môi trường

Đợt A: Web + Traceability + PostgreSQL; dùng fixture để QA layout/in có ghi rõ sandbox. Kiểm thật scope kho cần Identity/Keycloak, Catalog và Inventory cùng DB phụ thuộc. Chưa có authoritative scope thì không nghiệm thu bằng grant GLOBAL giả.

Đợt B: bổ sung Order, Fulfillment, Inventory và broker cho notification/proof. Chỉ thêm peer thật cần để tạo order fixture và seal; payment/carrier/media có thể dùng test ports được ghi rõ, không báo full end-to-end nếu giả lập. Docker Engine baseline đang chưa chạy; chưa có kết quả runtime. Muse kiểm browser local và trả ảnh/báo cáo theo commit vào thư mục test riêng, không thao tác dữ liệu production. Nhánh triển khai xuất phát dev, PR qua trưởng nhóm rồi release RC02; chưa có xác nhận deploy.

| Nhóm kiểm thử | Bắt buộc |
| --- | --- |
| Receipt | NEW/EXISTING; duplicate/revision conflict; tạo nháp lỗi không mất receipt; tìm đúng UUID lô mới |
| Scope | Kho A/B, GLOBAL, cấp/thu hồi, scoped replay/cursor, provider thiếu warehouse, token và /me lệch, giả warehouse từ UI |
| Workflow | Hai actor, contributor không tự duyệt, snapshot change conflict, publish timeout, draft không in |
| Print | SVG/PNG và trang in giải mã đúng qr_url, mobile/desktop, quiet zone/độ tương phản, không cắt QR; reprint cùng mã; lỗi/410 chặn in mới; test máy in thật trước phát tem |
| Packing | Một dòng nhiều lô, nhiều dòng, unique SKU/dòng, duplicate-SKU bị chặn; proof đúng sealed version; mã lô trùng nhưng UUID khác; thiếu trường producer cũ |
| Events | Duplicate/out-of-order/restart/peer timeout; không publish QR hoặc trừ kho lần hai; Order lifecycle không phụ thuộc resolver |
| Customer | Ownership/guest grant hết hạn; zero collections; resolver mixed results/down; pending/cancelled; deep-link QR anonymous 200/404/410/503 |
| Contract | OpenAPI lint, schema/examples, proto compatibility + codegen ở từng service; provider/consumer tests trước stable |

## 10. Tài liệu liên quan

- [EPIC 19, quyết định nghiệp vụ](../01_requirements/epics/EPIC_19_OCOP_Traceability.md#65-quyết-định-nghiệp-vụ-qr-theo-lô--đã-chốt-ngày-21112026).
- [P8 batch snapshot](../02_architecture/services/traceability-service/08_batch_snapshot_2026-11-13.md), [P8 scope GLOBAL baseline](../02_architecture/services/traceability-service/09_staff_authorization_2026-11-13.md).
- [Registry MVP](../../packages/contracts/mvp/registry.yaml), [Packing integration](../../packages/contracts/mvp/fulfillment-integration.md).
- [Kafka catalog](02_kafka_event_catalog.md), [Events README](../../packages/events/README.md).

## 11. Kết quả rà soát tài liệu và tình trạng chốt

- REST draft bổ sung ba operation; delta receipt được ghi riêng, không ghi đè stable. gRPC chỉ đề xuất field mới, chưa sửa proto/codegen. Không thay event schema hoặc generated frontend rows.
- Đã đối chiếu HLD, container diagram, ghi chú tổng quan, P8 handoff/scope/batch snapshot, Kafka catalog, Events README và MVP registry; cập nhật những mô tả packing kích hoạt QR cho đúng phạm vi mới.
- Redocly lint draft mới và bốn file customer/admin stable/draft-planned: exit 0. Baseline có warning ví dụ/unused component/license; không tự sửa các ví dụ sinh tự động hoặc nội dung ngoài phạm vi. Draft mới có một warning: schema delta receipt chưa gắn vào operation (chỉ được tham chiếu qua extension); chưa sửa operation stable. Các draft chung tham chiếu operation và security scheme bằng $ref, cùng chuẩn OpenAPI 3.1.
- Parse YAML, ID registry duy nhất, artifact registry tồn tại, đường dẫn tài liệu mới và git diff --check: PASS. Đây là kiểm tra tài liệu, chưa chứng minh wire compatibility của delta chưa implement, scope kho hoặc end-to-end.
- Bàn giao peer review: Inventory xác nhận warehouse/transfer và receipt delta; Fulfillment xác nhận field/proof; Order xác nhận line mapping và projection lịch sử; Identity xác nhận assignment; Web/Gateway xác nhận routing. Owner không cần phản hồi ACCEPT để bắt đầu. Trạng thái runtime integration: NOT_VERIFIED; chỉ promote stable sau contract tests và bằng chứng thật.

**Kết quả review0.2.2:** [report](traceability-workflow-review-2026-11-25.md) và [service spec mục10](traceability-workflow-service-spec.md#10-sửa-sau-review-code-2511--target022) thay các giả định baseline còn thiếu về Product bootstrap, BFF credentials, one-SKU-per-line, evidence hydration và idempotency version namespace. Các role/stock/public semantics nghiệp vụ đã chốt không đổi.
