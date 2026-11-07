package com.hvduong.identity.dto.response;

import com.fasterxml.jackson.annotation.JsonInclude;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Response DTO for a role assignment record (GET /api/v1/admin/role-assignments).
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class RoleAssignmentResponse {
    private UUID assignmentId;
    private UUID principalId;
    private UUID roleId;
    private String roleCode;
    private String scopeType;
    private String scopeId;
    private String assignmentStatus;
    private UUID assignedByPrincipalId;
    private String reason;
    private OffsetDateTime validFrom;
    private OffsetDateTime validUntil;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
    private long version;
}
