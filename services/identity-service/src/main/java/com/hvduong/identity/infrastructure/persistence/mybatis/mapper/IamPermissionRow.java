package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.time.OffsetDateTime;

@Data
public class IamPermissionRow {
    private String permissionCode;
    private String resourceCode;
    private String actionCode;
    private String riskLevel;
    private Boolean scopeRequired;
    private String description;
    private OffsetDateTime createdAt;
}
