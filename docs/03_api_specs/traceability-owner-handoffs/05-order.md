# PBL6 — Bàn giao Order: ORD-TRC-01

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
| `docs/03_api_specs/traceability-workflow-candidates/packing.proto` | [Mở đúng commit](https://github.com/hvduong2k5/DUT.PBL6/blob/89e15ca19432749b606c6d0a61700e98fdb7fd67/docs/03_api_specs/traceability-workflow-candidates/packing.proto) |

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

## Phần việc của Order

**Mục đích:** người có quyền xem đơn đọc được danh sách nguồn gốc theo từng lô thực giao, không làm lỗi phần detail/payment/shipping đang có.

**Đầu ra bắt buộc:** ORD-OUT-01: Web lấy được từng line/batch/quantity và link/trạng thái truy xuất đúng đơn; ORD-OUT-02: chỉ chủ đơn hoặc guest capability hợp lệ được đọc; ORD-OUT-03: proof/Traceability thiếu hoặc lỗi không phá detail/lifecycle, không gắn nhầm lô. Owner đối chiếu API/read model hiện có trước; endpoint riêng bên dưới là contract đích để triển khai. Có thể dùng logic/read model hiện có phía sau route/adapter đúng contract đó; không cần consumer duyệt phương án nội bộ. Không tự thêm field vào stable detail hoặc đổi response ngoài phạm vi.

**Consumer:** Fulfillment proof candidate và Traceability internal resolver. **Provider đích:** `GET /api/v1/orders/{order_id}/traceability` trong OpenAPI tích hợp, endpoint riêng; không sửa frozen commercial snapshot hoặc lifecycle chỉ vì lookup nguồn gốc lỗi.

1. Dùng proof exact version từ luồng notification hiện có hoặc read model đã được kiểm chứng. Kiểm order/task/package/sealed version/generation/digest/warehouse và invariant dòng-lô-số lượng. Đầu ra nhận diện theo `(order, package, sealed_version, order_line_id, batch_id)`; duplicate input không cộng số lượng lần hai, event cũ không ghi lùi phiên bản. Persist projection là một phương án; owner có thể tái sử dụng projection/proof reader đã có nếu bảo đảm cùng semantics, ownership, deadline và không lấy latest/live allocation thay lịch sử. GET không tạo command nghiệp vụ để “bù” dữ liệu thiếu.

   Tách kiểm chứng base package proof dùng cho lifecycle hiện có khỏi extension packed_lines dùng cho nguồn gốc. Base proof sai vẫn bị từ chối như baseline. Riêng packed_lines thiếu/sai pair/quantity không được làm adapter trả lỗi khiến event PACKING_SEALED và shipping kẹt: ghi traceability capability UNAVAILABLE, giữ xử lý base proof/lifecycle hợp lệ. Không nới kiểm tra parcel/media/order/generation/digest/seal để test traceability qua.
2. line_id lấy từ frozen Order snapshot; customer projection hiện không gán ItemID nên UI có thể dùng orderId-index. Subresource trả canonical order_line_id và sku_code. Current manifest unique SKU: Web/BFF giữ typed skuCode từ raw detail và map exact unique SKU, kiểm tổng quantity trước gắn link; không parse skuLabel, không join index/tên. Nếu duplicate/không chứng minh mapping, state UNAVAILABLE; không gắn tạm SKU khác. Không cần sửa frozen commercial snapshot hoặc stable detail để có bridge này.
3. Root state: không có proof và đơn chưa bị hủy trước packing → PACKING_PENDING; hủy trước packing → NOT_APPLICABLE; có proof hợp lệ → READY; proof/capability không xác minh được → UNAVAILABLE. Có sealed proof rồi hủy/hoàn sau đó vẫn cho đọc lô lịch sử; không xóa lịch sử. items rỗng có total0; READY không đồng nghĩa mọi link AVAILABLE.
4. Kiểm auth/ownership trước mọi peer call. Dùng cơ chế customer/guest detail của đúng release; tại dev1564a7bc guest session do Order cấp có issuer `order-service-guest-session`,audience `order-storefront`,typ `guest-session+jwt`,live session được xác minh. Payment read token issuer `order-service-guest`/aud `order-payment` không đủ đọc nguồn gốc đơn. Không thêm token/OTP/challenge mới.
5. Resolver input chỉ `source_namespace=inventory`, batch UUID và SKU từ proof. ≤100/request, chia chunk và reconstruct output. Không cần product_id vì Traceability unique namespace+batch và đã verify product lúc publish. Timeout ≤1.5s/call, outer request budget5s; không retry đồng bộ trong cùng request, UI có thể retry read sau. Không gửi PII/actor token customer cho internal resolver. Concurrency≤2 và total budget5s, chunk chưa xong báoUNAVAILABLE. Quá1000line-batch rows hoặc response512KiB → rootUNAVAILABLE, không truncate thành READY.
6. AVAILABLE/NOT_AVAILABLE/NO_LONGER_PUBLIC/UNAVAILABLE từ resolver map nguyên nghĩa. Resolver auth/config lỗi làm các row UNAVAILABLE và có operational log an toàn, không biến thành NOT_AVAILABLE. Full Order DB lỗi trả503. Root total là số line-batch row, không là số sản phẩm.

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

Nộp kết quả cuối/PR: OUT đã đáp ứng, SHA, phần tái sử dụng/bổ sung, mapping adapter nếu có, PASS/FAIL/SKIPPED từng test và bằng chứng đã lọc. Chưa có peer chạy thật vẫn code/test bằng fixture đúng contract, ghi `IMPLEMENTED_NOT_INTEGRATED`; không báo full E2E bằng mock. Chỉ báo blocker vượt scope, không gửi phương án rồi chờ duyệt trước sửa code.

Đọc AGENTS.md và rule service trong repository trước khi sửa. Commit mới dùng tiếng Anh, giữ một type/scope; contract thay trước hoặc cùng code, không sửa migration cũ/generated rows sai quy trình. Không tự push dev/release hoặc deploy production; tạo PR theo quy trình nhóm.

## Contract HTTP nhúng — không cần tải thêm file để đọc schema

Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.

```yaml
openapi: 3.1.0
info:
  title: PBL6 Order traceability handoff
  version: 0.2.2-draft
  license:
    name: Proprietary
    identifier: LicenseRef-Proprietary
  description: Distribution snapshot from documentation commit 89e15ca19432749b606c6d0a61700e98fdb7fd67. Direct
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
  /api/v1/internal/traceability/batch-links/resolve:
    post:
      tags:
      - Order
      operationId: resolveOrderBatchTraceabilityDraft
      summary: Resolve published links for exact batch bindings
      description: Read-only bounded bulk lookup. Authenticated order-service identity with a dedicated credential
        over trusted internal transport; never public Kong routing. Does not create records, publish, or mutate
        stock, so no create Idempotency-Key is required. Preserve input order and one output per input. Unknown
        and unpublished bindings both return NOT_AVAILABLE. No internal profile IDs, evidence or actor data. Cache-Control
        no-store. AVAILABLE requires eligible public revision and qr_url; NO_LONGER_PUBLIC returns only the previously
        public URL; if it cannot be reconstructed safely, return UNAVAILABLE. Other states have null qr_url.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
              - items
              properties:
                items:
                  type: array
                  minItems: 1
                  maxItems: 100
                  items:
                    $ref: '#/components/schemas/Workflow_Binding'
      responses:
        '200':
          description: One result per binding; per-item source failures remain UNAVAILABLE
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
                required:
                - items
                - total
                properties:
                  items:
                    type: array
                    maxItems: 100
                    items:
                      $ref: '#/components/schemas/Workflow_ResolvedBindingResult'
                    minItems: 1
                  total:
                    type: integer
                    minimum: 1
                    maximum: 100
              examples:
                mixed:
                  value:
                    items:
                    - sku_code: MX-001
                      source_namespace: inventory
                      batch_id: 33333333-3333-4333-8333-333333333333
                      status: AVAILABLE
                      qr_url: https://shop.example.test/trace/abcdefghijklmnopqrstuv
                    - sku_code: MX-001
                      source_namespace: inventory
                      batch_id: 44444444-4444-4444-8444-444444444444
                      status: NOT_AVAILABLE
                      qr_url: null
                    - sku_code: MX-002
                      source_namespace: inventory
                      batch_id: 55555555-5555-4555-8555-555555555555
                      status: NO_LONGER_PUBLIC
                      qr_url: https://shop.example.test/trace/zyxwvutsrqponmlkjihgfe
                    - sku_code: MX-003
                      source_namespace: inventory
                      batch_id: 66666666-6666-4666-8666-666666666666
                      status: UNAVAILABLE
                      qr_url: null
                    total: 4
        '401':
          $ref: '#/components/responses/Workflow_Unauthorized'
        '403':
          $ref: '#/components/responses/Workflow_Forbidden'
        '422':
          $ref: '#/components/responses/Workflow_Invalid'
        '503':
          $ref: '#/components/responses/Workflow_Unavailable'
      security:
      - Workflow_serviceBearer: []
      servers:
      - url: http://127.0.0.1:8003
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
    Workflow_Binding:
      type: object
      required:
      - sku_code
      - source_namespace
      - batch_id
      properties:
        sku_code:
          type: string
          minLength: 1
          maxLength: 100
        source_namespace:
          type: string
          enum:
          - inventory
        batch_id:
          type: string
          format: uuid
    Workflow_ResolvedBindingResult:
      oneOf:
      - type: object
        additionalProperties: false
        required:
        - sku_code
        - source_namespace
        - batch_id
        - status
        - qr_url
        properties:
          sku_code:
            type: string
            minLength: 1
            maxLength: 100
          source_namespace:
            type: string
            enum:
            - inventory
          batch_id:
            type: string
            format: uuid
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
        - sku_code
        - source_namespace
        - batch_id
        - status
        - qr_url
        properties:
          sku_code:
            type: string
            minLength: 1
            maxLength: 100
          source_namespace:
            type: string
            enum:
            - inventory
          batch_id:
            type: string
            format: uuid
          status:
            type: string
            enum:
            - NOT_AVAILABLE
            - UNAVAILABLE
          qr_url:
            type: 'null'
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
    Workflow_Invalid:
      description: Invalid or incomplete binding; code INVALID_BINDING
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Workflow_Error'
          examples:
            INVALID_BINDING:
              value:
                code: INVALID_BINDING
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
    Workflow_serviceBearer:
      type: http
      scheme: bearer
      description: Dedicated Order internal credential, TLS required outside isolated local tests
x-source-commit: 89e15ca19432749b606c6d0a61700e98fdb7fd67
```

## Proto target nhúng: `docs/03_api_specs/traceability-workflow-candidates/packing.proto`

Không compile cạnh proto canonical cùng package/message vì đây là candidate full definition. Khi implement cập nhật canonical/copies/codegen theo contract, giữ field/method cũ; không hand-edit generated code.

```proto
// Implementation target 0.2.2-draft, 2026-11-25. NOT canonical/runtime code.
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

Nguồn commit: `89e15ca19432749b606c6d0a61700e98fdb7fd67`. SHA-256 tập source: `fbd951d3a39c452b41ab6cd7101cfe22e23f8f7d57714a66c06c65c1129bad47`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn và nội dung nhúng dùng cùng commit; không có token/password trong file.
