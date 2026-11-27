# Transport nội bộ Traceability — 0.3.0-draft

Chốt27/11/2026 theo định hướng nhóm: business calls East-West dùng gRPC; REST là bề mặt Web/Mobile. Bản này thay transport nội bộ của target0.2.2, giữ nguyên nghiệp vụ, scope kho, state machine, nguồn dữ liệu, snapshot/hash và dữ liệu public. Các phiếu0.2.2 cũ chỉ là lịch sử; phát lại phiếu0.3.0 từ commit nguồn mới, không dùng lẫn contract giữa hai bản.

## Ranh giới và contract authoritative

| Kết nối | Contract đích | Owner |
| --- | --- | --- |
| Traceability → Inventory | `GetTraceabilityBatch`, candidate field9 warehouse hiện có | Inventory |
| Traceability → Catalog | [CatalogTraceabilitySource.GetSkuBinding](../../packages/proto/catalog/v1/traceability_source.proto) | Catalog |
| Traceability → Identity | [StaffAuthorization.GetCurrentStaffGrants](../../packages/proto/identity/v1/staff_authorization.proto) | Identity |
| Order → Traceability | [BatchLinkResolver.ResolveBatchLinks](../../packages/proto/traceability/v1/batch_links.proto) | Traceability |
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
