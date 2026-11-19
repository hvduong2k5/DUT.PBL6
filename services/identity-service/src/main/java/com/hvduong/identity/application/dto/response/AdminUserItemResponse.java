package com.hvduong.identity.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.List;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminUserItemResponse {
    private UUID principalId;
    private String idpSubject;
    private String email;
    private String firstName;
    private String lastName;
    private String principalType;
    private String provisioningStatus;
    private OffsetDateTime createdAt;
    private List<String> roles;
}

