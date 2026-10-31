package com.hvduong.identity.application.account;

import com.hvduong.identity.dto.common.PageResponse;
import com.hvduong.identity.dto.response.AdminUserDetailResponse;
import com.hvduong.identity.dto.response.AdminUserItemResponse;
import com.hvduong.identity.dto.response.UserProfileResponse;

import java.util.Optional;
import java.util.UUID;

public interface UserLifecycleService {
    Optional<UserProfileResponse> getProfileBySubject(String idpSubject, String email);

    void suspendAccount(UUID principalId, String reason, UUID actorPrincipalId);

    void reactivateAccount(UUID principalId, UUID actorPrincipalId);

    PageResponse<AdminUserItemResponse> listUsers(String type, String status, int page, int size, UUID actorId);

    AdminUserDetailResponse getUserDetail(UUID principalId, UUID actorId);

    void adminResetPassword(UUID principalId, String newPassword, boolean temporary, UUID actorId);

    void revokeUserSessions(UUID principalId, UUID actorId);

    void changeOwnPassword(UUID actorId, String currentPassword, String newPassword);
}
