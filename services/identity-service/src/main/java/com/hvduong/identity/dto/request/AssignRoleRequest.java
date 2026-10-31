package com.hvduong.identity.dto.request;

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
public class AssignRoleRequest {
    private UUID principalId;
    private UUID roleId;
    private String scopeType;
    private String scopeId;
    private OffsetDateTime validFrom;
    private OffsetDateTime validUntil;
    private String reason;
}

