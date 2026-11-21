# Agent Rules — DUT.PBL6 (Repository-wide)

## Commit Message Convention (MANDATORY)

**All commit messages MUST be written in 100% English.** No Vietnamese text in commit subject or body.

### Format
```
<type>(<scope>): <short description in English>

- Detail of change 1
- Detail of change 2
- ./gradlew test --offline: PASS
```

### Types
| Type | Meaning |
|------|---------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Refactor without behavior change |
| `docs` | Documentation only |
| `chore` | Build, config, maintenance |
| `build` | Build system changes |
| `ci` | CI/CD changes |
| `test` | Adding or updating tests |
| `style` | Formatting, missing semicolons, etc. |
| `perf` | Performance improvement |

### Rules
- Each commit must do **one thing** only (one type, one scope).
- Do not squash unrelated changes into one commit.
- Commit message must be clear. Do not use vague words like "fix", "update", "done".
- Subject line: `<type>(<scope>): <short description>` — English only.
- Body bullets: English only.

### Examples (correct)
```
feat(catalog): add PATCH product and SKU admin endpoints

- Implement PATCH /products/{id} with full content update including story
- Implement PATCH /skus/{id} for variant attributes and sale status
- skuCode is immutable; price changes go through /prices endpoint
- Add unit tests for service and controller layers
- ./gradlew test --offline: PASS
```

```
fix(catalog): escape XML comparison operators in ProductMapper.xml

- Replace raw '<' and '>' with '&lt;' and '&gt;' in MyBatis mapper XML
- Fix SAXParseException causing Spring context startup failure
- ./gradlew test --offline: PASS
```

## Repository Structure and File Placement (MANDATORY)

Updated: 2026-11-21.

### Kiểm tra trước khi sửa hoặc tạo file

1. Đọc `AGENTS.md` ở gốc và các quy tắc áp dụng trong thư mục sẽ sửa. Đối chiếu `PROJECT_STRUCTURE.md`, README của module và cấu trúc thực tế; không tự tạo thư mục theo một ví dụ cũ trong tài liệu.
2. Chạy `git status --short`, dùng `rg --files` để kiểm tra module liên quan và `rg` để tìm tên endpoint, nghiệp vụ hoặc nội dung tương tự. Xác định file nguồn chính thức, các tham chiếu/import, registry và generator liên quan trước khi chỉnh sửa.
3. Ưu tiên cập nhật file hiện có. Trước khi tạo file/thư mục mới, nêu ngắn gọn đường dẫn, mục đích và vì sao file hiện có không phù hợp trong kế hoạch hoặc cập nhật công việc. Không tạo thêm một file kế hoạch chỉ để thực hiện bước này.
4. Chỉ tạo file mới khi cần cho chức năng đã được yêu cầu và đúng cấu trúc module hiện hành. Không tự mở thêm thư mục cấp gốc, service/package, bộ contract hoặc hệ thống tài liệu song song ngoài phạm vi công việc đã được giao. Nếu thật sự thiếu quyết định về nơi quản lý, làm rõ quyết định đó trước khi tạo cấu trúc mới; không hỏi lại các quyết định đã được chấp thuận.

### Quy tắc chọn nơi lưu

| Loại thay đổi | Nơi quản lý và nguyên tắc |
|---|---|
| Source code và test | Đặt trong module/service sở hữu chức năng, theo cấu trúc và cách đặt tên hiện có. Không tạo một module khác chỉ để chứa bản sửa. |
| REST API contract | Cập nhật bộ `customer-*.openapi.yaml` hoặc `admin-*.openapi.yaml` và state matrix tương ứng trong `docs/03_api_specs/`, theo lifecycle ở phần Frontend API Contract Handoff. Không tự tạo spec riêng theo service, task hoặc ngày tháng. |
| REST vận hành nội bộ | Vẫn dùng bộ contract chung phù hợp; ghi rõ phạm vi nội bộ, server và security của từng operation. Không tạo thêm spec riêng và không biến endpoint nội bộ thành route gateway công khai chỉ để khớp tài liệu. |
| Kafka / gRPC contract | Cập nhật schema/proto trong `packages/events/`, `packages/proto/` hoặc nơi sở hữu đã được xác lập. Không dùng thay cho REST contract. |
| Tài liệu | Cập nhật tài liệu hiện có về cùng nội dung trước. Chỉ tạo tài liệu riêng khi có nhu cầu độc lập rõ ràng hoặc người dùng yêu cầu; đặt trong nhánh tài liệu đang quản lý chủ đề đó. Tài liệu Catalog phải nằm trong `docs/03_api_specs/`. |
| Migration | Dùng thư mục và quy ước migration của service; kiểm tra version hiện hành để không trùng số. Không sửa migration đã apply để tránh tạo file mới. File down chỉ hoàn tác thay đổi thuộc up cùng version. |
| File generated | Dùng generator đã được xác lập; không chỉnh tay hoặc tạo bản sao để né generator. Nếu generator được nhắc trong tài liệu nhưng không tồn tại, báo rõ thay vì tự tạo một generator khác. |
| Log, output thử nghiệm, bản nháp và file tạm | Dùng thư mục tạm của hệ điều hành hoặc thư mục generated/ignored đã được dự án quy định. Không commit các file này như tài liệu nguồn chính thức; không đưa secret hoặc dump dữ liệu thật vào repository. |

Các contract/tài liệu cũ đang được registry hoặc code tham chiếu phải được giữ theo quy tắc của chúng. Không xóa hàng loạt file cũ chỉ vì có quy tắc ưu tiên file chung. Khi hợp nhất hoặc đổi đường dẫn, cập nhật mọi tham chiếu và xác nhận không còn tham chiếu hỏng trước khi bỏ file thay thế.

### Kiểm tra trước khi commit hoặc bàn giao

- Chạy `git status --short`, `git diff --name-status` và `git diff --cached --name-status`; rà cả file untracked, file thêm/xóa/đổi tên và nội dung staged, không chỉ các file đã sửa.
- Đối chiếu từng đường dẫn mới với phạm vi đã công bố. Bỏ khỏi thay đổi của mình các file tạm hoặc trùng nội dung; không xóa hay ghi đè file người khác đang làm.
- Kiểm tra import, đường dẫn trong tài liệu, registry, CI và generator khi thêm/xóa/di chuyển file. Chạy lint/contract checks phù hợp với thay đổi.
- Trong mô tả PR, liệt kê ngắn các file/thư mục mới hoặc bị xóa cùng lý do, nhất là contract, migration và tài liệu. Nếu không thêm cấu trúc mới, không cần tạo báo cáo riêng.
- Nếu thay đổi cấu trúc thực sự được chấp thuận, cập nhật mục liên quan trong `PROJECT_STRUCTURE.md`; không viết lại các phần ngoài phạm vi.

## Contract Docs
- Admin catalog API contract: `docs/03_api_specs/admin-stable.openapi.yaml`
- Admin state matrix: `docs/03_api_specs/admin-api-state-matrix.md`
- Do not create new catalog doc files outside `docs/03_api_specs/`.
- `catalog-checkout.openapi.yaml` must be kept (referenced by `packages/contracts/mvp/registry.yaml`).

## Frontend API Contract Handoff

Thư mục và file hợp đồng nằm ở `docs/03_api_specs/`. Không được tạo thêm file doc catalog ngoài thư mục này.

### Files
| File | Mô tả |
|------|-------|
| `customer-stable.openapi.yaml` | Customer API đã sẵn sàng triển khai (implementation-ready). |
| `customer-draft-planned.openapi.yaml` | Khả năng Customer đề xuất, còn chờ quyết định sản phẩm, bảo mật, lưu trữ hoặc quy trình làm việc. |
| `admin-stable.openapi.yaml` | API nhân viên đã sẵn sàng (employee session, management-read, approved workbench API). Phải được giữ lại (referenced by `packages/contracts/mvp/registry.yaml`). |
| `admin-draft-planned.openapi.yaml` | Đề xuất biến thể nhân viên chưa ổn định về chính sách (employee mutations not yet policy-stable). |
| `customer-api-state-matrix.md` | Bao phủ trạng thái bắt buộc của Customer (success/loading/empty/error/auth). |
| `admin-api-state-matrix.md` | Bao phủ trạng thái bắt buộc của Admin (success/loading/empty/error/auth). |

### Quy tắc bắt buộc: Mọi thay đổi REST API phải cập nhật contract
Bất kỳ thay đổi nào liên quan đến REST API đều phải được phản ánh vào contract trước hoặc song song với việc implement code:
- Thêm endpoint mới → phải thêm vào file `.openapi.yaml` tương ứng (stable hoặc draft-planned).
- Sửa endpoint (path, method, query parameter, request body, response schema, status code, error code) → phải cập nhật lại file `.openapi.yaml` đó.
- Xóa endpoint → phải đánh dấu xóa hoặc xóa khỏi file contract, đồng thời cập nhật state matrix.
- Thay đổi phân quyền, permission, scope, auth requirement → phải cập nhật cả `.openapi.yaml` và `.md` state matrix.
- Thay đổi trạng thái giao diện (success/loading/empty/error/auth) → phải cập nhật file `*-api-state-matrix.md` tương ứng.
- **Không được implement code REST API mà bỏ qua cập nhật contract.** Nếu thay đổi là breaking change đối với `stable`, yêu cầu thay đổi phiên bản hợp đồng (contract version change).

### Quy tắc vòng đời (Lifecycle)
- **stable**: Backend có thể triển khai ngay; thay đổi phá vỡ (breaking change) yêu cầu thay đổi phiên bản hợp đồng.
- **draft**: Hình dạng endpoint đủ cụ thể để đánh giá, nhưng ít nhất một quyết định chính sách hoặc quy trình làm việc đã đặt tên vẫn chưa được giải quyết.
- **planned**: Khả năng được yêu cầu bởi backlog, nhưng hình dạng endpoint và/hoặc quy trình làm việc vẫn có thể thay đổi lớn.
- Các file `draft-planned` không được import bởi các file `stable`.
- Việc chuyển một operation sang `stable` yêu cầu: quyết định rõ ràng, quy tắc phân quyền đã hoàn thành, ví dụ cho mọi trạng thái áp dụng, và contract tests.

### Quy tắc chung (Shared rules)
- Tất cả lỗi sử dụng Error envelope chung với stable code, safe message, và requestId.
- Empty collections trả về 200 với empty arrays và zero totals.
- 401 nghĩa là authentication là absent hoặc invalid; 403 nghĩa là identity là known nhưng permission, capability, ownership, hoặc scope là insufficient.
- Admin authorization là permission-and-scope based. Role names là assignment bundles và không được dùng làm chỉ duy nhất authorization check.
- `Idempotency-Key` là required bởi implementation policy cho retryable create/action operations even where an older client has not supplied it yet.
- Upload proposals dùng direct-to-object-storage sessions; business APIs nhận immutable fileId references, không nhận binary bodies hoặc arbitrary URLs.

### Xác thực hợp đồng (Validation)
Chạy các lệnh sau để kiểm tra:
```powershell
npx @redocly/cli lint .\docs\03_api_specs\customer-stable.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\customer-draft-planned.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\admin-stable.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\admin-draft-planned.openapi.yaml
```

### Sản phẩm tạo ra tự động (Generated artifacts)
Các sản phẩm tạo ra có thể được làm mới bằng `node scripts/generate-frontend-api-contracts.mjs` sau khi có thay đổi hợp đồng được phê duyệt. Không được chỉnh sửa generated rows hoặc inferred stable response examples nếu không cập nhật cũng theo generator.

## Catalog Service Specific Rules
See `services/catalog-service/CATALOG_SERVICE_AGENT_RULES.md`.

## Identity Service Specific Rules
See `services/identity-service/IDENTITY_SERVICE_AGENT_RULES.md`.


## Keycloak Service Specific Rules
See `services/keycloak-service/KEYCLOAK_SERVICE_AGENT_RULES.md`.
