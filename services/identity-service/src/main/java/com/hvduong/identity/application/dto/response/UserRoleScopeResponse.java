package com.hvduong.identity.application.dto.response;

import com.fasterxml.jackson.annotation.JsonInclude;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class UserRoleScopeResponse {
    private UUID assignmentId;
    private UUID roleId;
    private String roleCode;
    private String roleName;
    private String scopeType;
    private String scopeId;
    private String assignmentStatus;
    private OffsetDateTime validFrom;
    private OffsetDateTime validUntil;
    private UUID assignedByPrincipalId;
    private String reason;
    private OffsetDateTime createdAt;
}
