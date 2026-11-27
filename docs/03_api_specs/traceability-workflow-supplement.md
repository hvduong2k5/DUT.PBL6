# Phần bổ sung truy xuất ngoài plan Dương — 0.4.0-draft

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
