# PBL6 — Bàn giao Identity / Keycloak: IAM-TRC-01

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
| `docs/03_api_specs/admin-stable.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/admin-stable.openapi.yaml) |
| `docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-draft.openapi.yaml) |
| `packages/proto/identity/v1/staff_authorization.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/packages/proto/identity/v1/staff_authorization.proto) |

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

## Phần việc của Identity / Keycloak

**Mục đích:** cấp các permission đã có cho đúng nhân viên/kho. Không thêm role bypass hoặc scope claim mới.

**Đầu ra bắt buộc:** IAM-OUT-01: token và `/users/me` biểu diễn đúng permission gắn scopeId kho; IAM-OUT-02: nhân viên soạn/duyệt được gán đúng nhóm quyền, thời hạn và thu hồi có hiệu lực theo contract; IAM-OUT-03: có bộ actor test kiểm kho A/B và tách nhiệm vụ. Phần IAM grants có thể chỉ cần cấu hình/test. IAM-OUT-04: trong cookie mode cung cấp các transport trong [BFF contract](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml), guard ACTIVE EMPLOYEE cho cả v2, giữ Origin/CSRF và chuyển cookie thành bearer server-side; không mặc định Identity chỉ cần cấu hình. Direct OIDC mode không cần cookie adapter. Không bắt tạo role mới.

| Bundle nhiệm vụ | Permissions được gán |
| --- | --- |
| Soạn hồ sơ | read_internal + create + edit; thêm submit nếu được giao |
| Duyệt/công bố | read_internal + approve + publish, người duyệt độc lập với contributors |
| Xem bằng chứng | read_evidence riêng khi cần, không cấp tự động cho mọi warehouse staff |
| Dừng công khai | suspend/archive riêng cho nhiệm vụ quản lý |

Mã đầy đủ dùng prefix `traceability.`. Role code do owner Identity chọn theo danh mục hiện có; không bắt buộc tạo tên role mới. Assignment WAREHOUSE scopeId là UUID kho thật. `/users/me.permissionGrants` phải giữ permissionCode,scopeType,scopeId; JWT giữ chuỗi `permission#WAREHOUSE#uuid` hoặc `permission#GLOBAL`. Không chỉ flat permissions. Traceability kiểm intersection JWT với current grants qua RPC Identity (không gọi /users/me nội bộ); /users/me vẫn phục vụ Web/BFF. Hai nguồn phải giữ tuple permission/scope; quyền role bootstrap ADMIN hiện có không chứng minh warehouse staff đã có assignment.

Nhân viên thực hiện cả nhập kho phải giữ các quyền Inventory nền INVENTORY_VIEW, WAREHOUSE_WORKBENCH_VIEW, INVENTORY_RECEIPT_CREATE đúng kho theo contract hiện có. Traceability grants được bổ sung cho cùng người, không thay thế các quyền này. Batch-context không yêu cầu cấp thêm CATALOG management/BATCH_VIEW chỉ để lấy Product ID.

Giữ thời hạn assignment, sync token mới, revoke/disable/thu hồi theo cơ chế service hiện có. Không sửa V20/V21 để gắn user test; nếu cần seed mới dùng migration mới hoặc API quản trị trong môi trường test. Cấp hai actor soạn/duyệt ở kho A, actor kho B, GLOBAL auditor và actor thiếu read_evidence; lưu ID test không lưu password/token trong báo cáo.

**IAM-OUT-05 — phần mới bắt buộc:** triển khai `/omamx.identity.v1.StaffAuthorization/GetCurrentStaffGrants` theo [shared proto](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/packages/proto/identity/v1/staff_authorization.proto) và [gRPC guide](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-grpc.md#identity--iam-out-05). Request rỗng; xác minh cả service credential trong authorization và employee JWT trong x-staff-authorization qua TLS. Không lấy userId do caller truyền để bỏ kiểm JWT. Trả current scoped traceability grants đầy đủ, cùng principalId/subject, EMPLOYEE ACTIVE; cap128grant/32KiB, budget2s. ErrorInfo phân biệt lỗi staff401/403 với service config503. Cập nhật gRPC interceptor đúng full method; HTTP security filter không tự bảo vệ RPC. Không log token/metadata đầy đủ. Tái sử dụng principal resolver/effective-grant query và revocation logic hiện có; không dùng RPC tài chính thay thế. Nghiệm thu GRPC-IAM-01…04 cùng T-SCOPE-03; proto/copies/codegen theo file canonical mới, không đổi method cũ.

**Nộp:** cấu hình/seed test actor và kết quả token+me đã lọc, case revoke/expiry và không mất scopeId. Nghiệm thu T-SCOPE-01…03, IAM-01. Thay đổi consumer GLOBAL-only thành scoped nằm ở Traceability, không yêu cầu Identity viết code đọc DB Traceability.

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| T-SCOPE-01 | makerA create/read/edit profileA rồi thửB | A thành công, B403 không nội dung; create không tin warehouse browser |
| T-SCOPE-02 | Revoke scope sau command rồi replay cùng key; đổi scope khi dùng cursor | Replay403; cursor authorization đổi bị từ chối; không rò cached response |
| T-SCOPE-03 | Token có quyềnA nhưng current-grants RPC chỉB; Identity timeout; forged headers | Không cấp quyềnA, timeout503, header không đổi actor/scope |
| IAM-01 | Grant/revoke/expire WAREHOUSE, active/inactive | JWT/me giữ scopeId; quyền mới/thu hồi theo chính sách thật, không role bypass |
| IAM-02 | Cookie → trace v2 và Inventory receipt/workbench qua Identity; CUSTOMER token vào v2 | Đúng bearer server-side, ACTIVE EMPLOYEE required, CUSTOMER403, không token ở browser URL/log |
| WEB-05 | Thiếu/sai Origin trên cookie mutation; body256KiB/profile proxy; proxy response quá lớn | Origin403; không disable CSRF; header hợp lệ được forward; bounded response502 không giả thành công |
| GRPC-IAM-01 | Service credential + employee JWT, wrong userId/subject/customer/inactive | Identity xác minh hai danh tính, Traceability kiểm ID/subject và EMPLOYEE ACTIVE, không lấy ID từ request body |
| GRPC-IAM-02 | STAFF_UNAUTHENTICATED/FORBIDDEN và SERVICE_AUTHENTICATION_FAILED/PERMISSION_DENIED; missing details | ErrorInfo staff→401/403; service/missing/unknown→503; không re-login loop để che lỗi config |
| GRPC-IAM-03 | Reply129grant hoặc >32KiB, deadline và old provider |503 fail closed, không truncate/cached token fallback |
| GRPC-IAM-04 | Mixed GLOBAL/WAREHOUSE, revoked grant và department | Intersection tuple đúng scope hiện hành, không role/cross-product bypass; /users/me vẫn phục vụ Web |

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

## Contract HTTP nhúng — không cần tải thêm file để đọc schema

Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.

```yaml
openapi: 3.1.0
info:
  title: PBL6 Identity / Keycloak traceability handoff
  version: 0.3.0-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 1d4fba6477032419d894a32b1dd6866490b3e95c. Direct
    implementation target, not runtime acceptance. All references are local to this embedded contract.
tags:
- name: Identity / Keycloak
  description: Owner-specific provider and dependency contracts
servers:
- url: http://127.0.0.1
paths:
  /api/v2/admin/trace-api/records:
    get:
      operationId: listTraceabilityProfilesScopedV2IdentityBff
      parameters:
      - name: cursor
        in: query
        schema:
          type: string
          maxLength: 256
      - name: page_size
        in: query
        schema:
          type: integer
          minimum: 1
          maximum: 100
          default: 20
      responses:
        '200':
          description: Instance-scoped list; internal evidence requires its own permission.
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required:
                - items
                - next_cursor
                - total
                properties:
                  items:
                    type: array
                    items:
                      $ref: '#/components/schemas/Scoped_Profile'
                  next_cursor:
                    type:
                    - string
                    - 'null'
                  total:
                    type: integer
                    minimum: 0
                    description: Rows returned in this page, not a count of all matching database rows.
              examples:
                empty:
                  value:
                    items: []
                    next_cursor: null
                    total: 0
                success:
                  value:
                    items:
                    - id: 11111111-1111-4111-8111-111111111111
                      mode: integration
                      product_id: 22222222-2222-4222-8222-222222222222
                      sku_code: MX-001
                      source_namespace: inventory
                      batch_id: 33333333-3333-4333-8333-333333333333
                      warehouse_id: 44444444-4444-4444-8444-444444444444
                      qr_code: abcdefghijklmnopqrstuv
                      qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                      version: 1
                      status: ACTIVE
                      public_revision: null
                      revisions:
                      - number: 1
                        status: DRAFT
                        content:
                          product_name: Me xung LOT-A
                          producer: Co so O Ma
                          workshop_location: Hue
                          artisan_name: null
                          process: Che bien va dong goi theo ho so lo
                          ingredients:
                          - name: Me
                            origin: Quang Dien
                          evidence:
                          - id: 55555555-5555-4555-8555-555555555555
                            reference: document:66666666-6666-4666-8666-666666666666
                            kind: CERTIFICATE
                            description: Ho so OCOP da kiem tra thu cong
                          certificates:
                          - id: 77777777-7777-4777-8777-777777777777
                            type: OCOP
                            number: OCOP-TEST-01
                            issuer: Don vi cap chung nhan
                            scope: BATCH
                            scope_id: 33333333-3333-4333-8333-333333333333
                            issued_on: '2026-01-01'
                            valid_until: '2027-12-31'
                            no_expiry_stated: false
                            review_due_at: null
                            stars: 4
                            verification_method: MANUAL
                            verified_at: '2026-11-20T00:00:00Z'
                            revoked: false
                            evidence_ids:
                            - 55555555-5555-4555-8555-555555555555
                          crafting_video_url: null
                    next_cursor: null
                    total: 1
          headers:
            Cache-Control:
              schema:
                type: string
                enum:
                - no-store
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: listTraceabilityProfilesScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
    post:
      operationId: createTraceabilityProfileScopedV2IdentityBff
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required:
              - product_id
              - sku_code
              - source_namespace
              - batch_id
              - content
              properties:
                product_id:
                  type: string
                  format: uuid
                sku_code:
                  type: string
                  minLength: 1
                  maxLength: 100
                source_namespace:
                  type: string
                  minLength: 1
                  maxLength: 100
                batch_id:
                  type: string
                  minLength: 1
                  maxLength: 160
                content:
                  $ref: '#/components/schemas/Scoped_Content'
            example:
              product_id: 22222222-2222-4222-8222-222222222222
              sku_code: MX-001
              source_namespace: inventory
              batch_id: 33333333-3333-4333-8333-333333333333
              content:
                product_name: Me xung LOT-A
                producer: Co so O Ma
                workshop_location: Hue
                artisan_name: null
                process: Che bien va dong goi theo ho so lo
                ingredients:
                - name: Me
                  origin: Quang Dien
                evidence:
                - id: 55555555-5555-4555-8555-555555555555
                  reference: document:66666666-6666-4666-8666-666666666666
                  kind: CERTIFICATE
                  description: Ho so OCOP da kiem tra thu cong
                certificates:
                - id: 77777777-7777-4777-8777-777777777777
                  type: OCOP
                  number: OCOP-TEST-01
                  issuer: Don vi cap chung nhan
                  scope: BATCH
                  scope_id: 33333333-3333-4333-8333-333333333333
                  issued_on: '2026-01-01'
                  valid_until: '2027-12-31'
                  no_expiry_stated: false
                  review_due_at: null
                  stars: 4
                  verification_method: MANUAL
                  verified_at: '2026-11-20T00:00:00Z'
                  revoked: false
                  evidence_ids:
                  - 55555555-5555-4555-8555-555555555555
                crafting_video_url: null
      responses:
        '201':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 1
                    status: ACTIVE
                    public_revision: null
                    revisions:
                    - number: 1
                      status: DRAFT
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: createTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.create
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    get:
      operationId: getTraceabilityProfileScopedV2IdentityBff
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 1
                    status: ACTIVE
                    public_revision: null
                    revisions:
                    - number: 1
                      status: DRAFT
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: getTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: createTraceabilityRevisionScopedV2IdentityBff
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required:
              - expected_version
              - from_revision
              properties:
                expected_version:
                  type: integer
                  format: int64
                  minimum: 1
                from_revision:
                  type: integer
                  minimum: 1
                reason:
                  type: string
                  minLength: 1
                  maxLength: 1000
                  description: Required when withdrawing an existing working revision
            example:
              expected_version: 5
              from_revision: 1
              reason: Prepare a corrected revision
      responses:
        '201':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 6
                    status: ACTIVE
                    public_revision: 1
                    revisions:
                    - number: 1
                      status: PUBLISHED
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
                    - number: 2
                      status: DRAFT
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: createTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.edit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    patch:
      operationId: replaceDraftContentScopedV2IdentityBff
      description: Replaces the whole Draft content snapshot; not a JSON merge patch. Does not change binding or
        accept contributor identities from the caller. All operations require read_internal plus their command permission
        for the same authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill
        and contributor policy. Identity transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id
        and Traceparent; no automatic mutation retry, no redirects. Preserve backend status/body and no-store. Next
        BFF must pass validated Origin to Identity for cookie mutations.
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required:
              - expected_version
              - content
              properties:
                expected_version:
                  type: integer
                  format: int64
                  minimum: 1
                content:
                  $ref: '#/components/schemas/Scoped_Content'
            example:
              expected_version: 1
              content:
                product_name: Me xung LOT-A
                producer: Co so O Ma
                workshop_location: Hue
                artisan_name: null
                process: Che bien va dong goi theo ho so lo
                ingredients:
                - name: Me
                  origin: Quang Dien
                evidence:
                - id: 55555555-5555-4555-8555-555555555555
                  reference: document:66666666-6666-4666-8666-666666666666
                  kind: CERTIFICATE
                  description: Ho so OCOP da kiem tra thu cong
                certificates:
                - id: 77777777-7777-4777-8777-777777777777
                  type: OCOP
                  number: OCOP-TEST-01
                  issuer: Don vi cap chung nhan
                  scope: BATCH
                  scope_id: 33333333-3333-4333-8333-333333333333
                  issued_on: '2026-01-01'
                  valid_until: '2027-12-31'
                  no_expiry_stated: false
                  review_due_at: null
                  stars: 4
                  verification_method: MANUAL
                  verified_at: '2026-11-20T00:00:00Z'
                  revoked: false
                  evidence_ids:
                  - 55555555-5555-4555-8555-555555555555
                crafting_video_url: null
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 2
                    status: ACTIVE
                    public_revision: null
                    revisions:
                    - number: 1
                      status: DRAFT
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: replaceDraftContentScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.edit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/submit:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: submitTraceabilityRevisionScopedV2IdentityBff
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Scoped_Command'
            example:
              expected_version: 2
              reason: Independent review or authorized change
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 3
                    status: ACTIVE
                    public_revision: null
                    revisions:
                    - number: 1
                      status: IN_REVIEW
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: submitTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.submit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/submit
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/approve:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: approveTraceabilityRevisionScopedV2IdentityBff
      description: Reviewer must not be a contributor to this revision, including inherited contributors. Never
        publishes automatically. All operations require read_internal plus their command permission for the same
        authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor
        policy. Identity transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent;
        no automatic mutation retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass
        validated Origin to Identity for cookie mutations.
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
              - $ref: '#/components/schemas/Scoped_Command'
              - type: object
                required:
                - reason
            example:
              expected_version: 3
              reason: Independent review or authorized change
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 4
                    status: ACTIVE
                    public_revision: null
                    revisions:
                    - number: 1
                      status: APPROVED
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: approveTraceabilityRevisionScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.approve
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-contributor-policy: actor must not be a contributor of the revision
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/approve
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/reject:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: rejectTraceabilityRevisionScopedV2IdentityBff
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
              - $ref: '#/components/schemas/Scoped_Command'
              - type: object
                required:
                - reason
            example:
              expected_version: 3
              reason: Independent review or authorized change
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 4
                    status: ACTIVE
                    public_revision: null
                    revisions:
                    - number: 1
                      status: REJECTED
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: rejectTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.approve
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-contributor-policy: actor must not be a contributor of the revision
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/reject
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/publish:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: publishTraceabilityRevisionScopedV2IdentityBff
      description: Rechecks dependency snapshot and certificate validity; no published result if source changed
        or contract is unavailable. All operations require read_internal plus their command permission for the same
        authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor
        policy. Identity transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent;
        no automatic mutation retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass
        validated Origin to Identity for cookie mutations.
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Scoped_Command'
            example:
              expected_version: 4
              reason: Independent review or authorized change
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 5
                    status: ACTIVE
                    public_revision: 1
                    revisions:
                    - number: 1
                      status: PUBLISHED
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: publishTraceabilityRevisionScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.publish
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/publish
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/suspend:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: suspendTraceabilityProfileScopedV2IdentityBff
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
              - $ref: '#/components/schemas/Scoped_Command'
              - type: object
                required:
                - reason
            example:
              expected_version: 5
              reason: Independent review or authorized change
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 6
                    status: SUSPENDED
                    public_revision: 1
                    revisions:
                    - number: 1
                      status: PUBLISHED
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: suspendTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.suspend
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/suspend
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/archive:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: archiveTraceabilityProfileScopedV2IdentityBff
      parameters:
      - $ref: '#/components/parameters/Scoped_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
              - $ref: '#/components/schemas/Scoped_Command'
              - type: object
                required:
                - reason
            example:
              expected_version: 5
              reason: Independent review or authorized change
      responses:
        '200':
          description: Profile after the command, including its current version; replay returns the original response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scoped_Profile'
              examples:
                success:
                  value:
                    id: 11111111-1111-4111-8111-111111111111
                    mode: integration
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    qr_code: abcdefghijklmnopqrstuv
                    qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    version: 6
                    status: ARCHIVED
                    public_revision: 1
                    revisions:
                    - number: 1
                      status: PUBLISHED
                      content:
                        product_name: Me xung LOT-A
                        producer: Co so O Ma
                        workshop_location: Hue
                        artisan_name: null
                        process: Che bien va dong goi theo ho so lo
                        ingredients:
                        - name: Me
                          origin: Quang Dien
                        evidence:
                        - id: 55555555-5555-4555-8555-555555555555
                          reference: document:66666666-6666-4666-8666-666666666666
                          kind: CERTIFICATE
                          description: Ho so OCOP da kiem tra thu cong
                        certificates:
                        - id: 77777777-7777-4777-8777-777777777777
                          type: OCOP
                          number: OCOP-TEST-01
                          issuer: Don vi cap chung nhan
                          scope: BATCH
                          scope_id: 33333333-3333-4333-8333-333333333333
                          issued_on: '2026-01-01'
                          valid_until: '2027-12-31'
                          no_expiry_stated: false
                          review_due_at: null
                          stars: 4
                          verification_method: MANUAL
                          verified_at: '2026-11-20T00:00:00Z'
                          revoked: false
                          evidence_ids:
                          - 55555555-5555-4555-8555-555555555555
                        crafting_video_url: null
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '401':
          $ref: '#/components/responses/Scoped_Unauthorized'
        '403':
          $ref: '#/components/responses/Scoped_Forbidden'
        '404':
          $ref: '#/components/responses/Scoped_NotFound'
        '409':
          $ref: '#/components/responses/Scoped_Conflict'
        '503':
          $ref: '#/components/responses/Scoped_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: archiveTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.archive
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/archive
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/by-batch:
    get:
      tags:
      - Identity / Keycloak
      operationId: findScopedTraceabilityByBatchDraftIdentityBff
      summary: Find the existing profile before creating a draft
      description: Requires traceability.read_internal for the authoritative warehouse from Inventory. Validate
        scope even when no profile exists. Warehouse is never trusted from browser input. Existing profile responses
        follow evidence redaction rules. Return at most one exact-binding match; no fuzzy SKU lookup. This literal
        route must take precedence over /records/{id}. Identity transport forwards Idempotency-Key, Content-Type,
        Accept, X-Request-Id and Traceparent; no automatic mutation retry, no redirects. Preserve backend status/body
        and no-store. Next BFF must pass validated Origin to Identity for cookie mutations.
      parameters:
      - $ref: '#/components/parameters/Workflow_ProductId'
      - $ref: '#/components/parameters/Workflow_SkuCode'
      - $ref: '#/components/parameters/Workflow_BatchId'
      - name: source_namespace
        in: query
        required: true
        schema:
          type: string
          enum:
          - inventory
      responses:
        '200':
          description: Exact match or empty collection; never 404 for an empty lookup
          content:
            application/json:
              schema:
                type: object
                required:
                - items
                - total
                properties:
                  items:
                    type: array
                    maxItems: 1
                    items:
                      type: object
                      required:
                      - profile_id
                      - version
                      properties:
                        profile_id:
                          type: string
                          format: uuid
                        version:
                          type: integer
                          format: int64
                          minimum: 1
                  total:
                    type: integer
                    minimum: 0
                    maximum: 1
              examples:
                empty:
                  value:
                    items: []
                    total: 0
                found:
                  value:
                    items:
                    - profile_id: 11111111-1111-4111-8111-111111111111
                      version: 1
                    total: 1
        '401':
          $ref: '#/components/responses/Workflow_Unauthorized'
        '403':
          $ref: '#/components/responses/Workflow_Forbidden'
        '503':
          $ref: '#/components/responses/Workflow_Unavailable'
        '409':
          $ref: '#/components/responses/Workflow_Conflict'
        '400':
          $ref: '#/components/responses/Workflow_BadRequest'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/by-batch
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/batches/{batch_id}/context:
    get:
      tags:
      - Identity / Keycloak
      operationId: getScopedBatchTraceabilityContextIdentityBff
      summary: Bootstrap a traceability form from an authoritative receipt batch
      description: Traceability-owned. Requires read_internal for the resource warehouse. Read Inventory first,
        check scope, then resolve Product by exact SKU from Catalog within a shared 2-second budget. No client-supplied
        product/warehouse identity. Does not create a draft or change stock. Restricted/expired source may be displayed
        for draft preparation, but existing submit/publish guards still apply. No dependency on unimplemented Inventory
        SKU detail endpoints or Catalog prices. Identity transport forwards Idempotency-Key, Content-Type, Accept,
        X-Request-Id and Traceparent; no automatic mutation retry, no redirects. Preserve backend status/body and
        no-store. Next BFF must pass validated Origin to Identity for cookie mutations.
      parameters:
      - name: batch_id
        in: path
        required: true
        schema:
          type: string
          format: uuid
      - $ref: '#/components/parameters/Workflow_SkuCode'
      responses:
        '200':
          description: Verified binding for prefilled form; not a public eligibility approval
          headers:
            Cache-Control:
              schema:
                type: string
                enum:
                - no-store
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required:
                - product_id
                - sku_code
                - source_namespace
                - batch_id
                - warehouse_id
                - batch_code
                - production_date
                - expiry_date
                - restricted
                - depleted
                - catalog_version
                - batch_version
                properties:
                  product_id:
                    type: string
                    format: uuid
                  sku_code:
                    type: string
                  source_namespace:
                    type: string
                    enum:
                    - inventory
                  batch_id:
                    type: string
                    format: uuid
                  warehouse_id:
                    type: string
                    format: uuid
                  batch_code:
                    type: string
                  production_date:
                    type: string
                    format: date
                  expiry_date:
                    type: string
                    format: date
                  restricted:
                    type: boolean
                  depleted:
                    type: boolean
                  catalog_version:
                    type: string
                  batch_version:
                    type: string
              examples:
                success:
                  value:
                    product_id: 22222222-2222-4222-8222-222222222222
                    sku_code: MX-001
                    source_namespace: inventory
                    batch_id: 33333333-3333-4333-8333-333333333333
                    warehouse_id: 44444444-4444-4444-8444-444444444444
                    batch_code: LOT-A
                    production_date: '2026-11-01'
                    expiry_date: '2027-11-01'
                    restricted: false
                    depleted: false
                    catalog_version: variant:1:1
                    batch_version: '1'
        '400':
          $ref: '#/components/responses/Workflow_BadRequest'
        '401':
          $ref: '#/components/responses/Workflow_Unauthorized'
        '403':
          $ref: '#/components/responses/Workflow_Forbidden'
        '404':
          $ref: '#/components/responses/Workflow_NotFound'
        '409':
          $ref: '#/components/responses/Workflow_Conflict'
        '503':
          $ref: '#/components/responses/Workflow_Unavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      x-forward-to: /api/v2/admin/trace/batches/{batch_id}/context
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v1/admin/inventory-api/workbench:
    get:
      tags:
      - Identity / Keycloak
      operationId: getAdminInventoryWorkbenchTraceabilityIdentityBff
      summary: Warehouse Operations Workbench context for Epic 09 MVP. Balance values and approval counts are upstream
        projections; the UI does not recalculate them.
      x-lifecycle: stable
      x-backlog:
      - EPIC_09
      x-ui-states:
      - loading
      - success
      - error
      - unauthorized
      - forbidden
      x-test-latency-ms: 1200
      responses:
        '200':
          description: Successful response.
          content:
            application/json:
              schema:
                type: object
                required:
                - skus
                - recentMovements
                - pendingApprovalCount
                - calculatedAt
                - policyNotice
                properties:
                  skus:
                    type: array
                    items:
                      type: object
                      required:
                      - skuId
                      - skuCode
                      - skuLabel
                      - productName
                      - balance
                      - revision
                      - batches
                      properties:
                        skuId:
                          type: string
                        skuCode:
                          type: string
                        skuLabel:
                          type: string
                        productName:
                          type: string
                        balance:
                          type: object
                          required:
                          - onHand
                          - reserved
                          - nonSellable
                          - available
                          properties:
                            onHand:
                              type: integer
                            reserved:
                              type: integer
                            nonSellable:
                              type: integer
                            available:
                              type: integer
                          additionalProperties: false
                        revision:
                          type: integer
                        batches:
                          type: array
                          items:
                            type: object
                            required:
                            - batchId
                            - batchCode
                            - manufacturingDate
                            - expiresAt
                            - expiryState
                            - expiryLabel
                            - balance
                            properties:
                              batchId:
                                type: string
                              batchCode:
                                type: string
                              manufacturingDate:
                                type: string
                                format: date-time
                              expiresAt:
                                type: string
                                format: date-time
                              expiryState:
                                type: string
                              expiryLabel:
                                type: string
                              balance:
                                type: object
                                required:
                                - onHand
                                - reserved
                                - nonSellable
                                - available
                                properties:
                                  onHand:
                                    type: integer
                                  reserved:
                                    type: integer
                                  nonSellable:
                                    type: integer
                                  available:
                                    type: integer
                                additionalProperties: false
                            additionalProperties: false
                      additionalProperties: false
                  recentMovements:
                    type: array
                    items:
                      type: object
                      required:
                      - movementId
                      - type
                      - typeLabel
                      - quantity
                      - occurredAt
                      - actorLabel
                      - reference
                      - skuCode
                      - batchCode
                      properties:
                        movementId:
                          type: string
                        type:
                          type: string
                        typeLabel:
                          type: string
                        quantity:
                          type: integer
                        occurredAt:
                          type: string
                          format: date-time
                        actorLabel:
                          type: string
                        reference:
                          type: string
                        skuCode:
                          type: string
                        batchCode:
                          type: string
                      additionalProperties: false
                  pendingApprovalCount:
                    type: integer
                  calculatedAt:
                    type: string
                    format: date-time
                  policyNotice:
                    type: string
                additionalProperties: false
              examples:
                success:
                  summary: Successful response
                  value:
                    skus:
                    - skuId: prod-me-xung-truyen-thong-sku-500
                      skuCode: OMA-MXT-500
                      skuLabel: Hộp 500 g
                      productName: Mè Xửng Truyền Thống
                      balance:
                        onHand: 80
                        reserved: 10
                        nonSellable: 8
                        available: 62
                      revision: 9
                      batches:
                      - batchId: batch-mxt-safe
                        batchCode: LOT-2026-08-A
                        manufacturingDate: '2026-08-01T00:00:00.000Z'
                        expiresAt: '2027-02-01T00:00:00.000Z'
                        expiryState: SAFE
                        expiryLabel: Còn hạn
                        balance:
                          onHand: 40
                          reserved: 10
                          nonSellable: 0
                          available: 30
                      - batchId: batch-mxt-expiring
                        batchCode: LOT-2026-04-B
                        manufacturingDate: '2026-04-20T00:00:00.000Z'
                        expiresAt: '2026-10-20T00:00:00.000Z'
                        expiryState: EXPIRING
                        expiryLabel: Cận hạn sử dụng
                        balance:
                          onHand: 32
                          reserved: 0
                          nonSellable: 0
                          available: 32
                      - batchId: batch-mxt-expired
                        batchCode: LOT-2026-03-C
                        manufacturingDate: '2026-03-15T00:00:00.000Z'
                        expiresAt: '2026-09-30T00:00:00.000Z'
                        expiryState: EXPIRED
                        expiryLabel: Đã hết hạn
                        balance:
                          onHand: 8
                          reserved: 0
                          nonSellable: 8
                          available: 0
                    - skuId: prod-tra-sen-tinh-tam-sku-200
                      skuCode: OMA-TSTT-200
                      skuLabel: Hộp trà 200 g
                      productName: Trà Sen Tịnh Tâm
                      balance:
                        onHand: 45
                        reserved: 5
                        nonSellable: 10
                        available: 30
                      revision: 7
                      batches:
                      - batchId: batch-tstt-safe
                        batchCode: LOT-2026-09-T
                        manufacturingDate: '2026-09-01T00:00:00.000Z'
                        expiresAt: '2027-03-15T00:00:00.000Z'
                        expiryState: SAFE
                        expiryLabel: Còn hạn
                        balance:
                          onHand: 35
                          reserved: 5
                          nonSellable: 0
                          available: 30
                      - batchId: batch-tstt-expired
                        batchCode: LOT-2026-02-T
                        manufacturingDate: '2026-02-01T00:00:00.000Z'
                        expiresAt: '2026-09-15T00:00:00.000Z'
                        expiryState: EXPIRED
                        expiryLabel: Đã hết hạn
                        balance:
                          onHand: 10
                          reserved: 0
                          nonSellable: 10
                          available: 0
                    recentMovements:
                    - movementId: mov-wh-001
                      type: RECEIPT
                      typeLabel: Nhập kho
                      quantity: 24
                      occurredAt: '2026-10-03T01:40:00.000Z'
                      actorLabel: Nguyễn Văn Bình · Warehouse Staff
                      reference: SR-2026-0103
                      skuCode: OMA-MXT-500
                      batchCode: LOT-2026-08-A
                    - movementId: mov-wh-002
                      type: ISSUE
                      typeLabel: Xuất kho
                      quantity: -6
                      occurredAt: '2026-10-03T02:10:00.000Z'
                      actorLabel: Nguyễn Văn Bình · Warehouse Staff
                      reference: ORD-2026-1038
                      skuCode: OMA-TSTT-200
                      batchCode: LOT-2026-09-T
                    - movementId: mov-wh-003
                      type: DAMAGE
                      typeLabel: Ghi nhận hàng hỏng
                      quantity: -2
                      occurredAt: '2026-10-03T02:25:00.000Z'
                      actorLabel: Nguyễn Văn Bình · Warehouse Staff
                      reference: DMG-2026-004
                      skuCode: OMA-MXT-500
                      batchCode: LOT-2026-04-B
                    pendingApprovalCount: 1
                    calculatedAt: '2026-10-03T02:30:00.000Z'
                    policyNotice: Mọi thay đổi được ghi bằng biến động, không sửa trực tiếp số dư. Inventory API
                      kiểm tra tồn âm, revision, tham chiếu trùng và quyết định áp dụng ngay hay chuyển chờ phê
                      duyệt.
        '400':
          $ref: '#/components/responses/AdminStable_BadRequest'
        '401':
          $ref: '#/components/responses/AdminStable_Unauthorized'
        '403':
          $ref: '#/components/responses/AdminStable_Forbidden'
        '500':
          $ref: '#/components/responses/AdminStable_InternalError'
        '503':
          $ref: '#/components/responses/AdminStable_ServiceUnavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      x-forward-to: /api/v1/admin/inventory/workbench
      x-body-limit-bytes: 0
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v1/admin/inventory-api/workbench/receipts:
    post:
      tags:
      - Identity / Keycloak
      operationId: postAdminInventoryWorkbenchReceiptsTraceabilityIdentityBff
      summary: Idempotent stock receipt command for US-INV-02. The real Inventory service validates SKU, Batch identity
        and source reference before updating balances.
      description: Requires INVENTORY_RECEIPT_CREATE with GLOBAL scope or WAREHOUSE scope matching the SKU. Fresh
        APPLIED receipts include batchId/batchCode/skuCode/warehouseId. Legacy replay may omit these fields; preserve
        its result and let UI bootstrap from a verified batch, never repeat stock receipt.
      x-lifecycle: stable
      x-backlog:
      - EPIC_09
      x-ui-states:
      - loading
      - success
      - error
      - unauthorized
      - forbidden
      x-test-latency-ms: 1200
      responses:
        '200':
          description: Successful response.
          content:
            application/json:
              schema:
                type: object
                required:
                - operationId
                - status
                - statusLabel
                - movementId
                - approvalRequestId
                - occurredAt
                - revision
                - message
                properties:
                  operationId:
                    type: string
                  status:
                    type: string
                  statusLabel:
                    type: string
                  movementId:
                    type: string
                  approvalRequestId:
                    type:
                    - string
                    - 'null'
                    description: approvalRequestId may be null.
                  occurredAt:
                    type: string
                    format: date-time
                  revision:
                    type: integer
                  message:
                    type: string
                  batchId:
                    type: string
                    format: uuid
                  batchCode:
                    type: string
                  skuCode:
                    type: string
                  warehouseId:
                    type: string
                    format: uuid
                additionalProperties: false
              examples:
                success:
                  summary: Successful response
                  value:
                    operationId: INV-OP-REC-0104
                    status: APPLIED
                    statusLabel: Đã áp dụng
                    movementId: mov-receipt-0104
                    approvalRequestId: null
                    occurredAt: '2026-10-03T03:10:00.000Z'
                    revision: 10
                    message: Đã ghi nhận nhập kho bằng một biến động có tham chiếu và chống gửi trùng.
        '400':
          $ref: '#/components/responses/AdminStable_BadRequest'
        '401':
          $ref: '#/components/responses/AdminStable_Unauthorized'
        '403':
          $ref: '#/components/responses/AdminStable_Forbidden'
        '409':
          $ref: '#/components/responses/AdminStable_Conflict'
        '422':
          $ref: '#/components/responses/AdminStable_UnprocessableEntity'
        '500':
          $ref: '#/components/responses/AdminStable_InternalError'
        '503':
          $ref: '#/components/responses/AdminStable_ServiceUnavailable'
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
              - skuId
              - quantity
              - expectedRevision
              properties:
                skuId:
                  type: string
                  minLength: 1
                  maxLength: 50
                batchMode:
                  type: string
                  enum:
                  - NEW
                  - EXISTING
                batchId:
                  type: string
                  format: uuid
                batchCode:
                  type: string
                  minLength: 1
                  maxLength: 100
                supplierId:
                  type: string
                  format: uuid
                manufacturingDate:
                  type: string
                  format: date
                expiresAt:
                  type: string
                  format: date
                quantity:
                  type: integer
                  minimum: 1
                  maximum: 2147483647
                sourceReference:
                  type: string
                  minLength: 1
                  maxLength: 160
                reasonCode:
                  type: string
                  maxLength: 80
                expectedRevision:
                  type: integer
                  minimum: 0
                idempotencyKey:
                  type: string
                  description: Must match the header if supplied.
              additionalProperties: false
              oneOf:
              - required:
                - batchMode
                - batchId
                - sourceReference
                properties:
                  batchMode:
                    const: EXISTING
              - required:
                - batchMode
                - batchCode
                - supplierId
                - manufacturingDate
                - expiresAt
                - sourceReference
                properties:
                  batchMode:
                    const: NEW
              - required:
                - batchCode
                - reasonCode
                - idempotencyKey
                properties:
                  batchMode: false
            example:
              skuId: MX-GION-500G
              batchMode: NEW
              batchCode: LOT-20261005-01
              supplierId: b2222222-2222-2222-2222-222222222222
              manufacturingDate: '2026-10-05'
              expiresAt: '2027-03-05'
              quantity: 100
              sourceReference: PRODUCTION-20261005-01
              expectedRevision: 8
      parameters:
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
      x-forward-to: /api/v1/admin/inventory/workbench/receipts
      x-body-limit-bytes: 8192
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
components:
  schemas:
    Scoped_Profile:
      type: object
      required:
      - id
      - mode
      - version
      - status
      - product_id
      - sku_code
      - batch_id
      - qr_code
      - revisions
      - warehouse_id
      - source_namespace
      - qr_url
      - public_revision
      properties:
        id:
          type: string
          format: uuid
        mode:
          type: string
          enum:
          - sandbox
          - integration
        version:
          type: integer
          format: int64
          minimum: 1
        status:
          type: string
          enum:
          - ACTIVE
          - SUSPENDED
          - ARCHIVED
        product_id:
          type: string
          format: uuid
        sku_code:
          type: string
        source_namespace:
          type: string
        batch_id:
          type: string
        qr_code:
          type: string
        qr_url:
          type: string
          format: uri
          description: Configured Web page URL derived from the stable code; API public-read URL is separate
        public_revision:
          type:
          - integer
          - 'null'
        revisions:
          type: array
          items:
            type: object
            required:
            - number
            - status
            - content
            properties:
              number:
                type: integer
                minimum: 1
              status:
                type: string
                enum:
                - DRAFT
                - IN_REVIEW
                - APPROVED
                - REJECTED
                - PUBLISHED
                - ARCHIVED
              content:
                $ref: '#/components/schemas/Scoped_AdminContent'
              dependency_snapshot:
                type:
                - object
                - 'null'
                readOnly: true
                description: Frozen source snapshot. Historical snapshots are not rewritten during warehouse metadata
                  backfill.
                properties:
                  warehouse_id:
                    type: string
                    format: uuid
                  checked_at:
                    type: string
                    format: date-time
                  batch_code:
                    type: string
                  batch_version:
                    type: string
                  catalog_version:
                    type: string
        warehouse_id:
          type:
          - string
          - 'null'
          format: uuid
          readOnly: true
          description: Server-derived warehouse ownership; null only for unmigrated legacy records visible to GLOBAL
            readers.
    Scoped_AdminContent:
      type: object
      additionalProperties: false
      description: Content response; internal evidence reference is omitted without traceability.read_evidence,
        including on replay.
      properties:
        product_name:
          $ref: '#/components/schemas/Scoped_Content/properties/product_name'
        producer:
          $ref: '#/components/schemas/Scoped_Content/properties/producer'
        workshop_location:
          $ref: '#/components/schemas/Scoped_Content/properties/workshop_location'
        artisan_name:
          $ref: '#/components/schemas/Scoped_Content/properties/artisan_name'
        process:
          $ref: '#/components/schemas/Scoped_Content/properties/process'
        ingredients:
          $ref: '#/components/schemas/Scoped_Content/properties/ingredients'
        certificates:
          $ref: '#/components/schemas/Scoped_Content/properties/certificates'
        crafting_video_url:
          $ref: '#/components/schemas/Scoped_Content/properties/crafting_video_url'
        evidence:
          type: array
          maxItems: 50
          items:
            type: object
            additionalProperties: false
            required:
            - id
            - kind
            - description
            properties:
              id:
                type: string
                format: uuid
              kind:
                type: string
              description:
                type: string
              reference:
                type: string
                description: Empty string when read_evidence is absent, matching service redaction; never replace
                  a stored reference with this empty redaction.
    Scoped_Content:
      type: object
      additionalProperties: false
      description: Draft may be incomplete; submit additionally requires producer, origin, process and a verified
        in-scope OCOP certificate.
      properties:
        product_name:
          type: string
          maxLength: 250
        producer:
          type: string
          maxLength: 250
        workshop_location:
          type: string
          maxLength: 1000
        artisan_name:
          type:
          - string
          - 'null'
          maxLength: 250
        process:
          type: string
          maxLength: 8000
        ingredients:
          type: array
          maxItems: 20
          items:
            type: object
            additionalProperties: false
            required:
            - name
            - origin
            properties:
              name:
                type: string
                minLength: 1
                maxLength: 250
              origin:
                type: string
                minLength: 1
                maxLength: 1000
        certificates:
          type: array
          maxItems: 20
          items:
            $ref: '#/components/schemas/Scoped_Certificate'
        evidence:
          type: array
          maxItems: 50
          items:
            $ref: '#/components/schemas/Scoped_Evidence'
        crafting_video_url:
          type:
          - string
          - 'null'
          format: uri
          maxLength: 2048
          description: Legacy optional HTTPS URL accepted only under configured host allowlist. No arbitrary URL
            or upload workflow is introduced; the new UI does not offer free-form URL input.
    Scoped_Certificate:
      type: object
      additionalProperties: false
      required:
      - id
      - type
      - number
      - issuer
      - scope
      - scope_id
      - issued_on
      - verification_method
      - verified_at
      - evidence_ids
      properties:
        id:
          type: string
          format: uuid
        type:
          type: string
          enum:
          - OCOP
          - ATTP
          - OTHER
        number:
          type: string
          minLength: 1
          maxLength: 250
        issuer:
          type: string
          minLength: 1
          maxLength: 250
        scope:
          type: string
          enum:
          - PRODUCT
          - SKU
          - BATCH
        scope_id:
          type: string
          minLength: 1
          maxLength: 160
        issued_on:
          type: string
          format: date
        valid_until:
          type:
          - string
          - 'null'
          format: date
        no_expiry_stated:
          type: boolean
          default: false
        review_due_at:
          type:
          - string
          - 'null'
          format: date-time
        stars:
          type:
          - integer
          - 'null'
          minimum: 1
          maximum: 5
        verification_method:
          type: string
          enum:
          - MANUAL
        verified_at:
          type: string
          format: date-time
        revoked:
          type: boolean
          default: false
        evidence_ids:
          type: array
          minItems: 1
          maxItems: 50
          items:
            type: string
            format: uuid
    Scoped_Evidence:
      type: object
      additionalProperties: false
      required:
      - id
      - kind
      - description
      properties:
        id:
          type: string
          format: uuid
        reference:
          type: string
          minLength: 0
          maxLength: 1000
          description: 'On edit only: omission preserves the stored reference for the same evidence ID. New evidence
            requires a valid internal reference. No arbitrary upload URL. This rule is validated after restoring
            authorized stored references.'
        kind:
          type: string
          minLength: 1
          maxLength: 100
        description:
          type: string
          minLength: 1
          maxLength: 1000
    Scoped_Error:
      type: object
      additionalProperties: false
      required:
      - code
      - message
      - requestId
      properties:
        code:
          type: string
        message:
          type: string
        requestId:
          type: string
    IdentityBff_ProxyError:
      type: object
      required:
      - code
      - message
      - requestId
      properties:
        code:
          type: string
        message:
          type: string
        requestId:
          type: string
    Scoped_Command:
      type: object
      additionalProperties: false
      required:
      - expected_version
      properties:
        expected_version:
          type: integer
          format: int64
          minimum: 1
        reason:
          type: string
          minLength: 1
          maxLength: 1000
    Workflow_Error:
      type: object
      required:
      - code
      - message
      - requestId
      properties:
        code:
          type: string
          description: Stable machine-readable code
        message:
          type: string
          description: Safe user-visible message
        requestId:
          type: string
    AdminStable_Error:
      type: object
      required:
      - code
      - message
      - requestId
      properties:
        code:
          type: string
          description: Stable machine-readable error code.
        message:
          type: string
          description: Safe user-facing message in Vietnamese or the negotiated locale.
        requestId:
          type: string
          description: Correlation identifier for support and logs.
        errors:
          type: array
          items:
            type: object
            required:
            - field
            - code
            - message
            properties:
              field:
                type: string
              code:
                type: string
              message:
                type: string
            additionalProperties: false
        retryAfterSeconds:
          type: integer
          minimum: 1
        details:
          type: object
          additionalProperties: true
      additionalProperties: false
  responses:
    Scoped_Unauthorized:
      description: Missing or invalid verified identity
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Scoped_Error'
          examples:
            UNAUTHENTICATED:
              value:
                code: UNAUTHENTICATED
                message: Request cannot be completed.
                requestId: req-traceability-example
    Scoped_Forbidden:
      description: Permission denied or contributor trying to approve their own revision
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Scoped_Error'
          examples:
            FORBIDDEN:
              value:
                code: FORBIDDEN
                message: Request cannot be completed.
                requestId: req-traceability-example
    Scoped_Unavailable:
      description: Required dependency or contract unavailable; no fixture fallback in integration
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Scoped_Error'
          examples:
            DEPENDENCY_UNAVAILABLE:
              value:
                code: DEPENDENCY_UNAVAILABLE
                message: Request cannot be completed.
                requestId: req-traceability-example
    Scoped_BadRequest:
      description: Invalid input, missing evidence or certificate validation failed
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Scoped_Error'
          examples:
            INVALID_INPUT:
              value:
                code: INVALID_INPUT
                message: Request cannot be completed.
                requestId: req-traceability-example
    Scoped_NotFound:
      description: Resource not found or never published; no internal details
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Scoped_Error'
          examples:
            NOT_FOUND:
              value:
                code: NOT_FOUND
                message: Request cannot be completed.
                requestId: req-traceability-example
    Scoped_Conflict:
      description: Stale version, invalid state, immutable binding, source changed or idempotency conflict
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Scoped_Error'
          examples:
            CONFLICT:
              value:
                code: CONFLICT
                message: Request cannot be completed.
                requestId: req-traceability-example
            IDEMPOTENCY_CONFLICT:
              value:
                code: IDEMPOTENCY_CONFLICT
                message: Request conflicts with current authoritative state.
                requestId: req-traceability-example
            WAREHOUSE_BINDING_CONFLICT:
              value:
                code: WAREHOUSE_BINDING_CONFLICT
                message: Request conflicts with current authoritative state.
                requestId: req-traceability-example
    Workflow_Unauthorized:
      description: Absent, expired or invalid authentication; code UNAUTHENTICATED
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Workflow_Error'
          examples:
            UNAUTHENTICATED:
              value:
                code: UNAUTHENTICATED
                message: Request cannot be completed.
                requestId: req-traceability-example
    Workflow_Forbidden:
      description: Known identity without required permission, scope or capability; code FORBIDDEN
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Workflow_Error'
          examples:
            FORBIDDEN:
              value:
                code: FORBIDDEN
                message: Request cannot be completed.
                requestId: req-traceability-example
    Workflow_Unavailable:
      description: Required authority or storage unavailable; code DEPENDENCY_UNAVAILABLE
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Workflow_Error'
          examples:
            DEPENDENCY_UNAVAILABLE:
              value:
                code: DEPENDENCY_UNAVAILABLE
                message: Request cannot be completed.
                requestId: req-traceability-example
    Workflow_Conflict:
      description: Authoritative warehouse changed for an existing profile; code WAREHOUSE_BINDING_CONFLICT
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Workflow_Error'
          examples:
            WAREHOUSE_BINDING_CONFLICT:
              value:
                code: WAREHOUSE_BINDING_CONFLICT
                message: Request cannot be completed.
                requestId: req-traceability-example
    Workflow_BadRequest:
      description: Invalid staff binding input; code INVALID_INPUT
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Workflow_Error'
          example:
            code: INVALID_INPUT
            message: Invalid request.
            requestId: req-traceability-example
    Workflow_NotFound:
      description: Resource does not exist; code NOT_FOUND
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Workflow_Error'
          examples:
            NOT_FOUND:
              value:
                code: NOT_FOUND
                message: Request cannot be completed.
                requestId: req-traceability-example
    AdminStable_BadRequest:
      description: Malformed request or invalid query parameters.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/AdminStable_Error'
          example:
            code: BAD_REQUEST
            message: Malformed request or invalid query parameters.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    AdminStable_Unauthorized:
      description: Authentication is missing, invalid, or expired.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/AdminStable_Error'
          example:
            code: AUTHENTICATION_REQUIRED
            message: Authentication is missing, invalid, or expired.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    AdminStable_Forbidden:
      description: Authenticated actor lacks the required permission, capability, ownership, or data scope.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/AdminStable_Error'
          example:
            code: FORBIDDEN
            message: Authenticated actor lacks the required permission, capability, ownership, or data scope.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    AdminStable_InternalError:
      description: Unexpected server error.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/AdminStable_Error'
          example:
            code: INTERNAL_SERVER_ERROR
            message: Unexpected server error.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    AdminStable_ServiceUnavailable:
      description: A required service is temporarily unavailable.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/AdminStable_Error'
          example:
            code: SERVICE_UNAVAILABLE
            message: A required service is temporarily unavailable.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    AdminStable_Conflict:
      description: State transition, revision, idempotency key, or unique value conflicts with current state.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/AdminStable_Error'
          example:
            code: STATE_CONFLICT
            message: State transition, revision, idempotency key, or unique value conflicts with current state.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    AdminStable_UnprocessableEntity:
      description: Payload is syntactically valid but violates field or business validation rules.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/AdminStable_Error'
          example:
            code: VALIDATION_FAILED
            message: Payload is syntactically valid but violates field or business validation rules.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
  securitySchemes:
    IdentityBff_employeeSessionCookie:
      type: apiKey
      in: cookie
      name: OMAMA_SESSION
      description: Existing HttpOnly session resolved server-side; no browser access-token exposure.
    IdentityBff_identityStaffBearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: Verified Identity/Keycloak employee token; same active principal checks apply. Not a forged user/role
        header.
  parameters:
    Scoped_IdempotencyKey:
      name: Idempotency-Key
      in: header
      required: true
      schema:
        type: string
        minLength: 16
        maxLength: 128
        pattern: ^[A-Za-z0-9._~-]+$
    Scoped_ProfileID:
      name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
    Scoped_RevisionNumber:
      name: revision
      in: path
      required: true
      schema:
        type: integer
        minimum: 1
    Workflow_ProductId:
      name: product_id
      in: query
      required: true
      schema:
        type: string
        format: uuid
    Workflow_SkuCode:
      name: sku_code
      in: query
      required: true
      schema:
        type: string
        minLength: 1
        maxLength: 100
    Workflow_BatchId:
      name: batch_id
      in: query
      required: true
      schema:
        type: string
        format: uuid
x-source-commit: 1d4fba6477032419d894a32b1dd6866490b3e95c
```

## Proto target nhúng: `packages/proto/identity/v1/staff_authorization.proto`

Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng. Không hand-edit generated code.

```proto
// Integration target 0.3.0-draft. Provider: Identity; consumer: Traceability.
syntax = "proto3";
package omamx.identity.v1;
option go_package = "github.com/omamx/proto/identity/v1;identityv1";
option java_multiple_files = true;
option java_package = "com.omamx.proto.identity.v1";

// TLS, authorization: Bearer <service credential>, and exactly one
// x-staff-authorization: Bearer <employee JWT>. Never log either credential.
// Identity verifies both, derives the principal from the employee JWT and reads
// current authoritative IAM grants. Do not accept a caller-supplied user ID.
service StaffAuthorization {
  rpc GetCurrentStaffGrants(CurrentStaffGrantsRequest) returns (CurrentStaffGrantsResponse);
}
message CurrentStaffGrantsRequest {}
message CurrentStaffGrantsResponse {
  string principal_id = 1;
  string idp_subject = 2;
  string principal_type = 3;
  string provisioning_status = 4;
  // Only traceability.* grants, at most 128; never silently truncate.
  repeated StaffPermissionGrant permission_grants = 5;
}
message StaffPermissionGrant {
  string permission_code = 1;
  // GLOBAL or WAREHOUSE. Consumers ignore unknown scope types, never widen them.
  string scope_type = 2;
  // Empty for GLOBAL; canonical warehouse UUID for WAREHOUSE.
  string scope_id = 3;
}
```

## Tính toàn vẹn bản gửi

Nguồn commit: `1d4fba6477032419d894a32b1dd6866490b3e95c`. SHA-256 tập source: `2eb647c459daba826ac697559e03db5b6dabef07408f5d857febc2cec111a2b3`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
