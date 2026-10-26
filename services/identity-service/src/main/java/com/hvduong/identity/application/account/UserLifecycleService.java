package com.hvduong.identity.application.account;

import lombok.Builder;
import lombok.Data;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

public interface UserLifecycleService {

    @Data
    @Builder
    class UserRoleScopeDto {
        private String roleCode;
        private String roleName;
        private String scopeType;
        private String scopeId;
    }

    @Data
    @Builder
    class UserProfileDto {
        private UUID principalId;
        private String idpSubject;
        private String email;
        private String principalType;
        private String provisioningStatus;
        private List<UserRoleScopeDto> roles;
        private List<String> permissions;
    }

    Optional<UserProfileDto> getProfileBySubject(String idpSubject, String email);

    void suspendAccount(UUID principalId, String reason, UUID actorPrincipalId);

    void reactivateAccount(UUID principalId, UUID actorPrincipalId);
}
