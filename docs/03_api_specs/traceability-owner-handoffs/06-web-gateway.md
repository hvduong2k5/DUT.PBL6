# PBL6 — Bàn giao Frontend / Gateway: WEB-TRC-01

Version nghiệp vụ/contract: **0.4.0-draft** · Snapshot tài liệu: **`89379118b32146acea8945dcde3d5b28a35fd693`**.

**Bản 0.4.0 thay phiếu 0.3.0/0.2.2:** theo Catalog/Media đã merge trong PR #175 và plan Dương; không triển khai proposal GetSkuBinding đã gỡ. Web/BFF vẫn REST. Identity current-grants và Order resolver vẫn là phần việc riêng, chưa có trong plan Dương.

**Checkpoint28/11:** phần bổ sung đã có trên nhánh này; đọc report12 để tránh code trùng, xem kết quả toàn luồng và giới hạn nghiệm thu. Owner bảo đảm tương thích khi ghép, không thay phần pricing/checkout trong plan Order Dương.

File này là bản `.md` độc lập để gửi qua Zalo. Đề bài, yêu cầu đầu ra, test và contract liên quan được nhúng bên dưới; không cần đọc lịch sử chat hoặc chờ phản hồi ACCEPT để bắt đầu code. Draft không đồng nghĩa chức năng đã chạy hoặc đã deploy.

## Nguồn tài liệu và cách lấy về

- Repository: [https://github.com/hvduong2k5/DUT.PBL6](https://github.com/hvduong2k5/DUT.PBL6). Nhánh tài liệu: `codex/traceability-order-flow`.
- **Đọc theo commit đã ghim**, không tự thay bằng HEAD của dev hoặc nhánh tài liệu đang tiếp tục thay đổi. Đây là nguồn tài liệu; owner giữ nhánh làm việc riêng cho service, không phải checkout đè công việc của mình.
- Có thể đọc toàn bộ đặc tả qua các link dưới. Người truy cập repository riêng tư cần quyền hiện có của nhóm. Contract nhúng trong file này vẫn đọc được khi chưa mở được link.

| Tài liệu liên quan trong repository | Link bất biến |
| --- | --- |
| `docs/03_api_specs/traceability-workflow-peer-handoff.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-peer-handoff.md) |
| `docs/03_api_specs/traceability-workflow-acceptance.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-acceptance.md) |
| `docs/03_api_specs/traceability-workflow-supplement.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-supplement.md) |
| `docs/03_api_specs/traceability-workflow-service-spec.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-service-spec.md) |
| `docs/03_api_specs/traceability-workflow-review-2026-11-25.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-review-2026-11-25.md) |
| `docs/03_api_specs/traceability-workflow-grpc.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-grpc.md) |
| `docs/04_testing/traceability-service/12_full_workflow_2026-11-28.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/04_testing/traceability-service/12_full_workflow_2026-11-28.md) |
| `docs/03_api_specs/customer-stable.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/customer-stable.openapi.yaml) |
| `docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-service.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-service.openapi.yaml) |
| `docs/03_api_specs/traceability-web-bff-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-web-bff-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml) |
| `services/api-gateway/config/kong.yml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/services/api-gateway/config/kong.yml) |

Trong repository đã clone, các lệnh đọc sau không checkout/reset hay ghi đè source đang làm:

```powershell
git fetch origin codex/traceability-order-flow
git show 89379118b32146acea8945dcde3d5b28a35fd693:docs/03_api_specs/traceability-workflow-grpc.md
git show 89379118b32146acea8945dcde3d5b28a35fd693:docs/03_api_specs/traceability-workflow-peer-handoff.md
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

## Phần việc của Frontend / Gateway

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

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| T-QR-01 | Public200 có draft mới; export QR rồi decode SVG/PNG | Tem dùng public revision, link đúng qr_url, mã không đổi khi reprint |
| T-QR-02 | Public404/410/503, Identity403 khi in, ảnh offline đã tải rồi suspend | Không export mới; ảnh offline không thể thu hồi nhưng link dẫn410, không profile cũ hợp lệ |
| T-QR-03 | Camera/print thực, footer `/trace`, mobile input mã | Đọc đúng mã/lô, quiet zone không cắt; không login public, không sample hardcoded |
| ORD-04 | Canonical line_id khác UI fallback, typed SKU bridge và malformed proof | Exact unique sku_code + quantity map đúng dòng, canonical ID giữ từ subresource; không parse label/index; duplicate/mismatchUNAVAILABLE |
| WEB-01 | Receipt thành công nhưng Traceability lỗi/thiếu quyền | Confirmation nhập kho giữ nguyên, retry nguồn gốc riêng |
| WEB-02 | Two staff create race; working draft vs public print | Lookup lại đúngprofile, không repeatreceipt, inpublic |
| WEB-03 | Direct scoped admin/public/customer routes qua BFF/Kong | V2 scope đúng, no-store/status đúng; resolver không public; không JWT trong URL |
| WEB-04 | Screenshots desktop/mobile/print, source states pending/error/empty | Đúng state matrix, QR download decode được; không dùng mock để ghi provider PASS |
| T-CTX-01 | Batch ở kho khác gọicontext, expired/restricted draft source, peer timeout | Kiểm Inventory và scope trước trả Product;403 ngoài kho; context không đồng nghĩa publish; timeout503, receipt giữ nguyên |
| WEB-05 | Thiếu/sai Origin trên cookie mutation; body256KiB/profile proxy; proxy response quá lớn | Origin403; không disable CSRF; header hợp lệ được forward; bounded response502 không giả thành công |
| WEB-06 | Lowercase traceability permissions, mixed GLOBAL/WAREHOUSE grants, source projection role ALL | Giữ scopeId đúng permission; không bị whitelist mất quyền hoặc suy GLOBAL từ role không liên quan |
| WEB-07 | Receipt đãAPPLIED nhưng trace binding mới thiếu/sai/timeout | Giữ confirmation APPLIED, chỉ trace panel báo chưa sẵn sàng; không POST lại receipt với key mới hoặc giả rollback |

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

Phần bổ sung Inventory/IAM/Fulfillment/Order/Web/Gateway đã được triển khai trên `codex/traceability-order-flow`; [report12 toàn luồng](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/04_testing/traceability-service/12_full_workflow_2026-11-28.md) là nguồn trạng thái hiện hành và [Web BFF contract](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-web-bff-draft.openapi.yaml) giữ contract0.4.0-draft. Lượt local đã kiểm receipt/replay → hồ sơ/tệp chứng nhận Media → người khác duyệt/publish → QR; đơn Order thật với hai lô FEFO từ exact sealed Fulfillment proof, public410, resolver outage và IAM revoke/expiry. Browser kiểm desktop/mobile, giải mã QR và render bản in; thiết bị in/camera vật lý vẫn NOT RUN. Report11 giữ checkpoint trước đó. Dương tiếp tục phần pricing/checkout theo plan riêng; không làm lại phần bổ sung đã có trên nhánh này, chỉ ghép và chạy regression khi review. CI theo HEAD kiểm tại PR177; chưa deploy production hoặc tự promote stable.

Identity cookie alias là `/api/v1/admin/trace-api/**` để tuân namespace rule của Identity; backend Traceability scoped vẫn `/api/v2/admin/trace/**`. Không route internal RPC qua Kong. Migration IAM V27 chỉ seed INVENTORY_VIEW và WAREHOUSE_WORKBENCH_VIEW còn thiếu, không tự gán GLOBAL.

## Quy định transport mới và contract gRPC

**Phạm vi bổ sung đã chốt 27/11:** [kế hoạch bổ sung và ranh giới tránh conflict](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-supplement.md) tách phần Inventory/IAM/Fulfillment/Order/Web còn thiếu khỏi plan Catalog/Media của Dương. IAM current-grants và resolver thuộc phần bổ sung, không gán là RPC đã có trong plan Dương. Giữ contract 0.4.0; owner triển khai theo phiếu và test, không cần ACCEPT trước code.

Chốt ngày 27/11/2026 theo plan của Dương và provider đã merge trong PR #175, `dev@ac5395c5`. Bản này thay phiếu 0.3.0 về Catalog và bổ sung tích hợp Media. Nghiệp vụ nhập kho độc lập hồ sơ, scope kho, duyệt độc lập và QR theo lô giữ nguyên. Không yêu cầu Dương triển khai CatalogTraceabilitySource/GetSkuBinding; target chưa phát hành đó đã được gỡ.

## Contract và ranh giới

| Kết nối | Canonical/target | Mức sẵn sàng |
| --- | --- | --- |
| Traceability → Catalog | [CatalogService.GetProductVariant/GetTraceabilityBinding](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/packages/proto/catalog/v1/catalog.proto) | Provider đã có trên dev; consumer theo đúng proto này |
| Traceability → Media | [MediaInternalService.ValidateMediaReference/BindMediaToEntity](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/packages/proto/media/v1/media_service.proto) | Provider đã có trên dev; adapter và durable binding của Traceability theo bản này |
| Traceability → Inventory | [BatchSource candidate warehouse9](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-candidates/batch_source.proto) | Giữ yêu cầu scope đã chốt |
| Traceability → Identity | [StaffAuthorization.GetCurrentStaffGrants](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/packages/proto/identity/v1/staff_authorization.proto) | Còn là yêu cầu mới cho owner IAM; PR #175 chỉ thêm Media client, chưa có RPC này |
| Order → Traceability | [BatchLinkResolver.ResolveBatchLinks](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/packages/proto/traceability/v1/batch_links.proto) | Traceability đã implement; consumer Order còn phải tích hợp |
| Order → Fulfillment | [Packing candidate field14](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-candidates/packing.proto) | Giữ exact sealed proof, không dùng live allocation |
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

Phần bổ sung Inventory/IAM/Fulfillment/Order/Web/Gateway đã được triển khai trên `codex/traceability-order-flow`; xem [kết quả local và phạm vi chưa nghiệm thu](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/04_testing/traceability-service/11_local_workflow_2026-11-28.md) và [Web BFF contract](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-web-bff-draft.openapi.yaml). Luồng nhập kho → draft → Media scan → người khác duyệt/publish → QR đã chạy với provider thật. Order read được kiểm PostgreSQL + gRPC peers fixture; không suy ra toàn bộ checkout/shipping/Kafka đã nghiệm thu. Dương tiếp tục plan Order của bạn ấy; không làm lại các phần bổ sung đã có trên nhánh này, chỉ ghép và bảo đảm đầu ra/contract khi review. Target vẫn0.4.0-draft, chưa deploy production.

Identity cookie alias là `/api/v1/admin/trace-api/**` để tuân namespace rule của Identity; backend Traceability scoped vẫn `/api/v2/admin/trace/**`. Không route internal RPC qua Kong. Migration IAM V27 chỉ seed INVENTORY_VIEW và WAREHOUSE_WORKBENCH_VIEW còn thiếu, không tự gán GLOBAL.

## Contract HTTP nhúng — không cần tải thêm file để đọc schema

Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.

```yaml
openapi: 3.1.0
info:
  title: PBL6 Frontend / Gateway traceability handoff
  version: 0.4.0-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 89379118b32146acea8945dcde3d5b28a35fd693. Direct
    implementation target, not runtime acceptance. All references are local to this embedded contract.
tags:
- name: Frontend / Gateway
  description: Owner-specific provider and dependency contracts
servers:
- url: http://127.0.0.1
paths:
  /api/v2/admin/trace/records:
    get:
      operationId: listTraceabilityProfilesScopedV2
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
      tags:
      - Frontend / Gateway
      summary: listTraceabilityProfilesScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
    post:
      operationId: createTraceabilityProfileScopedV2
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
      tags:
      - Frontend / Gateway
      summary: createTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.create
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    get:
      operationId: getTraceabilityProfileScopedV2
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
      tags:
      - Frontend / Gateway
      summary: getTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/revisions:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: createTraceabilityRevisionScopedV2
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
      tags:
      - Frontend / Gateway
      summary: createTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.edit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/revisions/{revision}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    patch:
      operationId: replaceDraftContentScopedV2
      description: Replaces the whole Draft content snapshot; not a JSON merge patch. Does not change binding or
        accept contributor identities from the caller. All operations require read_internal plus their command permission
        for the same authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill
        and contributor policy.
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
      tags:
      - Frontend / Gateway
      summary: replaceDraftContentScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.edit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/revisions/{revision}/submit:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: submitTraceabilityRevisionScopedV2
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
      tags:
      - Frontend / Gateway
      summary: submitTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.submit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/revisions/{revision}/approve:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: approveTraceabilityRevisionScopedV2
      description: Reviewer must not be a contributor to this revision, including inherited contributors. Never
        publishes automatically. All operations require read_internal plus their command permission for the same
        authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor
        policy.
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
      tags:
      - Frontend / Gateway
      summary: approveTraceabilityRevisionScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.approve
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-contributor-policy: actor must not be a contributor of the revision
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/revisions/{revision}/reject:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: rejectTraceabilityRevisionScopedV2
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
      tags:
      - Frontend / Gateway
      summary: rejectTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.approve
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-contributor-policy: actor must not be a contributor of the revision
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/revisions/{revision}/publish:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: publishTraceabilityRevisionScopedV2
      description: Rechecks dependency snapshot and certificate validity; no published result if source changed
        or contract is unavailable. All operations require read_internal plus their command permission for the same
        authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor
        policy.
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
      tags:
      - Frontend / Gateway
      summary: publishTraceabilityRevisionScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.publish
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/suspend:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: suspendTraceabilityProfileScopedV2
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
      tags:
      - Frontend / Gateway
      summary: suspendTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.suspend
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/{id}/archive:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: archiveTraceabilityProfileScopedV2
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
      tags:
      - Frontend / Gateway
      summary: archiveTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.archive
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/records/by-batch:
    get:
      tags:
      - Frontend / Gateway
      operationId: findScopedTraceabilityByBatchDraft
      summary: Find the existing profile before creating a draft
      description: Requires traceability.read_internal for the authoritative warehouse from Inventory. Validate
        scope even when no profile exists. Warehouse is never trusted from browser input. Existing profile responses
        follow evidence redaction rules. Return at most one exact-binding match; no fuzzy SKU lookup. This literal
        route must take precedence over /records/{id}.
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
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      security:
      - Scoped_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v2/admin/trace/batches/{batch_id}/context:
    get:
      tags:
      - Frontend / Gateway
      operationId: getScopedBatchTraceabilityContext
      summary: Bootstrap a traceability form from an authoritative receipt batch
      description: Traceability-owned. Requires read_internal for the resource warehouse. Read Inventory first,
        check scope, then resolve Product by exact SKU from Catalog within a shared 2-second budget. No client-supplied
        product/warehouse identity. Does not create a draft or change stock. Restricted/expired source may be displayed
        for draft preparation, but existing submit/publish guards still apply. No dependency on unimplemented Inventory
        SKU detail endpoints or Catalog prices.
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
      security:
      - Workflow_staffBearer: []
      servers:
      - url: http://127.0.0.1:8003
  /api/v1/orders/{order_id}/traceability:
    get:
      servers:
      - url: http://127.0.0.1:8004
      tags:
      - Frontend / Gateway
      operationId: getCustomerOrderTraceabilityDraft
      summary: Read traceability for the sealed batches of an owned order
      description: Owned by Order. Apply the same ownership and existing guest capability checks as order detail
        before calling peers. No new guest credential type. A known identity without order ownership returns 403
        under repository rules; a nonexistent order returns 404. Derive all bindings from validated sealed proof
        and frozen Order lines; never accept batch IDs from this request. Mixed resolver failures do not remove
        order lines and do not affect the existing order detail endpoint. state PACKING_PENDING or NOT_APPLICABLE
        has empty items and total 0; READY means sealed batch bindings exist, not that every link is available.
        Internal schema/capability mismatch yields UNAVAILABLE rather than guessed data. Current Order/Inventory
        manifest supports one line per SKU. Every output row includes exact sku_code for UI bridging. More than
        1000 rows or an unrepresentable response returns root UNAVAILABLE with empty items/total 0; never truncate
        valid-looking data. Resolver chunks not completed within the outer 5-second budget stay UNAVAILABLE.
      parameters:
      - name: order_id
        in: path
        required: true
        schema:
          type: string
          format: uuid
      responses:
        '200':
          description: Sealed batch rows, or an explicit pending/unavailable state
          headers:
            Cache-Control:
              schema:
                type: string
                enum:
                - no-store
          content:
            application/json:
              schema:
                oneOf:
                - type: object
                  required:
                  - order_id
                  - state
                  - items
                  - total
                  properties:
                    order_id:
                      type: string
                      format: uuid
                    state:
                      type: string
                      enum:
                      - READY
                    items:
                      type: array
                      items:
                        $ref: '#/components/schemas/Workflow_OrderBatchTraceResult'
                      maxItems: 1000
                      minItems: 1
                    total:
                      type: integer
                      minimum: 1
                      description: Number of line-batch rows, not number of products
                - type: object
                  required:
                  - order_id
                  - state
                  - items
                  - total
                  properties:
                    order_id:
                      type: string
                      format: uuid
                    state:
                      type: string
                      enum:
                      - PACKING_PENDING
                      - NOT_APPLICABLE
                      - UNAVAILABLE
                    items:
                      type: array
                      items:
                        $ref: '#/components/schemas/Workflow_OrderBatchTraceResult'
                      maxItems: 0
                    total:
                      type: integer
                      const: 0
              examples:
                pending:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: PACKING_PENDING
                    items: []
                    total: 0
                unavailable:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: UNAVAILABLE
                    items: []
                    total: 0
                cancelledBeforePacking:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: NOT_APPLICABLE
                    items: []
                    total: 0
                oneLineTwoBatches:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: READY
                    items:
                    - order_line_id: 77777777-7777-4777-8777-777777777777
                      batch_id: 33333333-3333-4333-8333-333333333333
                      batch_code: LOT-A
                      packed_quantity: 3
                      status: AVAILABLE
                      qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                      sku_code: MX-001
                    - order_line_id: 77777777-7777-4777-8777-777777777777
                      batch_id: 44444444-4444-4444-8444-444444444444
                      batch_code: LOT-B
                      packed_quantity: 2
                      status: NOT_AVAILABLE
                      qr_url: null
                      sku_code: MX-001
                    total: 2
        '401':
          $ref: '#/components/responses/Workflow_Unauthorized'
        '403':
          $ref: '#/components/responses/Workflow_Forbidden'
        '404':
          $ref: '#/components/responses/Workflow_NotFound'
        '503':
          $ref: '#/components/responses/Workflow_Unavailable'
      security:
      - Workflow_customerBearer: []
      - Workflow_guestCapability: []
  /api/v2/trace/{qr_code}:
    get:
      operationId: getPublishedTraceabilityGatewayV2
      description: Public Kong route. The upstream Traceability service handles the same request at its internal
        v1 path. The local server example is Kong's configured proxy port. Configure the gateway origin per environment;
        do not call Traceability port 8003 with this v2 path.
      parameters:
      - name: qr_code
        in: path
        required: true
        schema:
          type: string
          minLength: 22
          maxLength: 160
          pattern: ^[A-Za-z0-9_-]+$
      responses:
        '200':
          description: Eligible published revision without internal evidence or actor data.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Legacy_PublicRecord'
        '404':
          $ref: '#/components/responses/Legacy_NotFound'
        '410':
          description: Previously published record is no longer available.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Legacy_Error'
        '503':
          $ref: '#/components/responses/Legacy_Unavailable'
      summary: getPublishedTraceabilityGatewayV2
      security: []
      servers:
      - url: http://127.0.0.1:8000
      tags:
      - Frontend / Gateway
  /api/admin/trace/records:
    get:
      operationId: listTraceabilityProfilesScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: listTraceabilityProfilesScopedV2
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
    post:
      operationId: createTraceabilityProfileScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: createTraceabilityProfileScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
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
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    get:
      operationId: getTraceabilityProfileScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: getTraceabilityProfileScopedV2
      parameters: []
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '400':
          $ref: '#/components/responses/Scoped_BadRequest'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/revisions:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: createTraceabilityRevisionScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: createTraceabilityRevisionScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
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
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/revisions/{revision}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    patch:
      operationId: replaceDraftContentScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: replaceDraftContentScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
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
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/revisions/{revision}/submit:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: submitTraceabilityRevisionScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: submitTraceabilityRevisionScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Scoped_Command'
            example:
              expected_version: 2
              reason: Independent review or authorized change
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/revisions/{revision}/approve:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: approveTraceabilityRevisionScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: approveTraceabilityRevisionScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
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
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/revisions/{revision}/reject:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: rejectTraceabilityRevisionScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: rejectTraceabilityRevisionScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
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
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/revisions/{revision}/publish:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: publishTraceabilityRevisionScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: publishTraceabilityRevisionScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Scoped_Command'
            example:
              expected_version: 4
              reason: Independent review or authorized change
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/suspend:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: suspendTraceabilityProfileScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: suspendTraceabilityProfileScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
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
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/{id}/archive:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: archiveTraceabilityProfileScopedV2IdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: archiveTraceabilityProfileScopedV2
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
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
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/records/by-batch:
    get:
      operationId: findScopedTraceabilityByBatchDraftIdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: Find the existing profile before creating a draft
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '409':
          $ref: '#/components/responses/Workflow_Conflict'
        '400':
          $ref: '#/components/responses/Workflow_BadRequest'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace/batches/{batch_id}/context:
    get:
      operationId: getScopedBatchTraceabilityContextIdentityBffWeb
      tags:
      - Frontend / Gateway
      summary: Bootstrap a traceability form from an authoritative receipt batch
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
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace-media/upload-sessions:
    post:
      operationId: createTraceCertificateUploadWeb
      tags:
      - Frontend / Gateway
      summary: Create an employee-owned OCOP scan upload session
      description: Requires current EMPLOYEE read_internal and create or edit grant. Private Media receives only
        a derived X-User-Id. Web validates JSON at 16KiB and scan size at 20MiB. Reusing a key preserves client
        intent; legacy Media does not guarantee deduplication of session creation after a lost response. Unreferenced
        orphan sessions do not create duplicate Traceability evidence.
      parameters:
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/traceability_web_bff_draft_openapi_UploadInput'
      responses:
        '201':
          description: Successful response
          headers:
            Cache-Control:
              schema:
                type: string
                const: no-store
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_UploadSession'
        '400':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '401':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '403':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '404':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '503':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace-media/upload-sessions/{sessionId}/complete:
    post:
      operationId: completeTraceCertificateUploadWeb
      tags:
      - Frontend / Gateway
      summary: Complete only the upload session issued to this employee
      parameters:
      - name: sessionId
        in: path
        required: true
        schema:
          type: string
          pattern: ^[A-Za-z0-9_-]{1,128}$
      - name: Origin
        in: header
        required: true
        schema:
          type: string
          format: uri
      - name: Idempotency-Key
        in: header
        required: true
        schema:
          type: string
          minLength: 16
          maxLength: 128
          pattern: ^[!-~]+$
      - name: X-Upload-Permit
        in: header
        required: true
        schema:
          type: string
          maxLength: 2048
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: Successful response
          headers:
            Cache-Control:
              schema:
                type: string
                const: no-store
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_FileMetadata'
        '400':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '401':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '403':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '404':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '503':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace-media/files/{fileId}:
    get:
      operationId: readTraceCertificateUploadWeb
      tags:
      - Frontend / Gateway
      summary: Poll metadata of the current employee upload capability
      parameters:
      - name: fileId
        in: path
        required: true
        schema:
          type: string
          format: uuid
      - name: X-Upload-Permit
        in: header
        required: true
        schema:
          type: string
          maxLength: 2048
      responses:
        '200':
          description: Successful response
          headers:
            Cache-Control:
              schema:
                type: string
                const: no-store
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_FileMetadata'
        '400':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '401':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '403':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '404':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '413':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '503':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/admin/trace-files/{profileId}/{evidenceId}:
    get:
      operationId: openScopedTraceEvidenceWeb
      tags:
      - Frontend / Gateway
      summary: Open the scan after current Traceability evidence and warehouse authorization
      description: Read profile through the scoped backend first. A redacted/missing reference cannot be converted
        into a Media link. Only document:fileId with purpose OCOP_CERTIFICATE is accepted. The 307 storage destination
        is pinned to the public Media origin.
      parameters:
      - name: profileId
        in: path
        required: true
        schema:
          type: string
          format: uuid
      - name: evidenceId
        in: path
        required: true
        schema:
          type: string
          format: uuid
      - name: revision
        in: query
        required: true
        schema:
          type: integer
          minimum: 1
      responses:
        '307':
          description: No-store redirect to the permitted certificate scan
          headers:
            Location:
              schema:
                type: string
                format: uri
            Cache-Control:
              schema:
                type: string
                const: no-store
        '401':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '403':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '404':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '503':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/orders/{order_id}/traceability:
    get:
      operationId: readCustomerOrderTraceabilityWeb
      tags:
      - Frontend / Gateway
      summary: Read exact sealed batch links for an owned order
      description: Uses existing Order history capability and guest session policy. The response is limited to 512KiB
        and validates order echo, row uniqueness, status/URL presence and configured public origin before relaying.
        The UI joins only an explicit unique SKU field and verifies full frozen quantities; never parse a display
        label or array index.
      parameters:
      - name: order_id
        in: path
        required: true
        schema:
          type: string
          format: uuid
      responses:
        '200':
          description: Sealed batch rows, or an explicit pending/unavailable state
          headers:
            Cache-Control:
              schema:
                type: string
                enum:
                - no-store
          content:
            application/json:
              schema:
                oneOf:
                - type: object
                  required:
                  - order_id
                  - state
                  - items
                  - total
                  properties:
                    order_id:
                      type: string
                      format: uuid
                    state:
                      type: string
                      enum:
                      - READY
                    items:
                      type: array
                      items:
                        $ref: '#/components/schemas/Workflow_OrderBatchTraceResult'
                      maxItems: 1000
                      minItems: 1
                    total:
                      type: integer
                      minimum: 1
                      description: Number of line-batch rows, not number of products
                - type: object
                  required:
                  - order_id
                  - state
                  - items
                  - total
                  properties:
                    order_id:
                      type: string
                      format: uuid
                    state:
                      type: string
                      enum:
                      - PACKING_PENDING
                      - NOT_APPLICABLE
                      - UNAVAILABLE
                    items:
                      type: array
                      items:
                        $ref: '#/components/schemas/Workflow_OrderBatchTraceResult'
                      maxItems: 0
                    total:
                      type: integer
                      const: 0
              examples:
                pending:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: PACKING_PENDING
                    items: []
                    total: 0
                unavailable:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: UNAVAILABLE
                    items: []
                    total: 0
                cancelledBeforePacking:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: NOT_APPLICABLE
                    items: []
                    total: 0
                oneLineTwoBatches:
                  value:
                    order_id: 22222222-2222-4222-8222-222222222222
                    state: READY
                    items:
                    - order_line_id: 77777777-7777-4777-8777-777777777777
                      batch_id: 33333333-3333-4333-8333-333333333333
                      batch_code: LOT-A
                      packed_quantity: 3
                      status: AVAILABLE
                      qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                      sku_code: MX-001
                    - order_line_id: 77777777-7777-4777-8777-777777777777
                      batch_id: 44444444-4444-4444-8444-444444444444
                      batch_code: LOT-B
                      packed_quantity: 2
                      status: NOT_AVAILABLE
                      qr_url: null
                      sku_code: MX-001
                    total: 2
        '400':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '401':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '403':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '404':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '502':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '503':
          description: Safe error envelope. No saved receipt, draft or Order is rolled back.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/auth/me:
    get:
      operationId: readBrowserIdentityWithScopedTraceGrants
      tags:
      - Frontend / Gateway
      summary: Preserve scoped Identity grants in the existing browser session projection
      description: Additive permissionGrants field on principal; authentication uses the existing Identity session
        or OIDC cookies. ACTIVE employee/customer profile still comes from Identity. No tokens or service credentials
        are exposed. Malformed permission tuples fail closed.
      responses:
        '200':
          description: Authenticated principal projection, no-store
          content:
            application/json:
              schema:
                type: object
                required:
                - authenticated
                - principal
                - customerAccess
                - customer
                properties:
                  authenticated:
                    type: boolean
                    const: true
                  principal:
                    type: object
                    required:
                    - principalId
                    - principalType
                    - displayName
                    - roles
                    - permissions
                    properties:
                      principalId:
                        type: string
                        format: uuid
                      principalType:
                        type: string
                        enum:
                        - EMPLOYEE
                        - CUSTOMER
                      displayName:
                        type: string
                      email:
                        type: string
                      customerId:
                        type: string
                      roles:
                        type: array
                        items:
                          type: object
                          required:
                          - roleCode
                          properties:
                            roleCode:
                              type: string
                            roleName:
                              type: string
                            scopeType:
                              type: string
                            scopeId:
                              type:
                              - string
                              - 'null'
                      permissions:
                        type: array
                        items:
                          type: string
                      permissionGrants:
                        type: array
                        description: Optional legacy-compatible projection of current Identity permission/scope
                          tuples. A missing list never grants Traceability access. An unrelated GLOBAL role cannot
                          widen a WAREHOUSE Traceability grant.
                        items:
                          type: object
                          required:
                          - permissionCode
                          - scopeType
                          - scopeId
                          properties:
                            permissionCode:
                              type: string
                            scopeType:
                              type: string
                            scopeId:
                              type:
                              - string
                              - 'null'
                  customerAccess:
                    type: boolean
                  customer:
                    type:
                    - object
                    - 'null'
                  sessionExpiresAt:
                    type: string
                    format: date-time
        '401':
          description: Authentication absent, forbidden or Identity unavailable
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '403':
          description: Authentication absent, forbidden or Identity unavailable
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
        '503':
          description: Authentication absent, forbidden or Identity unavailable
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/traceability_web_bff_draft_openapi_Error'
      security:
      - traceability_web_bff_draft_openapi_identityCookie: []
      - traceability_web_bff_draft_openapi_oidcCookie: []
      servers:
      - url: https://shop.example.test
  /api/v1/media/upload-sessions:
    post:
      tags:
      - Frontend / Gateway
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
      - Frontend / Gateway
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
    Workflow_OrderBatchTraceResult:
      oneOf:
      - type: object
        additionalProperties: false
        required:
        - order_line_id
        - batch_id
        - batch_code
        - packed_quantity
        - sku_code
        - status
        - qr_url
        properties:
          order_line_id:
            type: string
            format: uuid
          batch_id:
            type: string
            format: uuid
            description: Stable row identity together with order_line_id; batch_code alone is not unique
          batch_code:
            type: string
          packed_quantity:
            type: integer
            minimum: 1
            format: int32
            maximum: 2147483647
          sku_code:
            type: string
            minLength: 1
            maxLength: 100
            description: Authoritative unique-per-order SKU to bridge customer detail rows that lack canonical line
              IDs. Never parse skuLabel.
          status:
            type: string
            enum:
            - AVAILABLE
            - NO_LONGER_PUBLIC
          qr_url:
            type: string
            format: uri
      - type: object
        additionalProperties: false
        required:
        - order_line_id
        - batch_id
        - batch_code
        - packed_quantity
        - sku_code
        - status
        - qr_url
        properties:
          order_line_id:
            type: string
            format: uuid
          batch_id:
            type: string
            format: uuid
            description: Stable row identity together with order_line_id; batch_code alone is not unique
          batch_code:
            type: string
          packed_quantity:
            type: integer
            minimum: 1
            format: int32
            maximum: 2147483647
          sku_code:
            type: string
            minLength: 1
            maxLength: 100
            description: Authoritative unique-per-order SKU to bridge customer detail rows that lack canonical line
              IDs. Never parse skuLabel.
          status:
            type: string
            enum:
            - NOT_AVAILABLE
            - UNAVAILABLE
          qr_url:
            type: 'null'
    Legacy_PublicRecord:
      type: object
      additionalProperties: false
      required:
      - qr_code
      - product_name
      - batch_code
      - scope
      - revision
      - mode
      - verification_method
      - verified_at
      - source_verified_at
      - ocop_star
      - ocop_certificate_no
      - production_date
      - expiry_date
      - producer
      - workshop_location
      - artisan_name
      - food_safety_cert
      - raw_materials_origin
      - process
      - crafting_video_url
      properties:
        qr_code:
          type: string
        product_name:
          type: string
        batch_code:
          type: string
        scope:
          type: string
          enum:
          - BATCH
        revision:
          type: integer
        mode:
          type: string
          enum:
          - sandbox
          - integration
        verification_method:
          type: string
          enum:
          - MANUAL
        verified_at:
          type: string
          format: date-time
        source_verified_at:
          type: string
          format: date-time
          description: Frozen source snapshot check time; distinct from the manual certificate check time
        ocop_star:
          type: integer
          minimum: 1
          maximum: 5
        ocop_certificate_no:
          type: string
        production_date:
          type: string
          format: date
        expiry_date:
          type: string
          format: date
        producer:
          type: string
        workshop_location:
          type: string
        artisan_name:
          type:
          - string
          - 'null'
        food_safety_cert:
          type:
          - string
          - 'null'
        raw_materials_origin:
          type: string
        process:
          type: string
        crafting_video_url:
          type:
          - string
          - 'null'
          format: uri
    Legacy_Error:
      type: object
      additionalProperties: false
      required:
      - error_code
      - user_message
      - request_id
      properties:
        error_code:
          type: string
        user_message:
          type: string
        request_id:
          type: string
    traceability_web_bff_draft_openapi_Error:
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
    traceability_web_bff_draft_openapi_UploadInput:
      type: object
      required:
      - purpose
      - fileName
      - mediaType
      - sizeBytes
      - sha256
      properties:
        purpose:
          type: string
          const: OCOP_CERTIFICATE
        fileName:
          type: string
          minLength: 1
          maxLength: 250
        mediaType:
          type: string
          enum:
          - application/pdf
          - image/jpeg
          - image/png
          - image/webp
        sizeBytes:
          type: integer
          minimum: 1
          maximum: 20971520
        sha256:
          type: string
          pattern: ^[0-9a-f]{64}$
    traceability_web_bff_draft_openapi_UploadSession:
      type: object
      required:
      - fileId
      - uploadSessionId
      - uploadUrl
      - requiredHeaders
      - uploadPermit
      properties:
        fileId:
          type: string
          format: uuid
        uploadSessionId:
          type: string
        uploadUrl:
          type: string
          format: uri
          description: Origin pinned to configured public storage; PUT directly with requiredHeaders and omit credentials.
        requiredHeaders:
          type: object
          additionalProperties:
            type: string
        uploadPermit:
          type: string
          description: Opaque server HMAC permit; send only to this BFF in X-Upload-Permit, never in URLs.
        expiresAt:
          type: string
          format: date-time
    traceability_web_bff_draft_openapi_FileMetadata:
      type: object
      required:
      - fileId
      - status
      - sha256Checksum
      properties:
        fileId:
          type: string
          format: uuid
        status:
          type: string
          enum:
          - AWAITING_UPLOAD
          - PROCESSING
          - READY
          - PROCESSING_FAILED
          - ARCHIVED
        sha256Checksum:
          type:
          - string
          - 'null'
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
    Legacy_NotFound:
      description: Resource not found or never published; no internal details
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Legacy_Error'
    Legacy_Unavailable:
      description: Required dependency or contract unavailable; no fixture fallback in integration
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Legacy_Error'
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
  securitySchemes:
    Scoped_staffBearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
    Workflow_staffBearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
    Workflow_customerBearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
    Workflow_guestCapability:
      type: http
      scheme: bearer
      description: Existing authoritative Order detail guest credential only; payment-read grants do not authorize
        this operation. No new credential or header is introduced.
    traceability_web_bff_draft_openapi_identityCookie:
      type: apiKey
      in: cookie
      name: OMAMA_SESSION
    traceability_web_bff_draft_openapi_oidcCookie:
      type: apiKey
      in: cookie
      name: oma_access_token
    MediaStable_bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
    MediaStable_customerSession:
      type: apiKey
      in: cookie
      name: customer_session
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
    MediaStable_IdempotencyKey:
      name: Idempotency-Key
      in: header
      required: false
      description: Required for retryable create/action operations. Same key and same payload must return the same
        outcome.
      schema:
        type: string
        format: uuid
x-source-commit: 89379118b32146acea8945dcde3d5b28a35fd693
```

## Mapping cookie transport do Identity cung cấp

| BFF route | Backend route qua Kong |
| --- | --- |
| `/api/v1/admin/trace-api/...` | `/api/v2/admin/trace/...` |
| `/api/v1/admin/inventory-api/workbench` | `/api/v1/admin/inventory/workbench` |
| `/api/v1/admin/inventory-api/workbench/receipts` | `/api/v1/admin/inventory/workbench/receipts` |

Cookie mode chuyển `OMAMA_SESSION` server-side qua Identity; forward Origin đã kiểm tra ở mutation, giữ Idempotency-Key. Bearer mode lấy credential tin cậy trong server, không cho client gán user/role headers. Internal resolver và Catalog binding không route public.

## Tính toàn vẹn bản gửi

Nguồn commit: `89379118b32146acea8945dcde3d5b28a35fd693`. SHA-256 tập source: `d45b0ae72b82291d172afc0855a67ff865362580dd5cca21bd43d94aecd9d4aa`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn đặc tả và nội dung nhúng dùng cùng commit; liên kết bằng chứng Git/PR lịch sử ghi SHA provider riêng trong phần đối chiếu. Không có token/password trong file.
