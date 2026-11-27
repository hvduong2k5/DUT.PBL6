# PBL6 — Bàn giao Fulfillment: FUL-TRC-01

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
| `docs/03_api_specs/traceability-workflow-candidates/packing.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-candidates/packing.proto) |

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

## Phần việc của Fulfillment

**Mục đích:** trả lô đã đóng gói của đúng dòng đơn từ bản niêm phong bất biến, không từ phân bổ kho hiện tại.

**Đầu ra bắt buộc:** FUL-OUT-01: mỗi cặp dòng đơn/lô có UUID, SKU, mã lô và số lượng thực đóng; FUL-OUT-02: dữ liệu gắn đúng order/package/sealed-version/generation/digest/kho, truy vấn lại cùng version không đổi; FUL-OUT-03: phân biệt thiếu proof lịch sử với đơn không có hàng. Nếu đã có read API/proof đáp ứng đủ, tự tái sử dụng logic và map ra field14 bên dưới để consumer giữ contract đích; không cần hỏi lại trước khi làm adapter. Không bắt làm lại cách lưu hoặc truy vấn nội bộ.

**Provider:** [packing candidate](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-candidates/packing.proto), RPC GetPackageRevision hiện có. Request giữ package_id+sealed_task_version và xác thực Order service. Giữ nguyên fields1–13.

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

## Proto target nhúng: `docs/03_api_specs/traceability-workflow-candidates/packing.proto`

Candidate full definition: compile riêng, cập nhật canonical/copies/codegen khi implement; không compile cạnh canonical cùng message. Không hand-edit generated code.

```proto
// Implementation target 0.3.0-draft, 2026-11-27. NOT canonical/runtime code.
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

Nguồn commit: `1d4fba6477032419d894a32b1dd6866490b3e95c`. SHA-256 tập source: `d6ad6c38ea0131e50cb28a7d97ecea70f73ae74f740beb0ea954a1e03a8e0747`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
