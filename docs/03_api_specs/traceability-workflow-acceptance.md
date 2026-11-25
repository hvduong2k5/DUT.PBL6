# Nghiệm thu và rollout Traceability workflow

Version0.2.2-draft · 25/11/2026. Mỗi ca bên dưới là yêu cầu, **chưa phải kết quả PASS runtime**. [Index](traceability-workflow-README.md) và [đặc tả service](traceability-workflow-service-spec.md) là context. Người thực hiện điền evidence theo SHA/runID.

Nghiệm thu theo đầu ra OUT và contract đích trong [phiếu handoff](traceability-workflow-peer-handoff.md); không có gate gửi ACCEPT trước code. Không bắt tạo bảng/PR mới nếu capability hiện tại đã đáp ứng. Owner tự map logic/API nội bộ ra contract đích, ghi mapping và bằng chứng trong kết quả cuối. Các ca semantics/quyền/lỗi/lịch sử luôn áp dụng; field9/14 phải đúng ở bề mặt proto đích, dù provider tái sử dụng logic nào phía sau. Không dùng khác biệt implementation để bỏ test hoặc đổi wire consumer.

## 1. Fixture dùng chung

- Kho A/B UUID khác nhau; Product P, SKU S1 tạiA và SKU S2 tạiB; hai Batch UUID có thể trùng mã hiển thị để chứng minh không join theo batch_code. Hai lô S1 tạiA dùng cho ca một dòng nhiều lô. Không seed cùng SKU ở hai warehouse: baseline inventory_items dùng SKU làm primary key. Product khác cho negative binding; thêm lô depleted còn hạn, expired và restricted.
- Nhân viên makerA/read+create+edit+submit tạiA; reviewerA/read+approve+publish tạiA; operatorB cùng quyền tạo ởB; readerA thiếu evidence; GLOBAL auditor; actor token khác quyền hiện hành để kiểm thu hồi. Maker làm cả nhập kho còn có INVENTORY_VIEW, WAREHOUSE_WORKBENCH_VIEW và INVENTORY_RECEIPT_CREATE đúng kho theo nghiệp vụ Inventory hiện có; không mặc định grant Traceability thay quyền nhập. Không dùng chung maker/reviewer identity.
- Hồ sơ chưa public, eligible public, public có draft mới, suspended, archived, cert expired/revoked/review_due. Đơn một dòng hai lô3+2, hai SKU/dòng với batch riêng, duplicate-SKU manifest làm negative fixture, proof old không packed_lines, foreign order và guest session hết hạn.
- Đồng hồ kiểm thử cố định; test thời gian certificate valid_until ở23:59:59 và00:00 ngày kế tiếp theo Việt Nam, review_due_at đúng instant. Không dùng dữ liệu công khai thật để giả chứng nhận đã cấp.

## 2. Traceability phải đóng trước bàn giao tích hợp

| ID | Thực hiện | Expected |
| --- | --- | --- |
| T-SCOPE-01 | makerA create/read/edit profileA rồi thửB | A thành công, B403 không nội dung; create không tin warehouse browser |
| T-SCOPE-02 | Revoke scope sau command rồi replay cùng key; đổi scope khi dùng cursor | Replay403; cursor authorization đổi bị từ chối; không rò cached response |
| T-SCOPE-03 | Token có quyềnA nhưng `/me` chỉB; Identity timeout; forged headers | Không cấp quyềnA, timeout503, header không đổi actor/scope |
| T-SCOPE-04 | Provider thiếu warehouse hoặc đổi kho cùng UUID; legacy null binding | Không scoped create/submit/publish; mismatch409; null chỉ GLOBAL, không gán default |
| T-SCOPE-05 | List tậpA với nhiều rowB xen kẽ, page_size nhỏ | Filter tại DB trước page, không rò ID/count B, empty200 total0 |
| T-DATA-01 | Hai create đồng thời cùng namespace/batch, key khác | Chỉ một profile/QR, một201 và một409; lookup dẫn cùng profile |
| T-DATA-02 | Cùng key/body replay; khác body; stale version | Replay giữ resource/result, key mismatch409, stale409 không ghi đè |
| T-DATA-03 | Backfill warehouse legacy; resubmit snapshot cũ | Không sửa published JSON/hash, audit backfill; approved cũ không publish thiếu source kho |
| T-DATA-04 | Reader không thấy reference; edit content với reference omitted/empty cùng evidenceID | Không tiết lộ hoặc xóa reference; preserve trước validation. New evidence không reference bị400 |
| T-WF-01 | DRAFT thiếu required content/cert rồi submit | Draft có thể lưu, submit400; tồn/receipt không bị sửa |
| T-WF-02 | Maker có approve nhưng là contributor; clone revision | Tự duyệt403, contributor không mất khi clone; reviewer độc lập duyệt được |
| T-WF-03 | Source version/expiry/restriction/kho đổi giữa submit và publish | Publish409; không public pointer/outbox mới; cần revision/review mới |
| T-WF-04 | Ghi audit/outbox lỗi tại publish | Rollback toàn local command; retry cùng key không tạo hai public events |
| T-QR-01 | Public200 có draft mới; export QR rồi decode SVG/PNG | Tem dùng public revision, link đúng qr_url, mã không đổi khi reprint |
| T-QR-02 | Public404/410/503, Identity403 khi in, ảnh offline đã tải rồi suspend | Không export mới; ảnh offline không thể thu hồi nhưng link dẫn410, không profile cũ hợp lệ |
| T-QR-03 | Camera/print thực, footer `/trace`, mobile input mã | Đọc đúng mã/lô, quiet zone không cắt; không login public, không sample hardcoded |
| T-RES-01 | Input duplicates, SKU mismatch, draft/missing/published/gone/corrupt | Giữ1:1 thứ tự; NOT_AVAILABLE không lộ draft; gone link cũ; corruptUNAVAILABLE |
| T-RES-02 | 101items, body quá lớn, service credential sai, DB down | Invalid input theo contract;401/403;DB503, không giả NOT_AVAILABLE; không query peer |

## 3. Peer acceptance

| ID | Provider / consumer | Expected |
| --- | --- | --- |
| INV-01 | BatchSource exact UUID+SKU, depleted/expired/restricted | Giữ flags/version và trả kho authoritative đúng; saiSKU không lấy lô gần đúng |
| INV-02 | Receipt NEW/EXISTING và replay | batchId/code/SKU/warehouse đúng, transaction+idempotency không nhân đôi tồn |
| INV-03 | Legacy receipt thiếu binding, DB/network failure | Không đoán UUID; không tạo receipt mới để lấy binding |
| INV-04 | Contract Inventory đích và mapping | Giữ field/method cũ, cung cấp warehouse9 ở proto đích; API nội bộ được tái sử dụng phải map kho/version chính xác. Cùng source version không đổi nội dung âm thầm |
| IAM-01 | Grant/revoke/expire WAREHOUSE, active/inactive | JWT/me giữ scopeId; quyền mới/thu hồi theo chính sách thật, không role bypass |
| FUL-01 | One line two batches3+2, hai SKU/dòng riêng; duplicate-SKU invalid | packed_lines đúng pair/quantity; không ép provider hỗ trợ duplicate SKU ngoài current manifest; invalid proof không gắn nhầm lô |
| FUL-02 | Query exact sealed version sau khi live allocation/mã hiển thị đổi | Proof lịch sử bất biến, không dùng latest |
| FUL-03 | Old proof thiếu allocation hoặc wrong version/order | Không fake rows; trả lỗi/capability thiếu theo target; không sửa dữ liệu cũ bằng tồn hiện tại |
| FUL-04 | Contract Fulfillment đích, old Shipping consumer | Field14/codegen giữ wire cũ; logic/API được tái sử dụng phía sau adapter có test. Cả hai không phá Shipping/lifecycle cũ |
| EVT-01 | Duplicate event/restart, notification before peer ready | Inbox dedup, retry read proof, không double quantity/QR publish/stock mutation |
| EVT-02 | Out-of-order source version/generation | Không ghi lùi projection; event đúngorder/version, không bỏ kiểm proof |
| ORD-01 | Own/foreign/missing order, invalid/expired guest, payment-only token | Ownership kiểm trước peer;401/403/404 theo contract; không mới guest capability |
| ORD-02 | Pending packing/cancelled before packing/proof exists then returned | State đúng; không xóa lô lịch sử sau return |
| ORD-03 | Mixed resolver results/DB down/peer auth config lỗi | Detail vẫn dùng được; partialUNAVAILABLE khác NOT_AVAILABLE; không block shipping |
| ORD-04 | Canonical line_id khác UI fallback, typed SKU bridge và malformed proof | Exact unique sku_code + quantity map đúng dòng, canonical ID giữ từ subresource; không parse label/index; duplicate/mismatchUNAVAILABLE |
| ORD-05 | 101+line-batch rows, legacy no proof | Chunk≤100 và reconstruct; không backfill bằng live stock |
| WEB-01 | Receipt thành công nhưng Traceability lỗi/thiếu quyền | Confirmation nhập kho giữ nguyên, retry nguồn gốc riêng |
| WEB-02 | Two staff create race; working draft vs public print | Lookup lại đúngprofile, không repeatreceipt, inpublic |
| WEB-03 | Direct scoped admin/public/customer routes qua BFF/Kong | V2 scope đúng, no-store/status đúng; resolver không public; không JWT trong URL |
| WEB-04 | Screenshots desktop/mobile/print, source states pending/error/empty | Đúng state matrix, QR download decode được; không dùng mock để ghi provider PASS |

## 4. Evidence bắt buộc

Mỗi báo cáo gồm runID, thời gian Việt Nam, commit/image của producer+consumer, mode sandbox/integration, testID và PASS/FAIL/SKIPPED, request/response đã lọc, requestId, ảnh browser/print khi áp dụng. Ghi rõ fixture/mocked peer nào và giới hạn; không log token/password/secret/evidence nhạy cảm. Không gọi “full E2E” nếu chỉ unit hoặc provider mock.

Muse được giao bài browser localhost bằng URL truy cập được, actor test, bước và expected cụ thể, folder đích theo runID trên máy này; xác nhận ảnh/file thực sự đã nhận trước báo PASS. Tool/API failure không phải UI failure; không restart stack khác hoặc sửa production để cứu ca test.

## 5. Trình tự và điều kiện release

1. Giao trực tiếp target0.2.2-draft/SHA và task handoff. Owner bắt đầu kiểm code, tái sử dụng/bổ sung và test, không gửi ACCEPT hoặc chờ duyệt phương án. Chỉ ghi blocker/amendment nếu không thể đáp ứng contract trong scope; vẫn làm phần độc lập.
2. Implement Traceability scoped domain/repository/transport/migration bằng fixture và tests T-*; v1 wire vẫn có tests bảo vệ. Chưa expose scoped traffic nếu Inventory/Identity chưa đạt.
3. Inventory+Identity capability thật → chạy scope/workflow integration. Web in tem đi qua provider thật; screenshot sandbox chỉ chứng minh layout.
4. Order consumer support packed_lines và empty-legacy fallback trước; Fulfillment provider thêmfield14 sau; broker notification schema v2 không đổi. Deploy resolver trước bật customer traceability link.
5. Local stack tối thiểu cho từng đợt; không chạy18service. PostgreSQL+Traceability+Web đủ layout/public sandbox; scope thật cần Identity/Keycloak/Catalog/Inventory+DB; full order proof thêm Order/Fulfillment/Kafka và phụ thuộc test ports có nhãn. Không báo Docker/môi trường đang chạy nếu chưa kiểm lại.
6. Review PR vào dev, trưởng nhóm duyệt rồi release RC02; xác nhận tên nhánh release thực tế khi tạo PR/deploy, không suy đoán từ tên chat. Không test trên deploy cũ rồi coi là bản sửa mới.
7. Promotion stable chỉ khi đầy đủ quyết định/quyền + schema/examples/contract tests + các ca auth/empty/error/loading áp dụng có bằng chứng. Docs lint xanh không là runtime acceptance. Rollback: tắt route/feature mới, giữ dữ liệu/QR/snapshot; không drop migration hoặc replay command bằng key mới.

## 6. Kết quả kiểm tra artifact ngày 25/11/2026

Đây là checkpoint trước review0.2.2. Kết quả mới nhất ở report review:7OpenAPI exit0,305expanded examples,6negative cases và proto compatibility compile PASS; runtime vẫn NOT RUN. Không dùng số129examples bên dưới làm kết quả của target mới.

| Kiểm tra đã chạy | Kết quả | Giới hạn |
| --- | --- | --- |
| Redocly full scoped admin v2 và named admin draft | PASS exit0 | Không có lỗi; không chứng minh provider đã implement |
| Redocly integration draft và customer draft | PASS exit0 | Integration còn1warning schema receipt delta chưa gắn operation; customer còn1warning unused component baseline |
| Redocly admin/customer stable | PASS exit0 | Giữ nguyên15/22warning baseline; không sửa generated examples ngoài scope |
| JSON Schema2020-12 sau expand local/external refs, request/response examples | PASS129examples | Kiểm shape/type/format, không thay workflow/domain tests |
| Biên dịch2candidate proto và so descriptor với canonical | PASS | 17message types và4service definitions cũ giữ field/signature; xác nhận thêm warehouse9 và packed_lines14, chưa codegen runtime |
| Links bộ tài liệu và registry | PASS30links /21entries | Kiểm file tồn tại và ID registry không trùng, không kiểm service deployment |
| git diff --check và phạm vi thay đổi | PASS | Chưa sửa runtime code, stable/v1 contract, canonical proto hoặc event schema |

Có thể chạy lại lint bằng `npx @redocly/cli lint` với sáu file: traceability-scoped-admin-draft, traceability-workflow-draft, admin-draft-planned, customer-draft-planned, admin-stable, customer-stable trong `docs/03_api_specs/` (đuôi `.openapi.yaml`). Proto candidates phải compile riêng thư mục candidates, không cùng canonical files vì chúng cố ý giữ cùng package/message names. Kiểm tương thích so descriptor theo full message/service name, giữ mọi field number/type/label/type_name và RPC signature cũ; không chỉ so tên file.

Tất cả test runtime T-*/INV-*/IAM-*/FUL-*/EVT-*/ORD-*/WEB-* ở mục2–3: **NOT RUN trong lượt tài liệu này**. Chưa có kết quả triển khai/kiểm thử peer, chưa push/PR/deploy; không yêu cầu peer ACCEPT để bắt đầu. Không dùng kết quả static để thay thế điều kiện stable.

## 7. Ca bổ sung từ review0.2.2

| ID | Ca kiểm | Expected |
| --- | --- | --- |
| CAT-01 | Exact-SKU bootstrap active/hidden/stopped và không có giá | Trả đúng Product/SKU/version, không cần public/saleable/price |
| CAT-02 | Credential sai, SKU không tồn tại, khác case, DB lỗi | Không fuzzy match;400/401/403/404/503 đúng contract, không lộ internal binding public |
| CAT-03 | Bootstrap Product rồi verifier cũ phát hiện SKU đổi binding/version | Create/submit/publish chặn conflict, không tin context cũ từ browser |
| T-CTX-01 | Batch ở kho khác gọicontext, expired/restricted draft source, peer timeout | Kiểm Inventory và scope trước trả Product;403 ngoài kho; context không đồng nghĩa publish; timeout503, receipt giữ nguyên |
| T-REPLAY-01 | Cùng key v1/v2, edit reference omit/empty rồi retry sau khi dữ liệu đổi | Namespace response không lẫn version; hash client request trước hydrate; replay không mutate; resource/current grants được kiểm lại |
| T-ROLL-01 | Old writer coexist, metadata backfill, rollback | Không bật scoped khi còn writer tạo null; guard new rows, public JSON cũ bất biến; rollback không bỏ guard âm thầm |
| IAM-02 | Cookie → trace v2 và Inventory receipt/workbench qua Identity; CUSTOMER token vào v2 | Đúng bearer server-side, ACTIVE EMPLOYEE required, CUSTOMER403, không token ở browser URL/log |
| WEB-05 | Thiếu/sai Origin trên cookie mutation; body256KiB/profile proxy; proxy response quá lớn | Origin403; không disable CSRF; header hợp lệ được forward; bounded response502 không giả thành công |
| WEB-06 | Lowercase traceability permissions, mixed GLOBAL/WAREHOUSE grants, source projection role ALL | Giữ scopeId đúng permission; không bị whitelist mất quyền hoặc suy GLOBAL từ role không liên quan |
| INV-05 | GET workbench provider thật + receipt và bootstrap end-to-end | SKU/batch/revision đúng kho; không chỉ context Mockoon; scoped trace không đòi quyền Catalog management/BATCH_VIEW để bootstrap |
| ORD-06 | 1000+rows, response vượt512KiB, chunk chưa hoàn thành budget | RootUNAVAILABLE khi quá cap, không truncate; chunk chưa resolveUNAVAILABLE, detail/lifecycle vẫn chạy |
| ORD-07 | Base proof hợp lệ nhưng packed_lines thiếu/sai; sau đó thử base proof sai thật | Chỉ extension lỗi → traceUNAVAILABLE, PACKING_SEALED/shipping baseline vẫn tiến; base proof sai vẫn bị từ chối, không nới guard cũ |
| WEB-07 | Receipt đãAPPLIED nhưng trace binding mới thiếu/sai/timeout | Giữ confirmation APPLIED, chỉ trace panel báo chưa sẵn sàng; không POST lại receipt với key mới hoặc giả rollback |

Các ca trên cùng bộ test cũ là yêu cầu runtime, chưa chạy trong lượt tài liệu. Code baseline tăng batch_version với mọi update: thêm kiểm stock mutation giữa review/publish gây409 và UI hướng dẫn đúng, không tự sửa hash để test qua.
