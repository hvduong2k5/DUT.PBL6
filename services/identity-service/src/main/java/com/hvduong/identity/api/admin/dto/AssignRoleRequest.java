package com.hvduong.identity.api.admin.dto;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class AssignRoleRequest {
    private UUID principalId;
    private UUID roleId;
    private String scopeType;
    private String scopeId;
    private OffsetDateTime validFrom;
    private OffsetDateTime validUntil;
    private String reason;
}
