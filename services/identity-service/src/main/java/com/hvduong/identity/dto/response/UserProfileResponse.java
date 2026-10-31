package com.hvduong.identity.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UserProfileResponse {
    private UUID principalId;
    private String idpSubject;
    private String email;
    private String principalType;
    private String provisioningStatus;
    private List<UserRoleScopeResponse> roles;
    private List<String> permissions;
}

