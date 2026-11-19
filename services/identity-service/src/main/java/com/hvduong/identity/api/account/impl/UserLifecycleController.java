package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.api.account.UserLifecycleApi;
import com.hvduong.identity.application.account.UserLifecycleService;
import com.hvduong.identity.common.response.ApiResponse;
import com.hvduong.identity.application.dto.request.UpdateUserStatusRequest;
import com.hvduong.identity.application.dto.response.UserProfileResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RestController;

import java.util.UUID;

@RestController
@RequiredArgsConstructor
public class UserLifecycleController implements UserLifecycleApi {

    private final UserLifecycleService userLifecycleService;

    @Override
    public ResponseEntity<ApiResponse<UserProfileResponse>> getCurrentUser(@AuthenticationPrincipal Jwt jwt) {
        if (jwt == null) {
            return ResponseEntity.status(HttpStatus.UNAUTHORIZED)
                    .body(ApiResponse.error("UNAUTHORIZED", "Yêu cầu đăng nhập để truy cập"));
        }

        String subject = jwt.getSubject();
        String email = jwt.getClaimAsString("email");

        return userLifecycleService.getProfileBySubject(jwt.getIssuer().toString(), subject, email)
                .map(profile -> {
                    if (!"ACTIVE".equals(profile.getProvisioningStatus())
                            || !profile.getPrincipalId().toString().equals(jwt.getClaimAsString("user_id"))) {
                        return ResponseEntity.status(HttpStatus.FORBIDDEN)
                                .body(ApiResponse.<UserProfileResponse>error("PRINCIPAL_FORBIDDEN", "Tài khoản chưa được phép truy cập"));
                    }
                    return ResponseEntity.ok(ApiResponse.success(profile));
                })
                .orElseGet(() -> ResponseEntity.status(HttpStatus.NOT_FOUND)
                        .body(ApiResponse.error("USER_NOT_FOUND", "Không tìm thấy thông tin tài khoản")));
    }

    @Override
    @PreAuthorize("hasAuthority('USER_STATUS_MANAGE')")
    public ResponseEntity<ApiResponse<Void>> updateUserStatus(
            @PathVariable UUID principalId,
            @RequestBody UpdateUserStatusRequest request,
            @AuthenticationPrincipal Jwt jwt
    ) {
        UUID actorId = null;
        if (jwt != null) {
            try {
                actorId = UUID.fromString(jwt.getClaimAsString("user_id"));
            } catch (Exception ignored) {}
        }

        String status = request != null && request.getStatus() != null ? request.getStatus().trim().toUpperCase() : "SUSPENDED";
        String reason = request != null && request.getReason() != null ? request.getReason() : "Status updated by admin";

        if ("ACTIVE".equals(status)) {
            userLifecycleService.reactivateAccount(principalId, actorId);
            return ResponseEntity.ok(ApiResponse.success("Kích hoạt lại tài khoản thành công", null));
        } else {
            userLifecycleService.suspendAccount(principalId, reason, actorId);
            return ResponseEntity.ok(ApiResponse.success("Khóa tài khoản người dùng thành công", null));
        }
    }

    @Override
    public ResponseEntity<ApiResponse<Void>> changeOwnPassword(
            @jakarta.validation.Valid @RequestBody com.hvduong.identity.application.dto.request.ChangePasswordRequest request,
            @AuthenticationPrincipal Jwt jwt,
            String actorHeader
    ) {
        UUID actorId = extractActorId(jwt, actorHeader);
        if (actorId == null) {
            return ResponseEntity.status(HttpStatus.UNAUTHORIZED)
                    .body(ApiResponse.error("UNAUTHORIZED", "Yêu cầu đăng nhập để đổi mật khẩu"));
        }

        userLifecycleService.changeOwnPassword(actorId, request.getCurrentPassword(), request.getNewPassword());
        return ResponseEntity.ok(ApiResponse.success("Đổi mật khẩu thành công", null));
    }

    private UUID extractActorId(Jwt jwt, String actorHeader) {
        if (jwt == null) return null;
        try { return UUID.fromString(jwt.getClaimAsString("user_id")); }
        catch (IllegalArgumentException | NullPointerException error) { return null; }
    }
}
