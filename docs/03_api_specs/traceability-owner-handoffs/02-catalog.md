# PBL6 — Bàn giao Catalog: CAT-TRC-01

Version nghiệp vụ/contract: **0.3.0-draft** · Snapshot tài liệu: **`1d4fba6477032419d894a32b1dd6866490b3e95c`**.

**Bản phát lại0.3.0 thay toàn bộ phiếu0.2.2 cũ:** giữ nghiệp vụ, chuyển business service-to-service sang gRPC. Nếu đã làm REST nội bộ theo bản cũ, tái sử dụng logic và bổ sung gRPC adapter đúng contract. Web/BFF vẫn REST.

File này là bản `.md` độc lập để gửi qua Zalo. Đề bài, yêu cầu đầu ra, test và contract liên quan được nhúng bên dưới; không cần đọc lịch sử chat hoặc chờ phản hồi ACCEPT để bắt đầu code. Draft không đồng nghĩa chức năng đã chạy hoặc đã deploy.

## Nguồn tài liệu và cách lấy về

- Repository: [https://github.com/hvduong2k5/DUT.PBL6](https://github.com/hvduong2k5/DUT.PBL6). Nhánh tài liệu: `codex/traceability-order-flow`.
- **Đọc theo commit đã ghim**, không tự thay bằng HEAD của dev hoặc nhánh tài liệu đang tiếp tục thay đổi. Đây là nguồn tài liệu; owner giữ nhánh làm việc riêng cho service, không phải checkout đè công việc của mình.
- Có thể đọc toàn bộ đặc tả qua các link dưới. Người truy cập repository riêng tư cần quyền hiện có của nhóm. Contract nhúng trong file này vẫn đọc được khi chưa mở được link.

| Tài liệu liên quan trong repository | Link bất biến |
| --- | --- |
| `docs/03_api_specs/traceability-workflow-peer-handoff.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-peer-handoff.md) |
| `docs/03_api_specs/traceability-workflow-acceptance.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-acceptance.md) |
| `docs/03_api_specs/traceability-workflow-service-spec.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-service-spec.md) |
| `docs/03_api_specs/traceability-workflow-review-2026-11-25.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-review-2026-11-25.md) |
| `docs/03_api_specs/traceability-workflow-grpc.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-grpc.md) |
| `packages/proto/catalog/v1/traceability_source.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/packages/proto/catalog/v1/traceability_source.proto) |

Trong repository đã clone, các lệnh đọc sau không checkout/reset hay ghi đè source đang làm:

```powershell
git fetch origin codex/traceability-order-flow
git show 1d4fba6477032419d894a32b1dd6866490b3e95c:docs/03_api_specs/traceability-workflow-grpc.md
git show 1d4fba6477032419d894a32b1dd6866490b3e95c:docs/03_api_specs/traceability-workflow-peer-handoff.md
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

- Business calls giữa service dùng gRPC; REST Web/BFF và OIDC protocol giữ nguyên theo guide. Wire/case/ID và permission là contract đích, không đổi tên cho giống convention riêng. Không fake provider verification bằng mock, không sửa stable/generated examples hoặc migration cũ.
- HTTP error chuẩn mới `{code,message,requestId}`; mutation mới có Idempotency-Key. Legacy route giữ contract riêng. Internal RPC/call dùng danh tính service và transport tin cậy, không tin X-User-ID/role do browser tự gửi.
- Không kích hoạt/publish QR khi nhận packing hoặc GoodsReceived. Không thêm trừ kho, tạo profile, cấp mã hoặc payment side effect trong read flow.

## Phần việc của Catalog

### Catalog: CAT-TRC-01 — triển khai trực tiếp

CAT-OUT-01: triển khai `/omamx.catalog.v1.CatalogTraceabilitySource/GetSkuBinding` theo [shared proto](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/packages/proto/catalog/v1/traceability_source.proto). Input exact sku_code; output product_id/sku_code/catalog_version, không phụ thuộc giá hoặc trạng thái public, có hidden/stopped variants còn lưu. Traceability dùng cùng RPC bootstrap và reverify binding khi create/submit/publish. Version semantics giữ như verifier Product–SKU cũ; tái sử dụng query/domain/mapping, không sửa public variant response hoặc checkout RPC chỉ để bootstrap.

CAT-OUT-02: chỉ credential Traceability riêng được đọc qua TLS và một authorization Bearer metadata. INVALID_ARGUMENT/NOT_FOUND/UNAUTHENTICATED/PERMISSION_DENIED/UNAVAILABLE theo [gRPC guide](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-grpc.md#catalog--cat-out-0102), reply≤8KiB, budget trong tổng2s source check. Giữ REST verifier cũ cho legacy consumers nếu cần; target mới không gọi hoặc yêu cầu thêm REST exact-SKU nữa. Không gọi Catalog khi khách scan QR hoặc Order resolve links.

**CAT-OUT-03 — guard RPC mới:** CatalogAccess hiện chỉ kiểm prefix CatalogService/ và x-internal-token chung. Service mới `CatalogTraceabilitySource/` không được tự đi qua nhánh không kiểm auth; mở guard cho đúng full method và credential Traceability riêng theo contract, không tái sử dụng token checkout chung hoặc cho token đó đọc mặc định. Giữ auth cũ của các method khác. Nghiệm thu CAT-01…03 và GRPC-CAT-01/02, gồm missing/wrong/duplicate credential, wrong caller, hidden/no-price/version và DB timeout. Thêm canonical proto vào pipeline copy/codegen/service registration hiện có; không hand-edit generated Java.

Nếu đã làm REST0.2.2, giữ application/query logic và thêm adapter gRPC đúng wire; không yêu cầu owner phải viết lại DB hoặc phản hồi ACCEPT trước code.

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| CAT-01 | Exact-SKU bootstrap active/hidden/stopped và không có giá | Trả đúng Product/SKU/version, không cần public/saleable/price |
| CAT-02 | Credential sai, SKU không tồn tại, khác case, DB lỗi | Không fuzzy match; RPC INVALID_ARGUMENT/UNAUTHENTICATED/PERMISSION_DENIED/NOT_FOUND/UNAVAILABLE đúng nguyên nhân; Traceability map sang HTTP theo guide, không lộ internal binding public |
| CAT-03 | Bootstrap Product rồi verifier cũ phát hiện SKU đổi binding/version | Create/submit/publish chặn conflict, không tin context cũ từ browser |
| T-CTX-01 | Batch ở kho khác gọicontext, expired/restricted draft source, peer timeout | Kiểm Inventory và scope trước trả Product;403 ngoài kho; context không đồng nghĩa publish; timeout503, receipt giữ nguyên |
| GRPC-CAT-01 | Exact SKU active/hidden/stopped/no-price, wrong/missing/duplicate service credential | Product/SKU/version authoritative, TLS + authorization Bearer riêng; interceptor bảo vệ cả full method mới |
| GRPC-CAT-02 | NOT_FOUND/timeout/UNIMPLEMENTED/DB failure, version đổi | Phân biệt absent/conflict/503, không HTTP fallback hoặc constant version |

Nộp kết quả cuối/PR: OUT đã đáp ứng, SHA, phần tái sử dụng/bổ sung, mapping adapter nếu có, PASS/FAIL/SKIPPED từng test và bằng chứng đã lọc. Chưa có peer chạy thật vẫn code/test bằng fixture đúng contract, ghi `IMPLEMENTED_NOT_INTEGRATED`; không báo full E2E bằng mock. Chỉ báo blocker vượt scope, không gửi phương án rồi chờ duyệt trước sửa code.

Đọc AGENTS.md và rule service trong repository trước khi sửa. Commit mới dùng tiếng Anh, giữ một type/scope; contract thay trước hoặc cùng code, không sửa migration cũ/generated rows sai quy trình. Không tự push dev/release hoặc deploy production; tạo PR theo quy trình nhóm.

## Quy định transport mới và contract gRPC

Chốt27/11/2026 theo định hướng nhóm: business calls East-West dùng gRPC; REST là bề mặt Web/Mobile. Bản này thay transport nội bộ của target0.2.2, giữ nguyên nghiệp vụ, scope kho, state machine, nguồn dữ liệu, snapshot/hash và dữ liệu public. Các phiếu0.2.2 cũ chỉ là lịch sử; phát lại phiếu0.3.0 từ commit nguồn mới, không dùng lẫn contract giữa hai bản.

## Ranh giới và contract authoritative

| Kết nối | Contract đích | Owner |
| --- | --- | --- |
| Traceability → Inventory | `GetTraceabilityBatch`, candidate field9 warehouse hiện có | Inventory |
| Traceability → Catalog | [CatalogTraceabilitySource.GetSkuBinding](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/packages/proto/catalog/v1/traceability_source.proto) | Catalog |
| Traceability → Identity | [StaffAuthorization.GetCurrentStaffGrants](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/packages/proto/identity/v1/staff_authorization.proto) | Identity |
| Order → Traceability | [BatchLinkResolver.ResolveBatchLinks](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/packages/proto/traceability/v1/batch_links.proto) | Traceability |
| Order → Fulfillment | packing gRPC exact sealed proof, candidate field14 hiện có | Fulfillment/Order |
| Web/Mobile → public/admin/customer APIs | REST/OpenAPI hiện có; scoped staff vẫn v2 | Web/Gateway và backend tương ứng |

Ba file proto mới trong `packages/proto/` là shared integration target, chưa phải tuyên bố provider đã chạy. Không đổi service/method/field cũ trong catalog.proto hoặc identity.proto. Owner thêm implementation/copies/codegen của file mới bằng pipeline hiện có, không sửa tay generated code.

Catalog và Identity hiện đã có `sourceSets.main.proto.srcDir '../../packages/proto'`: lấy đúng file canonical mới từ source commit/khối proto trong phiếu vào đường dẫn ghi ở bảng, rồi chạy pipeline `generateProto`/build của service. Không chép thêm một bản cùng definition vào src/main/proto. Identity implementation cần google.rpc.ErrorInfo + io.grpc.protobuf.StatusProto để tạo rich status details; tái sử dụng dependency đang có hoặc bổ sung dependency tương thích gRPC/protobuf đã pin, không đổi wire để né detail bắt buộc.

Pipeline Java hiện pin protoc3.25.3/gRPC1.62.2; ba proto mới đã được kiểm codegen với đúng hai version này. Không dùng toolchain Python của Traceability để thay pin Java.

### Listener TLS mới và tương thích deployment

Baseline Catalog9005 cấu hình security.enabled=false; Identity9016 chưa có TLS trong application.yml. Không đổi transport của các port legacy và làm hỏng client đang dùng. Chốt listener TLS riêng cho hai RPC mới: Catalog mặc định9105 (`CATALOG_TRACEABILITY_GRPC_PORT`), Identity mặc định9116 (`IDENTITY_TRACEABILITY_GRPC_PORT`); cho phép override env khi hạ tầng yêu cầu, ghi địa chỉ thực trong delivery. Chỉ bind/register service mới trên listener TLS đó, không mở thêm plaintext bản sao của RPC mới qua annotation auto-registration lên legacy listener.

Server env: Catalog dùng `CATALOG_TRACEABILITY_GRPC_TLS_CERT_FILE/KEY_FILE/TOKEN`, Identity dùng `IDENTITY_TRACEABILITY_GRPC_TLS_CERT_FILE/KEY_FILE/TOKEN`. TOKEN là credential caller Traceability riêng≥32bytes (có thể map alias tới secret Traceability hiện có đã riêng), không dùng checkout/financial token chung. Cert SAN khớp DNS service, key chỉ service đọc. New server phải tự gắn interceptor xác thực; annotation global interceptor của server legacy không tự bảo vệ một manual TLS server. Gắn lifecycle vào ứng dụng để drain rồi stop khi shutdown và fail startup nếu bật mà thiếu TLS/secret. Không publish host/public port ở deployment; chỉ network nội bộ.

Traceability client cấu hình theo địa chỉ thực, ví dụ trong Docker network `catalog-service:9105` và `identity-service:9116`, trusted CA tương ứng. Owner cung cấp env example/port/CA contract và test legacy RPC còn hoạt động. Có thể tái sử dụng TLS listener riêng đã tồn tại nếu bảo đảm cùng kết quả, không yêu cầu consumer đổi method/message/auth hoặc hỏi duyệt trước.

Proxy BFF cookie-to-bearer tiếp tục chuyển tiếp request REST của Web tới Gateway theo north-south contract, giữ Origin/CSRF. Nó không phải domain read East-West. OIDC discovery/JWKS vẫn HTTPS theo giao thức xác thực; không đổi sang gRPC. Kafka/outbox tiếp tục thông báo publication/status sau local transaction; không dùng event để tự tạo hồ sơ, publish QR hoặc thay kiểm quyền/nguồn đồng bộ. Không thêm request/reply bằng Kafka trong đợt này.

## Catalog — CAT-OUT-01/02

RPC full method: `/omamx.catalog.v1.CatalogTraceabilitySource/GetSkuBinding`.

- Input exact `sku_code`,1–100 ký tự, giữ case, không trim/fuzzy; canonical response product UUID, đúng SKU và opaque `catalog_version` không rỗng.
- Bao gồm SKU ẩn/ngừng bán còn lưu; không yêu cầu giá, trạng thái public hoặc checkout eligibility. Tái sử dụng query/verifier nội bộ đã có.
- Traceability dùng cùng RPC khi bootstrap và create/submit/publish, tự đối chiếu product_id với binding đã lưu. Version phải khớp semantics verifier cũ; không thay version bằng thời gian hoặc giá trị hằng.
- CatalogAccess hiện chỉ bảo vệ prefix CatalogService/ bằng token chung x-internal-token. RPC mới phải được thêm guard riêng đúng full method + dedicated Traceability Bearer; test không credential phải fail closed, không để service mới lọt nhánh không kiểm auth.
- TLS, credential Traceability riêng qua một metadata `authorization: Bearer ...`. Không dùng staff JWT làm service credential. Giữ credential hiện có nếu đã riêng và đủ an toàn; chuyển cách truyền từ X-Internal-Token sang gRPC metadata.
- INVALID_ARGUMENT: input sai; NOT_FOUND: SKU không tồn tại chính xác; UNAUTHENTICATED: credential invalid; PERMISSION_DENIED: known caller không có capability; UNAVAILABLE: DB/provider lỗi. Reply≤8KiB. Context Inventory+Catalog có budget chung2s; không retry vô hạn.

`GetProductVariant` hiện có thiếu catalog_version và phụ thuộc saleability/price; không được dùng thay RPC này khi chưa đáp ứng đủ semantics. Legacy Catalog internal REST có consumer khác vẫn giữ tương thích; target mới không gọi các endpoint đó và không yêu cầu bổ sung REST exact-SKU nữa.

## Identity — IAM-OUT-05

RPC full method: `/omamx.identity.v1.StaffAuthorization/GetCurrentStaffGrants`.

Hai credential riêng qua TLS:

| Metadata | Mục đích |
| --- | --- |
| `authorization: Bearer <Traceability-to-Identity credential>` | Xác thực caller service; chỉ Traceability được gọi RPC này |
| `x-staff-authorization: Bearer <employee access token>` | Xác thực actor nhân viên; không nhận principalId/role từ caller để bỏ qua kiểm JWT |

Mỗi metadata đúng một giá trị, không log/interceptor dump token. Employee JWT≤16KiB. Request message rỗng; Identity tự xác minh JWT issuer/audience/signature/expiry, suy principal từ mapping authoritative, kiểm EMPLOYEE/ACTIVE, đọc effective grants hiện hành bằng query IAM hiện có. Lọc assignment expired/revoked, disabled role và giữ scopeId; không gọi lại Traceability, không role ADMIN bypass. Không sửa claims/refresh semantics hiện có.

Response gồm principal_id/idp_subject/principal_type/provisioning_status và đầy đủ grant traceability.* hiện hành, tối đa128grant/32KiB; vượt cap trả RESOURCE_EXHAUSTED, không truncate. GLOBAL scope_id rỗng, WAREHOUSE có UUID. Traceability vẫn xác minh JWT cục bộ, đối chiếu response ID/subject và intersection token/current grants theo permission+scope; không dùng response để mở rộng token hoặc bỏ kiểm hiện hành. Call budget2s, không cache grants qua request.

Auth failures phải có `google.rpc.ErrorInfo`, domain `identity.omama.internal` để phân biệt lỗi service với actor:

| gRPC code | ErrorInfo.reason | Consumer xử lý |
| --- | --- | --- |
| UNAUTHENTICATED | STAFF_UNAUTHENTICATED | HTTP401, xác thực lại nhân viên |
| PERMISSION_DENIED | STAFF_FORBIDDEN | HTTP403, actor không có capability/EMPLOYEE ACTIVE |
| UNAUTHENTICATED | SERVICE_AUTHENTICATION_FAILED | HTTP503, sai cấu hình service; không bắt nhân viên login để che lỗi |
| PERMISSION_DENIED | SERVICE_PERMISSION_DENIED | HTTP503, service không được cấp RPC |
| UNAVAILABLE / DEADLINE_EXCEEDED / RESOURCE_EXHAUSTED / UNIMPLEMENTED | Không yêu cầu detail | HTTP503, fail closed |

Auth error thiếu detail/unknown reason →503. JWT hỏng phát hiện cục bộ →401 trước RPC. Grant rỗng cho nhân viên hợp lệ trả200/RPC OK; action không đủ quyền trả403 tại Traceability. ID/subject mismatch hoặc response malformed →503. Không dùng `CheckSpecializedPermission` tài chính để thay current grants.

## Traceability resolver — ORD-OUT-04

RPC full method: `/omamx.traceability.v1.BatchLinkResolver/ResolveBatchLinks`.

- Order kiểm ownership/guest capability trước RPC, dùng credential Order riêng≥32bytes qua đúng một authorization Bearer trên TLS. Không route qua Kong/public wildcard; không gửi token qua URL.
- Input1–100 bindings, inventory namespace, canonical Batch UUID, exact SKU; body≤256KiB. Read-only, không Idempotency-Key. Deadline caller≤1.5s, server tự giới hạn1.5s; Order vẫn giữ budget toàn5s/concurrency≤2 và response caps đã chốt.
- Response `items[i].binding` lặp lại input i; enum status không UNSPECIFIED; total=len(input), giữ duplicates/thứ tự. qr_url có presence chỉ AVAILABLE/NO_LONGER_PUBLIC. Map protobuf absence → JSON null cho NOT_AVAILABLE/UNAVAILABLE; map enum prefix sang bốn status REST đã chốt. Không phát proto JSON trực tiếp ra customer vì tên enum/wrapper khác DTO customer.
- Order dùng switch enum tường minh: AVAILABLE→AVAILABLE, NOT_AVAILABLE→NOT_AVAILABLE/null, NO_LONGER_PUBLIC→NO_LONGER_PUBLIC, UNAVAILABLE→UNAVAILABLE/null. UNSPECIFIED/unknown enum, thiếu URL bắt buộc, total/length/binding echo sai → chunk UNAVAILABLE; không cắt prefix rồi chấp nhận status lạ hoặc ghép theo index không đối chiếu binding. URL phải hợp lệ theo public origin đã cấu hình, không dựng từ batchCode/PII.
- AVAILABLE chỉ khi public revision eligible; NOT_AVAILABLE cho missing/unpublished/SKU mismatch; NO_LONGER_PUBLIC có URL đã public nếu dựng an toàn; corrupt public row→UNAVAILABLE. DB lỗi toàn bộ→UNAVAILABLE gRPC status, không giả reply toàn NOT_AVAILABLE. Không gọi peer khi resolve và không chặn Order lifecycle/shipping nếu lỗi trace.
- INVALID_ARGUMENT: binding/count sai; RESOURCE_EXHAUSTED: vượt message cap; UNAUTHENTICATED: credential absent/invalid; UNAVAILABLE/DEADLINE_EXCEEDED: storage/budget; PERMISSION_DENIED chỉ khi đã biết caller nhưng thiếu capability. Safe status messages, không lộ SQL/credential/evidence. Header `x-request-id` cho RPC được xử lý, sinh mới khi thiếu/invalid; runtime decoding/size rejection có thể do gRPC runtime trả trước handler.

Internal REST resolver của0.2.2 chưa phát hành không được đăng ký nữa. Không thêm REST fallback hoặc dual-write. Public QR/admin/customer REST vẫn giữ error/header/body riêng như OpenAPI.

## Cấu hình, rollout và đầu ra owner

Traceability bổ sung Catalog/Identity gRPC target+CA+credential; giữ OIDC issuer HTTPS và Inventory TLS. Resolver có listener nội bộ `GRPC_ADDR` (ví dụ9003), TLS cert/key bắt buộc ở integration khi bật; sandbox chỉ loopback có thể dùng plaintext explicit. Feature flags scoped/resolver vẫn độc lập và mặc định false. Không mở insecure integration hoặc fallback HTTP khi RPC thiếu.

Owner Catalog triển khai RPC GetSkuBinding; Identity triển khai current staff grants + auth details và giữ phần BFF/assignments; Order chuyển consumer resolver sang generated gRPC và map đúng DTO. Inventory/Fulfillment giữ đầu ra đã chốt. Web tiếp tục REST; không cần gRPC client trong browser.

Nghiệm thu mới: TLS/auth/credentials riêng; exact SKU hidden/no-price; JWT/current-grant intersection và revoke; ID mismatch; RPC timeout/UNIMPLEMENTED fail closed; resolver100/101/byte caps/duplicates/status/presence/DB down; lifecycle/shutdown hai listener; kiểm không còn internal business REST trong live adapters. Peer fixtures phải ghi nhãn; provider thật vẫn cần test trước bật flags. Không yêu cầu owner phản hồi ACCEPT trước code.

## Proto target nhúng: `packages/proto/catalog/v1/traceability_source.proto`

Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng. Không hand-edit generated code.

```proto
// Integration target 0.3.0-draft. Provider: Catalog; consumer: Traceability.
syntax = "proto3";
package omamx.catalog.v1;
option go_package = "github.com/omamx/proto/catalog/v1;catalogv1";
option java_multiple_files = true;
option java_package = "com.omamx.proto.catalog.v1";

// TLS plus one authorization: Bearer <dedicated Traceability credential>.
// This is not a public saleability/price lookup. See the normative gRPC guide.
service CatalogTraceabilitySource {
  rpc GetSkuBinding(TraceabilitySkuBindingRequest) returns (TraceabilitySkuBindingResponse);
}
message TraceabilitySkuBindingRequest {
  string sku_code = 1;
}
message TraceabilitySkuBindingResponse {
  string product_id = 1;
  string sku_code = 2;
  // Same opaque version semantics as the legacy Product-SKU verifier.
  string catalog_version = 3;
}
```

## Tính toàn vẹn bản gửi

Nguồn commit: `1d4fba6477032419d894a32b1dd6866490b3e95c`. SHA-256 tập source: `e0028c449c799162bcb6f03dadbd5d5c7785a37582e0e4e1a27117fecd092646`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
