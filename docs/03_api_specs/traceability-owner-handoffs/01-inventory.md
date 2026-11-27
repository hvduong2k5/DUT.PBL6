# PBL6 — Bàn giao Inventory: INV-TRC-01

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
| `docs/03_api_specs/traceability-workflow-candidates/batch_source.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-candidates/batch_source.proto) |

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

## Phần việc của Inventory

**Mục đích:** sau nhập kho biết chính xác UUID lô; Traceability xác minh kho để chặn nhân viên ngoài phạm vi.

**Đầu ra bắt buộc, có thể tái sử dụng phần đã có:**

| Mã yêu cầu | Consumer nhận được gì | Điều kiện để được coi là đáp ứng |
| --- | --- | --- |
| INV-OUT-01 | Kết quả xác minh lô gồm UUID, SKU, mã lô, NSX/HSD, trạng thái hạn chế/hết tồn, version và kho sở hữu | Binding chính xác; đọc được lô lịch sử còn lưu; kho lấy từ nguồn authoritative và không âm thầm đổi ý nghĩa |
| INV-OUT-02 | Binding lô vừa nhập, đủ để mở/lập đúng hồ sơ | NEW/EXISTING xác định đúng UUID; cùng receipt replay không tạo thêm tồn hoặc lô. Có thể tái sử dụng read API xác định duy nhất theo receipt/movement bên trong adapter, nhưng đầu ra receipt mà consumer nhận vẫn đúng contract đích |
| INV-OUT-03 | Phân biệt binding sai, thiếu quyền và provider lỗi | Status/error và timeout đúng contract; không trả lô gần đúng hoặc gán kho mặc định |

**Provider contracts:** [BatchSource candidate](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-candidates/batch_source.proto), receipt delta trong [OpenAPI tích hợp](https://github.com/hvduong2k5/DUT.PBL6/blob/1d4fba6477032419d894a32b1dd6866490b3e95c/docs/03_api_specs/traceability-workflow-draft.openapi.yaml). Không đổi Reserve/Finalize/Release hay FEFO.

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

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| T-SCOPE-04 | Provider thiếu warehouse hoặc đổi kho cùng UUID; legacy null binding | Không scoped create/submit/publish; mismatch409; null chỉ GLOBAL, không gán default |
| INV-01 | BatchSource exact UUID+SKU, depleted/expired/restricted | Giữ flags/version và trả kho authoritative đúng; saiSKU không lấy lô gần đúng |
| INV-02 | Receipt NEW/EXISTING và replay | batchId/code/SKU/warehouse đúng, transaction+idempotency không nhân đôi tồn |
| INV-03 | Legacy receipt thiếu binding, DB/network failure | Không đoán UUID; không tạo receipt mới để lấy binding |
| INV-04 | Contract Inventory đích và mapping | Giữ field/method cũ, cung cấp warehouse9 ở proto đích; API nội bộ được tái sử dụng phải map kho/version chính xác. Cùng source version không đổi nội dung âm thầm |
| INV-05 | GET workbench provider thật + receipt và bootstrap end-to-end | SKU/batch/revision đúng kho; không chỉ context Mockoon; scoped trace không đòi quyền Catalog management/BATCH_VIEW để bootstrap |

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
  title: PBL6 Inventory traceability handoff
  version: 0.3.0-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 1d4fba6477032419d894a32b1dd6866490b3e95c. Direct
    implementation target, not runtime acceptance. All references are local to this embedded contract.
tags:
- name: Inventory
  description: Owner-specific provider and dependency contracts
servers:
- url: http://127.0.0.1
paths:
  /api/v1/admin/inventory/workbench:
    get:
      tags:
      - Inventory
      operationId: getAdminInventoryWorkbench
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
      security:
      - AdminStable_bearerAuth: []
      - AdminStable_adminSession: []
      servers:
      - url: https://api.omama.vn
        description: Production API gateway
  /api/v1/admin/inventory/workbench/receipts:
    post:
      tags:
      - Inventory
      operationId: postAdminInventoryWorkbenchReceipts
      summary: Idempotent stock receipt command for US-INV-02. The real Inventory service validates SKU, Batch identity
        and source reference before updating balances.
      description: Requires INVENTORY_RECEIPT_CREATE with GLOBAL scope or WAREHOUSE scope matching the SKU. Fresh
        APPLIED receipts include batchId, batchCode, skuCode and warehouseId from the committed transaction. Legacy
        idempotent replay may omit them; preserve the receipt result and do not repeat the stock mutation.
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
      security:
      - AdminStable_bearerAuth: []
      servers:
      - url: https://api.omama.vn
        description: Production API gateway
components:
  responses:
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
  schemas:
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
  securitySchemes:
    AdminStable_bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
    AdminStable_adminSession:
      type: apiKey
      in: cookie
      name: admin_session
x-source-commit: 1d4fba6477032419d894a32b1dd6866490b3e95c
```

## Proto target nhúng: `docs/03_api_specs/traceability-workflow-candidates/batch_source.proto`

Candidate full definition: compile riêng, cập nhật canonical/copies/codegen khi implement; không compile cạnh canonical cùng message. Không hand-edit generated code.

```proto
// Implementation target 0.3.0-draft, 2026-11-27. NOT canonical/runtime code.
// Compile this directory separately; do not compile beside the canonical files.
syntax = "proto3";

package omamx.traceability.v1;

option go_package = "github.com/omamx/proto/traceability/v1;traceabilityv1";

service BatchSource {
  rpc GetTraceabilityBatch(BatchRequest) returns (BatchSnapshot);
}

message BatchRequest {
  string batch_id = 1;
  string sku_code = 2;
}

message BatchSnapshot {
  string batch_id = 1;
  string sku_code = 2;
  string batch_code = 3;
  string production_date = 4;
  string expiry_date = 5;
  bool restricted = 6;
  bool depleted = 7;
  int64 batch_version = 8;
  // Authoritative owning warehouse; empty means provider lacks scoped capability.
  string warehouse_id = 9;
}
```

## Tính toàn vẹn bản gửi

Nguồn commit: `1d4fba6477032419d894a32b1dd6866490b3e95c`. SHA-256 tập source: `1cf3061967c403a3806896564c0eb6c9a51df160135435ecaf6057b7e2f0c481`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
