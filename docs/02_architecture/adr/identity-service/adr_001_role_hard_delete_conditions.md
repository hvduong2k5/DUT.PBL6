# ADR-001 — Điều kiện xóa vật lý (hard delete) một IAM role

- Ngày: 11/11/2026
- Trạng thái: Accepted — implemented trong `RoleManagementServiceImpl.deleteRole`.
- Scope: Identity Service, vòng đời IAM role (baseline 7-B mục 4-B/4-A: role đã dùng → disable rồi archive; role tạo nhầm/chưa dùng → được xóa vật lý, điều kiện cụ thể trước đây chưa chốt).

## Bối cảnh và gap

Baseline đã chốt hướng vòng đời nhưng chưa chốt điều kiện xóa vật lý. Code hiện tại (trước ADR này) có ba gap:

1. Role đang disabled bị `deleteRole` coi như "không tồn tại" (404). Luồng tạo-nhầm phổ biến nhất — tạo role, disable, rồi mới phát hiện nhầm — đi vào ngõ cụt: role kẹt ở trạng thái disabled vĩnh viễn, không xóa cứng được nữa.
2. Assignment hẹn trong tương lai (`assignment_status = 'ACTIVE'`, `valid_from > now`) không bị chặn: role chỉ có assignment tương lai rơi vào nhánh xóa mềm, và thao tác đó âm thầm thu hồi cả assignment chưa có hiệu lực.
3. `KeycloakAdminClient.deleteRealmRole` nuốt mọi exception. Xóa role trong PostgreSQL có thể commit thành công trong khi role phía Keycloak vẫn còn (mồ côi), không ai biết.

Ngoài ra có một ràng buộc tích hợp đã kiểm chứng trong code: Identity **không lưu Keycloak role ID**; mọi thao tác phía Keycloak định danh role theo **tên/mã**. Hệ quả: việc tái dùng một role code chỉ an toàn khi thao tác xóa phía Keycloak còn đồng bộ trong cùng request. Nếu sau này chuyển sang xóa bất đồng bộ (outbox), một sự kiện xóa đến trễ định danh theo tên có thể xóa nhầm role mới được tạo lại cùng mã.

## Quyết định

`DELETE role` áp dụng bảng quy tắc sau (thứ tự đúng như code):

| Tình huống | Kết quả |
|---|---|
| Role không tồn tại | 404 (`NoSuchElementException`) |
| Role protected (hệ thống) | Từ chối — 409 |
| Còn assignment đang hiệu lực | Từ chối — 409 |
| Còn assignment hẹn trong tương lai | Từ chối — 409 |
| Chưa từng có assignment nào (`countAllByRoleId == 0`) | **Xóa vật lý**, kể cả khi role đang disabled |
| Đã disabled và có lịch sử assignment | Từ chối — 409 (trạng thái cuối đã đạt; không xóa mềm lần hai) |
| Đang enabled và có lịch sử assignment | Chỉ xóa mềm (disable + thu hồi + audit) |

Nguyên tắc đi kèm:

- `is_enabled` (role còn phân quyền được không) và tư cách được xóa (role đã từng được dùng chưa) là hai chiều độc lập; không gộp disabled vào "không tồn tại".
- Tồn tại bản ghi audit **không** chặn xóa vật lý. Thay vào đó, audit phải tự chứa snapshot (role UUID, roleCode tại thời điểm, actor, hành động) và không bao giờ bị xóa cascade theo role — hiện trạng code đã thỏa (audit ghi target dạng String + `roleCode` trong details).
- Xóa phía Keycloak giữ **đồng bộ trong transaction xóa** và từ nay **ném lỗi khi thất bại** để transaction rollback, thay vì nuốt lỗi. Đây là điều kiện để tái dùng role code an toàn ở thời điểm hiện tại.
- Cổng "đã từng dùng" hiện tính theo **tổng số assignment từng tồn tại** (kể cả assignment bị hủy trước khi có hiệu lực). Đây là lựa chọn bảo thủ có chủ đích: nó giữ lại dấu vết thay vì xóa, và chỉ nới lỏng khi có nhu cầu thực tế kèm kiểm chứng riêng.

Mã lỗi giữ theo handler hiện có: `NoSuchElementException` → 404 `RESOURCE_NOT_FOUND`, `IllegalStateException` → 409 `STATE_CONFLICT`.

## Phương án không chọn

- **Không bao giờ xóa vật lý** (chỉ logical delete cho mọi role): an toàn mặc định, nhưng thêm trạng thái vòng đời thứ ba chỉ để "DB sạch" là phức tạp không cần thiết ở quy mô MVP; role chưa từng dùng thì chưa từng có quyết định phân quyền nào đi qua nó, xóa nó không viết lại lịch sử.
- **Outbox bất đồng bộ cho xóa Keycloak ngay bây giờ**: đúng hướng dài hạn khi có thêm consumer, nhưng hiện tại xóa đồng bộ + ném lỗi đã khép được rủi ro mồ côi mà không cần thêm worker/reconciliation. Điều kiện kích hoạt xem lại: khi xóa chuyển sang bất đồng bộ, **bắt buộc** hoặc (a) lưu Keycloak role ID và xóa theo ID bất biến, hoặc (b) giữ role code ở trạng thái chờ tới khi xác nhận đã xóa phía Keycloak — không được tái dùng code ngay.
- **Thêm điều kiện "không có audit nào tham chiếu role" vào cổng xóa vật lý**: nếu cả hành động tạo role cũng được audit thì điều kiện này khiến xóa vật lý không bao giờ tới được; vấn đề đúng nằm ở snapshot của audit (đã thỏa), không nằm ở việc audit tồn tại.

## Hệ quả và phần còn mở

- Admin xóa được role tạo nhầm ngay cả khi đã lỡ disable trước đó; assignment tương lai không còn bị âm thầm thu hồi bởi thao tác xóa role.
- Keycloak sập giữa lúc xóa cứng → toàn bộ thao tác rollback, role còn nguyên ở cả hai phía; đổi lại, xóa role phụ thuộc Keycloak khả dụng (chấp nhận được với thao tác quản trị tần suất thấp).
- Còn mở: (1) chuyển đổi mô hình xóa bất đồng bộ + định danh theo Keycloak role ID khi scale; (2) nới cổng "đã từng dùng" từ tổng-số-assignment sang chỉ-assignment-từng-hiệu-lực nếu nghiệp vụ cần.

## Kiểm chứng

- Unit tests `RoleManagementServiceTest`: protected, active assignments, hard delete khi không có lịch sử, soft delete khi có lịch sử (giữ nguyên), cộng ba ca mới — disabled + chưa từng gán → xóa vật lý; còn assignment tương lai → từ chối; đã disabled + có lịch sử → từ chối.
- Đối chiếu code trước khi chốt: `countActiveByRoleId` chỉ đếm assignment đang hiệu lực tại `now`; `deleteRealmRole` trước ADR này nuốt exception (đã sửa); Identity không có cột/field nào lưu Keycloak role ID.
