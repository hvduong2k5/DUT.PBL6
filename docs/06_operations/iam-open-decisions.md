# Sổ quyết định còn mở và chênh lệch implementation IAM

**Ngày tổng hợp:** 07/11/2026. Đọc [hồ sơ kiến thức và nguồn](iam-knowledge-audit.md) trước khi dùng bảng này.

## 1. Cách dùng và thẩm quyền

- Đây là danh sách để đối chiếu/chốt, không phải phê duyệt mở quyền hoặc bật API.
- 54 ID dưới đây giữ nguyên vấn đề và trạng thái từ [Access Management Policy Decisions](../ui_web/ACCESS_MANAGEMENT_POLICY_DECISIONS.md). Không đổi một mục thành đã chốt chỉ vì implementation đã chọn một phương án.
- Cột quan sát là kết quả rà code/docs tại snapshot ghi trong hồ sơ. Không phải exploit report hoặc chứng nhận production.
- Owner được nêu theo vai trò cần tham gia: PO/BA, Identity, domain owner, Frontend, Security/Audit, Platform/DBA. Chưa có người nhận việc hay ngày phê duyệt nếu nguồn chưa ghi.
- Các nhóm K và AUD bên dưới liên quan/đè lên một số ID gốc; không cộng chúng thành tổng số quyết định độc lập.

## 2. 54 mục policy gốc và điều cần đối chiếu

| ID | Vấn đề từ nguồn | Trạng thái nguồn, giữ nguyên | Quan sát cần xử lý trước khi đóng mục |
|---|---|---|---|
| `AUTH-01` | Quan hệ giữa Role và Permission | **Đã chốt** | Schema tách roles, permissions và role_permissions. Phù hợp nguyên tắc đã chốt. |
| `AUTH-02` | Cấp Permission trực tiếp cho Employee | **Tạm áp dụng** — Direct Permission không có trong backlog | Không thấy mô hình Direct Permission hoàn chỉnh; không mở editor cấp quyền trực tiếp. |
| `AUTH-03` | Một Employee có nhiều Role | **Chưa chốt** — Epic có scenario nhiều Role nhưng Mục 8 vẫn đặt câu hỏi | DB và /users/me xử lý nhiều assignments; cần giải quyết mâu thuẫn scenario với mục 8 Epic trước khi coi là policy sản phẩm. |
| `AUTH-04` | Hợp nhất quyền từ nhiều Role | **Tạm áp dụng** | Backend có union; #71 giữ cặp permission–scope. Quy tắc overlap/merchant/assigned-only toàn hệ thống chưa thống nhất. |
| `AUTH-05` | Role inheritance | **Chưa chốt** | Không có role hierarchy trong model đã rà; tên manager/admin không tự tạo quan hệ kế thừa. |
| `AUTH-06` | Explicit Deny | **Chưa chốt** | Grant model hiện là allow; chưa có deny precedence. Không thêm deny bằng suy đoán UI. |
| `AUTH-07` | Taxonomy scope | **Chưa chốt** | V2 có enum scope; Inventory MVP một kho, Profile có department. Cần chốt loại scope được phép cho từng resource/action. |
| `AUTH-08` | Permission và điều kiện nghiệp vụ động | **Đã chốt bởi Epic 22** | Order/Care/Fulfillment có invariant và permit/assignee guards. Không biến IAM permission thành business approval. |
| `ACC-01` | Nguồn tạo Account | **Chưa chốt** | Invitation service tạo principal PENDING; chưa ràng buộc bắt buộc có Employee Profile trước. Cần chọn flow và xử lý orphan. |
| `ACC-02` | Định danh đăng nhập duy nhất | **Chưa chốt** | OIDC identity gắn issuer/sub + principal UUID. Định danh nhập khi mời/đăng nhập và uniqueness vẫn cần contract chung. |
| `ACC-03` | Trường dữ liệu được sửa tại Epic 22 | **Đã chốt bởi Epic 22** | Profile và System Access tách owner; không sửa hồ sơ HR trong Access Management. |
| `ACC-04` | Luồng lời mời và kích hoạt | **Chưa chốt** | Có invitation create/accept/revoke/expiry; chưa chứng minh kênh gửi, one-time proof và toàn bộ onboarding đã nghiệm thu. |
| `ACC-05` | MFA và reset credential | **Chưa chốt** | Keycloak owner credential/MFA; Finance có check MFA riêng. Chưa có policy áp dụng đồng đều cho IAM và HR mutations. |
| `ACC-06` | Bộ trạng thái Account | **Tạm áp dụng** | DB principal dùng PENDING/ACTIVE/FAILED/SUSPENDED, khác enum Account UI. Cần bảng ánh xạ chính thức. |
| `ACC-07` | Mở khóa/reactivate | **Chưa chốt** | Có thao tác lifecycle ở Identity; semantics unlock/reactivate, ai được làm và audit chưa được đồng nhất với UI. |
| `ACC-08` | Khóa ngay hoặc theo lịch | **Đã chốt về kết quả; chưa chốt timezone/lịch chạy** | Không dùng endpoint suspend/reactivate hiện có làm bằng chứng đã có scheduled lock và timezone policy. |
| `ACC-09` | Phiên đang hoạt động khi khóa/thu hồi quyền | **Chưa chốt** | Fresh grants ở #71 khác JWT authorities ở Catalog/Identity và mapping ADMIN Fulfillment; cần SLA revoke xuyên các đường này. |
| `ACC-10` | Bảo vệ Admin hợp lệ cuối cùng | **Mục tiêu đã chốt; cơ chế chưa chốt** | Role protected được chặn disable/delete/change permissions; chưa tương đương bảo vệ người Admin hợp lệ cuối cùng. |
| `ACC-11` | HR Manager | **Chưa chốt** | Epic 22 vẫn thiếu Actor ID/thẩm quyền HR Manager. Quyền hồ sơ HR không tự cấp quyền cấp/khóa tài khoản. |
| `ACC-12` | Cập nhật đồng thời | **Kết quả chống ghi đè đã chốt; cơ chế chưa chốt** | UpdateRoleRequest không nhận expectedVersion; service đọc version hiện hành rồi update. Cần test stale form phía client, không chỉ race DB. |
| `ROLE-01` | Có cho tạo Custom Role | **Chưa chốt** | Identity đã có POST create custom role, trong khi decision register còn mở và UI read-only. |
| `ROLE-02` | Danh sách Role hệ thống/đặc quyền | **Chưa chốt** | V7 seed ADMIN protected. Chưa có registry role nghiệp vụ mẫu/privileged đầy đủ được xác nhận. |
| `ROLE-03` | Tính bất biến của `roleCode` | **Chưa chốt** | Update DTO không sửa roleCode; bất biến đang là lựa chọn code, cần khớp contract/migration policy. |
| `ROLE-04` | Quy tắc mã/tên duy nhất | **Chưa chốt** | Code create trim/uppercase, DB unique role_code; chưa suy ra uniqueness tên, tenant hoặc scope đã được chốt. |
| `ROLE-05` | Sửa Role đặc quyền | **Chưa chốt** | Protected role có guard nhưng chưa thấy workflow approval/break-glass đầy đủ. Cần actor ladder và self-approval policy. |
| `ROLE-06` | Xóa Role có lịch sử | **Đã chốt bởi Epic 22** | Mâu thuẫn rõ: RoleManagementService + mapper có DELETE vật lý. FK history có thể chặn; không phải archive lifecycle. |
| `ROLE-07` | Role inactive đang còn assignment | **Chưa chốt** | #71 bỏ disabled role khỏi grants; JWT attribute/token paths không có cùng semantics revoke. Cần impact và resync. |
| `ROLE-08` | Clone Role | **Chưa chốt** | Không coi copy permission list bằng create role là clone feature đã được duyệt. |
| `ROLE-09` | Thay tập Permission của Role | **Kết quả impact/Audit đã chốt; quy trình chưa chốt** | Có replace/add/remove permissions và audit; chưa đủ impact preview, approval, stale-client guard và propagation xuyên service. |
| `PERM-01` | Taxonomy mã Permission | **Chưa chốt** | UPPER_SNAKE_CASE và payment.*/fulfillment.* cùng tồn tại. Cần registry mã, owner, action và scope, tránh alias làm rộng quyền. |
| `PERM-02` | Custom Permission | **Chưa chốt** | Có mapper insert permission nhưng API quan sát chủ yếu quản lý permission của role. Không suy ra có custom permission CRUD end to end. |
| `PERM-03` | Tính bất biến của `permissionCode` | **Chưa chốt** | Seed dùng code làm key; nhánh Order cũ và #71 có cùng code nhưng metadata khác. Cần quy tắc deprecation/forward migration. |
| `PERM-04` | Permission đã được module thực thi | **Mục tiêu bảo vệ đã chốt; danh sách code chưa chốt** | Protected ADMIN không được đổi permissions qua các method đã rà; chưa phải registry bảo vệ mọi permission hệ thống. |
| `PERM-05` | Vòng đời Permission | **Chưa chốt** | Permission DB chưa có trạng thái lifecycle như UI ACTIVE/DEPRECATED/ARCHIVED; không map ngầm. |
| `PERM-06` | Archive Permission đang thuộc Role | **Chưa chốt** | Chưa có archive workflow, dependency migration và replacement policy được chứng minh. |
| `PERM-07` | Thu hồi một nguồn quyền trùng | **Đã chốt bởi Epic 22** | Nhiều nguồn cấp cần giữ provenance; revoke một assignment không được xóa hiệu lực từ assignment còn hợp lệ. |
| `PERM-08` | Quyền xem và quyền mutation | **Đã chốt bởi Epic 22** | VIEW không suy ra mutation/sensitive/export; BFF và backend phải dùng cùng resource/action meaning. |
| `ASSIGN-01` | Ma trận ai được gán Role nào | **Chưa chốt** | ROLE_ASSIGN/ADMIN hiện là guard coarse; assignRole chưa kiểm đầy đủ role/scope ceiling của người gán. |
| `ASSIGN-02` | Self-assignment và self-approval | **Chưa chốt** | Chưa tìm thấy bộ guard self-assignment/self-approval thống nhất; cần negative tests theo role đặc quyền. |
| `ASSIGN-03` | Separation of Duties | **Chưa chốt** | Schema/assign path đã rà chưa có registry tổ hợp role xung đột; cần owner Finance/HR/Operations xác nhận. |
| `ASSIGN-04` | Approval quyền đặc biệt | **Chưa chốt** | Không coi approval nghiệp vụ Care/refund là approval cấp role IAM; cần workflow riêng nếu sản phẩm yêu cầu. |
| `ASSIGN-05` | Assignment có thời hạn | **Khả năng có thời hạn đã chốt; chi tiết chưa chốt** | valid_from/until và query hiệu lực đã có; thời hạn tối đa/bắt buộc expiry theo loại quyền còn mở. |
| `ASSIGN-06` | Timezone và thời điểm hiệu lực | **Chưa chốt** | Dùng OffsetDateTime và DB timestamptz. Lệch clock host/container đã ảnh hưởng fixture; cần clock sync/SLA và cách hiển thị. |
| `ASSIGN-07` | Gán Role cho Account locked | **Đã nêu trong Epic 22, cần xác nhận sản phẩm** | Assign path kiểm EMPLOYEE, không bắt buộc principal ACTIVE; hiệu lực truy cập vẫn phụ thuộc persona/status. Cần xác nhận configured access cho locked/pending. |
| `ASSIGN-08` | Break-glass | **Chưa chốt** | Chưa có break-glass workflow được chứng minh. Không dùng ADMIN bypass làm substitute. |
| `ASSIGN-09` | Lý do bắt buộc | **Chưa chốt** | Một số endpoint có reason; chưa có reason catalog/validation áp dụng đồng đều cho tất cả mutation. |
| `GOV-01` | Nội dung Audit tối thiểu | **Đã chốt ở mức nguyên tắc** | Có AuditService/local audit rows; cần đối chiếu before/after/actor/outcome/correlation với catalog từng action. |
| `GOV-02` | Ghi thao tác denied/failed | **Chưa chốt** | Không coi HTTP access log là audit denied/failed; chưa có catalog thống nhất action nhạy cảm/read reveal. |
| `GOV-03` | Retention và legal hold | **Chưa chốt** | Retention/legal hold/disposal còn mở; không tự chọn một năm/vĩnh viễn trong code hoặc đề xuất này. |
| `GOV-04` | Chu kỳ Access Review | **Chưa chốt** | Access Review popup read-only không phải campaign định kỳ hay bằng chứng tái phê duyệt quyền. |
| `GOV-05` | Owner và bằng chứng review | **Chưa chốt** | Cần chỉ định reviewer/owner dữ liệu, bằng chứng chấp thuận và trách nhiệm thu hồi. |
| `GOV-06` | Export Access Review | **Chưa chốt** | AUDIT_LOG_EXPORT seed không chứng minh có export Access Review/Audit, ACL file, expiry hoặc download audit. |
| `GOV-07` | Dữ liệu Employee được hiển thị | **Nguyên tắc đã chốt; field cụ thể chưa chốt** | PII gates #71 và projection BFF là lựa chọn ở từng lát cắt; field/purpose/reveal audit toàn hệ thống chưa chốt. |
| `GOV-08` | Notification | **Chưa chốt** | Outbox đăng ký Customer đã có; không coi đó là notification invite/revoke/expiry cho mọi Employee mutation. |

## 3. Hai mươi vấn đề cần owner giải quyết từ đối chiếu liên nguồn

**P0** là việc cần giải quyết trước khi mở mutation/rollout hoặc merge phần liên quan; không có nghĩa toàn bộ luồng mua hàng hiện tại phải dừng. **P1** cần chốt để hoàn thiện vận hành và contract. Những phương án MVP dưới đây là lựa chọn để thảo luận, không phải quyết định tự động áp dụng.

| ID / ưu tiên | Câu hỏi cần quyết định / bằng chứng còn thiếu | Owner cần tham gia | Điều kiện đóng / tác động |
|---|---|---|---|
| K01 / P0 | Nguồn nào có quyền phê duyệt cuối: Epic/UI read-only hay Identity contract ghi CHỐT mutation? Ai ký các thay đổi về quyền? | PO/BA + Identity + Frontend | ADR/decision có phạm vi, người duyệt, ngày và tài liệu superseded; chưa coi backend CRUD là phê duyệt UI mutation. |
| K02 / P0 | MVP một kho hay đa kho; HR toàn doanh nghiệp hay chia department? Có merchant/tenant/branch boundary nào thực sự áp dụng? | PO + Order/Inventory/Profile | Ma trận actor × resource × action × scope; sửa mâu thuẫn HLD đa kho với MVP single pool. Không ép HR theo department chỉ vì có FK. |
| K03 / P0 | Ai là authority quyền ở request time, revoke SLA bao lâu; chọn JWT, fresh grants, cache hay hybrid theo action? | Identity + domain owners + Platform/Security | SLO latency/availability, TTL/grace/invalidation, high-risk fresh checks, outage semantics và test revoke xuyên service. Hai giây là timeout HTTP, không phải chi phí cố định. |
| K04 / P0 | Ai được gán/thu hồi role nào, trần scope/quyền của người gán, last Admin, self-grant, SoD, approval? | PO + Security + Identity + Finance | Bộ chính sách và test negative trước mở editor/privileged mutations. Protected role chỉ là một guard; không thay last-active-admin workflow. |
| K05 / P0 | Role/permission/account dùng archive/inactive hay DELETE? Xử lý lịch sử và FK thế nào? | Identity + Audit + DBA + PO | Sửa mâu thuẫn ROLE-06 với DELETE SQL; migration/retention/error contract rõ. Không quảng bá hard delete là archive. |
| K06 / P1 | Client gửi expectedVersion/ETag ở đâu, lỗi stale là gì, idempotency và impact preview áp dụng action nào? | Identity + Frontend | DTO/OpenAPI/mapper thống nhất và test stale form sau người khác đã commit; version đọc mới ở server chưa đủ. |
| K07 / P0 khi nối UI thật | EMPLOYEE_ACCOUNT_VIEW/ACCESS_REVIEW_VIEW so với USER_VIEW là quyền khác hay alias? Route admin/access và roleCode/roleId map thế nào? | Identity + Frontend + Profile + PO | API projection stable, field ownership, cùng permission meaning; không alias rộng quyền để làm UI hoạt động. |
| K08 / P0 | Scope enum/type/ID hợp lệ theo permission nào, nguồn danh mục kho/phòng ban ở đâu, GLOBAL/ALL được định nghĩa thế nào? | Identity + domain owners | Validation/preflight từ nguồn thật; thông báo sai cấu hình an toàn, audit lý do từ chối; SQL scope_required không tự xác minh ID. |
| K09 / P0 | Resync Keycloak user attributes/realm roles khi role disabled, đổi permissions, assignment future/expired hoặc sync thất bại bằng cách nào? | Identity + Platform | Reconciliation/outbox hoặc cơ chế tương đương, observability và acceptance token cũ/token mới. TTL ngắn không sửa nguồn attributes stale. |
| K10 / P1 | UI account/assignment/role/permission states map provisioning/status DB ra sao; scheduled/expired là stored hay computed? | PO + Identity + Frontend | State machine và transition/error mapping có version; configured access khác actual access; không map FAILED thành LOCKED theo suy đoán. |
| K11 / P0 cho high-risk | Specialized policy chọn đúng scope/actor bằng gì, caller gRPC được xác thực ở đâu, context ai chứng thực, approval/evidence gắn resource thế nào? | Identity + Security/Platform + Finance/Order/B2B | E2E authenticated caller → scoped decision → domain enforcement; test khác actor/scope, nhiều policies và timeout. Chưa thấy đủ caller/interceptor/deploy evidence để coi production-ready. |
| K12 / P0 theo rollout | Chuyển Catalog/Identity JWT ADMIN bypass và Fulfillment ADMIN mapping sang permission model nào? Inventory/Shipping staff API nào sẽ mở? | Các service owners | Mỗi service có endpoint × permission × object scope × business guards × test matrix; giữ credential nội bộ riêng. Không gắn nhãn cả hệ thống đã phân quyền. |
| K13 / P0 theo exposure | Identity của workload, token rotation, TLS, per-method ACL, trusted headers và delegated actor được bảo vệ thế nào? | Platform + Security + service owners | Network/API tests chống giả header/caller; hạn chế token read/write; không coi URL internal/admin tự bảo đảm an toàn. |
| K14 / P0 cho dữ liệu nhạy cảm | PII nào cần purpose/MFA/approval/reveal audit; quyền xem video khác metadata/upload ra sao; revoke presigned URL có cần tức thời? | PO + Profile/Order/Fulfillment/Care + Security | Field/purpose matrix, masking, audit, signed URL TTL hoặc proxy theo yêu cầu. Không tự biến risk_level HIGH thành MFA đã enforce. |
| K15 / P1 | Role nghiệp vụ mẫu nào được chấp thuận, scope ID dev lấy ở đâu; ai nghiệm thu theo quyền tối thiểu? | PO + domain owners + Identity + QA | Acceptance actor thật/role tối thiểu đúng scope và denial; CI ADMIN fixture không phải xác nhận ma trận nghiệp vụ. |
| K16 / P0 trước merge nhánh cũ | Giữ gì từ codex/order-admin-integration; V8 Order trùng V8 Catalog và permission metadata khác V9 xử lý sao? | Order + Identity + Frontend + DBA | Port UI/BFF/DTO có chọn lọc; một auth path, một migration ledger. Xem Flyway history nếu migration cũ đã chạy; không sửa tùy tiện file đã áp dụng. |
| K17 / P1 | HR Profile trước Account hay đồng thời; HR được mời/khóa đến đâu; email/activation proof/MFA/offboarding thế nào? | PO + Profile + Identity + Notification | Flow onboarding/offboarding có actor, proof, expiry, failure recovery; xác nhận HR actor ID và không khóa tài khoản Customer cá nhân theo Employee. |
| K18 / P0 cho mutation nhạy cảm | Audit bắt buộc action nào, before/after ra sao, lưu lỗi denied/reveal thế nào; ghi audit lỗi thì action nào fail closed? | Audit/Security + Identity + domain owners | Producer catalog, schema/outbox, denied/failure tests, integrity/retention/export design theo EPIC 23. Seed AUDIT_LOG_* không phải Audit API đã có. |
| K19 / P1 trước mở rộng Analytics | Giữ pilot allowlist/ADMIN GLOBAL bao lâu; Supply/Finance/Auditor xem dataset nào; dùng quyền Analytics hay Inventory/BATCH? | Analytics + PO + Identity + Supply/Finance | Ma trận dataset/action/scope riêng và nghiệm thu pilot; auth nhánh Analytics chưa merge không được tính vào coverage dev. |
| K20 / P1 | Registry permission chuẩn UPPER_SNAKE/dotted, version/owner/deprecate, custom permission, aliases, risk level và scope taxonomy do ai quản lý? | Identity + service owners + PO/Security | Registry có endpoint enforcement và migration consumers; không đồng nhất các mã gần nghĩa nếu quyền dữ liệu khác nhau. |

## 4. Mười bốn câu hỏi Audit chưa được chốt ở EPIC 23 mục 8

Giữ riêng vì ranh giới Governance không chỉ là một bảng identity_audit_log. Nguồn: [EPIC 23](../01_requirements/epics/EPIC_23_Audit_Security.md). Các dòng này giao với GOV-01~08/K18; không phải 14 yêu cầu mới do bản rà soát tự thêm.

1. Catalog action bắt buộc Audit theo từng Epic; có ghi denied/failed/read access không và mức chi tiết cho mỗi loại.
2. Schema chuẩn, versioning, định danh actor/object khi nguồn bị xóa logic, correlation và quy tắc occurred/recorded time.
3. Trường before/after bắt buộc, cách diff collection/quan hệ, giới hạn payload và xử lý media/tài liệu lớn.
4. Danh sách secret/PII/financial fields cấm ghi hoặc phải mask; quyền xem field nhạy cảm và quy trình redaction khi producer gửi nhầm.
5. Source fields được coi là đáng tin cậy, cách xử lý proxy/multiple hop, device/location và thời hạn lưu dữ liệu network.
6. Action nào bắt buộc reason, validation/độ dài/danh mục lý do và quy tắc annotation/case điều tra.
7. Permission matrix cho search/detail/sensitive fields/export/annotation/verify và phạm vi module/đơn vị/kho/kênh.
8. Retention chính thức theo loại event thay cho ví dụ “1 năm hoặc vĩnh viễn”; archive, legal hold, disposal và bằng chứng đã hủy theo chính sách.
9. Cơ chế tamper-evident cụ thể, đơn vị/segment của continuity, key management, rotation, backup/restore và verification schedule.
10. Quy tắc với late/duplicate/out-of-order event để không tạo integrity alert giả và vẫn nhận biết event thật sự bị thiếu.
11. Chính sách fail-open/fail-closed khi không ghi được Audit cho từng nhóm action nhạy cảm; retry/dead-letter/reconciliation.
12. SLA cảnh báo, dedup/escalation, owner Incident Response và quy trình điều tra/khắc phục sau khi integrity invalid.
13. Export format, giới hạn thời gian/kích thước, phê duyệt, watermark, expiry, storage và audit việc tải/chia sẻ.
14. Mục tiêu hiệu năng truy vấn/ghi/verify, giới hạn filter/range và cách tránh Audit làm chậm giao dịch nghiệp vụ.

## 5. Trình tự thảo luận và bằng chứng phê duyệt

1. **Phạm vi và authority:** K01/K02/K03, AUTH-03/04/07, ACC-09/11. Chốt single-pool MVP, phạm vi HR và semantics quyền runtime trước khi chọn role mẫu.
2. **Control plane đặc quyền:** K04/K08/K09, ASSIGN-01~04/08, ACC-10. Chốt ai được cấp quyền cho ai và cơ chế thu hồi thực tế.
3. **Contract/lifecycle:** K05/K06/K07/K10/K20, ROLE/PERM. Đồng bộ DTO/UI/API/migration, archive và concurrency.
4. **Theo service:** K11/K12/K13/K14/K16/K19. Giữ business admission/approval guards, chuyển từng route theo test cùng lúc.
5. **Rollout và governance:** K15/K17/K18, GOV/Audit. Nghiệm thu ít quyền, failure paths, role templates, recovery và người chịu trách nhiệm.

Mẫu ghi quyết định sau thảo luận:

| Trường | Nội dung phải ghi |
|---|---|
| ID và nguồn | AUTH/ACC/... hoặc K/AUD tương ứng; tài liệu nào được thay thế |
| Owner và người duyệt | Cá nhân/vai trò có thẩm quyền; ngày duyệt; còn phụ thuộc ai |
| Hành vi được phép | Actor, principal type, resource/action, scope loại/ID, hiệu lực/thời hạn |
| Hành vi bị từ chối | Thiếu quyền, sai scope, không ACTIVE, self-grant, stale input và trạng thái không hợp lệ |
| Nhất quán/quyền đã phát | JWT/cache/revoke deadline, request đang chạy, capability URL đã phát |
| Vận hành | SLO, outage/fail-closed, audit/PII, clock/timezone, sync/reconciliation |
| Contract/test | Mã permission/route/DTO/version và positive/negative acceptance |
| Migration/rollout | Dev/review/live, nguồn ID thật, backward compatibility và kế hoạch khôi phục |

Cho đến khi có bằng chứng duyệt, giữ nhãn “chưa chốt/tạm áp dụng” của nguồn. Không dùng danh sách này làm lệnh tự động seed permission, gán GLOBAL hoặc bật UI mutation.

## JWT clock skew

F-05 sets the Keycloak JWT verification tolerance to 30 seconds for Identity, Catalog, Order, Profile, Fulfillment, Inventory, Analytics, and the Web OIDC BFF. Keep service clocks synchronized. Access-token lifetime and refresh timing are independent of this tolerance.

## Limited grant verification

A local Keycloak and Identity test gave identical role-list access before
and after PermissionSync wrote the same three global grants. The original
limited-grant denial was reproduced with the same signed token when the
matching ACTIVE EMPLOYEE record was absent from Identity. Check issuer,
subject, provisioning status and internal principal ID alongside the
grant list when investigating an environment that still denies access.
