package com.hvduong.identity.application.account;

import com.hvduong.identity.common.response.PageResponse;
import com.hvduong.identity.application.dto.response.AdminUserDetailResponse;
import com.hvduong.identity.application.dto.response.AdminUserItemResponse;
import com.hvduong.identity.application.dto.response.UserProfileResponse;

import java.util.Optional;
import java.util.UUID;

public interface UserLifecycleService {
    Optional<UserProfileResponse> getProfileBySubject(String idpIssuer, String idpSubject, String email);

    void suspendAccount(UUID principalId, String reason, UUID actorPrincipalId);

    void reactivateAccount(UUID principalId, UUID actorPrincipalId);

    default PageResponse<AdminUserItemResponse> listUsers(String type, String status, int page, int size, UUID actorId) {
        return listUsers(type, status, null, null, null, null, null, page, size, actorId);
    }

    PageResponse<AdminUserItemResponse> listUsers(
            String type,
            String status,
            UUID roleId,
            String roleCode,
            String scopeType,
            String scopeId,
            String search,
            int page,
            int size,
            UUID actorId
    );

    AdminUserDetailResponse getUserDetail(UUID principalId, UUID actorId);

    void adminResetPassword(UUID principalId, String newPassword, boolean temporary, UUID actorId);

    void revokeUserSessions(UUID principalId, UUID actorId);

    void changeOwnPassword(UUID actorId, String currentPassword, String newPassword);
}
