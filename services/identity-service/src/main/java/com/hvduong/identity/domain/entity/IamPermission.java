package com.hvduong.identity.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Domain entity representing an IAM permission.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class IamPermission {
    private UUID permissionId;
    private String permissionCode;
    private String description;
    private String resourceType;
    private String actionType;
    private OffsetDateTime createdAt;
}
