# PBL6 — Bàn giao Catalog / Media: CAT-TRC-01

Version nghiệp vụ/contract: **0.4.0-draft** · Snapshot tài liệu: **`95f9ab117cca0cc231412dc3cbdfc878a9437386`**.

**Bản 0.4.0 thay phiếu 0.3.0/0.2.2:** theo Catalog/Media đã merge trong PR #175 và plan Dương; không triển khai proposal GetSkuBinding đã gỡ. Web/BFF vẫn REST. Identity current-grants và Order resolver vẫn là phần việc riêng, chưa có trong plan Dương.

**Checkpoint28/11:** phần bổ sung đã có trên nhánh này; đọc report11 để tránh code trùng và phân biệt live-provider checks với fixture. Owner bảo đảm tương thích khi ghép, không thay phần pricing/checkout trong plan Order Dương.

File này là bản `.md` độc lập để gửi qua Zalo. Đề bài, yêu cầu đầu ra, test và contract liên quan được nhúng bên dưới; không cần đọc lịch sử chat hoặc chờ phản hồi ACCEPT để bắt đầu code. Draft không đồng nghĩa chức năng đã chạy hoặc đã deploy.

## Nguồn tài liệu và cách lấy về

- Repository: [https://github.com/hvduong2k5/DUT.PBL6](https://github.com/hvduong2k5/DUT.PBL6). Nhánh tài liệu: `codex/traceability-order-flow`.
- **Đọc theo commit đã ghim**, không tự thay bằng HEAD của dev hoặc nhánh tài liệu đang tiếp tục thay đổi. Đây là nguồn tài liệu; owner giữ nhánh làm việc riêng cho service, không phải checkout đè công việc của mình.
- Có thể đọc toàn bộ đặc tả qua các link dưới. Người truy cập repository riêng tư cần quyền hiện có của nhóm. Contract nhúng trong file này vẫn đọc được khi chưa mở được link.

| Tài liệu liên quan trong repository | Link bất biến |
| --- | --- |
| `docs/03_api_specs/traceability-workflow-peer-handoff.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-peer-handoff.md) |
| `docs/03_api_specs/traceability-workflow-acceptance.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-acceptance.md) |
| `docs/03_api_specs/traceability-workflow-supplement.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-supplement.md) |
| `docs/03_api_specs/traceability-workflow-service-spec.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-service-spec.md) |
| `docs/03_api_specs/traceability-workflow-review-2026-11-25.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-review-2026-11-25.md) |
| `docs/03_api_specs/traceability-workflow-grpc.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-grpc.md) |
| `docs/04_testing/traceability-service/11_local_workflow_2026-11-28.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/04_testing/traceability-service/11_local_workflow_2026-11-28.md) |
| `docs/03_api_specs/customer-stable.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/customer-stable.openapi.yaml) |
| `packages/proto/catalog/v1/catalog.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/catalog/v1/catalog.proto) |
| `packages/proto/media/v1/media_service.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/media/v1/media_service.proto) |
| `packages/proto/common/v1/types.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/common/v1/types.proto) |
| `packages/proto/common/v1/errors.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/common/v1/errors.proto) |

Trong repository đã clone, các lệnh đọc sau không checkout/reset hay ghi đè source đang làm:

```powershell
git fetch origin codex/traceability-order-flow
git show 95f9ab117cca0cc231412dc3cbdfc878a9437386:docs/03_api_specs/traceability-workflow-grpc.md
git show 95f9ab117cca0cc231412dc3cbdfc878a9437386:docs/03_api_specs/traceability-workflow-peer-handoff.md
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

## Phần việc của Catalog / Media

### Catalog / Media: CAT-TRC-01 — tái sử dụng provider đã merge

CAT-OUT-01: giữ [CatalogService.GetTraceabilityBinding](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/catalog/v1/catalog.proto) của PR #175; input product_id + exact sku_code, output version_fingerprint và echo authoritative. Traceability dùng fingerprint nguyên trạng cho catalog_version. Không thêm CatalogTraceabilitySource/GetSkuBinding hoặc khôi phục REST đã gỡ.

CAT-OUT-02: bootstrap từ SKU dùng GetProductVariant(channel UNSPECIFIED) lấy product_id rồi GetTraceabilityBinding xác minh/version. Đảm bảo hidden/stopped/no-price còn đọc được, exact SKU, không fuzzy; không buộc UI biết trước Product UUID. Hai read + Inventory dùng budget 2 giây và cap reply 8 KiB; source thay đổi chặn submit/publish theo source hash cũ.

CAT-OUT-03: giữ x-internal-token theo CatalogAccess và CATALOG_INTERNAL_TOKEN ≥32 ký tự; không bỏ credential trong Traceability. Giữ port thực mặc định9005 thay ví dụ9090 trong plan. TLS nếu đã cấu hình, hoặc explicit private plaintext như guide; không mở RPC public. Nộp test provider thật cho binding/missing SKU/credential/hidden-no-price/fingerprint; không viết lại module đã đáp ứng.

MED-OUT-01: giữ [MediaInternalService](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/media/v1/media_service.proto), purpose OCOP_CERTIFICATE, fileId/READY/MIME/size/checksum theo [guide](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-grpc.md#media--chứng-nhận-ocop-theo-plan-dương). Traceability không nhận binary/raw object key; evidence.reference=document:<UUID> ánh xạ sang file_id. Không đổi schema certificates đã frozen.

MED-OUT-02: BindMediaToEntity(file_id, TraceabilityProfile, profile_id) phải retry/idempotent được; ràng buộc file không bị GC sau gắn kết. Traceability enqueue media_bind_jobs cùng publish transaction, worker gọi sau commit. Media hiện chỉ có một bound_entity: giữ file được bảo vệ khi dùng chung qua nhiều lô; không dùng trường đó làm quyền sở hữu thay IAM của Traceability.

MED-OUT-03: ghi rõ OCOP_CERTIFICATE baseline PUBLIC_READ và RPC9007 chưa có auth/TLS interceptor. Chỉ upload bản scan được phép công khai, chỉ expose RPC trong mạng tin cậy. Không gọi PublishMedia để đổi sang bucket Catalog. Nếu triển khai có policy khác, cung cấp cấu hình/test chính xác, không coi việc giấu reference trong Traceability là thu hồi file đã public.

**Đầu ra cuối:** SHA/provider config đang dùng, test IDs CAT-*/GRPC-CAT-*/MED-* và bằng chứng đã lọc. Provider có sẵn thì test/tái sử dụng; chỉ sửa gap thực sự trong phần phụ trách, không viết proposal RPC thay thế hoặc chờ ACCEPT.

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
| GRPC-CAT-01 | GetProductVariant UNSPECIFIED → GetTraceabilityBinding, hidden/stopped/no-price, sai x-internal-token | Bootstrap/verify dùng canonical Dương, fingerprint nguyên trạng; credential được kiểm theo CatalogAccess |
| GRPC-CAT-02 | NOT_FOUND/timeout/UNIMPLEMENTED/DB failure, version đổi | Phân biệt absent/conflict/503, không HTTP fallback hoặc constant version |
| MED-01 | OCOP evidence document:<UUID>, duplicate fileId, raw object key/URL | Dedup fileId; raw URL/key không vượt kiểm Media, giữ schema certificates cũ |
| MED-02 | READY đúng purpose/MIME/size/SHA và file missing/processing/wrong purpose | READY hợp lệ mới submit/publish; invalid400, malformed/provider lỗi503, không đổi receipt |
| MED-03 | Publish enqueue job lỗi; retry cùng key; repeated publication replay | Rollback cả publish nếu không ghi job; retry giữ một publication/obligation; hash/QR không đổi |
| MED-04 | Bind RPC lỗi/crash/restart/concurrent workers | Pending được giữ; lease độc quyền, retry sau lỗi/lease hết; success không bị reset, peer call idempotent |
| MED-05 | Legacy frozen evidence reference và Media flag | Không sửa snapshot cũ; muốn publish với Media enabled phải draft/reference/review hợp lệ |
| MED-06 | OCOP file PUBLIC_READ, worker/media port private | Chỉ upload scan được phép public; không coi evidence permission là bảo vệ object public; không PublishMedia sang bucket Catalog |

Nộp kết quả cuối/PR: OUT đã đáp ứng, SHA, phần tái sử dụng/bổ sung, mapping adapter nếu có, PASS/FAIL/SKIPPED từng test và bằng chứng đã lọc. Chưa có peer chạy thật vẫn code/test bằng fixture đúng contract, ghi `IMPLEMENTED_NOT_INTEGRATED`; không báo full E2E bằng mock. Chỉ báo blocker vượt scope, không gửi phương án rồi chờ duyệt trước sửa code.

Đọc AGENTS.md và rule service trong repository trước khi sửa. Commit mới dùng tiếng Anh, giữ một type/scope; contract thay trước hoặc cùng code, không sửa migration cũ/generated rows sai quy trình. Không tự push dev/release hoặc deploy production; tạo PR theo quy trình nhóm.

## Phần bổ sung riêng và ranh giới phối hợp

Chốt ngày 27/11/2026 theo chủ nghiệp vụ: phần chưa có trong plan Dương được triển khai thành phần bổ sung riêng, tương thích với phần Dương đang làm. Baseline đã fetch lại: `dev@ac5395c5` / PR #175. Đây là phân chia phạm vi triển khai, không thay wire contract 0.4.0 và không tuyên bố các provider bổ sung đã chạy.

## 1. Hai phạm vi công việc

| Phạm vi | Nguồn chuẩn | Trách nhiệm |
| --- | --- | --- |
| Theo plan Dương | CatalogService trong catalog.proto; MediaInternalService trong media_service.proto; pricing, Media và video workflows tương ứng | Tái sử dụng implementation đã merge. Traceability làm consumer/adapters theo canonical, giữ fingerprint và semantics fileId. Không giao lại GetSkuBinding đã gỡ |
| Bổ sung truy xuất | Guide, OpenAPI/candidates và phiếu owner Traceability 0.4.0 | Inventory warehouse/receipt binding; IAM scope/current grants/BFF; packed_lines exact sealed proof; Order customer traceability; UI nhập kho/hồ sơ/QR/đơn hàng |

Owner contract truy xuất duy trì nguồn chuẩn và phát lại snapshot. Theo quyền chủ dự án xác nhận ngày27/11, người phụ trách Traceability được sửa tối thiểu các service khác để hoàn thành luồng, không phải chờ mỗi service có một owner riêng. Dương tiếp tục plan của mình; ta triển khai phần bổ sung bằng commit/PR riêng, giới hạn đúng adapter/endpoint/permission/proof/UI cần thiết. Không refactor diện rộng hoặc đổi nghiệp vụ của service khác. Không nhét phần bổ sung vào PR pricing/Media đang làm mà không thể review riêng. Không tự sửa worktree hoặc nhánh đang làm của Dương. Nhánh riêng lấy nền dev hiện hành; không cherry-pick toàn bộ nhánh Traceability vào service khác chỉ để lấy tài liệu.

## 2. Phần việc và ranh giới sở hữu

| Task bổ sung | Owner triển khai | Đầu ra / phần giữ tương thích |
| --- | --- | --- |
| INV-TRC-01 | Inventory | warehouse_id field9 của BatchSource và receipt binding. Giữ field1–8, UUID lô và nghiệp vụ tồn/FEFO; không thêm auto-create profile |
| IAM-TRC-01 | Identity/Keycloak | Scope kho đúng permission, current-grants RPC và BFF/Origin theo phiếu. Tái sử dụng query/principal/assignment hiện có; không đổi Media client, login/refresh hoặc RPC tài chính của Dương |
| FUL-TRC-01 | Fulfillment | packed_lines field14 của đúng sealed proof. Giữ base proof, seal/video/parcel và Shipping; không lấy live allocation thay lịch sử |
| ORD-TRC-01 | Order | Customer traceability read + consumer resolver, kiểm ownership trước peer. Giữ pricing snapshot/ValidateCheckoutPricing và lifecycle/payment/shipping hiện có |
| WEB-TRC-01 | Web/Gateway | Nối receipt→context/profile, upload OCOP qua Media, in QR và nguồn gốc từ đơn. Dùng REST hiện có; không đổi pricing UI/API hoặc gọi gRPC trong browser |
| Traceability backend | Traceability | Profile/revision, scoped authorization, resolver, consumer Catalog/Media, durable media binding. Không ghi DB của service khác |

IAM current-grants và BatchLinkResolver là target của **phần bổ sung**, không phải RPC được lấy từ plan Dương hoặc đã có trong PR #175. Người phụ trách triển khai trực tiếp theo phiếu, không cần vòng ACCEPT trước code; kết quả vẫn phải review/test trước merge và rollout.

## 3. Các điểm tránh conflict

- **Canonical Catalog/Media:** giữ nguyên packages/proto/catalog/v1/catalog.proto và packages/proto/media/v1/media_service.proto của baseline khi làm phần bổ sung. Dùng adapter bên consumer; không tạo definition trùng, đổi method/field hoặc khôi phục REST đã decommission. Nếu phát hiện gap thật, nêu đúng OUT/test bị chặn và giữ compatibility, không âm thầm đổi contract của plan Dương.
- **Proto mới:** IAM dùng file staff_authorization.proto riêng; resolver dùng batch_links.proto riêng, package/service names riêng. Không chèn RPC mới vào identity.proto/CatalogService chỉ để làm giống naming. Inventory/Packing candidate compile riêng; chỉ owner provider đưa field9/14 vào canonical, giữ các field/RPC cũ. Consumer lấy đúng canonical đã merge rồi codegen lại, không tự publish thêm một bản definition.
- **Migration:** chỉ thêm migration mới trong service sở hữu DB. Không sửa V15/V17 pricing/media, V20/V21 IAM hoặc migration Traceability000001/000002. Kiểm số mới nhất trên dev và nhánh owner trước khi đặt tên; nếu trùng số chưa merge thì đổi số migration mới trước rollout, không sửa checksum bản đã áp dụng. Traceability000003 chỉ thêm local media_bind_jobs.
- **OpenAPI/state matrix/registry:** Traceability owner duy trì bộ draft bổ sung và exporter. Phiếu gửi là generated snapshot, không sửa tay. Owner không ghi đè cả admin/customer stable hoặc generator output bằng bản từ nhánh khác; delta REST thật cập nhật contract tương ứng theo AGENTS.md. Contract stable không import draft.
- **Route/port/auth:** Catalog giữ port cấu hình thực9005 + x-internal-token; Media9007 giữ policy provider. Identity RPC bổ sung dùng TLS listener riêng (mặc định9116 theo guide), resolver9003 nội bộ; kiểm deployment override thực tế trước bind. Không chuyển legacy Identity9016 sang TLS làm hỏng caller cũ; không expose RPC mới qua public wildcard. Không dùng permission/role của Media thay scope kho.
- **Dữ liệu/events:** giữ namespace/Batch UUID/SKU, fingerprint, public hash và QR. Không đổi packing event v2, thêm activation event hoặc tự publish QR từ GoodsReceived. Lỗi traceability không đảo receipt hay chặn Order lifecycle; lỗi base proof thật vẫn bị từ chối theo baseline.

Những ranh giới này giảm trùng sửa và breaking change; Git vẫn có thể conflict ở index/registry/state matrix dùng chung. Khi đó merge từng phần theo nguồn chuẩn, không chọn toàn bộ ours/theirs hoặc xóa nội dung owner khác để hết marker.

## 4. Thứ tự tích hợp và yêu cầu đầu ra

1. Ghim baseline dev và source SHA của phiếu trong PR. Owner làm phần độc lập bằng fixture đúng contract, ghi rõ IMPLEMENTED_NOT_INTEGRATED nếu chưa có provider thật. Không ghi E2E PASS từ mock.
2. Tích hợp Inventory/IAM cho scoped workflow; Catalog/Media đi theo provider Dương đã có. Chỉ bật Media sau migrate/regrant và kiểm upload complete/READY; giữ job pending khi rollback flag.
3. Order hỗ trợ proof extension và fail-soft trước khi Fulfillment phát field14; resolver sẵn sàng trước bật customer link. Thiếu extension chỉ làm traceability UNAVAILABLE, không phá base packing/shipping.
4. Web dùng các provider đã đạt OUT; kiểm receipt độc lập, scope, review, upload, QR decode/print và nguồn gốc đơn. Không nghiệm thu bằng deploy chưa chứa commit mới.
5. Trước merge, fetch/rebase dev mới nhất và kiểm diff chỉ còn phần bổ sung của service. Build/test cả khả năng mới và đường cũ bị chạm: Catalog binding/pricing, Media upload/validation/bind, IAM login/grants, Inventory receipt, sealed proof/shipping, Order checkout và traceability tùy phạm vi. Chạy gate contract chung nếu sửa shared contract; chưa đủ peer giữ flag tắt.

PR nộp: task/OUT, base SHA + head SHA, files/contracts/migrations thực sự đổi, phần tái sử dụng từ plan Dương, PASS/FAIL/SKIPPED và giới hạn fixture/provider thật, thứ tự bật/rollback. Đây là kết quả sau triển khai, không phải mẫu xin duyệt trước. Trưởng nhóm vẫn duyệt PR vào dev/release; không tự deploy production.

## 5. Đối chiếu Git/PR và phân công thực tế — 27/11/2026

Đã fetch lại dev và nhánh Order của Dương; kiểm tra read-only, không checkout/merge nhánh đang làm của Dương vào nhánh Traceability.

### Catalog: baseline đã hoàn thành và merge, còn follow-up hỗ trợ Order

[PR #175](https://github.com/hvduong2k5/DUT.PBL6/pull/175) do hvduong2k5 mở đã MERGED vào dev lúc20:21 ngày27/11 (Việt Nam), merge SHA ac5395c5ec916c61b627fffef7322af07960dd59. Job [Catalog service tests](https://github.com/hvduong2k5/DUT.PBL6/actions/runs/46002489496/job/140517391691) SUCCESS; step Run Gradle tests thực sự success, không phải skip. Đây là kết quả CI của PR, không phải full provider/UI E2E chạy lại trong lượt xác nhận.

| Phần plan Catalog | Code cụ thể trên dev | Kết luận |
| --- | --- | --- |
| Pricing gRPC và binding truy xuất | [CatalogGrpcService.java](https://github.com/hvduong2k5/DUT.PBL6/blob/ac5395c5ec916c61b627fffef7322af07960dd59/services/catalog-service/src/main/java/com/hvduong/catalog/grpc/CatalogGrpcService.java#L54): validateCheckoutPricing; getTraceabilityBinding tại462, fingerprint variant_id:product_version:variant_version | Có implementation và unit tests success/not-found/product-mismatch; Traceability đã dùng canonical này |
| Giá theo thời gian | [CatalogAdminController.java](https://github.com/hvduong2k5/DUT.PBL6/blob/ac5395c5ec916c61b627fffef7322af07960dd59/services/catalog-service/src/main/java/com/hvduong/catalog/api/CatalogAdminController.java#L592): schedulePrice và deleteScheduledPrice tại627; service impl tại1364/1461, tests tương ứng tại621–801 | Có code thật; hai checkbox chưa tick trong bản plan local không phản ánh hết code đã merge |
| Bỏ snapshot Catalog, tăng price revision | V15__add_price_revision_and_drop_checkout_snapshots.sql; ProductVariantMapper.xml increment price_revision; internal REST controller/snapshot classes đã bị xóa | Đã có trong baseline; không tạo lại HTTP adapter hoặc bảng snapshot cũ |
| Media ảnh và moderation MVP | MediaIntegrationClient.java; V17__mvp_drop_scan_status_requirement.sql; CatalogAdminServiceImpl gọi validate/bind/publish | Có adapter/flag và migration; chưa suy cấu hình provider thật đã bật hoặc public UI đã nghiệm thu |

Kết luận chính xác: **Catalog baseline của plan đã merge, có code và CI PASS**. Không diễn đạt “toàn bộ Catalog không còn thay đổi”: nhánh Order đang sửa thêm CatalogAdminServiceImpl.java và test để bổ sung event dữ liệu giá cho projection.

### Order: có code đang làm theo plan, chưa có PR cho nhánh đã tìm được

Nhánh remote `feat/order-catalog-pricing-projection`, HEAD a45da2176698dbf8528cace849e69908f891e664; commit author hvduong, subject “feat(catalog,order): add Kafka price projection, gRPC circuit-breaker, and refactor CatalogEvidence schema”. Truy vấn PR theo đúng head trả danh sách rỗng tại thời điểm kiểm tra; không suy công việc chưa push hoặc đã merge chỉ từ lời nhắn.

- [catalog_projection.go](https://github.com/hvduong2k5/DUT.PBL6/blob/a45da2176698dbf8528cace849e69908f891e664/services/order-service/internal/app/catalog_projection.go): consumer catalog.events.v1, UPSERT sku_prices_projection có price_version guard; migration000013 tạo bảng.
- [checkout.go](https://github.com/hvduong2k5/DUT.PBL6/blob/a45da2176698dbf8528cace849e69908f891e664/services/order-service/internal/app/checkout.go#L48): enrichCart đọc projection, thiếu SKU mới fallback gRPC. catalog.go/core_checkout.go chuyển checkout/revalidate sang ValidateCheckoutPricing; runtime.go khởi chạy consumeCatalog.
- internal/domain/core/snapshot.go đổi CatalogEvidence; adapter/test/customer_views/persistence và hai file Catalog cũng đang đổi. Đã đối chiếu source với plan, chưa chạy test nhánh này hoặc xác nhận toàn bộ DoD đã đạt. Plan local còn ghi ValidatePriceAndSKU nhưng canonical hiện hành là ValidateCheckoutPricing; không thêm RPC tên cũ.

### Phần ta có thể làm trước và vùng tránh sửa chồng

Ta có quyền thực hiện các bổ sung Inventory warehouse/receipt, Identity grants/BFF, Fulfillment packed_lines, Traceability và Web cần thiết, với diff nhỏ và tests cho đường cũ bị chạm. Không phải chờ đủ owner service phản hồi. Giữ scope nhập kho/hồ sơ/review/QR/đơn đã chốt.

Bên Order, ưu tiên file riêng cho traceability read/provider/DTO/tests và giữ cơ chế ownership. Không refactor pricing, projection, cart enrichment, core snapshot/hash, catalogprovider hoặc migration000013 mà Dương đang làm. Wiring/route/customer_views dùng chung chỉ sửa tối thiểu trên baseline đã tích hợp phần Dương khi cần; kiểm lại head trước động vào. CatalogAdminServiceImpl và test của nó vẫn là vùng Dương sửa cho event giá: ta tái sử dụng binding RPC, không chạm các file này nếu không có gap bắt buộc.

Không đảm bảo Git tuyệt đối không conflict: nhánh remote có thể tiếp tục thay đổi hoặc Dương còn code local chưa push. Phần làm độc lập vẫn tiếp tục; trước tích hợp cập nhật baseline, so diff theo file và chạy regression/contract gate. Không gửi tin nhắn cho Dương hoặc sửa nhánh của bạn ấy bằng công cụ trong lượt xác nhận này.


## Checkpoint triển khai local — 28/11/2026

Phần bổ sung Inventory/IAM/Fulfillment/Order/Web/Gateway đã được triển khai trên `codex/traceability-order-flow`; xem [kết quả local và phạm vi chưa nghiệm thu](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/04_testing/traceability-service/11_local_workflow_2026-11-28.md) và [Web BFF contract](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-web-bff-draft.openapi.yaml). Luồng nhập kho → draft → Media scan → người khác duyệt/publish → QR đã chạy với provider thật. Order read được kiểm PostgreSQL + gRPC peers fixture; không suy ra toàn bộ checkout/shipping/Kafka đã nghiệm thu. Dương tiếp tục plan Order của bạn ấy; không làm lại các phần bổ sung đã có trên nhánh này, chỉ ghép và bảo đảm đầu ra/contract khi review. Target vẫn0.4.0-draft, chưa deploy production.

Identity cookie alias là `/api/v1/admin/trace-api/**` để tuân namespace rule của Identity; backend Traceability scoped vẫn `/api/v2/admin/trace/**`. Không route internal RPC qua Kong. Migration IAM V27 chỉ seed INVENTORY_VIEW và WAREHOUSE_WORKBENCH_VIEW còn thiếu, không tự gán GLOBAL.

## Quy định transport mới và contract gRPC

**Phạm vi bổ sung đã chốt 27/11:** [kế hoạch bổ sung và ranh giới tránh conflict](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-supplement.md) tách phần Inventory/IAM/Fulfillment/Order/Web còn thiếu khỏi plan Catalog/Media của Dương. IAM current-grants và resolver thuộc phần bổ sung, không gán là RPC đã có trong plan Dương. Giữ contract 0.4.0; owner triển khai theo phiếu và test, không cần ACCEPT trước code.

Chốt ngày 27/11/2026 theo plan của Dương và provider đã merge trong PR #175, `dev@ac5395c5`. Bản này thay phiếu 0.3.0 về Catalog và bổ sung tích hợp Media. Nghiệp vụ nhập kho độc lập hồ sơ, scope kho, duyệt độc lập và QR theo lô giữ nguyên. Không yêu cầu Dương triển khai CatalogTraceabilitySource/GetSkuBinding; target chưa phát hành đó đã được gỡ.

## Contract và ranh giới

| Kết nối | Canonical/target | Mức sẵn sàng |
| --- | --- | --- |
| Traceability → Catalog | [CatalogService.GetProductVariant/GetTraceabilityBinding](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/catalog/v1/catalog.proto) | Provider đã có trên dev; consumer theo đúng proto này |
| Traceability → Media | [MediaInternalService.ValidateMediaReference/BindMediaToEntity](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/media/v1/media_service.proto) | Provider đã có trên dev; adapter và durable binding của Traceability theo bản này |
| Traceability → Inventory | [BatchSource candidate warehouse9](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-candidates/batch_source.proto) | Giữ yêu cầu scope đã chốt |
| Traceability → Identity | [StaffAuthorization.GetCurrentStaffGrants](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/identity/v1/staff_authorization.proto) | Còn là yêu cầu mới cho owner IAM; PR #175 chỉ thêm Media client, chưa có RPC này |
| Order → Traceability | [BatchLinkResolver.ResolveBatchLinks](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/packages/proto/traceability/v1/batch_links.proto) | Traceability đã implement; consumer Order còn phải tích hợp |
| Order → Fulfillment | [Packing candidate field14](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-workflow-candidates/packing.proto) | Giữ exact sealed proof, không dùng live allocation |
| Web/BFF/admin/customer/public | REST/OpenAPI | Không viết gRPC browser client |

Không thay canonical Catalog/Media method/message để khớp proposal riêng. Generate clients từ đúng file và common imports. Root AGENTS.md hiện đã bỏ internal REST Catalog và catalog-checkout OpenAPI; không khôi phục các artifact đã decommission trong PR #175. OIDC discovery/JWKS giữ HTTPS, BFF proxy giữ REST/Origin/CSRF; Kafka vẫn thông báo sau local commit, không kích hoạt QR.

## Catalog — tái sử dụng provider của Dương

- Verify create/submit/publish: `/omamx.catalog.v1.CatalogService/GetTraceabilityBinding`, gửi product_id + exact sku_code đã lưu. Đối chiếu response echo; lưu **nguyên version_fingerprint** vào catalog_version của dependency snapshot. Giữ fingerprint `variant_id:product_version:variant_version`, không đổi hash/QR/snapshot đã phát hành.
- Bootstrap khi chỉ có SKU: `GetProductVariant` với channel UNSPECIFIED lấy Product UUID (provider hiện đọc cả hidden/stopped và không bắt buộc giá), sau đó gọi `GetTraceabilityBinding` để xác nhận binding và lấy fingerprint. Hai read dùng cùng context với Inventory, budget toàn 2 giây. Không suy Product từ tên hoặc Inventory SKU detail.
- Giữ metadata **x-internal-token** và credential CATALOG_INTERNAL_TOKEN đúng CatalogAccess hiện có. Traceability cấu hình giá trị đó qua TRACEABILITY_CATALOG_TOKEN; không bỏ credential như gợi ý cũ trong plan vì server hiện vẫn bắt buộc token ít nhất 32 ký tự. Không dùng staff JWT làm service token.
- Target cấu hình `TRACEABILITY_CATALOG_GRPC_ADDR`, baseline `catalog-service:9005`; port 9090 trong plan chỉ là ví dụ, dùng port deployment thật. Không yêu cầu listener 9105 hoặc Bearer adapter mới.
- TLS qua trusted CA là mặc định. Provider dev hiện plaintext: chỉ được bật `TRACEABILITY_CATALOG_GRPC_PLAINTEXT=true` rõ ràng cho loopback/private IP hoặc DNS catalog-service trong mạng nội bộ đã cô lập. Không có TLS→plaintext fallback tự động; không publish host/public port.
- Reply cap 8 KiB; exact case/ID/echo; NOT_FOUND → absent/conflict theo action, INVALID_ARGUMENT → input lỗi, các lỗi auth/config/UNIMPLEMENTED/timeout/oversize → dependency503 an toàn. Không HTTP fallback.

## Media — chứng nhận OCOP theo plan Dương

1. Web tạo upload session với purpose OCOP_CERTIFICATE qua Media REST hiện có, upload binary thẳng object storage, complete session. Business API Traceability chỉ nhận immutable fileId dưới **evidence.reference = document:<canonical UUID>**; certificate.evidence_ids trỏ tới evidence ID như contract cũ. Không thêm raw object_key/URL hoặc thay hình dạng certificate đã lưu. Nếu đã có file hợp lệ thì tái sử dụng fileId.
2. Khi bật TRACEABILITY_MEDIA_ENABLED, trước submit và publish Traceability gọi ValidateMediaReference(file_id, expected_purpose=OCOP_CERTIFICATE) cho các evidence của chứng nhận OCOP, dedup theo fileId. Yêu cầu is_valid=true, READY, size>0, checksum SHA-256 64 hex, object key không rỗng và MIME PDF/JPEG/PNG/WebP. Không tuyên bố đã quét antivirus. Read source + Media cùng budget 2 giây; metadata malformed/RPC lỗi →503, file không hợp lệ/chưa READY →400. Không đưa object key/checksum vào public DTO.
3. Publish transaction ghi public pointer/audit/outbox và **media_bind_jobs** nguyên tử. Worker sau commit gọi BindMediaToEntity(file_id, TraceabilityProfile, profile_id), không gọi peer trong DB transaction. Lease 15 giây, mỗi RPC 2 giây, lỗi retry sau 5 giây không giới hạn; worker restart tiếp tục job còn pending. RPC có thể lặp nếu crash sau peer ACK, provider binding phải idempotent. Completed job không reset; replay publish không tạo hai obligation.
4. Media tạm lỗi khi binding không đảo publish đã commit, receipt hoặc Order lifecycle. Job pending là nghĩa vụ vận hành phải theo dõi và drain trước rollback. Không tự gọi PublishMedia cho OCOP vì method hiện chuyển file sang bucket Catalog, không phải nghiệp vụ Traceability.
5. Không backfill/di chuyển evidence của revision đã frozen/published. Legacy reference không theo document:<UUID> vẫn đọc nguyên trạng; khi bật kiểm Media, muốn submit/publish phải tạo/sửa draft hợp lệ rồi duyệt lại. Cờ Media mặc định false cho rollout, không gọi provider giả để đánh dấu verified.

Provider Media baseline dùng :9007 và chưa có interceptor xác thực/TLS. Client dùng TRACEABILITY_MEDIA_GRPC_ADDR + CA, hoặc explicit TRACEABILITY_MEDIA_GRPC_PLAINTEXT=true chỉ cho loopback/private IP/DNS media-service. Mạng nội bộ là biên tin cậy của baseline; không thêm token rồi gọi đó là authorization nếu provider không kiểm. Owner Media phải bảo đảm không expose RPC public và fileId/READY/checksum/binding đúng semantics; ghi rõ cấu hình thực trong delivery.

**Giới hạn chính sách Media phải giữ rõ:** OCOP_CERTIFICATE hiện được gán PUBLIC_READ ngay từ upload theo plan/provider. Đây chỉ dành cho bản scan đã được phép công khai; không upload tài liệu nội bộ/PII vào purpose này. Quyền traceability.read_evidence chỉ che reference trong Traceability, không thể thu hồi object đã public. Muốn kiểm duyệt trước khi public hoặc chứng từ private phải có policy/contract Media riêng, không giả đã được bảo vệ bằng QR eligibility. Media hiện chỉ lưu một bound_entity; tái dùng cùng file cho nhiều lô chỉ đạt retention nếu binding không làm file mất bảo vệ hoặc ownership cũ. Không suy grant scope từ bound_entity của Media.

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

## Rollout và đầu ra

Catalog/Media dùng implementation đã merge, không giao lại việc viết RPC GetSkuBinding. Owner đảm bảo provider contract/test/config đúng; Traceability chịu adapter, validate và durable binding. Identity current grants và Order resolver là các kết nối còn thiếu trong plan Dương, vẫn phải giao riêng theo phần trên; không suy PR #175 đã làm đủ chúng.

Migrate 000003 rồi cấp lại GrantRuntime để bổ sung media_bind_jobs. Không sửa checksum migration000001/000002. Bật Media sau khi upload complete/READY và provider integration đã kiểm; khi tắt flag, worker dừng, job cũ được giữ và cần bật worker lại để drain. Không drop pending jobs khi rollback. Scoped/resolver flags độc lập và giữ yêu cầu Inventory/Identity hiện có. Không merge/release/deploy production trong đợt local này.


## Checkpoint triển khai local — 28/11/2026

Phần bổ sung Inventory/IAM/Fulfillment/Order/Web/Gateway đã được triển khai trên `codex/traceability-order-flow`; xem [kết quả local và phạm vi chưa nghiệm thu](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/04_testing/traceability-service/11_local_workflow_2026-11-28.md) và [Web BFF contract](https://github.com/hvduong2k5/DUT.PBL6/blob/95f9ab117cca0cc231412dc3cbdfc878a9437386/docs/03_api_specs/traceability-web-bff-draft.openapi.yaml). Luồng nhập kho → draft → Media scan → người khác duyệt/publish → QR đã chạy với provider thật. Order read được kiểm PostgreSQL + gRPC peers fixture; không suy ra toàn bộ checkout/shipping/Kafka đã nghiệm thu. Dương tiếp tục plan Order của bạn ấy; không làm lại các phần bổ sung đã có trên nhánh này, chỉ ghép và bảo đảm đầu ra/contract khi review. Target vẫn0.4.0-draft, chưa deploy production.

Identity cookie alias là `/api/v1/admin/trace-api/**` để tuân namespace rule của Identity; backend Traceability scoped vẫn `/api/v2/admin/trace/**`. Không route internal RPC qua Kong. Migration IAM V27 chỉ seed INVENTORY_VIEW và WAREHOUSE_WORKBENCH_VIEW còn thiếu, không tự gán GLOBAL.

## Contract HTTP nhúng — không cần tải thêm file để đọc schema

Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.

```yaml
openapi: 3.1.0
info:
  title: PBL6 Catalog / Media traceability handoff
  version: 0.4.0-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 95f9ab117cca0cc231412dc3cbdfc878a9437386. Direct
    implementation target, not runtime acceptance. All references are local to this embedded contract.
tags:
- name: Catalog / Media
  description: Owner-specific provider and dependency contracts
servers:
- url: http://127.0.0.1
paths:
  /api/v1/media/upload-sessions:
    post:
      tags:
      - Catalog / Media
      operationId: postMediaUploadSessions
      summary: Create a direct-to-object-storage presigned upload session.
      x-lifecycle: stable
      x-backlog:
      - EPIC_14
      - EPIC_16
      - EPIC_18
      x-ui-states:
      - loading
      - success
      - error
      - unauthorized
      - forbidden
      parameters:
      - $ref: '#/components/parameters/MediaStable_IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
              - purpose
              - fileName
              - mediaType
              - sizeBytes
              properties:
                purpose:
                  type: string
                  enum:
                  - CATALOG_PRODUCT
                  - CATALOG_CATEGORY
                  - OCOP_CERTIFICATE
                  - SHIPPING_VIDEO
                  - B2B_DOCUMENT
                  - REFUND_EVIDENCE
                  - AVATAR
                fileName:
                  type: string
                  maxLength: 500
                mediaType:
                  type: string
                  maxLength: 128
                sizeBytes:
                  type: integer
                  minimum: 1
                  maximum: 104857600
                sha256:
                  type: string
                  maxLength: 64
              additionalProperties: false
            example:
              purpose: REFUND_EVIDENCE
              fileName: package.jpg
              mediaType: image/jpeg
              sizeBytes: 420000
      responses:
        '201':
          description: Upload session created with presigned PUT URL.
          content:
            application/json:
              schema:
                type: object
                required:
                - uploadSessionId
                - fileId
                - objectKey
                - bucket
                - uploadUrl
                - expiresAt
                - requiredHeaders
                properties:
                  uploadSessionId:
                    type: string
                  fileId:
                    type: string
                    format: uuid
                  objectKey:
                    type: string
                  bucket:
                    type: string
                  uploadUrl:
                    type: string
                    format: uri
                  expiresAt:
                    type: string
                    format: date-time
                  requiredHeaders:
                    type: object
                    additionalProperties:
                      type: string
                additionalProperties: false
              examples:
                success:
                  value:
                    uploadSessionId: sess_01JDJ89K20
                    fileId: 9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d
                    objectKey: refunds/2026/11/25/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d.jpg
                    bucket: om-documents
                    uploadUrl: https://s3.omama.vn/om-documents/refunds/2026/11/25/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d.jpg?X-Amz-Expires=900
                    expiresAt: '2026-11-25T16:30:00Z'
                    requiredHeaders:
                      Content-Type: image/jpeg
        '400':
          $ref: '#/components/responses/MediaStable_BadRequest'
        '401':
          $ref: '#/components/responses/MediaStable_Unauthorized'
        '403':
          $ref: '#/components/responses/MediaStable_Forbidden'
        '422':
          $ref: '#/components/responses/MediaStable_UnprocessableEntity'
        '500':
          $ref: '#/components/responses/MediaStable_InternalError'
        '503':
          $ref: '#/components/responses/MediaStable_ServiceUnavailable'
      security:
      - MediaStable_bearerAuth: []
      - MediaStable_customerSession: []
      servers:
      - url: https://api.omama.vn
        description: Production API gateway
  /api/v1/media/upload-sessions/{uploadSessionId}/complete:
    post:
      tags:
      - Catalog / Media
      operationId: postMediaUploadSessionsComplete
      summary: Confirm direct S3 upload completion and trigger WebP/thumbnail optimization.
      x-lifecycle: stable
      x-backlog:
      - EPIC_14
      - EPIC_16
      - EPIC_18
      x-ui-states:
      - loading
      - success
      - error
      - unauthorized
      - forbidden
      parameters:
      - name: uploadSessionId
        in: path
        required: true
        schema:
          type: string
          minLength: 1
      - $ref: '#/components/parameters/MediaStable_IdempotencyKey'
      responses:
        '200':
          description: Object verified in storage; async optimization started.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/MediaStable_MediaFileMetadata'
              examples:
                success:
                  value:
                    fileId: 9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d
                    objectKey: refunds/2026/11/25/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d.jpg
                    bucket: om-documents
                    fileName: package.jpg
                    mediaType: image/jpeg
                    sizeBytes: 420000
                    sha256Checksum: null
                    purpose: REFUND_EVIDENCE
                    accessLevel: PRIVATE_RESTRICTED
                    status: PROCESSING
                    variantsMeta: {}
                    createdAt: '2026-11-25T16:15:00Z'
                    updatedAt: '2026-11-25T16:16:00Z'
        '400':
          $ref: '#/components/responses/MediaStable_BadRequest'
        '401':
          $ref: '#/components/responses/MediaStable_Unauthorized'
        '403':
          $ref: '#/components/responses/MediaStable_Forbidden'
        '404':
          $ref: '#/components/responses/MediaStable_NotFound'
        '500':
          $ref: '#/components/responses/MediaStable_InternalError'
        '503':
          $ref: '#/components/responses/MediaStable_ServiceUnavailable'
      security:
      - MediaStable_bearerAuth: []
      - MediaStable_customerSession: []
      servers:
      - url: https://api.omama.vn
        description: Production API gateway
components:
  parameters:
    MediaStable_IdempotencyKey:
      name: Idempotency-Key
      in: header
      required: false
      description: Required for retryable create/action operations. Same key and same payload must return the same
        outcome.
      schema:
        type: string
        format: uuid
  responses:
    MediaStable_BadRequest:
      description: Malformed request or invalid query parameters.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/MediaStable_Error'
          example:
            code: BAD_REQUEST
            message: Malformed request or invalid query parameters.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    MediaStable_Unauthorized:
      description: Authentication is missing, invalid, or expired.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/MediaStable_Error'
          example:
            code: AUTHENTICATION_REQUIRED
            message: Authentication is missing, invalid, or expired.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    MediaStable_Forbidden:
      description: Authenticated actor lacks the required permission, capability, ownership, or data scope.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/MediaStable_Error'
          example:
            code: FORBIDDEN
            message: Authenticated actor lacks the required permission, capability, ownership, or data scope.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    MediaStable_UnprocessableEntity:
      description: Payload is syntactically valid but violates field or business validation rules.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/MediaStable_Error'
          example:
            code: VALIDATION_FAILED
            message: Payload is syntactically valid but violates field or business validation rules.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    MediaStable_InternalError:
      description: Unexpected server error.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/MediaStable_Error'
          example:
            code: INTERNAL_SERVER_ERROR
            message: Unexpected server error.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    MediaStable_ServiceUnavailable:
      description: A required service is temporarily unavailable.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/MediaStable_Error'
          example:
            code: SERVICE_UNAVAILABLE
            message: A required service is temporarily unavailable.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
    MediaStable_NotFound:
      description: Resource does not exist or is intentionally hidden outside the actor scope.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/MediaStable_Error'
          example:
            code: RESOURCE_NOT_FOUND
            message: Resource does not exist or is intentionally hidden outside the actor scope.
            requestId: req-01J9YAJ6CX8G5E0K7M8N9PQRST
            errors: []
  schemas:
    MediaStable_Error:
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
    MediaStable_MediaFileMetadata:
      type: object
      required:
      - fileId
      - objectKey
      - bucket
      - fileName
      - mediaType
      - sizeBytes
      - purpose
      - accessLevel
      - status
      - variantsMeta
      - createdAt
      - updatedAt
      properties:
        fileId:
          type: string
          format: uuid
        objectKey:
          type: string
        bucket:
          type: string
        fileName:
          type: string
        mediaType:
          type: string
        sizeBytes:
          type: integer
        sha256Checksum:
          type:
          - string
          - 'null'
        purpose:
          type: string
          enum:
          - CATALOG_PRODUCT
          - CATALOG_CATEGORY
          - OCOP_CERTIFICATE
          - SHIPPING_VIDEO
          - B2B_DOCUMENT
          - REFUND_EVIDENCE
          - AVATAR
        accessLevel:
          type: string
          enum:
          - PUBLIC_READ
          - AUTHENTICATED_READ
          - PRIVATE_RESTRICTED
        status:
          type: string
          enum:
          - AWAITING_UPLOAD
          - PROCESSING
          - READY
          - PROCESSING_FAILED
          - ARCHIVED
        variantsMeta:
          type: object
          additionalProperties:
            type: string
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
      additionalProperties: false
  securitySchemes:
    MediaStable_bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
    MediaStable_customerSession:
      type: apiKey
      in: cookie
      name: customer_session
x-source-commit: 95f9ab117cca0cc231412dc3cbdfc878a9437386
```

## Proto target nhúng: `packages/proto/catalog/v1/catalog.proto`

Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng. Không hand-edit generated code.

```proto
syntax = "proto3";

package omamx.catalog.v1;

import "google/protobuf/timestamp.proto";
import "common/v1/types.proto";
import "common/v1/errors.proto";

option go_package = "github.com/omamx/proto/catalog/v1;catalogv1";
option java_multiple_files = true;
option java_package = "com.omamx.proto.catalog.v1";

// =============================================================================
// HỢP ĐỒNG DỊCH VỤ DANH MỤC & GIÁ BÁN OCOP HUẾ (MS-05: CATALOG SERVICE)
// Điểm neo trên Critical Path: Thẩm định giá bán chống gian lận F12/Inspect
// =============================================================================

service CatalogService {
  // Thẩm định giá, SKU và trả metadata (quy cách, khối lượng, revision) cần thiết cho Order tạo Snapshot
  rpc ValidateCheckoutPricing(ValidateCheckoutPricingRequest) returns (ValidateCheckoutPricingResponse);

  // Truy vấn thông tin chi tiết của một biến thể sản phẩm theo mã SKU và kênh bán
  rpc GetProductVariant(GetProductVariantRequest) returns (GetProductVariantResponse);

  // Truy vấn danh sách biến thể sản phẩm theo bộ lọc (Có phân trang)
  rpc ListProductVariants(ListProductVariantsRequest) returns (ListProductVariantsResponse);

  // Truy vấn thông tin liên kết sản phẩm và SKU phục vụ truy xuất nguồn gốc (OCOP Traceability)
  // Bao gồm cả các biến thể chưa mở bán công khai
  rpc GetTraceabilityBinding(GetTraceabilityBindingRequest) returns (GetTraceabilityBindingResponse);
}

// ── Enums chuẩn hóa ─────────────────────────────────────────────────────────

enum SalesChannel {
  SALES_CHANNEL_UNSPECIFIED = 0;
  SALES_CHANNEL_D2C_WEB = 1;
  SALES_CHANNEL_MOBILE_APP = 2;
  SALES_CHANNEL_B2B_WHOLESALE = 3;
  SALES_CHANNEL_POS_QUAY = 4;
  SALES_CHANNEL_SHOPEE = 5;
  SALES_CHANNEL_TIKTOK = 6;
}

enum OcopRating {
  OCOP_RATING_UNSPECIFIED = 0;
  OCOP_RATING_3_STAR = 3;
  OCOP_RATING_4_STAR = 4;
  OCOP_RATING_5_STAR = 5;
}

enum PackagingType {
  PACKAGING_TYPE_UNSPECIFIED = 0;
  PACKAGING_TYPE_HOP_GIAY_KRAFT = 1;
  PACKAGING_TYPE_HOP_THIEC = 2;
  PACKAGING_TYPE_GOI_HUT_CHAN_KHONG = 3;
  PACKAGING_TYPE_HOP_QUA_TANG = 4;
  PACKAGING_TYPE_DANG_ROI = 5;
}

enum StockStatus {
  STOCK_STATUS_UNSPECIFIED = 0;
  STOCK_STATUS_UNKNOWN = 1;
  STOCK_STATUS_IN_STOCK = 2;
  STOCK_STATUS_OUT_OF_STOCK = 3;
}

enum ValidationIssue {
  VALIDATION_ISSUE_UNSPECIFIED = 0;
  VALIDATION_ISSUE_PRICE_CHANGED = 1;
  VALIDATION_ISSUE_SKU_NOT_FOUND = 2;
  VALIDATION_ISSUE_SKU_INACTIVE = 3;
  VALIDATION_ISSUE_PRODUCT_NOT_APPROVED = 4;
  VALIDATION_ISSUE_PRICE_NOT_CONFIGURED = 5;
  VALIDATION_ISSUE_CURRENCY_MISMATCH = 6;
}

// ── Messages cho ValidateCheckoutPricing ───────────────────────────────────

message CheckoutItemInput {
  // Mã SKU của biến thể kẹo
  string sku_code = 1;

  // Số lượng mua (1 - 999)
  int32 quantity = 2;

  // Đơn giá mà phía Client (Web/Mobile) gửi lên để kiểm tra
  omamx.common.v1.Money client_unit_price = 3;
}

message ValidateCheckoutPricingRequest {
  // Kênh bán hàng bắt buộc
  SalesChannel channel = 1;

  // Danh sách các mặt hàng cần kiểm định (1 - 100 items, cấm trùng SKU)
  repeated CheckoutItemInput items = 2;
}

message ValidatedCheckoutItem {
  string product_id = 1;
  string variant_id = 2;
  string sku_code = 3;

  string product_name = 4;
  string variant_name = 5;

  int32 quantity = 6;

  omamx.common.v1.Money authoritative_unit_price = 7;
  omamx.common.v1.Weight net_weight = 8;

  PackagingType packaging_type = 9;

  bool is_sellable = 10;

  int64 product_version = 11;
  int64 variant_version = 12;
  int64 price_version = 13; // price_revision tăng đơn điệu theo SKU
}

message PriceDiscrepancy {
  string sku_code = 1;
  omamx.common.v1.Money client_unit_price = 2;
  omamx.common.v1.Money authoritative_unit_price = 3;
  ValidationIssue discrepancy_reason = 4;
}

message ValidateCheckoutPricingResponse {
  // Kết quả hợp lệ: true nếu toàn bộ mã hàng hợp lệ và đúng giá 100%
  bool is_valid = 1;

  // Danh sách kết quả chi tiết của tất cả các SKU
  repeated ValidatedCheckoutItem items = 2;

  // Tổng số tiền chuẩn xác của giỏ hàng được tính toán lại từ Database gốc của Catalog
  omamx.common.v1.Money canonical_subtotal = 3;

  // Danh sách chi tiết các mặt hàng bị sai lệch giá hoặc ngừng bán (nếu có)
  repeated PriceDiscrepancy discrepancies = 4;

  // Thời điểm Catalog thực hiện đánh giá
  google.protobuf.Timestamp evaluated_at = 5;

  // Chi tiết lỗi chuẩn hóa nếu phát hiện gian lận giá
  optional omamx.common.v1.ErrorDetail error = 6;
}

// ── Messages cho GetProductVariant ─────────────────────────────────────────

message GetProductVariantRequest {
  string sku_code = 1;
  SalesChannel channel = 2;
}

message ProductVariantDetail {
  string variant_id = 1;
  string sku_code = 2;
  string product_id = 3;
  string product_name = 4;
  string variant_name = 5;
  optional omamx.common.v1.Money unit_price = 6;
  omamx.common.v1.Weight net_weight = 7;
  OcopRating ocop_rating = 8;
  PackagingType packaging_type = 9;
  optional int32 shelf_life_days = 10;
  bool is_active = 11;
  int64 price_version = 12;
  StockStatus stock_status = 13;
  optional int64 stock_available = 14;
}

message GetProductVariantResponse {
  ProductVariantDetail variant = 1;
}

// ── Messages cho ListProductVariants ───────────────────────────────────────

message ListProductVariantsRequest {
  omamx.common.v1.PaginationRequest pagination = 1;
  optional string category_id = 2;
  optional OcopRating ocop_rating = 3;
  optional bool active_only = 4;
  SalesChannel channel = 5;
}

message ListProductVariantsResponse {
  repeated ProductVariantDetail variants = 1;
  omamx.common.v1.PaginationResponse pagination = 2;
}

// ── Messages cho GetTraceabilityBinding ────────────────────────────────────

message GetTraceabilityBindingRequest {
  string product_id = 1;
  string sku_code = 2;
}

message GetTraceabilityBindingResponse {
  string product_id = 1;
  string sku_code = 2;
  string version_fingerprint = 3; // variant_id:product_version:variant_version
  int64 product_version = 4;
  int64 variant_version = 5;
  string variant_id = 6;
}
```

## Proto target nhúng: `packages/proto/media/v1/media_service.proto`

Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng. Không hand-edit generated code.

```proto
syntax = "proto3";

package omamx.media.v1;

option go_package = "github.com/omamx/proto/media/v1;mediav1";
option java_multiple_files = true;
option java_package = "com.omamx.proto.media.v1";

service MediaInternalService {
  // Kiểm tra file có tồn tại, đúng purpose và status = READY trước khi Business Service lưu DB
  rpc ValidateMediaReference (ValidateMediaRequest) returns (ValidateMediaResponse);

  // Cấp Presigned GET URL ngắn hạn cho tài liệu mật hoặc ảnh chưa duyệt (dùng cho Admin/CSKH/Khách)
  rpc GetPresignedAccessUrl (GetAccessUrlRequest) returns (GetAccessUrlResponse);
  rpc GetPresignedDownloadUrl (GetDownloadUrlRequest) returns (GetDownloadUrlResponse);

  // Đánh dấu gắn kết file với thực thể nghiệp vụ (chống bị Garbage Collector dọn dẹp nhầm)
  rpc BindMediaToEntity (BindMediaRequest) returns (BindMediaResponse);

  // Chuyển ảnh đã duyệt (APPROVED) từ vùng private sang vùng public (om-catalog / CDN)
  rpc PublishMedia (PublishMediaRequest) returns (PublishMediaResponse);
}

message ValidateMediaRequest {
  string file_id = 1;
  string object_key = 2;
  string expected_purpose = 3;
}

message ValidateMediaResponse {
  bool is_valid = 1;
  string status = 2; // AWAITING_UPLOAD, PROCESSING, READY, PROCESSING_FAILED
  string object_key = 3;
  string mime_type = 4;
  int64 size_bytes = 5;
  string sha256_checksum = 6;
}

message GetAccessUrlRequest {
  string file_id = 1;
  string requesting_user_id = 2;
  string user_role = 3;
  int32 expiry_seconds = 4; // Mặc định 900s (15 phút)
  string disposition = 5;   // "inline" (xem/stream) hoặc "attachment" (tải về máy)
}

message GetAccessUrlResponse {
  string access_url = 1;
  string expires_at = 2;
  string disposition = 3;
}

message GetDownloadUrlRequest {
  string file_id = 1;
  string requesting_user_id = 2;
  string user_role = 3;
  int32 expiry_seconds = 4;
}

message GetDownloadUrlResponse {
  string download_url = 1;
  string expires_at = 2;
}

message BindMediaRequest {
  string file_id = 1;
  string object_key = 2;
  string entity_type = 3; // Product, TraceabilityProfile, FulfillmentTask, OrderRefund
  string entity_id = 4;
}

message BindMediaResponse {
  bool success = 1;
}

message PublishMediaRequest {
  string file_id = 1;
  string object_key = 2;
}

message PublishMediaResponse {
  bool success = 1;
  string public_object_key = 2;
}
```

## Proto target nhúng: `packages/proto/common/v1/types.proto`

Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng. Không hand-edit generated code.

```proto
syntax = "proto3";

package omamx.common.v1;

option go_package = "github.com/omamx/proto/common/v1;commonv1";
option java_multiple_files = true;
option java_package = "com.omamx.proto.common.v1";

// =============================================================================
// THƯ VIỆN KIỂU DỮ LIỆU CHUẨN DÙNG CHUNG TOÀN HỆ THỐNG (COMMON TYPES)
// Tuân thủ triệt để: google.type.Money pattern, BigInt-safe, Decimal string
// =============================================================================

// Chuẩn tiền tệ quốc tế hóa, chống tràn số Number.MAX_SAFE_INTEGER trên Node.js
message Money {
  // Mã tiền tệ 3 ký tự theo chuẩn ISO 4217 (Ví dụ: "VND", "USD")
  string currency_code = 1;

  // Phần nguyên của số tiền (Ví dụ: 110000 trong 110.000 VND).
  // Chú ý: Ở phía Node.js, bắt buộc parse trường này bằng BigInt(units), tuyệt đối không dùng Number().
  int64 units = 2;

  // Số đơn vị nano (10^-9) của số tiền. Với đồng Việt Nam (VND) không có hào/xu, trường này luôn = 0.
  int32 nanos = 3;
}

// Địa chỉ giao nhận hàng hóa 4 cấp chuẩn hóa hành chính Việt Nam
message Address {
  // Họ và tên người nhận kiện hàng
  string recipient_name = 1;

  // Số điện thoại người nhận (Định dạng E.164 hoặc số di động VN 10 chữ số)
  string phone_number = 2;

  // Địa chỉ chi tiết (Số nhà, ngõ, tên đường)
  string street_address = 3;

  // Phường / Xã / Thị trấn (Ví dụ: "Phường Thuận Hòa")
  string ward = 4;

  // Quận / Huyện / Thị xã / Thành phố trực thuộc tỉnh (Ví dụ: "Thành phố Huế")
  string district = 5;

  // Tỉnh / Thành phố trực thuộc trung ương (Ví dụ: "Tỉnh Thừa Thiên Huế")
  string province = 6;

  // Mã quốc gia theo ISO 3166-1 alpha-2 (Mặc định là "VN")
  string country_code = 7;

  // Mã bưu chính (Optional)
  optional string postal_code = 8;
}

// Đơn vị đo lường trọng lượng kẹo mè xửng và nguyên liệu
enum WeightUnit {
  WEIGHT_UNIT_UNSPECIFIED = 0;
  WEIGHT_UNIT_GRAM = 1;
  WEIGHT_UNIT_KILOGRAM = 2;
}

// Kiểu định lượng trọng lượng với độ chính xác cao (Tránh sai số IEEE-754 giữa Node.js, Python và Go)
message Weight {
  // Giá trị khối lượng biểu diễn dưới dạng chuỗi số thập phân (Ví dụ: "500.00", "0.250")
  string decimal_value = 1;

  // Đơn vị đo lường
  WeightUnit unit = 2;
}

// Tham số phân trang dùng cho các truy vấn danh sách (Query List)
message PaginationRequest {
  // Số thứ tự trang cần lấy (Bắt đầu từ 1)
  int32 page = 1;

  // Số lượng bản ghi tối đa trên một trang (Mặc định: 20, Tối đa: 100)
  int32 page_size = 2;
}

// Kết quả phân trang trả về từ các truy vấn danh sách
message PaginationResponse {
  // Trang hiện tại
  int32 current_page = 1;

  // Số lượng phần tử trên trang hiện tại
  int32 page_size = 2;

  // Tổng số lượng bản ghi thỏa mãn điều kiện lọc
  int64 total_items = 3;

  // Tổng số trang
  int32 total_pages = 4;
}

// Bối cảnh kiểm toán (Audit Context) được truyền qua gRPC Metadata để phục vụ Audit Plane
message AuditContext {
  // Mã định danh người dùng hoặc hệ thống thực hiện thao tác (Ví dụ: "STAFF-WH-02", "CUST-9876")
  string actor_id = 1;

  // Vai trò của người thực hiện (Ví dụ: "CUSTOMER", "WAREHOUSE_STAFF", "SYSTEM_SAGA")
  string actor_role = 2;

  // Địa chỉ IP của client gọi yêu cầu
  string client_ip = 3;

  // Chuỗi vết dấu phân tán W3C Distributed Tracing (traceparent)
  string traceparent = 4;

  // User-Agent của ứng dụng client
  optional string user_agent = 5;
}
```

## Proto target nhúng: `packages/proto/common/v1/errors.proto`

Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng. Không hand-edit generated code.

```proto
syntax = "proto3";

package omamx.common.v1;

option go_package = "github.com/omamx/proto/common/v1;commonv1";
option java_multiple_files = true;
option java_package = "com.omamx.proto.common.v1";

// =============================================================================
// MÔ HÌNH LỖI NGHIỆP VỤ THỐNG NHẤT TOÀN HỆ THỐNG (UNIFIED ERROR MODEL)
// Map trực tiếp vào google.rpc.Status & google.rpc.ErrorInfo của gRPC
// Đồng thời cho phép API Gateway dịch chuẩn sang HTTP Status + JSON Body
// =============================================================================

enum ErrorCode {
  // 0 BẮT BUỘC là UNSPECIFIED (Tuân thủ luật cứng của Protobuf3)
  ERROR_CODE_UNSPECIFIED = 0;

  // --- LỖI HỆ THỐNG DÙNG CHUNG (1 - 99) ---
  ERR_INTERNAL_SERVER_ERROR = 1;
  ERR_INVALID_ARGUMENT = 2;
  ERR_UNAUTHENTICATED = 3;
  ERR_PERMISSION_DENIED = 4;
  ERR_RESOURCE_NOT_FOUND = 5;
  ERR_IDEMPOTENCY_CONFLICT = 6;       // Trùng Idempotency Key nhưng payload gửi lên khác bản ghi trước
  ERR_CIRCUIT_BREAKER_OPEN = 7;       // Cầu dao bảo vệ dịch vụ đang ngắt mạch
  ERR_DEADLINE_EXCEEDED = 8;          // Quá thời gian chờ phản hồi (Strict Timeout 2.0s)

  // --- LỖI MIỀN TỒN KHO & LÔ HÀNG (MS-01: INVENTORY - 100 - 199) ---
  ERR_INVENTORY_INSUFFICIENT_STOCK = 100;     // Hết hàng hoặc số lượng khả dụng < số lượng yêu cầu
  ERR_INVENTORY_SKU_NOT_FOUND = 101;          // Không tìm thấy mã sản phẩm trong kho
  ERR_INVENTORY_SKU_INACTIVE = 102;           // Mặt hàng kẹo đã ngừng sản xuất / ngừng bán
  ERR_INVENTORY_RESERVATION_NOT_FOUND = 103;  // Không tìm thấy phiếu khóa tồn kho
  ERR_INVENTORY_RESERVATION_EXPIRED = 104;    // Phiếu giữ chỗ đã hết hạn 15 phút
  ERR_INVENTORY_BATCH_QUARANTINED = 105;      // Lô kẹo đang bị cách ly kiểm định / cận date tiêu hủy

  // --- LỖI MIỀN DANH MỤC & GIÁ (MS-05: CATALOG - 200 - 299) ---
  ERR_CATALOG_PRICE_TAMPERED = 200;           // Phát hiện hành vi sửa giá từ phía client
  ERR_CATALOG_PRODUCT_OUT_OF_SEASON = 201;    // Kẹo mùa vụ (ví dụ: Mè xửng truyền thống Tết) chưa mở bán
  ERR_CATALOG_VARIANT_MISMATCH = 202;         // Biến thể SKU không khớp với sản phẩm cha
  ERR_CATALOG_PRODUCT_NOT_FOUND = 203;        // Không tìm thấy sản phẩm
  ERR_CATALOG_VARIANT_NOT_FOUND = 204;        // Không tìm thấy biến thể sản phẩm
  ERR_CATALOG_CATEGORY_NOT_FOUND = 205;       // Không tìm thấy danh mục
  ERR_CATALOG_PRODUCT_NOT_PUBLIC = 206;       // Sản phẩm chưa được duyệt hoặc ngừng bán
  ERR_CATALOG_SKU_DUPLICATE = 207;            // Trùng mã SKU
  ERR_CATALOG_PRODUCT_INVALID_STATE = 208;    // Sai trạng thái sản phẩm
  ERR_CATALOG_PRICE_NOT_FOUND = 209;          // Không tìm thấy giá
  ERR_CATALOG_SLUG_DUPLICATE = 210;           // Trùng slug
  ERR_CATALOG_VERSION_CONFLICT = 211;         // Xung đột phiên bản (Optimistic Locking)

  // --- LỖI MIỀN KHUYẾN MÃI & VOUCHER (MS-07: PROMOTION - 300 - 399) ---
  ERR_PROMOTION_VOUCHER_NOT_FOUND = 300;      // Mã voucher không tồn tại
  ERR_PROMOTION_VOUCHER_EXPIRED = 301;        // Mã giảm giá đã hết hạn sử dụng
  ERR_PROMOTION_USAGE_LIMIT_EXCEEDED = 302;   // Voucher đã hết lượt sử dụng trên toàn hệ thống / trên tài khoản
  ERR_PROMOTION_MIN_ORDER_AMOUNT_NOT_MET = 303; // Giá trị đơn hàng chưa đạt mức tối thiểu áp mã
  ERR_PROMOTION_VOUCHER_ALREADY_LOCKED = 304; // Voucher đang bị tạm giữ bởi một giao dịch checkout khác

  // --- LỖI MIỀN ĐƠN HÀNG & SAGA (MS-04: ORDER - 400 - 499) ---
  ERR_ORDER_NOT_FOUND = 400;                  // Không tìm thấy đơn hàng
  ERR_ORDER_INVALID_STATE_TRANSITION = 401;   // Chuyển trạng thái đơn không hợp lệ (Ví dụ: Đã CANCELLED mà đòi PAID)
  ERR_ORDER_ALREADY_PAID = 402;               // Đơn hàng này đã được thanh toán trước đó
  ERR_ORDER_PAYMENT_TIMEOUT = 403;            // Đơn hàng bị hủy tự động do quá hạn thanh toán 15 phút
  ERR_ORDER_CANCELLED_BY_USER = 404;          // Khách hàng chủ động hủy đơn
  ERR_ORDER_SAGA_COMPENSATION_FAILED = 405;   // Đền bù phân tán gặp sự cố (Cần can thiệp vận hành)

  // --- LỖI MIỀN THANH TOÁN (MS-19 / PAYMENT MODULE - 500 - 599) ---
  ERR_PAYMENT_GATEWAY_UNAVAILABLE = 500;      // Cổng thanh toán VietQR / Napas gặp sự cố
  ERR_PAYMENT_HMAC_VERIFICATION_FAILED = 501; // Chữ ký bảo mật Webhook ngân hàng không hợp lệ
  ERR_PAYMENT_AMOUNT_MISMATCH = 502;          // Số tiền ngân hàng báo về không khớp với giá trị đơn hàng
  ERR_PAYMENT_TRANSACTION_EXPIRED = 503;      // Giao dịch thanh toán hết hiệu lực

  // --- LỖI MIỀN VẬN CHUYỂN (MS-12: SHIPPING - 600 - 699) ---
  ERR_SHIPPING_SERVICE_UNAVAILABLE = 600;     // Đối tác giao vận 3PL (GHN/ViettelPost) không phản hồi
  ERR_SHIPPING_ADDRESS_UNSUPPORTED = 601;     // Địa chỉ nằm ngoài phạm vi giao hàng
  ERR_SHIPPING_WEIGHT_EXCEEDED = 602;         // Kiện kẹo vượt quá trọng lượng tối đa của một bưu kiện

  // --- LỖI MIỀN FULFILLMENT & TEM SEAL (MS-02: FULFILLMENT - 700 - 799) ---
  ERR_FULFILLMENT_VIDEO_NOT_FOUND = 700;      // Chưa có video đóng gói niêm phong trên AWS S3
  ERR_FULFILLMENT_SEAL_CODE_INVALID = 701;    // Mã tem seal O Mạ trên kiện hàng không khớp hệ thống
}

// Cấu trúc chi tiết mô tả lỗi trả về kèm theo gRPC status
message ErrorDetail {
  // Mã lỗi nghiệp vụ định danh duy nhất
  ErrorCode error_code = 1;

  // Thông điệp thân thiện hiển thị cho người dùng (Tiếng Việt)
  string user_message = 2;

  // Thông điệp kỹ thuật phục vụ lập trình viên debug
  string dev_message = 3;

  // Miền nghiệp vụ phát sinh lỗi (Ví dụ: "inventory", "payment", "order")
  string domain = 4;

  // Metadata bổ sung (Ví dụ: "sku_code": "MX-GION-500G", "available_qty": "1")
  map<string, string> metadata = 5;

  // Đường dẫn hướng dẫn xử lý hoặc tài liệu API (Optional)
  optional string help_link = 6;
}
```

## Tính toàn vẹn bản gửi

Nguồn commit: `95f9ab117cca0cc231412dc3cbdfc878a9437386`. SHA-256 tập source: `45c6d59966d951133c99c485e355035a24c01686fdb7b3bc0e8d478d2f8e549c`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn đặc tả và nội dung nhúng dùng cùng commit; liên kết bằng chứng Git/PR lịch sử ghi SHA provider riêng trong phần đối chiếu. Không có token/password trong file.
