package com.hvduong.identity.domain.entity;

import com.fasterxml.jackson.annotation.JsonIgnore;
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
    @JsonIgnore
    private OffsetDateTime archivedAt;
    private Long version;
    private UUID createdByPrincipalId;
    private UUID updatedByPrincipalId;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
}
