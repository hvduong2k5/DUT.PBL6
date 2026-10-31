package com.hvduong.identity.application.account;

import com.hvduong.identity.dto.response.UserProfileResponse;
import com.hvduong.identity.dto.response.UserRoleScopeResponse;

import java.util.Optional;
import java.util.UUID;

public interface UserLifecycleService {

    /**
     * @deprecated Sử dụng {@link UserRoleScopeResponse}
     */
    @Deprecated
    class UserRoleScopeDto extends UserRoleScopeResponse {}

    /**
     * @deprecated Sử dụng {@link UserProfileResponse}
     */
    @Deprecated
    class UserProfileDto extends UserProfileResponse {}

    Optional<UserProfileResponse> getProfileBySubject(String idpSubject, String email);

    void suspendAccount(UUID principalId, String reason, UUID actorPrincipalId);

    void reactivateAccount(UUID principalId, UUID actorPrincipalId);
}
