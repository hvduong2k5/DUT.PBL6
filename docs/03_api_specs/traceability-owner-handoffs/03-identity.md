# PBL6 — Bàn giao Identity / Keycloak: IAM-TRC-01

Version nghiệp vụ/contract: **0.2.2-draft** · Snapshot tài liệu: **`89e15ca19432749b606c6d0a61700e98fdb7fd67`**.

File này là bản `.md` độc lập để gửi qua Zalo. Đề bài, yêu cầu đầu ra, test và contract liên quan được nhúng bên dưới; không cần đọc lịch sử chat hoặc chờ phản hồi ACCEPT để bắt đầu code. Draft không đồng nghĩa chức năng đã chạy hoặc đã deploy.

## Nguồn tài liệu và cách lấy về

- Repository: [https://github.com/hvduong2k5/DUT.PBL6](https://github.com/hvduong2k5/DUT.PBL6). Nhánh tài liệu: `codex/traceability-order-flow`.
- **Đọc theo commit đã ghim**, không tự thay bằng HEAD của dev hoặc nhánh tài liệu đang tiếp tục thay đổi. Đây là nguồn tài liệu; owner giữ nhánh làm việc riêng cho service, không phải checkout đè công việc của mình.
- Có thể đọc toàn bộ đặc tả qua các link dưới. Người truy cập repository riêng tư cần quyền hiện có của nhóm. Contract nhúng trong file này vẫn đọc được khi chưa mở được link.

| Tài liệu liên quan trong repository | Link bất biến |
| --- | --- |
| `docs/03_api_specs/traceability-workflow-README.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-README.md) |
| `docs/03_api_specs/traceability-workflow-peer-handoff.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-peer-handoff.md) |
| `docs/03_api_specs/traceability-workflow-acceptance.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-acceptance.md) |
| `docs/03_api_specs/traceability-workflow-service-spec.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-service-spec.md) |
| `docs/03_api_specs/traceability-workflow-review-2026-11-25.md` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-review-2026-11-25.md) |
| `docs/03_api_specs/admin-stable.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/admin-stable.openapi.yaml) |
| `docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-draft.openapi.yaml) |

Trong repository đã clone, các lệnh đọc sau không checkout/reset hay ghi đè source đang làm:

```powershell
git fetch origin codex/traceability-order-flow
git show 89e15ca19432749b606c6d0a61700e98fdb7fd67:docs/03_api_specs/traceability-workflow-README.md
git show 89e15ca19432749b606c6d0a61700e98fdb7fd67:docs/03_api_specs/traceability-workflow-peer-handoff.md
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

- Wire/case/ID và permission là contract đích, không đổi tên cho giống convention riêng. Không fake provider verification bằng mock, không sửa stable/generated examples hoặc migration cũ.
- HTTP error chuẩn mới `{code,message,requestId}`; mutation mới có Idempotency-Key. Legacy route giữ contract riêng. Internal RPC/call dùng danh tính service và transport tin cậy, không tin X-User-ID/role do browser tự gửi.
- Không kích hoạt/publish QR khi nhận packing hoặc GoodsReceived. Không thêm trừ kho, tạo profile, cấp mã hoặc payment side effect trong read flow.

## Phần việc của Identity / Keycloak

**Mục đích:** cấp các permission đã có cho đúng nhân viên/kho. Không thêm role bypass hoặc scope claim mới.

**Đầu ra bắt buộc:** IAM-OUT-01: token và `/users/me` biểu diễn đúng permission gắn scopeId kho; IAM-OUT-02: nhân viên soạn/duyệt được gán đúng nhóm quyền, thời hạn và thu hồi có hiệu lực theo contract; IAM-OUT-03: có bộ actor test kiểm kho A/B và tách nhiệm vụ. Phần IAM grants có thể chỉ cần cấu hình/test. IAM-OUT-04: trong cookie mode cung cấp các transport trong [BFF contract](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml), guard ACTIVE EMPLOYEE cho cả v2, giữ Origin/CSRF và chuyển cookie thành bearer server-side; không mặc định Identity chỉ cần cấu hình. Direct OIDC mode không cần cookie adapter. Không bắt tạo role mới.

| Bundle nhiệm vụ | Permissions được gán |
| --- | --- |
| Soạn hồ sơ | read_internal + create + edit; thêm submit nếu được giao |
| Duyệt/công bố | read_internal + approve + publish, người duyệt độc lập với contributors |
| Xem bằng chứng | read_evidence riêng khi cần, không cấp tự động cho mọi warehouse staff |
| Dừng công khai | suspend/archive riêng cho nhiệm vụ quản lý |

Mã đầy đủ dùng prefix `traceability.`. Role code do owner Identity chọn theo danh mục hiện có; không bắt buộc tạo tên role mới. Assignment WAREHOUSE scopeId là UUID kho thật. `/users/me.permissionGrants` phải giữ permissionCode,scopeType,scopeId; JWT giữ chuỗi `permission#WAREHOUSE#uuid` hoặc `permission#GLOBAL`. Không chỉ flat permissions. Traceability kiểm intersection hai nguồn; quyền role bootstrap ADMIN hiện có không chứng minh warehouse staff đã có assignment.

Nhân viên thực hiện cả nhập kho phải giữ các quyền Inventory nền INVENTORY_VIEW, WAREHOUSE_WORKBENCH_VIEW, INVENTORY_RECEIPT_CREATE đúng kho theo contract hiện có. Traceability grants được bổ sung cho cùng người, không thay thế các quyền này. Batch-context không yêu cầu cấp thêm CATALOG management/BATCH_VIEW chỉ để lấy Product ID.

Giữ thời hạn assignment, sync token mới, revoke/disable/thu hồi theo cơ chế service hiện có. Không sửa V20/V21 để gắn user test; nếu cần seed mới dùng migration mới hoặc API quản trị trong môi trường test. Cấp hai actor soạn/duyệt ở kho A, actor kho B, GLOBAL auditor và actor thiếu read_evidence; lưu ID test không lưu password/token trong báo cáo.

**Nộp:** cấu hình/seed test actor và kết quả token+me đã lọc, case revoke/expiry và không mất scopeId. Nghiệm thu T-SCOPE-01…03, IAM-01. Thay đổi consumer GLOBAL-only thành scoped nằm ở Traceability, không yêu cầu Identity viết code đọc DB Traceability.

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| T-SCOPE-01 | makerA create/read/edit profileA rồi thửB | A thành công, B403 không nội dung; create không tin warehouse browser |
| T-SCOPE-02 | Revoke scope sau command rồi replay cùng key; đổi scope khi dùng cursor | Replay403; cursor authorization đổi bị từ chối; không rò cached response |
| T-SCOPE-03 | Token có quyềnA nhưng `/me` chỉB; Identity timeout; forged headers | Không cấp quyềnA, timeout503, header không đổi actor/scope |
| IAM-01 | Grant/revoke/expire WAREHOUSE, active/inactive | JWT/me giữ scopeId; quyền mới/thu hồi theo chính sách thật, không role bypass |
| IAM-02 | Cookie → trace v2 và Inventory receipt/workbench qua Identity; CUSTOMER token vào v2 | Đúng bearer server-side, ACTIVE EMPLOYEE required, CUSTOMER403, không token ở browser URL/log |
| WEB-05 | Thiếu/sai Origin trên cookie mutation; body256KiB/profile proxy; proxy response quá lớn | Origin403; không disable CSRF; header hợp lệ được forward; bounded response502 không giả thành công |

Nộp kết quả cuối/PR: OUT đã đáp ứng, SHA, phần tái sử dụng/bổ sung, mapping adapter nếu có, PASS/FAIL/SKIPPED từng test và bằng chứng đã lọc. Chưa có peer chạy thật vẫn code/test bằng fixture đúng contract, ghi `IMPLEMENTED_NOT_INTEGRATED`; không báo full E2E bằng mock. Chỉ báo blocker vượt scope, không gửi phương án rồi chờ duyệt trước sửa code.

Đọc AGENTS.md và rule service trong repository trước khi sửa. Commit mới dùng tiếng Anh, giữ một type/scope; contract thay trước hoặc cùng code, không sửa migration cũ/generated rows sai quy trình. Không tự push dev/release hoặc deploy production; tạo PR theo quy trình nhóm.

## Contract HTTP nhúng — không cần tải thêm file để đọc schema

Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.

```yaml
openapi: 3.1.0
info:
  title: PBL6 Identity / Keycloak traceability handoff
  version: 0.2.2-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 89e15ca19432749b606c6d0a61700e98fdb7fd67. Direct
    implementation target, not runtime acceptance. All references are local to this embedded contract.
tags:
- name: Identity / Keycloak
  description: Owner-specific provider and dependency contracts
servers:
- url: http://127.0.0.1
paths:
  /api/v2/admin/trace-api/records:
    get:
      operationId: listTraceabilityProfilesScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: listTraceabilityProfilesScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
    post:
      operationId: createTraceabilityProfileScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: createTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.create
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    get:
      operationId: getTraceabilityProfileScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: getTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: createTraceabilityRevisionScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: createTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.edit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    patch:
      operationId: replaceDraftContentScopedV2IdentityBff
      description: Replaces the whole Draft content snapshot; not a JSON merge patch. Does not change binding or
        accept contributor identities from the caller. All operations require read_internal plus their command permission
        for the same authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill
        and contributor policy. Identity transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id
        and Traceparent; no automatic mutation retry, no redirects. Preserve backend status/body and no-store. Next
        BFF must pass validated Origin to Identity for cookie mutations.
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: replaceDraftContentScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.edit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/submit:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: submitTraceabilityRevisionScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: submitTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.submit
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/submit
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/approve:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: approveTraceabilityRevisionScopedV2IdentityBff
      description: Reviewer must not be a contributor to this revision, including inherited contributors. Never
        publishes automatically. All operations require read_internal plus their command permission for the same
        authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor
        policy. Identity transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent;
        no automatic mutation retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass
        validated Origin to Identity for cookie mutations.
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: approveTraceabilityRevisionScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.approve
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-contributor-policy: actor must not be a contributor of the revision
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/approve
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/reject:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: rejectTraceabilityRevisionScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: rejectTraceabilityRevisionScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.approve
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-contributor-policy: actor must not be a contributor of the revision
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/reject
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/revisions/{revision}/publish:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    - $ref: '#/components/parameters/Scoped_RevisionNumber'
    post:
      operationId: publishTraceabilityRevisionScopedV2IdentityBff
      description: Rechecks dependency snapshot and certificate validity; no published result if source changed
        or contract is unavailable. All operations require read_internal plus their command permission for the same
        authoritative warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor
        policy. Identity transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent;
        no automatic mutation retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass
        validated Origin to Identity for cookie mutations.
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: publishTraceabilityRevisionScopedV2
      x-required-permissions:
      - traceability.read_internal
      - traceability.publish
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/revisions/{revision}/publish
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/suspend:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: suspendTraceabilityProfileScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: suspendTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.suspend
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/suspend
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/{id}/archive:
    parameters:
    - $ref: '#/components/parameters/Scoped_ProfileID'
    post:
      operationId: archiveTraceabilityProfileScopedV2IdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      tags:
      - Identity / Keycloak
      summary: archiveTraceabilityProfileScopedV2
      description: ' All operations require read_internal plus their command permission for the same authoritative
        warehouse. See service spec for GLOBAL/WAREHOUSE intersection, legacy backfill and contributor policy. Identity
        transport forwards Idempotency-Key, Content-Type, Accept, X-Request-Id and Traceparent; no automatic mutation
        retry, no redirects. Preserve backend status/body and no-store. Next BFF must pass validated Origin to Identity
        for cookie mutations.'
      x-required-permissions:
      - traceability.read_internal
      - traceability.archive
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/{id}/archive
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/records/by-batch:
    get:
      tags:
      - Identity / Keycloak
      operationId: findScopedTraceabilityByBatchDraftIdentityBff
      summary: Find the existing profile before creating a draft
      description: Requires traceability.read_internal for the authoritative warehouse from Inventory. Validate
        scope even when no profile exists. Warehouse is never trusted from browser input. Existing profile responses
        follow evidence redaction rules. Return at most one exact-binding match; no fuzzy SKU lookup. This literal
        route must take precedence over /records/{id}. Identity transport forwards Idempotency-Key, Content-Type,
        Accept, X-Request-Id and Traceparent; no automatic mutation retry, no redirects. Preserve backend status/body
        and no-store. Next BFF must pass validated Origin to Identity for cookie mutations.
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      x-required-permissions:
      - traceability.read_internal
      x-scope-policy: GLOBAL or resource-matching WAREHOUSE; token and current Identity grants must both cover each
        required permission.
      x-forward-to: /api/v2/admin/trace/records/by-batch
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v2/admin/trace-api/batches/{batch_id}/context:
    get:
      tags:
      - Identity / Keycloak
      operationId: getScopedBatchTraceabilityContextIdentityBff
      summary: Bootstrap a traceability form from an authoritative receipt batch
      description: Traceability-owned. Requires read_internal for the resource warehouse. Read Inventory first,
        check scope, then resolve Product by exact SKU from Catalog within a shared 2-second budget. No client-supplied
        product/warehouse identity. Does not create a draft or change stock. Restricted/expired source may be displayed
        for draft preparation, but existing submit/publish guards still apply. No dependency on unimplemented Inventory
        SKU detail endpoints or Catalog prices. Identity transport forwards Idempotency-Key, Content-Type, Accept,
        X-Request-Id and Traceparent; no automatic mutation retry, no redirects. Preserve backend status/body and
        no-store. Next BFF must pass validated Origin to Identity for cookie mutations.
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      x-forward-to: /api/v2/admin/trace/batches/{batch_id}/context
      x-body-limit-bytes: 262144
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v1/admin/inventory-api/workbench:
    get:
      tags:
      - Identity / Keycloak
      operationId: getAdminInventoryWorkbenchTraceabilityIdentityBff
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
      x-forward-to: /api/v1/admin/inventory/workbench
      x-body-limit-bytes: 0
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
  /api/v1/admin/inventory-api/workbench/receipts:
    post:
      tags:
      - Identity / Keycloak
      operationId: postAdminInventoryWorkbenchReceiptsTraceabilityIdentityBff
      summary: Idempotent stock receipt command for US-INV-02. The real Inventory service validates SKU, Batch identity
        and source reference before updating balances.
      description: Requires INVENTORY_RECEIPT_CREATE with GLOBAL scope or WAREHOUSE scope matching the SKU. Fresh
        APPLIED receipts include batchId/batchCode/skuCode/warehouseId. Legacy replay may omit these fields; preserve
        its result and let UI bootstrap from a verified batch, never repeat stock receipt.
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
        '502':
          description: Malformed or oversized upstream response; no successful body is fabricated.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: UPSTREAM_INVALID_RESPONSE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
        '413':
          description: Request exceeds this proxy route body limit; no upstream mutation was sent.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentityBff_ProxyError'
              example:
                code: REQUEST_TOO_LARGE
                message: Unable to forward this request safely.
                requestId: req-proxy-example
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
      x-forward-to: /api/v1/admin/inventory/workbench/receipts
      x-body-limit-bytes: 8192
      x-response-limit-bytes: 2097152
      security:
      - IdentityBff_employeeSessionCookie: []
      - IdentityBff_identityStaffBearer: []
      servers:
      - url: http://127.0.0.1:8016
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
    IdentityBff_ProxyError:
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
  securitySchemes:
    IdentityBff_employeeSessionCookie:
      type: apiKey
      in: cookie
      name: OMAMA_SESSION
      description: Existing HttpOnly session resolved server-side; no browser access-token exposure.
    IdentityBff_identityStaffBearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: Verified Identity/Keycloak employee token; same active principal checks apply. Not a forged user/role
        header.
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
x-source-commit: 89e15ca19432749b606c6d0a61700e98fdb7fd67
```

## Tính toàn vẹn bản gửi

Nguồn commit: `89e15ca19432749b606c6d0a61700e98fdb7fd67`. SHA-256 tập source: `7517c874fc16e4f83915dddd92e12476b93770abf209da498b9f6e1181e34573`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
