package com.hvduong.identity.infrastructure.persistence.entity;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class IamRolePermissionRow {
    private UUID roleId;
    private String permissionCode;
    private OffsetDateTime createdAt;
}
