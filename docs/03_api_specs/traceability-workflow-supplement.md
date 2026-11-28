# Phần bổ sung truy xuất ngoài plan Dương — 0.4.0-draft

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

Phần bổ sung Inventory/IAM/Fulfillment/Order/Web/Gateway đã được triển khai trên `codex/traceability-order-flow`; xem [kết quả local và phạm vi chưa nghiệm thu](../04_testing/traceability-service/11_local_workflow_2026-11-28.md) và [Web BFF contract](traceability-web-bff-draft.openapi.yaml). Luồng nhập kho → draft → Media scan → người khác duyệt/publish → QR đã chạy với provider thật. Order read được kiểm PostgreSQL + gRPC peers fixture; không suy ra toàn bộ checkout/shipping/Kafka đã nghiệm thu. Dương tiếp tục plan Order của bạn ấy; không làm lại các phần bổ sung đã có trên nhánh này, chỉ ghép và bảo đảm đầu ra/contract khi review. Target vẫn0.4.0-draft, chưa deploy production.

Identity cookie alias là `/api/v1/admin/trace-api/**` để tuân namespace rule của Identity; backend Traceability scoped vẫn `/api/v2/admin/trace/**`. Không route internal RPC qua Kong. Migration IAM V27 chỉ seed INVENTORY_VIEW và WAREHOUSE_WORKBENCH_VIEW còn thiếu, không tự gán GLOBAL.
