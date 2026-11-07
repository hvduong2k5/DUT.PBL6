package com.hvduong.identity.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Domain entity representing an IAM role.
 * Soft-deleted roles (isEnabled=false) cannot be reactivated.
 * Active role_code uniqueness is enforced by partial index (V9 migration).
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class IamRole {
    private UUID roleId;
    private String roleCode;
    private String roleName;
    private String description;
    private boolean isEnabled;
    private boolean isProtected;
    private int version;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
}
