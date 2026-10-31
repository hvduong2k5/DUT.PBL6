package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.api.account.UserLifecycleApi;
import com.hvduong.identity.application.account.UserLifecycleService;
import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.SuspendRequest;
import com.hvduong.identity.dto.request.UpdateUserStatusRequest;
import com.hvduong.identity.dto.response.UserProfileResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
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

        return userLifecycleService.getProfileBySubject(subject, email)
                .map(profile -> ResponseEntity.ok(ApiResponse.success(profile)))
                .orElseGet(() -> ResponseEntity.status(HttpStatus.NOT_FOUND)
                        .body(ApiResponse.error("USER_NOT_FOUND", "Không tìm thấy thông tin tài khoản")));
    }

    @Override
    public ResponseEntity<ApiResponse<Void>> updateUserStatus(
            @PathVariable UUID principalId,
            @RequestBody UpdateUserStatusRequest request,
            @AuthenticationPrincipal Jwt jwt
    ) {
        UUID actorId = null;
        if (jwt != null) {
            try {
                actorId = UUID.fromString(jwt.getSubject());
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
    public ResponseEntity<ApiResponse<Void>> suspendUser(
            @PathVariable UUID principalId,
            @RequestBody(required = false) SuspendRequest request,
            @AuthenticationPrincipal Jwt jwt
    ) {
        UUID actorId = null;
        if (jwt != null) {
            try {
                actorId = UUID.fromString(jwt.getSubject());
            } catch (Exception ignored) {}
        }

        String reason = request != null && request.getReason() != null ? request.getReason() : "Account suspended by admin via legacy API";
        userLifecycleService.suspendAccount(principalId, reason, actorId);
        return ResponseEntity.ok(ApiResponse.success("Khóa tài khoản người dùng thành công", null));
    }

    @Override
    public ResponseEntity<ApiResponse<Void>> reactivateUser(
            @PathVariable UUID principalId,
            @AuthenticationPrincipal Jwt jwt
    ) {
        UUID actorId = null;
        if (jwt != null) {
            try {
                actorId = UUID.fromString(jwt.getSubject());
            } catch (Exception ignored) {}
        }

        userLifecycleService.reactivateAccount(principalId, actorId);
        return ResponseEntity.ok(ApiResponse.success("Kích hoạt lại tài khoản thành công", null));
    }
}
