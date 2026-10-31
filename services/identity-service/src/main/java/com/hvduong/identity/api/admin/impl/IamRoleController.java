package com.hvduong.identity.api.admin.impl;

import java.util.List;
import java.util.UUID;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

import com.hvduong.identity.api.admin.IamRoleApi;
import com.hvduong.identity.application.iam.RoleManagementService;
import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.AddPermissionRequest;
import com.hvduong.identity.dto.request.CreateRoleRequest;
import com.hvduong.identity.dto.request.ReplacePermissionsRequest;
import com.hvduong.identity.dto.request.UpdateRoleRequest;
import com.hvduong.identity.dto.response.RoleDetailResponse;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleRow;

import lombok.RequiredArgsConstructor;

@RestController
@RequiredArgsConstructor
public class IamRoleController implements IamRoleApi {

    private final RoleManagementService roleManagementService;

    @Override
    public ResponseEntity<ApiResponse<List<IamRoleRow>>> getAllRoles() {
        List<IamRoleRow> roles = roleManagementService.getAllRoles();
        return ResponseEntity.ok(ApiResponse.success(roles));
    }

    @Override
    public ResponseEntity<ApiResponse<RoleDetailResponse>> getRoleById(@PathVariable UUID roleId) {
        RoleDetailResponse role = roleManagementService.getRoleById(roleId);
        return ResponseEntity.ok(ApiResponse.success(role));
    }

    @Override
    public ResponseEntity<ApiResponse<UUID>> createRole(
            @RequestBody CreateRoleRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        UUID roleId = roleManagementService.createRole(request, actorId);
        return ResponseEntity.status(HttpStatus.CREATED)
                .body(ApiResponse.success("Tạo vai trò thành công", roleId));
    }

    @Override
    public ResponseEntity<ApiResponse<Void>> updateRole(
            @PathVariable UUID roleId,
            @RequestBody UpdateRoleRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        roleManagementService.updateRole(roleId, request, actorId);
        return ResponseEntity.ok(ApiResponse.success("Cập nhật thông tin vai trò thành công", null));
    }

    @Override
    public ResponseEntity<ApiResponse<Void>> deleteRole(
            @PathVariable UUID roleId,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        roleManagementService.deleteRole(roleId, actorId);
        return ResponseEntity.ok(ApiResponse.success("Xóa vai trò thành công", null));
    }

    @Override
    public ResponseEntity<ApiResponse<List<String>>> getRolePermissions(@PathVariable UUID roleId) {
        List<String> permissions = roleManagementService.getPermissionsForRole(roleId);
        return ResponseEntity.ok(ApiResponse.success(permissions));
    }

    @Override
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

    @Override
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

    @Override
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
