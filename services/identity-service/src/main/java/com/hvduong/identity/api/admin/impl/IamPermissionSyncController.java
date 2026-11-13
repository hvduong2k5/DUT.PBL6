package com.hvduong.identity.api.admin.impl;

import java.util.List;
import java.util.UUID;

import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

import com.hvduong.identity.api.admin.IamPermissionSyncApi;
import com.hvduong.identity.application.dto.response.EffectivePermissionsResponse;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.common.exception.ErrorCode;
import com.hvduong.identity.common.exception.IdentityException;
import com.hvduong.identity.common.response.ApiResponse;
import com.hvduong.identity.security.VerifiedActor;

import lombok.RequiredArgsConstructor;

@RestController
@RequiredArgsConstructor
public class IamPermissionSyncController implements IamPermissionSyncApi {

    static final String TOKEN_NOTE =
            "Quyền này là cấu hình hiện tại trong IAM. Token đã phát không thể sửa; "
                    + "thay đổi chỉ áp dụng cho token phát sau lần refresh kế tiếp.";

    private final PermissionSyncService permissionSyncService;

    @Override
    @PreAuthorize("hasAuthority('ROLE_VIEW')")
    public ResponseEntity<ApiResponse<EffectivePermissionsResponse>> getEffectivePermissions(
            @PathVariable UUID principalId) {
        List<String> permissions = permissionSyncService.resolveEffectivePermissions(principalId);
        EffectivePermissionsResponse body = EffectivePermissionsResponse.builder()
                .principalId(principalId)
                .permissions(permissions)
                .tokenNote(TOKEN_NOTE)
                .build();
        return ResponseEntity.ok(ApiResponse.success(body));
    }

    @Override
    @PreAuthorize("hasAuthority('PERMISSION_MANAGE')")
    public ResponseEntity<ApiResponse<PermissionSyncService.SyncResult>> syncPermissions(
            @PathVariable UUID principalId,
            boolean forceRelogin) {
        UUID actorId = VerifiedActor.id();
        PermissionSyncService.SyncResult result = permissionSyncService.syncPrincipal(
                principalId, forceRelogin, PermissionSyncService.Trigger.MANUAL_RESYNC, actorId);

        if (!result.success()) {
            throw new IdentityException(ErrorCode.PERMISSION_SYNC_FAILED,
                    "Keycloak sync failed for principal " + principalId + ": " + result.errorCode());
        }
        return ResponseEntity.ok(ApiResponse.success("Đồng bộ quyền sang Keycloak thành công", result));
    }
}
