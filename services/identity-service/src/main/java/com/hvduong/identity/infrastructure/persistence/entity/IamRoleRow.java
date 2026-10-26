package com.hvduong.identity.infrastructure.persistence.entity;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class IamRoleRow {
    private UUID roleId;
    private String roleCode;
    private String roleName;
    private String description;
    private Boolean isProtected;
    private Boolean isEnabled;
    private Long version;
    private UUID createdByPrincipalId;
    private UUID updatedByPrincipalId;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
}
