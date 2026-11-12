# Đăng ký phát hiện triển khai (Deployment Findings Register) — W0–W4, 11–12/11/2026

Tài liệu này tổng hợp **mọi lỗi và phát hiện** ghi nhận trong quá trình triển khai PBL6 lên Coolify VPS (các wave W0–W4) và nghiệm thu E2E (W4). Mỗi mục gồm: triệu chứng/bằng chứng quan sát trên hệ thống thật, nguyên nhân (nếu đã xác định), trạng thái và hướng sửa đề xuất.

Nguyên tắc ghi: chỉ ghi điều đã kiểm chứng trên môi trường live hoặc trong log/DB; mục nào chưa cô lập được cơ chế tới dòng code thì ghi rõ "chưa cô lập".

Mức độ: **Blocker** = chặn một luồng nghiệp vụ chính; **High** = sai hành vi nghiêm trọng hoặc rủi ro vận hành lớn; **Medium** = sai lệch so với đặc tả, có đường vòng; **Low** = khiếm khuyết nhỏ/ghi chú vận hành.

---

## A. Identity & phân quyền

### F-01 — Identity biến từ chối phân quyền thành HTTP 500 · High · MỞ
- Bằng chứng: `POST /api/v1/admin/role-assignments` (token staff có `ROLE_ASSIGN#GLOBAL`) trả **500**; log identity ghi `GlobalExceptionHandler: Unhandled server exception` → `AuthorizationDeniedException` tại `IamAssignmentController.assignRole` (`@PreAuthorize hasAuthority('ROLE_ASSIGN')`). Tái hiện 3 lần (requestId req-01ce4683…, req-f6fad991…, req-df3abb87…).
- Nguyên nhân: exception của method-security phát sinh sau filter chain và rơi vào global handler → ánh xạ thành 500 thay vì 403.
- Hướng sửa: thêm handler riêng cho `AuthorizationDeniedException` → 403 tại identity-service.

### F-02 — Grants ghi tay vào Keycloak không được tính như grants do PermissionSync ghi · Medium · MỞ
- Bằng chứng: attribute `permission_grants` của e1-staff được đặt tay 3 giá trị (`CATALOG_VIEW#GLOBAL`, `ROLE_ASSIGN#GLOBAL`, `ROLE_VIEW#GLOBAL`) — token giải mã có đủ 3 grants nhưng method-security vẫn từ chối; sau khi thay bằng bộ 35 grants đúng định dạng PermissionSync ghi, cùng lời gọi đi qua được (kiểm chứng đối chứng hai token).
- Nguyên nhân: chưa cô lập tới dòng code (jar đã deploy có claim converter xử lý `permission_grants`; khác biệt có thể ở cách Keycloak serialize attribute đa trị trị khi ghi tay vs khi PermissionSync ghi).
- Hệ quả: bootstrap quyền bằng cách sửa attribute trực tiếp là **không đáng tin**; chỉ nên coi là đường vòng khẩn cấp có ghi nhãn.

### F-03 — Không tồn tại đường chính thống để tạo admin đầu tiên · High (thiết kế) · MỞ
- Bằng chứng: khi phân quyền đã qua, identity trả 400 `INVALID_ARGUMENT` — *"Self-assignment is forbidden: actor cannot assign a role to themselves"*. Seed chỉ tạo role ADMIN (35 permissions), không tạo principal/assignment nào; hệ thống trắng (`iam_role_assignments` = 0 dòng) không có bất kỳ API path nào tạo được admin đầu tiên.
- Cách đã dùng ở W4 (đường vòng, có ghi nhãn): insert assignment trực tiếp + ghi attribute grants đúng định dạng PermissionSync.
- Hướng sửa (team quyết định): cơ chế bootstrap một lần (CLI/env-gated init, hoặc tài khoản break-glass có audit) + runbook.

### F-04 — Health endpoints của identity trả 500 · Medium · MỞ
- `/api/v1/public/auth/health` và `/actuator/health` đều trả 500 trên bản deploy (phát hiện từ W3, còn nguyên ở W4). Chưa chẩn đoán nguyên nhân.

### F-05 — Dung sai hết hạn token 9–127 giây · Low · MỞ (cần quyết định)
- Bằng chứng (A4): token hết hạn 9 giây vẫn được chấp nhận (200); hết hạn 127 giây bị từ chối (401). Có clock-skew leeway ở đâu đó trong chuỗi (Keycloak/identity/services chưa xác định từng điểm).
- Hướng: team chốt leeway chấp nhận được và ghi vào tài liệu, hoặc siết về 0.

### F-06 — Client `identity-service-admin` không gọi được API admin qua đường token của nó · Medium · MỞ
- Token client_credentials (azp `identity-service-admin`, không có `aud`, không có `permission_grants`) bị identity trả 401 ở `GET /api/v1/admin/roles` vì `JwtAudienceConfig` yêu cầu `aud=omama-api`.
- Kèm việc rà soát least-privilege còn nợ: client này đang giữ thêm role `manage-realm`.

---

## B. Gateway (Kong)

### F-07 — Thiếu route cho các endpoint quản trị role của identity · High · MỞ
- `/api/v1/admin/role-assignments` và `/api/v1/admin/roles` **không có route nào trong Kong** → rơi vào catch-all `catalog-api` → 400 lỗi transcoding gRPC. Mặt quản trị role của staff không tới được từ bên ngoài; ở W4 phải gọi thẳng `identity-svc:8016` trong mạng nội bộ.
- Hướng sửa: bổ sung route identity-admin vào `kong.yml` + cấu hình live (kèm auth như các route admin khác).

### F-08 — Kong giữ upstream cũ sau khi container bị thay → 502 tới khi restart Kong · High · MỞ
- Bằng chứng (R1): sau mỗi lần restart/redeploy identity và fulfillment, gateway trả 502 (JWKS, users/me, fulfillment tasks) với timeout ~18,5s; restart Kong thì hết ngay (~0,1s). Lặp lại 2 lần trong một buổi.
- Hệ quả vận hành: **mọi lần redeploy một service đều có cửa sổ gateway 502** cho tới khi Kong được restart/nạp lại — cần runbook tạm thời và hướng sửa bền vững (cấu hình upstream/DNS TTL của Kong DB mode).

### F-09 — Không có rate limiting (D1 FAIL) · Medium · MỞ
- Kong không có plugin rate-limiting ở bất kỳ route/service/global nào. 12 POST đăng ký sai định dạng liên tiếp → 422 ×12, không có 429 (kế hoạch nghiệm thu đặt ngưỡng 10/phút cho route đăng ký).

### F-10 — Không có CORS ở gateway (D2 FAIL) · Medium · MỞ
- Kong không có plugin CORS; preflight `OPTIONS` từ origin web thật (`https://pbl.npanhbao.dev`) bị chính upstream catalog trả 403 `Invalid CORS request`; response thường cũng không có `Access-Control-Allow-Origin`. Web hiện không bị ảnh hưởng vì gọi qua BFF cùng origin, nhưng bất kỳ client cross-origin nào cũng không dùng được API.

### F-11 — Không có route cho payment callback · Medium · MỞ
- Không có route Kong cho callback thanh toán; bản thân order-service cũng không có path `/payments/sepay/callback` (đường callback đang dùng được là `/payments/vietqr/callback` qua gọi trực tiếp). Cần chốt hợp đồng callback trước khi nối cổng thanh toán thật.

---

## C. Web (web-user, chế độ OIDC)

### F-12 — Các route thương mại của web trỏ sai upstream ở chế độ OIDC · **Blocker** · MỞ (fix PR riêng)
- Bằng chứng: các route BFF `register`, `catalog`, `cart`, `checkout`, `orders` đều dựng upstream từ `CUSTOMER_CORE_UPSTREAM_URL` (= mount `/api/v1/customer-api` của identity) thay vì upstream của catalog/order:
  - Đăng ký: web gọi `.../customer-api/public/customer-registrations` → 401 (route công khai đúng là `/api/v1/public/customer-registrations` → 200). **Khách không tự đăng ký được qua web đã deploy.**
  - Sản phẩm: `GET /api/catalog/products/{slug}` → 401 `CATALOG_UPSTREAM_ERROR`; trang chi tiết sản phẩm hiện "KẾT NỐI GIÁN ĐOẠN".
- Phạm vi: PR #88 đã sửa các handler tài khoản (capabilities/profile/…); các route thương mại trên **chưa** được sửa — toàn bộ luồng mua hàng phía web đang đứt.
- Hướng sửa: PR riêng theo mẫu PR #88 (mỗi route dùng đúng upstream service của nó + test), cherry-pick lên release, redeploy, retest.

### F-13 — Đăng xuất trên UI không xóa cookie OIDC phía server · Medium · MỞ
- Sau khi bấm Đăng xuất, UI hiển thị khách nhưng `/api/auth/me` vẫn `authenticated: true` và capabilities vẫn 200 — phiên OIDC chưa bị thu hồi phía server.

### F-14 — `/account/orders` hiển thị lỗi chung thay vì dữ liệu đơn · Low · MỞ
- Trang hiển thị "Chưa thể mở lịch sử Order / Không thể xử lý Order lúc này". Một phần nguyên nhân có thể là F-12 (orders route sai upstream); cần kiểm lại sau khi F-12 được sửa, trước khi kết luận còn lỗi riêng ở order path.

---

## D. Order / Inventory / Fulfillment / Shipping

### F-15 — Lệch hợp đồng sự kiện order → fulfillment (v2 vs v3) · **Blocker** · MỞ
- Bằng chứng: order-service (bản sandbox đang deploy) phát `vn.omama.order.fulfillment_ready.v2` lên topic `order.events.v2`; fulfillment-service consume topic `order.events.v3` (group `fulfillment-ready-v1`). Replay nguyên sự kiện v2 vào topic v3: consumer đọc được (lag 0) nhưng **không tạo task** — payload v3 yêu cầu các hash provenance (`items_digest`, `stock_resource_version`) do order core tính và không lưu ở nơi đọc được, nên không dựng lại được một sự kiện v3 trung thực từ bên ngoài.
- Kèm: fulfillment không có endpoint tạo task thủ công; task chỉ sinh từ consumer này.
- Hệ quả: **đơn hàng không bao giờ tới được fulfillment** trong cấu hình hiện tại. Cần team chốt: căn chỉnh topic/version + schema sự kiện giữa hai service (đường bền vững là order core mode phát đúng v3), rồi redeploy.

### F-16 — Checkout sandbox bỏ qua inventory thật · High · MỞ (giới hạn đã biết của sandbox)
- Bằng chứng (C1): checkout giữ tồn **chỉ trong bảng simulator nội bộ của order-service** (available 10→9); `inventory_db` không đổi (physical 10 / reserved 0). Dependency facade chạy trong container, URL catalog để trống.
- Hệ quả: số liệu tồn kho thật không phản ánh đơn sandbox; C2 (trừ tồn khi đóng gói/giao) không có gì để nghiệm thu cho tới khi order chạy core mode hoặc nối facade tới inventory thật.

### F-17 — `PROFILE_INTERNAL_TOKEN` lệch nhau giữa các service (lỗi cấu hình deploy) · High · **ĐÃ SỬA 12/11**
- Ba giá trị khác nhau đồng thời tồn tại: trong order-service, trong profile-service (độ dài 65 ký tự — thừa 1 ký tự so với chuẩn 64), và bản staged tại `/root/.pbl6-secrets/profile_token.env`. Order Resolve bị profile trả 401 → mọi lời gọi giỏ hàng/checkout trả 503 `IDENTITY_RESOLUTION_UNAVAILABLE`.
- Đã sửa: đặt cả hai app về giá trị staged, redeploy cả hai; kiểm chứng Resolve 200 + `GET /api/v1/cart` 200.
- Bài học: các secret dùng chung phải có một nguồn sự thật duy nhất khi staged; checklist deploy cần bước đối chiếu hash của secret chung giữa các app (chỉ so hash, không in giá trị).

### F-18 — Inventory không có API ghi tồn kho; thiếu network alias · Medium · MỞ
- Inventory không expose RPC/HTTP nào để nhập tồn; đường duy nhất hiện có là SQL seed (`migrations/seed_sample_data.sql`). Ngoài ra inventory không có alias DNS ổn định trên mạng coolify (phải dùng IP container khi gọi gRPC) — các service khác đều đã có alias (`catalog-svc`, `profile-svc`, …).

### F-19 — Fulfillment trả 401 thay vì 403 cho staff thiếu grant (A5(i) FAIL) · Medium · MỞ
- Staff có danh tính hợp lệ nhưng không có grant FULFILLMENT_* gọi `GET /api/v1/admin/fulfillment/tasks` → **401 `UNAUTHENTICATED`** thay vì 403; log fulfillment không có dấu vết request — việc phân giải danh tính employee phía fulfillment thất bại trước khi tới kiểm tra grant.
- Việc từ chối vẫn xảy ra (an toàn), nhưng sai ngữ nghĩa status — client không phân biệt được "chưa đăng nhập" với "thiếu quyền".

### F-20 — Profile-service không có permission `LOYALTY_VIEW` · Low · MỞ
- Hợp đồng web kỳ vọng `LOYALTY_VIEW` cho `/account/loyalty`; profile-service không định nghĩa permission này → trang loyalty sẽ luôn bị từ chối. Cần chốt hợp đồng (thêm permission hay bỏ capability).

---

## E. Catalog

### F-21 — `OutboxRow` lỗi constructor (MyBatis) — tạo SKU trả 500, outbox kẹt · High · MỞ (fix PR riêng)
- Bằng chứng (B1): `POST` tạo variant → 500; log catalog: `Error instantiating class com.hvduong.catalog.infrastructure.outbox.OutboxRow … NoSuchMethodException`. Cùng lỗi ở outbox publisher → sự kiện `product.created` (1 dòng trong `outbox_events`) không đọc/phát được.
- Đường vòng đã dùng ở W4: insert trực tiếp `product_variants`/`channel_prices` mô phỏng đúng `addVariant`.
- Hướng sửa: sửa mapping/constructor của `OutboxRow` trong catalog-service (kèm test hồi quy cho addVariant + publisher).

### F-22 — Response chuyển trạng thái trả DTO cũ · Low · MỞ
- Sau `SUBMIT_FOR_REVIEW` thành công, body trả về vẫn là trạng thái trước cập nhật (DRAFT, version 0) trong khi DB đã là PENDING v1 (APPROVE sau đó → APPROVED v2). Client đọc body sẽ hiểu sai trạng thái; DB là nguồn sự thật.

### F-23 — Catalog không có API tạo category · Low · MỞ
- Seed dữ liệu phải insert thẳng DB. Cân nhắc bổ sung endpoint admin hoặc tài liệu hóa đường seed chính thức.

---

## F. Hạ tầng & vận hành

### F-24 — Thay thế tạm đã biết từ W2 (chấp nhận cho MVP, thay sau MVP) · Info · ĐÃ GHI NHẬN
- Object storage: Adobe S3Mock thay MinIO (image MinIO đã biến mất khỏi Docker Hub) — độ bền dữ liệu chỉ ở mức best-effort.
- ClamAV: bản build cộng đồng arm64 với DB virus 2019–2020; việc làm mới DB (freshclam) cần thực hiện và kiểm chứng định kỳ.
- Fulfillment xác thực JWT bằng file khóa công khai của realm được ghim sẵn — khóa realm rotate thì phải cập nhật file (runbook).

### F-25 — Hiệu chỉnh số liệu backup (R2) · Info · ĐÃ KIỂM CHỨNG LẠI 12/11
- Restore thử vào DB scratch: keycloak dump → **92 bảng**, kong dump → **35 bảng**, khớp đúng số bảng của DB live; scratch đã xóa sau kiểm chứng, DB production không bị đụng. (Ghi chú W0 trước đây gán "35 bảng" cho keycloak là nhầm — 35 là của kong.)

### F-26 — Dọn dẹp sau nghiệm thu · Info · CHỜ THỰC HIỆN SAU CÁC WAVE
- Xóa các tài khoản test (e1-customer, e1-staff, w4run-diag-*) và artifacts seed của W4 (category/product/SKU `W4-1794460692`, đơn sandbox `69d73e4b-…`) khi team xác nhận không cần nữa.
- Resource redis thừa chưa từng chạy trong project `database` cần xóa qua UI (cần mật khẩu tài khoản của Bao).
- Bản sao backup ngoài VPS: chưa thực hiện (chờ chốt nơi giữ).

---

## Kết quả nghiệm thu W4 tóm tắt (12/11/2026)

| Nhóm | Kết quả |
|---|---|
| A — Bảo mật/token | A1, A2, A3, A4, A5(ii), A5(iii), A6 PASS; A5(i) FAIL → F-19 |
| B — Đường vàng | B1 PASS (có đường vòng F-21/F-23); B2/B3 phía web FAIL → F-12; B2'/B3' mức API PASS (sau F-17); B4 bị chặn → F-15; B5 chưa chạy được (phụ thuộc B4) |
| C — Tồn kho/sự kiện | C1 chỉ đúng trong simulator → F-16; C2 chưa chạy (phụ thuộc B4/B5); C3 PASS cả hai nửa (~2,4s principal → profile; tồn kho → catalog projection) |
| D — Gateway | D1 FAIL → F-09; D2 FAIL → F-10 |
| R — Chịu lỗi | R1 restart PASS (phát hiện F-08); R2 restore PASS (F-25); R3 rollback A→B→A PASS, site kết thúc ở bản f540fde |

**Kết luận:** hạ tầng và lớp danh tính/phân quyền đã vận hành được; chuỗi thương mại đầu-cuối **chưa thông** do F-12 (web) và F-15 (sự kiện order→fulfillment). Chưa nên tuyên bố "deploy xong" cho phạm vi thương mại cho tới khi hai Blocker này được sửa và nghiệm thu lại.
