package com.hvduong.identity.api.admin;

import java.util.UUID;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;

import com.hvduong.identity.application.dto.response.EffectivePermissionsResponse;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.common.response.ApiResponse;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;

@Tag(name = "IAM Permission Sync", description = "Đối soát và chạy lại đồng bộ quyền Identity → Keycloak cho một principal")
@RequestMapping("/api/v1/admin/iam/principals/{principalId}")
public interface IamPermissionSyncApi {

    @Operation(
            summary = "Xem quyền đang cấu hình của một principal",
            description = "Tính lại quyền hiệu lực từ cơ sở dữ liệu IAM (nguồn sự thật). "
                    + "Đây là 'quyền đang cấu hình', không phải 'quyền token đang mang': token đã phát "
                    + "không thể sửa và chỉ nhận thay đổi ở lần refresh kế tiếp."
    )
    @SecurityRequirement(name = "bearerAuth")
    @GetMapping("/effective-permissions")
    ResponseEntity<ApiResponse<EffectivePermissionsResponse>> getEffectivePermissions(
            @Parameter(description = "ID nội bộ của principal", required = true)
            @PathVariable UUID principalId
    );

    @Operation(
            summary = "Chạy lại đồng bộ quyền Identity → Keycloak",
            description = "Đường đối soát khi lần đồng bộ trước thất bại (được ghi audit với action "
                    + "IAM_PERMISSION_SYNC_FAILED). Đặt forceRelogin=true để thu hồi phiên Keycloak và "
                    + "xóa phiên BFF sau khi đồng bộ."
    )
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/sync-permissions")
    ResponseEntity<ApiResponse<PermissionSyncService.SyncResult>> syncPermissions(
            @Parameter(description = "ID nội bộ của principal", required = true)
            @PathVariable UUID principalId,
            @RequestParam(required = false, defaultValue = "false") boolean forceRelogin
    );
}
