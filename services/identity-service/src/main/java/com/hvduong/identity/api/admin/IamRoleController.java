package com.hvduong.identity.api.admin;

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
import java.util.NoSuchElementException;
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
    public ResponseEntity<List<IamRoleRow>> getAllRoles() {
        return ResponseEntity.ok(roleManagementService.getAllRoles());
    }

    // 2. GET /api/v1/admin/roles/{roleId} — Chi tiết Role
    @GetMapping("/{roleId}")
    public ResponseEntity<RoleManagementService.RoleDetailDto> getRoleById(@PathVariable UUID roleId) {
        try {
            return ResponseEntity.ok(roleManagementService.getRoleById(roleId));
        } catch (NoSuchElementException ex) {
            return ResponseEntity.notFound().build();
        }
    }

    // 3. POST /api/v1/admin/roles — Tạo Role
    @PostMapping
    public ResponseEntity<?> createRole(
            @RequestBody RoleManagementService.CreateRoleDto request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        try {
            UUID roleId = roleManagementService.createRole(request, actorId);
            return ResponseEntity.status(HttpStatus.CREATED).body(roleId);
        } catch (IllegalArgumentException ex) {
            return ResponseEntity.badRequest().body(ex.getMessage());
        }
    }

    // 4. PATCH /api/v1/admin/roles/{roleId} — Cập nhật tên, mô tả hoặc isEnabled
    @PatchMapping("/{roleId}")
    public ResponseEntity<?> updateRole(
            @PathVariable UUID roleId,
            @RequestBody RoleManagementService.UpdateRoleDto request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        try {
            roleManagementService.updateRole(roleId, request, actorId);
            return ResponseEntity.noContent().build();
        } catch (NoSuchElementException ex) {
            return ResponseEntity.notFound().build();
        } catch (IllegalStateException ex) {
            return ResponseEntity.status(HttpStatus.CONFLICT).body(ex.getMessage());
        }
    }

    // 5. DELETE /api/v1/admin/roles/{roleId} — Xóa Role
    @DeleteMapping("/{roleId}")
    public ResponseEntity<?> deleteRole(
            @PathVariable UUID roleId,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        try {
            roleManagementService.deleteRole(roleId, actorId);
            return ResponseEntity.noContent().build();
        } catch (NoSuchElementException ex) {
            return ResponseEntity.notFound().build();
        } catch (IllegalStateException ex) {
            return ResponseEntity.status(HttpStatus.CONFLICT).body(ex.getMessage());
        }
    }

    // 6. GET /api/v1/admin/roles/{roleId}/permissions — Xem Permission của Role
    @GetMapping("/{roleId}/permissions")
    public ResponseEntity<List<String>> getRolePermissions(@PathVariable UUID roleId) {
        try {
            return ResponseEntity.ok(roleManagementService.getPermissionsForRole(roleId));
        } catch (NoSuchElementException ex) {
            return ResponseEntity.notFound().build();
        }
    }

    // 7. PUT /api/v1/admin/roles/{roleId}/permissions — Thay thế toàn bộ danh sách Permission
    @PutMapping("/{roleId}/permissions")
    public ResponseEntity<?> replacePermissions(
            @PathVariable UUID roleId,
            @RequestBody ReplacePermissionsRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        try {
            List<String> permissions = request != null ? request.getPermissions() : List.of();
            roleManagementService.replaceRolePermissions(roleId, permissions, actorId);
            return ResponseEntity.noContent().build();
        } catch (NoSuchElementException ex) {
            return ResponseEntity.notFound().build();
        }
    }

    // 8. POST /api/v1/admin/roles/{roleId}/permissions — Thêm một Permission vào Role
    @PostMapping("/{roleId}/permissions")
    public ResponseEntity<?> addPermission(
            @PathVariable UUID roleId,
            @RequestBody AddPermissionRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        if (request == null || request.getPermissionCode() == null) {
            return ResponseEntity.badRequest().body("permissionCode is required");
        }
        try {
            roleManagementService.addPermissionToRole(roleId, request.getPermissionCode(), actorId);
            return ResponseEntity.status(HttpStatus.CREATED).build();
        } catch (NoSuchElementException ex) {
            return ResponseEntity.notFound().build();
        }
    }

    // 9. DELETE /api/v1/admin/roles/{roleId}/permissions/{permissionCode} — Gỡ một Permission khỏi Role
    @DeleteMapping("/{roleId}/permissions/{permissionCode}")
    public ResponseEntity<?> removePermission(
            @PathVariable UUID roleId,
            @PathVariable String permissionCode,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        try {
            roleManagementService.removePermissionFromRole(roleId, permissionCode, actorId);
            return ResponseEntity.noContent().build();
        } catch (NoSuchElementException ex) {
            return ResponseEntity.notFound().build();
        }
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
