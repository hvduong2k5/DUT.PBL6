package com.hvduong.identity.api.admin;

import java.util.List;
import java.util.UUID;

import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;

import com.hvduong.identity.common.response.ApiResponse;
import com.hvduong.identity.application.dto.request.AddPermissionRequest;
import com.hvduong.identity.application.dto.request.CreateRoleRequest;
import com.hvduong.identity.application.dto.request.ReplacePermissionsRequest;
import com.hvduong.identity.application.dto.request.UpdateRoleRequest;
import com.hvduong.identity.application.dto.response.RoleDetailResponse;
import com.hvduong.identity.domain.entity.IamRoleRow;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;

@Tag(name = "IAM Roles & Permissions", description = "Quản lý danh mục Vai trò và Quyền hạn phân cấp (RBAC) theo chuẩn RESTful (Mục 3.3.1)")
@RequestMapping("/api/v1/admin/roles")
public interface IamRoleApi {

    @Operation(summary = "Lấy danh sách tất cả vai trò")
    @SecurityRequirement(name = "bearerAuth")
    @GetMapping
    ResponseEntity<ApiResponse<List<IamRoleRow>>> getAllRoles();

    @Operation(summary = "Xem chi tiết một vai trò kèm danh sách quyền")
    @SecurityRequirement(name = "bearerAuth")
    @GetMapping("/{roleId}")
    ResponseEntity<ApiResponse<RoleDetailResponse>> getRoleById(
            @Parameter(description = "ID vai trò (UUID)", required = true) @PathVariable UUID roleId
    );

    @Operation(summary = "Tạo mới một vai trò", description = "Đồng bộ tạo Realm Role trên Keycloak và lưu vào DB iam_roles.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping
    ResponseEntity<ApiResponse<UUID>> createRole(
            @RequestBody CreateRoleRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Cập nhật thông tin vai trò", description = "Cập nhật tên, mô tả hoặc bật/tắt (chặn sửa vai trò hệ thống).")
    @SecurityRequirement(name = "bearerAuth")
    @PatchMapping("/{roleId}")
    ResponseEntity<ApiResponse<Void>> updateRole(
            @Parameter(description = "ID vai trò (UUID)", required = true) @PathVariable UUID roleId,
            @RequestBody UpdateRoleRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Xóa vai trò", description = "Xóa vai trò khỏi Keycloak và DB (chặn xóa nếu có user đang được gán).")
    @SecurityRequirement(name = "bearerAuth")
    @DeleteMapping("/{roleId}")
    ResponseEntity<ApiResponse<Void>> deleteRole(
            @Parameter(description = "ID vai trò (UUID)", required = true) @PathVariable UUID roleId,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Xem danh sách mã quyền của vai trò")
    @SecurityRequirement(name = "bearerAuth")
    @GetMapping("/{roleId}/permissions")
    ResponseEntity<ApiResponse<List<String>>> getRolePermissions(
            @Parameter(description = "ID vai trò (UUID)", required = true) @PathVariable UUID roleId
    );

    @Operation(summary = "Thay thế toàn bộ quyền của vai trò (Batch Update)")
    @SecurityRequirement(name = "bearerAuth")
    @PutMapping("/{roleId}/permissions")
    ResponseEntity<ApiResponse<Void>> replacePermissions(
            @Parameter(description = "ID vai trò (UUID)", required = true) @PathVariable UUID roleId,
            @RequestBody ReplacePermissionsRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Gán thêm một quyền vào vai trò")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/{roleId}/permissions")
    ResponseEntity<ApiResponse<Void>> addPermission(
            @Parameter(description = "ID vai trò (UUID)", required = true) @PathVariable UUID roleId,
            @RequestBody AddPermissionRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Gỡ một quyền khỏi vai trò")
    @SecurityRequirement(name = "bearerAuth")
    @DeleteMapping("/{roleId}/permissions/{permissionCode}")
    ResponseEntity<ApiResponse<Void>> removePermission(
            @Parameter(description = "ID vai trò (UUID)", required = true) @PathVariable UUID roleId,
            @Parameter(description = "Mã quyền hạn", required = true) @PathVariable String permissionCode,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );
}
