# PBL6 — Bàn giao Fulfillment: FUL-TRC-01

Version nghiệp vụ/contract: **0.4.0-draft** · Snapshot tài liệu: **`34f3b4c7c2fb18de73efb749d5a083f70ff8429a`**.

**Bản 0.4.0 thay phiếu 0.3.0/0.2.2:** theo Catalog/Media đã merge trong PR #175 và plan Dương; không triển khai proposal GetSkuBinding đã gỡ. Web/BFF vẫn REST. Identity current-grants và Order resolver vẫn là phần việc riêng, chưa có trong plan Dương.

File này là bản `.md` độc lập để gửi qua Zalo. Đề bài, yêu cầu đầu ra, test và contract liên quan được nhúng bên dưới; không cần đọc lịch sử chat hoặc chờ phản hồi ACCEPT để bắt đầu code. Draft không đồng nghĩa chức năng đã chạy hoặc đã deploy.

## Nguồn tài liệu và cách lấy về

- Repository: [https://github.com/hvduong2k5/DUT.PBL6](https://github.com/hvduong2k5/DUT.PBL6). Nhánh tài liệu: `codex/traceability-order-flow`.
- **Đọc theo commit đã ghim**, không tự thay bằng HEAD của dev hoặc nhánh tài liệu đang tiếp tục thay đổi. Đây là nguồn tài liệu; owner giữ nhánh làm việc riêng cho service, không phải checkout đè công việc của mình.
- Có thể đọc toàn bộ đặc tả qua các link dưới. Người truy cập repository riêng tư cần quyền hiện có của nhóm. Contract nhúng trong file này vẫn đọc được khi chưa mở được link.

| Tài liệu liên quan trong repository | Link bất biến |
| --- | --- |
| `docs/03_api_specs/traceability-workflow-peer-handoff.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-peer-handoff.md) |
| `docs/03_api_specs/traceability-workflow-acceptance.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-acceptance.md) |
| `docs/03_api_specs/traceability-workflow-supplement.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-supplement.md) |
| `docs/03_api_specs/traceability-workflow-service-spec.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-service-spec.md) |
| `docs/03_api_specs/traceability-workflow-review-2026-11-25.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-review-2026-11-25.md) |
| `docs/03_api_specs/traceability-workflow-grpc.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-grpc.md) |
| `docs/03_api_specs/traceability-workflow-candidates/packing.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-candidates/packing.proto) |

Trong repository đã clone, các lệnh đọc sau không checkout/reset hay ghi đè source đang làm:

```powershell
git fetch origin codex/traceability-order-flow
git show 34f3b4c7c2fb18de73efb749d5a083f70ff8429a:docs/03_api_specs/traceability-workflow-grpc.md
git show 34f3b4c7c2fb18de73efb749d5a083f70ff8429a:docs/03_api_specs/traceability-workflow-peer-handoff.md
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

**Provider:** [packing candidate](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-candidates/packing.proto), RPC GetPackageRevision hiện có. Request giữ package_id+sealed_task_version và xác thực Order service. Giữ nguyên fields1–13.

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

## Phần bổ sung riêng và ranh giới phối hợp

Chốt ngày 27/11/2026 theo chủ nghiệp vụ: phần chưa có trong plan Dương được triển khai thành phần bổ sung riêng, tương thích với phần Dương đang làm. Baseline đã fetch lại: `dev@ac5395c5` / PR #175. Đây là phân chia phạm vi triển khai, không thay wire contract 0.4.0 và không tuyên bố các provider bổ sung đã chạy.

## 1. Hai phạm vi công việc

| Phạm vi | Nguồn chuẩn | Trách nhiệm |
| --- | --- | --- |
| Theo plan Dương | CatalogService trong catalog.proto; MediaInternalService trong media_service.proto; pricing, Media và video workflows tương ứng | Tái sử dụng implementation đã merge. Traceability làm consumer/adapters theo canonical, giữ fingerprint và semantics fileId. Không giao lại GetSkuBinding đã gỡ |
| Bổ sung truy xuất | Guide, OpenAPI/candidates và phiếu owner Traceability 0.4.0 | Inventory warehouse/receipt binding; IAM scope/current grants/BFF; packed_lines exact sealed proof; Order customer traceability; UI nhập kho/hồ sơ/QR/đơn hàng |

Owner contract truy xuất duy trì nguồn chuẩn và phát lại snapshot. Owner từng service thực hiện phần bổ sung trong nhánh/worktree riêng của service, hoặc commit/PR riêng nếu đang có PR plan Dương. Không nhét phần bổ sung vào PR pricing/Media đang làm mà không thể review riêng. Không tự sửa worktree hoặc nhánh đang làm của Dương. Nhánh riêng lấy nền dev hiện hành; không cherry-pick toàn bộ nhánh Traceability vào service khác chỉ để lấy tài liệu.

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

## Quy định transport mới và contract gRPC

**Phạm vi bổ sung đã chốt 27/11:** [kế hoạch bổ sung và ranh giới tránh conflict](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-supplement.md) tách phần Inventory/IAM/Fulfillment/Order/Web còn thiếu khỏi plan Catalog/Media của Dương. IAM current-grants và resolver thuộc phần bổ sung, không gán là RPC đã có trong plan Dương. Giữ contract 0.4.0; owner triển khai theo phiếu và test, không cần ACCEPT trước code.

Chốt ngày 27/11/2026 theo plan của Dương và provider đã merge trong PR #175, `dev@ac5395c5`. Bản này thay phiếu 0.3.0 về Catalog và bổ sung tích hợp Media. Nghiệp vụ nhập kho độc lập hồ sơ, scope kho, duyệt độc lập và QR theo lô giữ nguyên. Không yêu cầu Dương triển khai CatalogTraceabilitySource/GetSkuBinding; target chưa phát hành đó đã được gỡ.

## Contract và ranh giới

| Kết nối | Canonical/target | Mức sẵn sàng |
| --- | --- | --- |
| Traceability → Catalog | [CatalogService.GetProductVariant/GetTraceabilityBinding](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/packages/proto/catalog/v1/catalog.proto) | Provider đã có trên dev; consumer theo đúng proto này |
| Traceability → Media | [MediaInternalService.ValidateMediaReference/BindMediaToEntity](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/packages/proto/media/v1/media_service.proto) | Provider đã có trên dev; adapter và durable binding của Traceability theo bản này |
| Traceability → Inventory | [BatchSource candidate warehouse9](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-candidates/batch_source.proto) | Giữ yêu cầu scope đã chốt |
| Traceability → Identity | [StaffAuthorization.GetCurrentStaffGrants](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/packages/proto/identity/v1/staff_authorization.proto) | Còn là yêu cầu mới cho owner IAM; PR #175 chỉ thêm Media client, chưa có RPC này |
| Order → Traceability | [BatchLinkResolver.ResolveBatchLinks](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/packages/proto/traceability/v1/batch_links.proto) | Traceability đã implement; consumer Order còn phải tích hợp |
| Order → Fulfillment | [Packing candidate field14](https://github.com/hvduong2k5/DUT.PBL6/blob/34f3b4c7c2fb18de73efb749d5a083f70ff8429a/docs/03_api_specs/traceability-workflow-candidates/packing.proto) | Giữ exact sealed proof, không dùng live allocation |
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

Nguồn commit: `34f3b4c7c2fb18de73efb749d5a083f70ff8429a`. SHA-256 tập source: `238082252df9f096bebc16f61045ecd358b476251fef8253d023b9a982cb87ba`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
