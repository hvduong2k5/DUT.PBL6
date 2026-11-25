# PBL6 — Bàn giao Frontend / Gateway: WEB-TRC-01

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
| `docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-scoped-admin-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-service.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-service.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-draft.openapi.yaml) |
| `docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-bff-draft.openapi.yaml) |
| `services/api-gateway/docker-compose.yml` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/services/api-gateway/docker-compose.yml) |

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

## Contract HTTP nhúng — không cần tải thêm file để đọc schema

Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.

```yaml
openapi: 3.1.0
info:
  title: PBL6 Frontend / Gateway traceability handoff
  version: 0.2.2-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 89e15ca19432749b606c6d0a61700e98fdb7fd67. Direct
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

## Mapping cookie transport do Identity cung cấp

| BFF route | Backend route qua Kong |
| --- | --- |
| `/api/v2/admin/trace-api/...` | `/api/v2/admin/trace/...` |
| `/api/v1/admin/inventory-api/workbench` | `/api/v1/admin/inventory/workbench` |
| `/api/v1/admin/inventory-api/workbench/receipts` | `/api/v1/admin/inventory/workbench/receipts` |

Cookie mode chuyển `OMAMA_SESSION` server-side qua Identity; forward Origin đã kiểm tra ở mutation, giữ Idempotency-Key. Bearer mode lấy credential tin cậy trong server, không cho client gán user/role headers. Internal resolver và Catalog binding không route public.

## Tính toàn vẹn bản gửi

Nguồn commit: `89e15ca19432749b606c6d0a61700e98fdb7fd67`. SHA-256 tập source: `95f8fbc43812b4ad6b46ad9abec83fc01a017fd51c314a1aa10b98ba42998267`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
