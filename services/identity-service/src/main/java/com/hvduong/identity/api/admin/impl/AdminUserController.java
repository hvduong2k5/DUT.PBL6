package com.hvduong.identity.api.admin.impl;

import com.hvduong.identity.api.admin.AdminUserApi;
import com.hvduong.identity.application.account.UserLifecycleService;
import com.hvduong.identity.common.response.ApiResponse;
import com.hvduong.identity.common.response.PageResponse;
import com.hvduong.identity.application.dto.request.ResetPasswordRequest;
import com.hvduong.identity.application.dto.response.AdminUserDetailResponse;
import com.hvduong.identity.application.dto.response.AdminUserItemResponse;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.UUID;

@Slf4j
@RestController
@RequiredArgsConstructor
public class AdminUserController implements AdminUserApi {

    private final UserLifecycleService userLifecycleService;

    @Override
    @PreAuthorize("hasAuthority('USER_VIEW')")
    public ResponseEntity<ApiResponse<PageResponse<AdminUserItemResponse>>> listUsers(
            @RequestParam(value = "type", required = false) String type,
            @RequestParam(value = "status", required = false) String status,
            @RequestParam(value = "page", defaultValue = "0") int page,
            @RequestParam(value = "size", defaultValue = "20") int size,
            @AuthenticationPrincipal Jwt jwt,
            String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        PageResponse<AdminUserItemResponse> result = userLifecycleService.listUsers(type, status, page, size, actorId);
        return ResponseEntity.ok(ApiResponse.success(result));
    }

    @Override
    @PreAuthorize("hasAuthority('USER_VIEW')")
    public ResponseEntity<ApiResponse<AdminUserDetailResponse>> getUserDetail(
            @PathVariable UUID principalId,
            @AuthenticationPrincipal Jwt jwt,
            String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        AdminUserDetailResponse detail = userLifecycleService.getUserDetail(principalId, actorId);
        return ResponseEntity.ok(ApiResponse.success(detail));
    }

    @Override
    @PreAuthorize("hasAuthority('USER_PASSWORD_RESET')")
    public ResponseEntity<ApiResponse<Void>> resetUserPassword(
            @PathVariable UUID principalId,
            @Valid @RequestBody ResetPasswordRequest request,
            @AuthenticationPrincipal Jwt jwt,
            String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        userLifecycleService.adminResetPassword(principalId, request.getNewPassword(), request.isTemporary(), actorId);
        return ResponseEntity.ok(ApiResponse.success("Đặt lại mật khẩu người dùng thành công", null));
    }

    @Override
    @PreAuthorize("hasAuthority('USER_SESSION_REVOKE')")
    public ResponseEntity<ApiResponse<Void>> revokeUserSessions(
            @PathVariable UUID principalId,
            @AuthenticationPrincipal Jwt jwt,
            String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        userLifecycleService.revokeUserSessions(principalId, actorId);
        return ResponseEntity.ok(ApiResponse.success("Đã thu hồi toàn bộ phiên đăng nhập của người dùng", null));
    }

    private UUID extractActorId(Jwt jwt, String actorHeader) {
        if (jwt == null) return null;
        try { return UUID.fromString(jwt.getClaimAsString("user_id")); }
        catch (IllegalArgumentException | NullPointerException error) { return null; }
    }
}
