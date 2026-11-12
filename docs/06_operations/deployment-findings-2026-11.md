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

---

## Cập nhật chiều 12/11/2026 (sau W4 — đợt sửa F-12 và nghiệm thu lại)

- **F-12 → ĐÃ SỬA (PR #90), còn một điểm nghẽn chuyển sang F-10.** PR #90 (stack trên PR #88) đã deploy live (release head `65f6f88`, kèm `API_GATEWAY_UPSTREAM_URL` tường minh). Nghiệm thu lại: trang sản phẩm PASS (giá/tồn kho hiển thị đúng), giỏ hàng PASS (thêm/xóa/số lượng chạy; giá trong giỏ hiện "Chờ báo giá" theo đúng thiết kế MVP — quote mới chốt giá), capabilities/profile không hồi quy. Riêng **đăng ký**: trang đăng ký gọi thẳng gateway từ trình duyệt (client service dùng `NEXT_PUBLIC_API_URL` — đúng thiết kế mà cấu hình CORS của kong.yml phục vụ), nên sau khi sửa upstream, điểm nghẽn còn lại là preflight CORS bị Kong từ chối → thuộc F-10.
- **F-10 → NGUYÊN NHÂN GỐC ĐÃ RÕ; mức độ tăng thành Blocker cho luồng đăng ký.** `kong.yml` của repo KHAI ĐẦY ĐỦ: plugin `cors` toàn cục (origins `omama.hvduong.qzz.io` + `localhost:3000`, `credentials: true`), `rate-limiting` cho registration (10/phút), checkout (30/phút) và callback (600/phút), `rate-limiting` toàn cục 300/phút/IP, `correlation-id`, `request-transformer`, `prometheus`, và cả service `payment-webhook-handler` với route callback. **Bản Kong live được dựng tay chỉ gồm services/routes — toàn bộ plugin và service webhook đều thiếu** (đây cũng là nguyên nhân gốc của F-09 và F-11). Ngoài ra danh sách origins đã khai chưa có origin web hiện tại `https://pbl.npanhbao.dev` (web chuyển domain ở W3) — đã bổ sung vào kong.yml bằng PR riêng. Kế hoạch áp dụng lên Kong live: Stage 1 = `cors` + các `rate-limiting` (đúng cấu hình đã khai); Stage 2 = `correlation-id`, `request-transformer`, `prometheus`, `payment-webhook-handler`.
- **F-14 → NGUYÊN NHÂN ĐÃ CÔ LẬP.** Sau F-12, lời gọi từ web tới order-service đã thành công (`GET /api/v1/orders` → 200, có payload) nhưng trang vẫn báo lỗi: payload của order-service bản sandbox (dạng legacy) không khớp public contract mà web handler kiểm chứng (`isOwnedOrdersResponse`). Web chủ đích chỉ nới chuẩn legacy ở môi trường dev (hàm `normalizeCart` của giỏ hàng có cùng chính sách). Hướng xử lý đúng: order core mode trả đúng public contract; không nới validation ở production.
- **F-13 → TÁI XÁC NHẬN (12/11).** Đăng xuất trên web chỉ xóa cookie của app, không kết thúc phiên SSO ở Keycloak: bấm đăng nhập lại là vào thẳng tài khoản cũ, UI không có cách chuyển tài khoản; phải gọi trực tiếp `end_session_endpoint` của Keycloak mới đăng xuất thật.
- **F-27 (MỚI) — Reviews chưa có service sở hữu · Low · MỞ.** Widget đánh giá trên trang sản phẩm gọi qua customer-core route tới `{CUSTOMER_CORE}/reviews/products/{id}` (identity customer-api) → 404. Cùng họ với F-20 (`LOYALTY_VIEW`): web đã kỳ vọng các bề mặt reviews/loyalty nhưng chưa service nào implement chúng.

---

## Cập nhật tối 12/11/2026 (đợt sửa Kong Stage 1 + catalog + đóng F-12)

- **F-09/F-10 → ĐÃ ĐÓNG (Kong Stage 1 áp dụng live ~14:31).** Đã tạo trên Kong live đúng cấu hình kong.yml khai: plugin `cors` toàn cục (origins gồm `https://pbl.npanhbao.dev`), `rate-limiting` toàn cục 300/phút/IP, `rate-limiting` 10/phút cho route đăng ký, 30/phút cho route checkout. Kiểm chứng: D2 PASS (preflight từ origin web → 200 kèm ACAO + credentials; origin lạ không nhận ACAO), D1 PASS (POST đăng ký thứ 11 trong phút → 429). **Đăng ký qua UI chạy đầu-cuối lần đầu tiên** (tài khoản kiểm chứng `w4cors-1794468675@test.local`, đăng nhập + trang profile hiển thị đúng). Origin web hiện tại cũng đã được bổ sung vào kong.yml của repo (PR #91). Stage 2 (correlation-id, request-transformer, prometheus, `payment-webhook-handler`) vẫn còn nợ — xem F-11.
- **F-12 → ĐÃ ĐÓNG.** Vòng CI đầu của PR #90 fail chính check web-identity: fixture chạy web ở chế độ identity-bff, nơi mount `customer-api` là proxy dịch session — bản sửa v1 gửi credential về gateway root nên order-service không dịch được session id. Bản sửa v2 tách resolver theo chế độ (`customerCommerceUpstreamUrl()`: identity-bff giữ mount customer-api, OIDC dùng gateway root; ADR-002 đã bổ sung amendment). CI web-identity **xanh lại** trên head v2 (run #102, 18 phút, mọi stage PASS), gate local 189/189. PR #88 đã được merge vào dev (squash `e3fafa8`) cùng các PR #82–#87; PR #90 đã merge dev vào nhánh (commit `3367e46` — conflict ở đúng 3 file do squash chạm, giữ phiên bản của nhánh sau khi đối chiếu; CI đang chạy lại trên kết quả merge tại thời điểm ghi).
- **F-21/F-22 → ĐÃ ĐÓNG (PR #92 + nghiệm thu live).** Sau khi redeploy catalog (release head `14b482d`) và restart Kong: dòng outbox `product.created` bị kẹt từ W4 (305 lần thử phát) đã có `published_at`; sản phẩm tạo mới trong buổi nghiệm thu phát sự kiện sau ~1,3 giây; số dòng chưa phát = 0; log container mới không còn lỗi constructor. F-22: transition `SUBMIT_FOR_REVIEW` trả body đã là PENDING phiên bản 1 ngay trong response.
- **Hiệu chỉnh nhận định về lỗi addVariant 500 ở W4.** Trong buổi nghiệm thu lại, addVariant chỉ trả 500 khi dữ liệu thử sai enum `packaging_type` (lỗi nhập liệu của người kiểm chứng, log là `DataIntegrityViolationException`), và trả 200 khi input hợp lệ. Vì addVariant không đọc outbox trong code, lỗi 500 ở W4 **không còn cơ sở để gán cho F-21**; tác hại đã chứng minh được của F-21 chỉ là outbox không phát được sự kiện.
- **Ghi chú vận hành Keycloak.** Đổi cờ Direct Access Grants của client `frontend-web` bằng cách sửa thẳng DB không có hiệu lực ngay (cache client) — phải restart Keycloak thì password grant mới dùng được; sau buổi nghiệm thu cờ đã tắt lại và kiểm chứng bị từ chối (`unauthorized_client`). Ưu tiên thao tác qua Admin API/console thay vì sửa DB.
- **F-26 bổ sung danh sách dọn dẹp:** sản phẩm kiểm chứng F-22 (`0350fb42-3d42-49ed-9c3e-e24b2629c158`, SKU `F22-1794470583`) và các tài khoản `w4fix-diag2-1794467700@test.local`, `w4cors-1794468675@test.local`.
- **Trạng thái Blocker sau đợt này:** F-12 đã đóng; **Blocker duy nhất còn lại cho đường vàng là F-15** (lệch hợp đồng sự kiện order→fulfillment v2/v3 — cần team chốt hợp đồng), kèm F-14 chờ order core mode.

---

## Cập nhật 12/11/2026 — Kong Stage 2 hoàn tất, F-11 đóng

- **F-11 → ĐÃ ĐÓNG.** Stage 2 đã áp dụng lên Kong live (đúng cấu hình kong.yml khai): plugin toàn cục `correlation-id` (`X-Request-Id`, uuid#counter, echo downstream), `request-transformer` toàn cục (gỡ 6 header định danh khỏi request từ trình duyệt) và bản tương ứng trên route khách hàng của profile, `prometheus` (status/latency/bandwidth/upstream-health), service `payment-webhook-handler` (upstream `order-svc:8004` — dùng alias ổn định thay tên container như kong.yml viết) + route `vietqr-callback-route` (hai path callback VietQR/SePay, chỉ POST, rate-limit 600/phút). Tổng số plugin live: 10; Stage 1 và service/route gRPC của đồng đội không bị đụng.
- **Kiểm chứng:** response của API sản phẩm công khai đã kèm header `x-request-id` đúng định dạng; POST body rỗng tới `/api/v1/payments/vietqr/callback` nhận **401 ở tầng ứng dụng** từ order-service (kèm header rate-limit 600/phút) thay vì 404 "no Route matched" của Kong như trước — route đã thông tới upstream. Không giả lập callback có chữ ký trong buổi kiểm chứng này.
- Như vậy Kong live nay đã khớp toàn bộ phần plugin/service mà kong.yml khai báo; ba mục F-09, F-10, F-11 cùng một nguyên nhân gốc đều đã đóng.

---

## Cập nhật tối 12/11/2026 — W5: order core mode, đóng F-15/F-16, MỨC 1 ĐẠT

Bao chốt **Mức 1** làm vạch đích deploy (21:43): đường vàng chạy thật trên VPS = deploy hoàn thành; mọi mục còn lại là backlog sau deploy. W5 bật order core mode (phương án đã chốt cho F-15/F-14) và khép các khâu còn thiếu:

- **F-15 → ĐÃ ĐÓNG (bằng chứng Phase 4, 19:34).** Đơn core thật `222fe172-7585-4e59-9a25-27a00c558ab8`: checkout báo giá thật 100.000đ + phí ship 27.000đ (công thức hãng FAKE qua TLS) = 127.000đ; cả ba sự kiện `placed.v3` / `paid.v3` / `fulfillment_ready.v3` phát lên `order.events.v3`, outbox PUBLISHED; fulfillment tiêu thụ (lag 0). Sau FIX-F (bên dưới), admission ACCEPTED và đúng một task đóng gói được tạo.
- **F-16 → ĐÃ ĐÓNG.** Cùng đơn trên: tồn kho thật COMMITTED, kho vật lý 10 → 9 tại thời điểm thanh toán; reservation qua inventory terminal TLS thật (không còn simulator).
- **F-14 → ĐÃ ĐÓNG theo hệ quả.** Payload của order core khớp public contract mà web kiểm chứng; trang đơn của khách hiển thị đúng đơn PAID 127.000đ (kiểm chứng qua customer view, 12/11).
- **Sự cố Phase 3 + bài học vận hành (ghi để tránh lặp lại):** lần redeploy đầu khi bật core crash-loop order ~2h38m. Ba nguyên nhân: (1) trình sửa env hàng loạt của Coolify làm hỏng 4 giá trị có sẵn (một token bị cắt 88→87 ký tự, JWT secret, callback secret, encryption key) — **từ nay sửa env Coolify TỪNG BIẾN MỘT**; (2) file secret staging root-only trong khi container chạy uid dịch vụ → phải chown thư mục staging cho đúng uid; (3) cổng shipping của core checkout: entrypoint luôn export token file + dial blocking lúc khởi động + plaintext chỉ cho loopback → bắt buộc TLS thật. Đã xử lý ở Wave 1: CA nội bộ "PBL6 Internal CA" (hết hạn 2028-11-11), cert `shipping.internal`, kiểm chứng âm tính sai-CA/sai-hostname đều FAIL đúng kỳ vọng. Nhãn nghiệm thu trung thực cho Wave 1 là "Core Checkout Preparation — Shipping Gated" cho tới khi Phase 4 chứng minh chức năng.
- **FIX-W (webhook SePay) → ĐÃ ĐÓNG 22:56.** Nguyên nhân 401: giá trị Secret Key trong webhook #5524 khác secret đã staged trên VPS (scheme/header/route đều đúng — đã kiểm chứng bằng request ký có kiểm soát). Bao tự dán lại secret từ Dashboard (không qua chat); phát lại delivery #71307 → 200, observation cho giao dịch thật 48830 được ghi; toàn vẹn settlement giữ nguyên (intent SETTLED, đúng 1 receipt, allocated 127.000đ) — bất biến "một giao dịch → tối đa một settlement" được kiểm chứng qua cả hai đường quét + webhook.
- **FIX-F (đường permit order↔fulfillment) → ĐÃ ĐÓNG 21:16.** `FULFILLMENT_CAPABILITY_UNAVAILABLE` phát ra từ phía order: provider fulfillment của order là nil vì Wave 1 đã gỡ toàn bộ `ORDER_FULFILLMENT_*`. Khôi phục đúng contract: token hai chiều PHẢI khác nhau và mỗi bên giữ token của bên kia (snapshot cũ bị hoán chiều — khôi phục nguyên văn vẫn rớt); caller token là FILE (không có biến B64); cần thêm `ORDER_PACKING_POLICY_FILE`; route insecure bị code từ chối (loopback-only) → order dial fulfillment bằng TLS ghim CA + tên `fulfillment.internal` (fulfillment vẫn phục vụ plaintext ở local mode — các RPC order→fulfillment ngoài đường permit sẽ lỗi bắt tay TLS cho tới Wave 2). Endpoint inventory cũ của fulfillment (tên container lỗi thời) đã sửa thành `inventory-svc:8001`. Job admission `e3890469` thành công ở lần thử 30: task `5ab3fcad-b736-4457-9add-1bec622e0669`.
- **B5 → PASS (22:00).** Đóng gói đủ vòng đời trên task trên: gán việc → bắt đầu → nhặt hàng → checklist ITEMS_MATCH + SEAL_AND_VIDEO → video bằng chứng VERIFIED (quét SAFE) → niêm phong `SEAL-B5-20261112` → task COMPLETED, sự kiện `vn.omama.fulfillment.packing.completed.v2` đã publish lên Kafka. Điều kiện tiên quyết phát hiện giữa chừng: bucket `packing-evidence` (S3Mock) chưa từng bật versioning trong khi bước Verify bắt buộc VersionId thật → đã bật versioning (đây là một phần của F-24: các thay thế hạ tầng của W2 cần checklist riêng khi nghiệm thu sâu). File media cũ (upload trước khi bật versioning) còn kẹt VERIFYING/MANUAL_REVIEW — dư thừa vô hại, dọn sau.
- **Catalog Ô Mạ thật đã nạp (yêu cầu thêm của Bao khi chốt Mức 1):** 5 danh mục storefront (banh-cung-dinh, me-xung-keo-hue, mut-hat-sen, tra-cung-dinh, qua-bieu) + 5 sản phẩm có giá/SKU từ chính dữ liệu frontend (120.000đ / 85.000đ / 345.000đ / 450.000đ / 380.000đ), tồn kho 50/SKU (batch OMA-SEED-2026), tất cả APPROVED/ACTIVE, projection khớp. Kiểm chứng storefront: mọi danh mục trả 200 (kể cả `me-xung-keo-hue` từng 500 — xem F-28); riêng `mut-hat-sen` chưa có sản phẩm vì toàn bộ mã nguồn frontend không tồn tại sản phẩm có giá cho danh mục này (không tự bịa giá). Vết dữ liệu nhỏ: vài thumbnail bị ghép đôi tiền tố CDN (`cdn.example.com/https://…`) — team sửa ở dữ liệu nguồn.

### Finding mới từ W5

- **F-28 — Catalog: slug danh mục không tồn tại → web route trả 500 thay vì danh sách trống · Low · MỞ.** Trang danh mục hiện nguyên màn hình "gián đoạn" như sập hệ thống. Sửa đúng: catalog trả rỗng/404 sạch và/hoặc web render trạng thái trống. Đã giảm nhẹ thực tế nhờ nạp đủ danh mục storefront.
- **F-29 — Order: bug SQL trong `SweepManualRefunds` · Low · MỞ (code, cho team).** `payment_queues.go`: `version=version+1` mơ hồ vì cả `refunds` lẫn `receipts` đều có cột `version` → Postgres lỗi phân tích, vòng lặp 30 giây retry vĩnh viễn và chỉ log nhãn (không kèm error — `startPaymentLoop` nuốt lỗi). Đã tái hiện trong transaction ROLLBACK. Hiện không có refund nào nên tác động = 0 ngoài nhiễu log. Sửa: `f.version=f.version+1` + log kèm error.
- **F-30 — Fulfillment: dòng outbox audit kẹt retry · Low · MỞ.** Một dòng `audit.events.v1` (từ thời điểm admission 14:16Z) retry 231+ lần chưa publish; các dòng audit khác attempts 0. Cần điều tra publisher/đích của topic audit.
- **Quan sát không thành finding:** customer order view không có trường trạng thái fulfillment/shipping (theo thiết kế hiện tại — `task_id`/`shipment_id` null ở phía order); merchant_identity của SePay đã đối chiếu khớp cả ba nguồn (báo động trước đó không tái hiện; bootstrap của order fail-closed khi lệch thật). Chiều shipping (tạo shipment từ PackingCompleted) vẫn tắt có chủ ý từ Wave 1 — thuộc Wave 2/backlog cùng yêu cầu TLS client cho shipping→order.

### Tuyên bố Mức 1 (12/11/2026)

Đường vàng đã chạy thật đầu-cuối trên VPS: đăng ký/đăng nhập → duyệt catalog Ô Mạ thật → giỏ hàng → checkout core → thanh toán SePay test settle qua cả hai đường (quét tài khoản + webhook 200) → tồn kho thật trừ kho → task đóng gói tự tạo → đóng gói hoàn tất kèm video bằng chứng đã duyệt + niêm phong + sự kiện hoàn thành. **Deploy được tuyên bố HOÀN THÀNH ở Mức 1.** Backlog sau deploy: Wave 2 (TLS-everywhere: fulfillment production + order TLS-only + Kong HTTPS + mở lại chiều shipping), F-01/F-02/F-03/F-04/F-05/F-06/F-07/F-08/F-13/F-18 (nửa API)/F-19/F-20/F-23/F-24/F-26/F-27/F-28/F-29/F-30, CI/CD theo tag.
