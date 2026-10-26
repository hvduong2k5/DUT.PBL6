package com.hvduong.identity.api.admin;

import com.hvduong.identity.api.common.ApiResponse;
import com.hvduong.identity.application.iam.RoleManagementService;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleRow;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping({"/api/v1/admin/roles", "/api/v1/admin/iam/roles"})
@RequiredArgsConstructor
public class IamRoleController {

    private final RoleManagementService roleManagementService;

    @Data
    public static class AddPermissionRequest {
        private String permissionCode;
    }

    @Data
    public static class ReplacePermissionsRequest {
        private List<String> permissions;
    }

    // 1. GET /api/v1/admin/roles — Danh sách Role
    @GetMapping
    public ResponseEntity<ApiResponse<List<IamRoleRow>>> getAllRoles() {
        List<IamRoleRow> roles = roleManagementService.getAllRoles();
        return ResponseEntity.ok(ApiResponse.success(roles));
    }

    // 2. GET /api/v1/admin/roles/{roleId} — Chi tiết Role
    @GetMapping("/{roleId}")
    public ResponseEntity<ApiResponse<RoleManagementService.RoleDetailDto>> getRoleById(@PathVariable UUID roleId) {
        RoleManagementService.RoleDetailDto role = roleManagementService.getRoleById(roleId);
        return ResponseEntity.ok(ApiResponse.success(role));
    }

    // 3. POST /api/v1/admin/roles — Tạo Role
    @PostMapping
    public ResponseEntity<ApiResponse<UUID>> createRole(
            @RequestBody RoleManagementService.CreateRoleDto request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        UUID roleId = roleManagementService.createRole(request, actorId);
        return ResponseEntity.status(HttpStatus.CREATED)
                .body(ApiResponse.success("Tạo vai trò thành công", roleId));
    }

    // 4. PATCH /api/v1/admin/roles/{roleId} — Cập nhật tên, mô tả hoặc isEnabled
    @PatchMapping("/{roleId}")
    public ResponseEntity<ApiResponse<Void>> updateRole(
            @PathVariable UUID roleId,
            @RequestBody RoleManagementService.UpdateRoleDto request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        roleManagementService.updateRole(roleId, request, actorId);
        return ResponseEntity.ok(ApiResponse.success("Cập nhật thông tin vai trò thành công", null));
    }

    // 5. DELETE /api/v1/admin/roles/{roleId} — Xóa Role
    @DeleteMapping("/{roleId}")
    public ResponseEntity<ApiResponse<Void>> deleteRole(
            @PathVariable UUID roleId,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        roleManagementService.deleteRole(roleId, actorId);
        return ResponseEntity.ok(ApiResponse.success("Xóa vai trò thành công", null));
    }

    // 6. GET /api/v1/admin/roles/{roleId}/permissions — Xem Permission của Role
    @GetMapping("/{roleId}/permissions")
    public ResponseEntity<ApiResponse<List<String>>> getRolePermissions(@PathVariable UUID roleId) {
        List<String> permissions = roleManagementService.getPermissionsForRole(roleId);
        return ResponseEntity.ok(ApiResponse.success(permissions));
    }

    // 7. PUT /api/v1/admin/roles/{roleId}/permissions — Thay thế toàn bộ danh sách Permission
    @PutMapping("/{roleId}/permissions")
    public ResponseEntity<ApiResponse<Void>> replacePermissions(
            @PathVariable UUID roleId,
            @RequestBody ReplacePermissionsRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        List<String> permissions = request != null && request.getPermissions() != null ? request.getPermissions() : List.of();
        roleManagementService.replaceRolePermissions(roleId, permissions, actorId);
        return ResponseEntity.ok(ApiResponse.success("Cập nhật danh sách quyền cho vai trò thành công", null));
    }

    // 8. POST /api/v1/admin/roles/{roleId}/permissions — Thêm một Permission vào Role
    @PostMapping("/{roleId}/permissions")
    public ResponseEntity<ApiResponse<Void>> addPermission(
            @PathVariable UUID roleId,
            @RequestBody AddPermissionRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        if (request == null || request.getPermissionCode() == null || request.getPermissionCode().isBlank()) {
            throw new IllegalArgumentException("permissionCode là bắt buộc");
        }
        UUID actorId = extractActorId(jwt, actorHeader);
        roleManagementService.addPermissionToRole(roleId, request.getPermissionCode(), actorId);
        return ResponseEntity.status(HttpStatus.CREATED)
                .body(ApiResponse.success("Gán quyền vào vai trò thành công", null));
    }

    // 9. DELETE /api/v1/admin/roles/{roleId}/permissions/{permissionCode} — Gỡ một Permission khỏi Role
    @DeleteMapping("/{roleId}/permissions/{permissionCode}")
    public ResponseEntity<ApiResponse<Void>> removePermission(
            @PathVariable UUID roleId,
            @PathVariable String permissionCode,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        roleManagementService.removePermissionFromRole(roleId, permissionCode, actorId);
        return ResponseEntity.ok(ApiResponse.success("Gỡ quyền khỏi vai trò thành công", null));
    }

    private UUID extractActorId(Jwt jwt, String actorHeader) {
        if (jwt != null) {
            try {
                return UUID.fromString(jwt.getSubject());
            } catch (Exception ignored) {}
        }
        if (actorHeader != null) {
            try {
                return UUID.fromString(actorHeader);
            } catch (Exception ignored) {}
        }
        return null;
    }
}
