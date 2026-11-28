# PBL6 — Bàn giao Order: ORD-TRC-01

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
| `docs/03_api_specs/traceability-workflow-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-candidates/packing.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-candidates/packing.proto) |
| `packages/proto/traceability/v1/batch_links.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/packages/proto/traceability/v1/batch_links.proto) |

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

## Phần việc của Order

**Mục đích:** người có quyền xem đơn đọc được danh sách nguồn gốc theo từng lô thực giao, không làm lỗi phần detail/payment/shipping đang có.

**Đầu ra bắt buộc:** ORD-OUT-01: Web lấy được từng line/batch/quantity và link/trạng thái truy xuất đúng đơn; ORD-OUT-02: chỉ chủ đơn hoặc guest capability hợp lệ được đọc; ORD-OUT-03: proof/Traceability thiếu hoặc lỗi không phá detail/lifecycle, không gắn nhầm lô. Owner đối chiếu API/read model hiện có trước; endpoint riêng bên dưới là contract đích để triển khai. Có thể dùng logic/read model hiện có phía sau route/adapter đúng contract đó; không cần consumer duyệt phương án nội bộ. Không tự thêm field vào stable detail hoặc đổi response ngoài phạm vi.

**Consumer:** Fulfillment proof candidate và gRPC `BatchLinkResolver.ResolveBatchLinks` theo [shared proto](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/packages/proto/traceability/v1/batch_links.proto), [gRPC guide](https://github.com/hvduong2k5/DUT.PBL6/blob/89379118b32146acea8945dcde3d5b28a35fd693/docs/03_api_specs/traceability-workflow-grpc.md). Không gọi internal REST resolver0.2.2 nữa. **Provider đích:** `GET /api/v1/orders/{order_id}/traceability` trong OpenAPI tích hợp, endpoint riêng; không sửa frozen commercial snapshot hoặc lifecycle chỉ vì lookup nguồn gốc lỗi.

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

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| EVT-01 | Duplicate event/restart, notification before peer ready | Inbox dedup, retry read proof, không double quantity/QR publish/stock mutation |
| EVT-02 | Out-of-order source version/generation | Không ghi lùi projection; event đúngorder/version, không bỏ kiểm proof |
| ORD-01 | Own/foreign/missing order, invalid/expired guest, payment-only token | Ownership kiểm trước peer;401/403/404 theo contract; không mới guest capability |
| ORD-02 | Pending packing/cancelled before packing/proof exists then returned | State đúng; không xóa lô lịch sử sau return |
| ORD-03 | Mixed resolver results/DB down/peer auth config lỗi | Detail vẫn dùng được; partialUNAVAILABLE khác NOT_AVAILABLE; không block shipping |
| ORD-04 | Canonical line_id khác UI fallback, typed SKU bridge và malformed proof | Exact unique sku_code + quantity map đúng dòng, canonical ID giữ từ subresource; không parse label/index; duplicate/mismatchUNAVAILABLE |
| ORD-05 | 101+line-batch rows, legacy no proof | Chunk≤100 và reconstruct; không backfill bằng live stock |
| ORD-06 | 1000+rows, response vượt512KiB, chunk chưa hoàn thành budget | RootUNAVAILABLE khi quá cap, không truncate; chunk chưa resolveUNAVAILABLE, detail/lifecycle vẫn chạy |
| ORD-07 | Base proof hợp lệ nhưng packed_lines thiếu/sai; sau đó thử base proof sai thật | Chỉ extension lỗi → traceUNAVAILABLE, PACKING_SEALED/shipping baseline vẫn tiến; base proof sai vẫn bị từ chối, không nới guard cũ |
| GRPC-RES-01 | Duplicates/exact input echo, enum và optional qr_url |1:1 thứ tự; absent→null khi Order map REST; không UNSPECIFIED hoặc proto JSON trực tiếp |
| GRPC-RES-02 |1/100/101bindings, 256KiB/512KiB caps, wrong/missing/duplicate credential | Invalid count→INVALID_ARGUMENT, byte cap→RESOURCE_EXHAUSTED, auth→UNAUTHENTICATED; chỉ Order credential, không route HTTP cũ |
| GRPC-RES-03 | Deadline, DB down, corrupt/old/unpublished/withdrawn rows |gRPC storage/budget lỗi, per-row status đúng; không block Order lifecycle và không query peer |

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
  title: PBL6 Order traceability handoff
  version: 0.4.0-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 89379118b32146acea8945dcde3d5b28a35fd693. Direct
    implementation target, not runtime acceptance. All references are local to this embedded contract.
tags:
- name: Order
  description: Owner-specific provider and dependency contracts
servers:
- url: http://127.0.0.1
paths:
  /api/v1/orders/{order_id}/traceability:
    get:
      servers:
      - url: http://127.0.0.1:8004
      tags:
      - Order
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
components:
  schemas:
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
  responses:
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
  securitySchemes:
    Workflow_customerBearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
    Workflow_guestCapability:
      type: http
      scheme: bearer
      description: Existing authoritative Order detail guest credential only; payment-read grants do not authorize
        this operation. No new credential or header is introduced.
x-source-commit: 89379118b32146acea8945dcde3d5b28a35fd693
```

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

## Proto target nhúng: `packages/proto/traceability/v1/batch_links.proto`

Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng. Không hand-edit generated code.

```proto
// Integration target 0.3.0-draft. Provider: Traceability; consumer: Order.
syntax = "proto3";
package omamx.traceability.v1;
option go_package = "github.com/omamx/proto/traceability/v1;traceabilityv1";
option java_multiple_files = true;
option java_package = "com.omamx.proto.traceability.v1";

// Internal TLS plus a dedicated Order credential. Read-only, no idempotency key.
// Budget <= 1.5s, 1..100 items, <=256KiB request, <=512KiB response.
service BatchLinkResolver {
  rpc ResolveBatchLinks(ResolveBatchLinksRequest) returns (ResolveBatchLinksResponse);
}
message ResolveBatchLinksRequest {
  repeated TraceabilityBatchBinding items = 1;
}
message TraceabilityBatchBinding {
  string source_namespace = 1;
  string batch_id = 2;
  string sku_code = 3;
}
enum BatchLinkStatus {
  BATCH_LINK_STATUS_UNSPECIFIED = 0;
  BATCH_LINK_STATUS_AVAILABLE = 1;
  BATCH_LINK_STATUS_NOT_AVAILABLE = 2;
  BATCH_LINK_STATUS_NO_LONGER_PUBLIC = 3;
  BATCH_LINK_STATUS_UNAVAILABLE = 4;
}
message ResolvedBatchLink {
  TraceabilityBatchBinding binding = 1;
  BatchLinkStatus status = 2;
  // Present only for AVAILABLE/NO_LONGER_PUBLIC. No internal IDs or evidence.
  optional string qr_url = 3;
}
message ResolveBatchLinksResponse {
  // Exactly one result per input, including duplicates, in the original order.
  repeated ResolvedBatchLink items = 1;
  int32 total = 2;
}
```

## Tính toàn vẹn bản gửi

Nguồn commit: `89379118b32146acea8945dcde3d5b28a35fd693`. SHA-256 tập source: `7d026a87b511ce0e6fcc5770227d96ef6f2a32eb78b3b5f3ff622cd00f1a1852`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn đặc tả và nội dung nhúng dùng cùng commit; liên kết bằng chứng Git/PR lịch sử ghi SHA provider riêng trong phần đối chiếu. Không có token/password trong file.
