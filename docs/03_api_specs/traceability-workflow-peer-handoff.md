# Phiếu bàn giao contract theo service

Version 0.4.0-draft · 27/11/2026. Owner nghiệp vụ: Traceability. **Chế độ giao việc đã được chủ yêu cầu chốt: triển khai trực tiếp từ tài liệu, không cần gửi ACCEPT, đề xuất giải pháp hoặc chờ xác nhận trước khi sửa code.** Mỗi owner nhận section của mình, bảo đảm đầu ra và contract đích; đã có thì tái sử dụng, thiếu thì bổ sung. [Index](traceability-workflow-README.md), artifacts và [test IDs](traceability-workflow-acceptance.md) là một bộ đầu vào, không cần lịch sử chat. Không gửi secrets hoặc tài khoản cá nhân trong tài liệu.

**Bản 0.4.0 theo plan Dương và dev@ac5395c5 (PR #175):** thay phiếu 0.3.0/0.2.2 về Catalog, dùng CatalogService.GetTraceabilityBinding và MediaInternalService đã có. Gỡ target GetSkuBinding chưa phát hành; không giao lại việc viết RPC đó. Tích hợp Media validate trước submit/publish, durable binding sau commit; Web/BFF vẫn REST. Identity current-grants và Order resolver vẫn là phần việc riêng chưa có trong plan Dương.

## 0. Đề bài thực thi — đọc và bắt đầu làm

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

- Business calls giữa service dùng gRPC; REST Web/BFF và OIDC protocol giữ nguyên theo guide. Wire/case/ID và permission là contract đích, không đổi tên cho giống convention riêng. Không fake provider verification bằng mock, không sửa stable/generated examples hoặc migration cũ.
- HTTP error chuẩn mới `{code,message,requestId}`; mutation mới có Idempotency-Key. Legacy route giữ contract riêng. Internal RPC/call dùng danh tính service và transport tin cậy, không tin X-User-ID/role do browser tự gửi.
- Không kích hoạt/publish QR khi nhận packing hoặc GoodsReceived. Không thêm trừ kho, tạo profile, cấp mã hoặc payment side effect trong read flow.

## 1. Inventory: INV-TRC-01

**Mục đích:** sau nhập kho biết chính xác UUID lô; Traceability xác minh kho để chặn nhân viên ngoài phạm vi.

**Đầu ra bắt buộc, có thể tái sử dụng phần đã có:**

| Mã yêu cầu | Consumer nhận được gì | Điều kiện để được coi là đáp ứng |
| --- | --- | --- |
| INV-OUT-01 | Kết quả xác minh lô gồm UUID, SKU, mã lô, NSX/HSD, trạng thái hạn chế/hết tồn, version và kho sở hữu | Binding chính xác; đọc được lô lịch sử còn lưu; kho lấy từ nguồn authoritative và không âm thầm đổi ý nghĩa |
| INV-OUT-02 | Binding lô vừa nhập, đủ để mở/lập đúng hồ sơ | NEW/EXISTING xác định đúng UUID; cùng receipt replay không tạo thêm tồn hoặc lô. Có thể tái sử dụng read API xác định duy nhất theo receipt/movement bên trong adapter, nhưng đầu ra receipt mà consumer nhận vẫn đúng contract đích |
| INV-OUT-03 | Phân biệt binding sai, thiếu quyền và provider lỗi | Status/error và timeout đúng contract; không trả lô gần đúng hoặc gán kho mặc định |

**Provider contracts:** [BatchSource candidate](traceability-workflow-candidates/batch_source.proto), receipt delta trong [OpenAPI tích hợp](traceability-workflow-draft.openapi.yaml). Không đổi Reserve/Finalize/Release hay FEFO.

| Đầu vào/đầu ra | Yêu cầu |
| --- | --- |
| GetTraceabilityBatch request | Giữ batch_id field1 và sku_code field2; lookup exact UUID+SKU, trả cả lô hết tồn/hết hạn/cách ly còn lưu |
| Response fields1–8 | Giữ nguyên semantics và field numbers |
| warehouse_id field9 | UUID kho sở hữu authoritative của Batch; empty chỉ capability cũ, consumer không tự gán default |
| batch_version | Tăng khi các trường source ảnh hưởng publish/scope đổi; cùng version không được đổi nội dung source |
| APPLIED receipt result | Giữ toàn bộ trường cũ, thêm camelCase batchId/batchCode/skuCode/warehouseId; UUID server đã ghi thật; cả NEW và EXISTING |
| Receipt replay | Lưu kết quả bổ sung cùng idempotency response/transaction. Bản legacy chưa có field thì giữ replay cũ, không tạo receipt mới; Web dùng đọc lại có kiểm chứng hoặc báo chưa lấy được binding |

Baseline đã đọc cho thấy batch FK tới SKU và warehouse nằm ở inventory_items; JOIN là một gợi ý, không phải yêu cầu đổi DB hoặc bắt owner dùng câu query đó. Candidate warehouse là kho sở hữu để authorization, không phải “vị trí cuối cùng” đổi theo một request. Trong release này không hỗ trợ chuyển cùng UUID sang kho khác một cách âm thầm: provider từ chối hoặc chuyển giao theo version contract riêng. Nếu provider đã có ownership model tương đương, tự dùng lại và map ra contract đích. Nếu không thể bảo toàn semantics kho trong phần phụ trách thì ghi BLOCKED cho phần scoped bị ảnh hưởng; không tạo mapping tạm trong UI, vẫn làm các phần độc lập.

RPC đã có bearer riêng Traceability, giữ TLS và credential. Invalid input→INVALID_ARGUMENT; binding không tồn tại→NOT_FOUND; credential invalid→UNAUTHENTICATED; DB lỗi→UNAVAILABLE. Traceability timeout budget toàn Catalog+Inventory2s; provider không retry mutation vì RPC này chỉ read.

**Nộp:** bảng đáp ứng INV-OUT-01…03 với API/RPC/version hiện có, mapping và bằng chứng test receipt/idempotency/batch thật. Chỉ nộp PR/migration/proto/codegen nếu có gap cần sửa; nếu dùng candidate thì cập nhật canonical/copies/codegen đúng version đích. Nghiệm thu INV-01…04 và T-SCOPE-01/04. Không cần làm màn hình hồ sơ hoặc sinh QR. INV-OUT-04: bảo đảm provider GET /api/v1/admin/inventory/workbench theo admin contract hiện có để UI nhập kho đọc được SKU/lô/revision thật; trong source đã rà mới thấy receipt handler, không coi Mockoon là provider. Chỉ hiện thực phần read phục vụ luồng này nếu chưa có, không mở rộng Issue/Adjustment ngoài scope.

## 2. Identity/Keycloak: IAM-TRC-01

**Mục đích:** cấp các permission đã có cho đúng nhân viên/kho. Không thêm role bypass hoặc scope claim mới.

**Đầu ra bắt buộc:** IAM-OUT-01: token và `/users/me` biểu diễn đúng permission gắn scopeId kho; IAM-OUT-02: nhân viên soạn/duyệt được gán đúng nhóm quyền, thời hạn và thu hồi có hiệu lực theo contract; IAM-OUT-03: có bộ actor test kiểm kho A/B và tách nhiệm vụ. Phần IAM grants có thể chỉ cần cấu hình/test. IAM-OUT-04: trong cookie mode cung cấp các transport trong [BFF contract](traceability-workflow-bff-draft.openapi.yaml), guard ACTIVE EMPLOYEE cho cả v2, giữ Origin/CSRF và chuyển cookie thành bearer server-side; không mặc định Identity chỉ cần cấu hình. Direct OIDC mode không cần cookie adapter. Không bắt tạo role mới.

| Bundle nhiệm vụ | Permissions được gán |
| --- | --- |
| Soạn hồ sơ | read_internal + create + edit; thêm submit nếu được giao |
| Duyệt/công bố | read_internal + approve + publish, người duyệt độc lập với contributors |
| Xem bằng chứng | read_evidence riêng khi cần, không cấp tự động cho mọi warehouse staff |
| Dừng công khai | suspend/archive riêng cho nhiệm vụ quản lý |

Mã đầy đủ dùng prefix `traceability.`. Role code do owner Identity chọn theo danh mục hiện có; không bắt buộc tạo tên role mới. Assignment WAREHOUSE scopeId là UUID kho thật. `/users/me.permissionGrants` phải giữ permissionCode,scopeType,scopeId; JWT giữ chuỗi `permission#WAREHOUSE#uuid` hoặc `permission#GLOBAL`. Không chỉ flat permissions. Traceability kiểm intersection JWT với current grants qua RPC Identity (không gọi /users/me nội bộ); /users/me vẫn phục vụ Web/BFF. Hai nguồn phải giữ tuple permission/scope; quyền role bootstrap ADMIN hiện có không chứng minh warehouse staff đã có assignment.

Nhân viên thực hiện cả nhập kho phải giữ các quyền Inventory nền INVENTORY_VIEW, WAREHOUSE_WORKBENCH_VIEW, INVENTORY_RECEIPT_CREATE đúng kho theo contract hiện có. Traceability grants được bổ sung cho cùng người, không thay thế các quyền này. Batch-context không yêu cầu cấp thêm CATALOG management/BATCH_VIEW chỉ để lấy Product ID.

Giữ thời hạn assignment, sync token mới, revoke/disable/thu hồi theo cơ chế service hiện có. Không sửa V20/V21 để gắn user test; nếu cần seed mới dùng migration mới hoặc API quản trị trong môi trường test. Cấp hai actor soạn/duyệt ở kho A, actor kho B, GLOBAL auditor và actor thiếu read_evidence; lưu ID test không lưu password/token trong báo cáo.

**IAM-OUT-05 — phần mới bắt buộc:** triển khai `/omamx.identity.v1.StaffAuthorization/GetCurrentStaffGrants` theo [shared proto](../../packages/proto/identity/v1/staff_authorization.proto) và [gRPC guide](traceability-workflow-grpc.md#identity--iam-out-05). Request rỗng; xác minh cả service credential trong authorization và employee JWT trong x-staff-authorization qua TLS. Không lấy userId do caller truyền để bỏ kiểm JWT. Trả current scoped traceability grants đầy đủ, cùng principalId/subject, EMPLOYEE ACTIVE; cap128grant/32KiB, budget2s. ErrorInfo phân biệt lỗi staff401/403 với service config503. Cập nhật gRPC interceptor đúng full method; HTTP security filter không tự bảo vệ RPC. Không log token/metadata đầy đủ. Tái sử dụng principal resolver/effective-grant query và revocation logic hiện có; không dùng RPC tài chính thay thế. Nghiệm thu GRPC-IAM-01…04 cùng T-SCOPE-03; proto/copies/codegen theo file canonical mới, không đổi method cũ.

**Nộp:** cấu hình/seed test actor và kết quả token+me đã lọc, case revoke/expiry và không mất scopeId. Nghiệm thu T-SCOPE-01…03, IAM-01. Thay đổi consumer GLOBAL-only thành scoped nằm ở Traceability, không yêu cầu Identity viết code đọc DB Traceability.

## 3. Fulfillment: FUL-TRC-01

**Mục đích:** trả lô đã đóng gói của đúng dòng đơn từ bản niêm phong bất biến, không từ phân bổ kho hiện tại.

**Đầu ra bắt buộc:** FUL-OUT-01: mỗi cặp dòng đơn/lô có UUID, SKU, mã lô và số lượng thực đóng; FUL-OUT-02: dữ liệu gắn đúng order/package/sealed-version/generation/digest/kho, truy vấn lại cùng version không đổi; FUL-OUT-03: phân biệt thiếu proof lịch sử với đơn không có hàng. Nếu đã có read API/proof đáp ứng đủ, tự tái sử dụng logic và map ra field14 bên dưới để consumer giữ contract đích; không cần hỏi lại trước khi làm adapter. Không bắt làm lại cách lưu hoặc truy vấn nội bộ.

**Provider:** [packing candidate](traceability-workflow-candidates/packing.proto), RPC GetPackageRevision hiện có. Request giữ package_id+sealed_task_version và xác thực Order service. Giữ nguyên fields1–13.

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

## 4. Order: ORD-TRC-01

**Mục đích:** người có quyền xem đơn đọc được danh sách nguồn gốc theo từng lô thực giao, không làm lỗi phần detail/payment/shipping đang có.

**Đầu ra bắt buộc:** ORD-OUT-01: Web lấy được từng line/batch/quantity và link/trạng thái truy xuất đúng đơn; ORD-OUT-02: chỉ chủ đơn hoặc guest capability hợp lệ được đọc; ORD-OUT-03: proof/Traceability thiếu hoặc lỗi không phá detail/lifecycle, không gắn nhầm lô. Owner đối chiếu API/read model hiện có trước; endpoint riêng bên dưới là contract đích để triển khai. Có thể dùng logic/read model hiện có phía sau route/adapter đúng contract đó; không cần consumer duyệt phương án nội bộ. Không tự thêm field vào stable detail hoặc đổi response ngoài phạm vi.

**Consumer:** Fulfillment proof candidate và gRPC `BatchLinkResolver.ResolveBatchLinks` theo [shared proto](../../packages/proto/traceability/v1/batch_links.proto), [gRPC guide](traceability-workflow-grpc.md). Không gọi internal REST resolver0.2.2 nữa. **Provider đích:** `GET /api/v1/orders/{order_id}/traceability` trong OpenAPI tích hợp, endpoint riêng; không sửa frozen commercial snapshot hoặc lifecycle chỉ vì lookup nguồn gốc lỗi.

1. Dùng proof exact version từ luồng notification hiện có hoặc read model đã được kiểm chứng. Kiểm order/task/package/sealed version/generation/digest/warehouse và invariant dòng-lô-số lượng. Đầu ra nhận diện theo `(order, package, sealed_version, order_line_id, batch_id)`; duplicate input không cộng số lượng lần hai, event cũ không ghi lùi phiên bản. Persist projection là một phương án; owner có thể tái sử dụng projection/proof reader đã có nếu bảo đảm cùng semantics, ownership, deadline và không lấy latest/live allocation thay lịch sử. GET không tạo command nghiệp vụ để “bù” dữ liệu thiếu.

   Tách kiểm chứng base package proof dùng cho lifecycle hiện có khỏi extension packed_lines dùng cho nguồn gốc. Base proof sai vẫn bị từ chối như baseline. Riêng packed_lines thiếu/sai pair/quantity không được làm adapter trả lỗi khiến event PACKING_SEALED và shipping kẹt: ghi traceability capability UNAVAILABLE, giữ xử lý base proof/lifecycle hợp lệ. Không nới kiểm tra parcel/media/order/generation/digest/seal để test traceability qua.
2. line_id lấy từ frozen Order snapshot; customer projection hiện không gán ItemID nên UI có thể dùng orderId-index. Subresource trả canonical order_line_id và sku_code. Current manifest unique SKU: Web/BFF giữ typed skuCode từ raw detail và map exact unique SKU, kiểm tổng quantity trước gắn link; không parse skuLabel, không join index/tên. Nếu duplicate/không chứng minh mapping, state UNAVAILABLE; không gắn tạm SKU khác. Không cần sửa frozen commercial snapshot hoặc stable detail để có bridge này.
3. Root state: không có proof và đơn chưa bị hủy trước packing → PACKING_PENDING; hủy trước packing → NOT_APPLICABLE; có proof hợp lệ → READY; proof/capability không xác minh được → UNAVAILABLE. Có sealed proof rồi hủy/hoàn sau đó vẫn cho đọc lô lịch sử; không xóa lịch sử. items rỗng có total0; READY không đồng nghĩa mọi link AVAILABLE.
4. Kiểm auth/ownership trước mọi peer call. Dùng cơ chế customer/guest detail của đúng release; tại dev1564a7bc guest session do Order cấp có issuer `order-service-guest-session`,audience `order-storefront`,typ `guest-session+jwt`,live session được xác minh. Payment read token issuer `order-service-guest`/aud `order-payment` không đủ đọc nguồn gốc đơn. Không thêm token/OTP/challenge mới.
5. Resolver là unary gRPC qua TLS với credential Order riêng trong authorization metadata. Response dùng items[].binding + enum status + optional qr_url; map absence sang null và bỏ prefix enum khi dựng DTO customer REST, không phát proto JSON thẳng ra Web. RESOURCE_EXHAUSTED/UNIMPLEMENTED/auth/timeout làm row UNAVAILABLE; không fallback HTTP. Resolver input chỉ `source_namespace=inventory`, batch UUID và SKU từ proof. ≤100/request, chia chunk và reconstruct output. Không cần product_id vì Traceability unique namespace+batch và đã verify product lúc publish. Timeout ≤1.5s/call, outer request budget5s; không retry đồng bộ trong cùng request, UI có thể retry read sau. Không gửi PII/actor token customer cho internal resolver. Concurrency≤2 và total budget5s, chunk chưa xong báoUNAVAILABLE. Quá1000line-batch rows hoặc response512KiB → rootUNAVAILABLE, không truncate thành READY.
6. AVAILABLE/NOT_AVAILABLE/NO_LONGER_PUBLIC/UNAVAILABLE từ resolver map nguyên nghĩa. Resolver auth/config lỗi làm các row UNAVAILABLE và có operational log an toàn, không biến thành NOT_AVAILABLE. Full Order DB lỗi trả503. Root total là số line-batch row, không là số sản phẩm.

ORD-OUT-04: triển khai gRPC client TLS theo canonical batch_links.proto và guide0.4.0; exact response echo/count, switch enum tường minh, optional qr_url presence, deadline/chunk budget và fail-closed mapping. Nếu đã làm REST consumer0.2.2, giữ application/proof/ownership logic và thay transport adapter; không thêm HTTP fallback. Nộp codegen/version + GRPC-RES-01…03 evidence, không cần chờ Traceability lên để code/test bằng fixture đúng proto có nhãn.

Historical mặc định không backfill từ allocation live. Đơn cũ thiếu verified proof báo UNAVAILABLE và hướng dẫn scan QR trên bao bì; không chặn order lifecycle, không yêu cầu xóa đơn. Backfill có thể dùng exact sealed snapshot provider nếu đã chứng minh, nhưng là job có review riêng, không tự chạy khi khách GET.

**Nộp:** bảng đáp ứng ORD-OUT-01…03, API/read model/version hiện có hoặc gap cần bổ sung, canonical mapping và evidence tests missing/foreign/guest/revoked session/proof old/mixed resolver. Chỉ cần migration/PR mới khi giải pháp có gap thực tế; không bắt tạo bảng projection mới. Nghiệm thu ORD-01…05, EVT-01/02. Không cần Order tạo ảnh QR.

## 5. Web và Gateway: WEB-TRC-01

**Đầu ra bắt buộc:** WEB-OUT-01: nhập kho thành công nối được tới đúng hồ sơ mà không nhập lại binding hoặc mất receipt; WEB-OUT-02: tem tải/in giải mã đúng public URL của lô được công bố; WEB-OUT-03: khách tra cứu công khai hoặc từ đơn với đúng các trạng thái/quyền; WEB-OUT-04: BFF/Gateway route và credential/status/no-store đúng contract, internal resolver không public. Tái sử dụng màn hình/component/route hiện có nếu đã đạt; chỉ sửa phần thiếu, không bắt xây lại workspace.

**Contracts:** v2 scoped admin, v2 public gateway hiện có, receipt delta, customer subresource. Không giữ form mock/sample như dữ liệu thật.

- Sau receipt APPLIED: giữ confirmation, lấy binding lô server trả. Product ID và binding đọc từ Traceability batch-context endpoint, server kiểm Inventory scope rồi Catalog exact-SKU binding; không dựa SKU detail chưa có provider hoặc Catalog public variant thiếu productId. Nếu không đủ quyền/lookup lỗi, giữ receipt thành công và hành động retry nguồn gốc riêng.
- By-batch lookup scoped → existing profile hoặc create form với binding không editable. Nhập bổ sung cùng lô không tạo profile mới. Hai người tạo conflict → lookup lại; không lặp receipt.
- Workspace Traceability có list/detail, draft editor, submit/review/publish theo permission. Gửi version/key đúng contract; không tự gọi approve/publish chỉ vì create thành công. Không tin nút disabled thay backend authorization.
- In dùng public_revision đã công bố, không nội dung draft. Chặn export khi pending/403/404/410/503; tải lại/in lại cùng qr_url. Preview tên/mã lô/mã nhập tay/link+QR; SVG/PNG/print stylesheet; scan/decode asset trước nghiệm thu. Không claim chống giả, không dùng ảnh QR từ dịch vụ ngoài.
- Bỏ banner nhập mã lớn ở home; footer trỏ `/trace`, trang nhập mã giữ thủ công; `/trace/{code}` anonymous. Trang sản phẩm chỉ hướng dẫn, không tự chọn lô. Order history dẫn detail; detail gọi subresource riêng và render nhiều lô/quantity theo canonical IDs.
- Cookie-to-bearer/CSRF phải nối theo BFF contract mới: mode cookie qua Identity trace-api/inventory-api, forward Origin đã kiểm tra và giữ Idempotency-Key. Web thêm typed permissionGrants cho Traceability giữ warehouse scopeId; không suy scope từ GLOBAL role hoặc whitelist cũ. Direct OIDC dùng bearer server-trusted. Public requests không ép đăng nhập; admin/customer requests không gửi JWT vào URL. No-store và status codes được giữ/normalize có contract, không trả200 lỗi upstream.
- Gateway thêm route admin v2 (records và batch-context), Inventory GET workbench và customer subresource vào đúng owner; giữ public v2 rewrite tới v1. Internal resolver không expose ra Kong public/admin wildcard; health/metrics nội bộ. Không chuyển route traffic mới trước capability deployment.

**Nộp:** screenshot desktop/mobile/print theo SHA, decode QR thành đúng link, HTTP/status evidence, không có token trong ảnh/log. Muse có thể làm browser localhost theo network route được kiểm tra; request cụ thể trang nào/thao tác/expected và thư mục output theo runID, không tự dùng production. Nghiệm thu WEB-01…04, T-QR-01…03.

### Web: upload chứng nhận OCOP

WEB-OUT-MEDIA: dùng Media REST upload-session/direct-to-storage/complete đã có trên dev; lấy fileId, tạo evidence.reference=document:<fileId> và certificate.evidence_ids theo schema Traceability hiện có. Không gửi object key/URL hay binary vào Traceability. Chỉ cho chọn bản scan được phép công khai vì purpose OCOP_CERTIFICATE hiện PUBLIC_READ; không tự upload nội bộ/PII. READY chưa đạt hoặc provider503 phải giữ draft/receipt, hiển thị trạng thái và retry riêng. Không giả UI đã nghiệm thu chỉ từ Go fixtures.

## 6. Catalog / Procurement / Kafka / Audit

### Catalog / Media: CAT-TRC-01 — tái sử dụng provider đã merge

CAT-OUT-01: giữ [CatalogService.GetTraceabilityBinding](../../packages/proto/catalog/v1/catalog.proto) của PR #175; input product_id + exact sku_code, output version_fingerprint và echo authoritative. Traceability dùng fingerprint nguyên trạng cho catalog_version. Không thêm CatalogTraceabilitySource/GetSkuBinding hoặc khôi phục REST đã gỡ.

CAT-OUT-02: bootstrap từ SKU dùng GetProductVariant(channel UNSPECIFIED) lấy product_id rồi GetTraceabilityBinding xác minh/version. Đảm bảo hidden/stopped/no-price còn đọc được, exact SKU, không fuzzy; không buộc UI biết trước Product UUID. Hai read + Inventory dùng budget 2 giây và cap reply 8 KiB; source thay đổi chặn submit/publish theo source hash cũ.

CAT-OUT-03: giữ x-internal-token theo CatalogAccess và CATALOG_INTERNAL_TOKEN ≥32 ký tự; không bỏ credential trong Traceability. Giữ port thực mặc định9005 thay ví dụ9090 trong plan. TLS nếu đã cấu hình, hoặc explicit private plaintext như guide; không mở RPC public. Nộp test provider thật cho binding/missing SKU/credential/hidden-no-price/fingerprint; không viết lại module đã đáp ứng.

MED-OUT-01: giữ [MediaInternalService](../../packages/proto/media/v1/media_service.proto), purpose OCOP_CERTIFICATE, fileId/READY/MIME/size/checksum theo [guide](traceability-workflow-grpc.md#media--chứng-nhận-ocop-theo-plan-dương). Traceability không nhận binary/raw object key; evidence.reference=document:<UUID> ánh xạ sang file_id. Không đổi schema certificates đã frozen.

MED-OUT-02: BindMediaToEntity(file_id, TraceabilityProfile, profile_id) phải retry/idempotent được; ràng buộc file không bị GC sau gắn kết. Traceability enqueue media_bind_jobs cùng publish transaction, worker gọi sau commit. Media hiện chỉ có một bound_entity: giữ file được bảo vệ khi dùng chung qua nhiều lô; không dùng trường đó làm quyền sở hữu thay IAM của Traceability.

MED-OUT-03: ghi rõ OCOP_CERTIFICATE baseline PUBLIC_READ và RPC9007 chưa có auth/TLS interceptor. Chỉ upload bản scan được phép công khai, chỉ expose RPC trong mạng tin cậy. Không gọi PublishMedia để đổi sang bucket Catalog. Nếu triển khai có policy khác, cung cấp cấu hình/test chính xác, không coi việc giấu reference trong Traceability là thu hồi file đã public.

**Đầu ra cuối:** SHA/provider config đang dùng, test IDs CAT-*/GRPC-CAT-*/MED-* và bằng chứng đã lọc. Provider có sẵn thì test/tái sử dụng; chỉ sửa gap thực sự trong phần phụ trách, không viết proposal RPC thay thế hoặc chờ ACCEPT.

### Các miền khác không mở rộng trong đợt này

- Procurement: không cần consumer auto-create profile; GoodsReceived hiện thiếu Batch UUID/genealogy. Receipt chỉ dữ liệu tham khảo, không đại diện full nguồn gốc thành phẩm.
- Kafka: không thêm topic hoặc activation event. Event packing v2 chỉ notification cho Order lấy proof, Traceability outbox schema service-local giữ nguyên. Xem section7 thiết kế toàn luồng.
- Audit service: local audit Traceability giữ bền vững. Ingestion chung chưa có acceptance, không bịa hash hoặc cấp credit nghiệm thu bằng broker ACK. Việc chưa nối shared Audit không được che bằng ghi “toàn bộ audit đã tích hợp”.

## 7. Kết quả cuối công việc — không phải phiếu xin duyệt trước

Không có bước gửi ACCEPT/DEVIATION hoặc chờ trả lời để bắt đầu. Bảng này chỉ điền trong kết quả cuối/PR sau khi đã làm và kiểm thử, không phải báo cáo sơ bộ cần duyệt.

| Requirement ID | Trạng thái cuối | Contract/API/version/SHA cung cấp cho consumer | Tái sử dụng hoặc gap đã bổ sung | Test IDs / evidence | Blocker còn lại nếu có |
| --- | --- | --- | --- | --- | --- |
| Mã OUT của task | VERIFIED / IMPLEMENTED_NOT_INTEGRATED / BLOCKED | Route/RPC và commit thực tế | Ghi ngắn gọn phần đã làm | Kết quả thật, tách rõ fixture/provider thật | Chỉ phần không thể tự xử lý |

Đầu ra công việc gồm code/config/contract/migration nếu có sửa, mapping adapter trong tài liệu, tests và bằng chứng theo SHA, hướng dẫn rollout tương thích. Đã có đầy đủ thì ghi `NO CODE CHANGE REQUIRED` và bằng chứng; không tạo PR rỗng. Chưa có peer chạy thật thì `IMPLEMENTED_NOT_INTEGRATED`, ghi những ca integration chưa chạy; không dừng triển khai chỉ vì chưa nhận tin nhắn.

| Task | Chế độ bắt đầu | Tiêu chí hoàn tất |
| --- | --- | --- |
| Traceability | DIRECT_IMPLEMENTATION, không chờ owner khác | T-* và contract Traceability |
| Inventory INV-TRC-01 | DIRECT_IMPLEMENTATION | INV-OUT-01…04, INV-* và phần scope liên quan |
| Identity IAM-TRC-01 | DIRECT_IMPLEMENTATION | IAM-OUT-01…04, IAM-*, cookie adapters và scoped grants |
| Catalog CAT-TRC-01 | DIRECT_IMPLEMENTATION | CAT-OUT-01…02, CAT-* |
| Fulfillment FUL-TRC-01 | DIRECT_IMPLEMENTATION | FUL-OUT-01…03, FUL-*, EVT-* |
| Order ORD-TRC-01 | DIRECT_IMPLEMENTATION | ORD-OUT-01…03, ORD-*, EVT-* |
| Web/Gateway WEB-TRC-01 | DIRECT_IMPLEMENTATION | WEB-OUT-01…04, WEB-*, T-QR-* |

Ví dụ kết quả cuối: `INV-TRC-01 IMPLEMENTED_NOT_INTEGRATED, SHA..., receipt binding đã có nên tái sử dụng; BatchSource thêm warehouse9; INV-01…04 PASS với provider test, scope end-to-end chưa chạy vì Identity test chưa lên`. Hoặc `IAM-TRC-01 VERIFIED, NO CODE CHANGE REQUIRED, SHA..., grants/config hiện có đáp ứng IAM-OUT-01…03, evidence...`.

Thay đổi contract đích ngoài scope phải được ghi amendment/version trong quy trình review; không tự đổi semantics rồi coi nhiệm vụ đã xong. Quy trình review PR và promotion stable sau test vẫn áp dụng, nhưng không phải một vòng hỏi đáp trước sửa code.
