package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.application.account.EmployeeInvitationService;
import com.hvduong.identity.common.exception.ErrorCode;
import com.hvduong.identity.common.exception.IdentityException;
import com.hvduong.identity.common.response.ApiResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.UUID;

@RestController
@RequiredArgsConstructor
public class EmployeeInvitationAcceptanceController {

    public static final String ACCEPT_PATH = "/api/v1/public/auth/employee-invitations/{invitationId}/accept";

    private final EmployeeInvitationService invitationService;

    @PostMapping(ACCEPT_PATH)
    public ApiResponse<Void> accept(@PathVariable UUID invitationId, @AuthenticationPrincipal Jwt jwt) {
        if (jwt == null || jwt.getIssuer() == null || jwt.getSubject() == null || jwt.getSubject().isBlank()) {
            throw new IdentityException(ErrorCode.UNAUTHENTICATED);
        }
        invitationService.acceptInvitation(invitationId, jwt.getIssuer().toString(), jwt.getSubject());
        return ApiResponse.success("Đã chấp nhận lời mời nhân viên", null);
    }
}
