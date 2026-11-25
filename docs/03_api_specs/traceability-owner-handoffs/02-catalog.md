# PBL6 — Bàn giao Catalog: CAT-TRC-01

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

## Phần việc của Catalog

### Catalog: CAT-TRC-01 — triển khai trực tiếp

CAT-OUT-01: `GET /api/v1/internal/traceability/skus/{sku_code}` trả exact product_id/sku_code/catalog_version, không phụ thuộc giá hoặc trạng thái public, có hidden/stopped variants còn lưu. Tái sử dụng query/domain/mapping hiện có phía sau endpoint, không sửa public variant response chỉ để bootstrap.

CAT-OUT-02: chỉ credential Traceability riêng được đọc; lỗi400/401/403/404/503 và envelope theo [integration contract](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-draft.openapi.yaml); không route public. Dùng version semantics khớp verifier Product–SKU hiện có, giữ nguyên verifier cũ. Test CAT-01…03. Không gọi Catalog khi khách scan QR hoặc Order resolve links.

Nhận định trước đây Catalog chắc chắn không cần sửa được thay bằng task này vì public SKU response thiếu productId còn verifier cũ yêu cầu biết Product trước.

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

| Test ID | Ca kiểm | Expected |
| --- | --- | --- |
| CAT-01 | Exact-SKU bootstrap active/hidden/stopped và không có giá | Trả đúng Product/SKU/version, không cần public/saleable/price |
| CAT-02 | Credential sai, SKU không tồn tại, khác case, DB lỗi | Không fuzzy match;400/401/403/404/503 đúng contract, không lộ internal binding public |
| CAT-03 | Bootstrap Product rồi verifier cũ phát hiện SKU đổi binding/version | Create/submit/publish chặn conflict, không tin context cũ từ browser |
| T-CTX-01 | Batch ở kho khác gọicontext, expired/restricted draft source, peer timeout | Kiểm Inventory và scope trước trả Product;403 ngoài kho; context không đồng nghĩa publish; timeout503, receipt giữ nguyên |

Nộp kết quả cuối/PR: OUT đã đáp ứng, SHA, phần tái sử dụng/bổ sung, mapping adapter nếu có, PASS/FAIL/SKIPPED từng test và bằng chứng đã lọc. Chưa có peer chạy thật vẫn code/test bằng fixture đúng contract, ghi `IMPLEMENTED_NOT_INTEGRATED`; không báo full E2E bằng mock. Chỉ báo blocker vượt scope, không gửi phương án rồi chờ duyệt trước sửa code.

Đọc AGENTS.md và rule service trong repository trước khi sửa. Commit mới dùng tiếng Anh, giữ một type/scope; contract thay trước hoặc cùng code, không sửa migration cũ/generated rows sai quy trình. Không tự push dev/release hoặc deploy production; tạo PR theo quy trình nhóm.

## Contract HTTP nhúng — không cần tải thêm file để đọc schema

Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.

```yaml
openapi: 3.1.0
info:
  title: PBL6 Catalog traceability handoff
  version: 0.2.2-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 89e15ca19432749b606c6d0a61700e98fdb7fd67. Direct
    implementation target, not runtime acceptance. All references are local to this embedded contract.
tags:
- name: Catalog
  description: Owner-specific provider and dependency contracts
servers:
- url: http://127.0.0.1
paths:
  /api/v1/internal/traceability/skus/{sku_code}:
    get:
      servers:
      - url: http://127.0.0.1:8005
      tags:
      - Catalog
      operationId: getTraceabilityCatalogBindingBySku
      summary: Resolve exact SKU to authoritative Product without a saleability dependency
      description: Owned by Catalog. Read-only exact case-sensitive SKU; include retained hidden or stopped variants,
        no price requirement, no fuzzy search. Reuse the dedicated Traceability token and version semantics of the
        existing Product-SKU verifier. Not routed publicly.
      parameters:
      - name: sku_code
        in: path
        required: true
        schema:
          type: string
          minLength: 1
          maxLength: 100
      responses:
        '200':
          description: Authoritative exact Product-SKU binding
          content:
            application/json:
              schema:
                type: object
                required:
                - product_id
                - sku_code
                - catalog_version
                properties:
                  product_id:
                    type: string
                    format: uuid
                  sku_code:
                    type: string
                  catalog_version:
                    type: string
                    minLength: 1
              example:
                product_id: 22222222-2222-4222-8222-222222222222
                sku_code: MX-001
                catalog_version: variant:1:1
        '400':
          $ref: '#/components/responses/Workflow_BadRequest'
        '401':
          $ref: '#/components/responses/Workflow_Unauthorized'
        '403':
          $ref: '#/components/responses/Workflow_Forbidden'
        '404':
          $ref: '#/components/responses/Workflow_NotFound'
        '503':
          $ref: '#/components/responses/Workflow_Unavailable'
      security:
      - Workflow_catalogTraceabilityToken: []
components:
  responses:
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
  schemas:
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
  securitySchemes:
    Workflow_catalogTraceabilityToken:
      type: apiKey
      in: header
      name: X-Internal-Token
      description: Dedicated Traceability-to-Catalog credential; existing verification credential, never exposed
        to browser.
x-source-commit: 89e15ca19432749b606c6d0a61700e98fdb7fd67
```

## Tính toàn vẹn bản gửi

Nguồn commit: `89e15ca19432749b606c6d0a61700e98fdb7fd67`. SHA-256 tập source: `13f279b3a1f8be29cd2064ceb02b8c06e391007992b4458d2a3c7afed3c699b5`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
