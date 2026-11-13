package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.domain.entity.*;

import org.apache.ibatis.annotations.Param;
import java.util.UUID;
import java.time.OffsetDateTime;
import java.util.List;

public interface IamRoleAssignmentMapper {
    void insert(IamRoleAssignmentRow row);

    IamRoleAssignmentRow findById(@Param("assignmentId") UUID assignmentId);

    List<IamRoleAssignmentRow> findActiveByPrincipal(@Param("principalId") UUID principalId, @Param("now") OffsetDateTime now);

    int revokeAssignment(
        @Param("assignmentId") UUID assignmentId,
        @Param("revokedBy") UUID revokedBy,
        @Param("reason") String reason,
        @Param("expectedVersion") Long expectedVersion
    );

    int updateAssignment(
        @Param("assignmentId") UUID assignmentId,
        @Param("validUntil") OffsetDateTime validUntil,
        @Param("clearValidUntil") Boolean clearValidUntil,
        @Param("reason") String reason,
        @Param("expectedVersion") Long expectedVersion
    );

    /** Count currently active assignments for a role (used before soft delete). */
    int countActiveByRoleId(@Param("roleId") UUID roleId, @Param("now") OffsetDateTime now);

    int countPendingByRoleId(@Param("roleId") UUID roleId, @Param("now") OffsetDateTime now);

    /** Find all currently active assignments for a role (used to sync/revoke sessions when role permissions change). */
    List<IamRoleAssignmentRow> findActiveByRoleId(@Param("roleId") UUID roleId, @Param("now") OffsetDateTime now);

    /** Count ALL assignments ever made for a role (active + revoked + expired). Used to decide hard vs soft delete. */
    int countAllByRoleId(@Param("roleId") UUID roleId);

    /**
     * Revoke all ACTIVE assignments for a role in bulk (called on role soft-delete).
     * Sets assignment_status = REVOKED for all active assignments where role_id matches.
     */
    int revokeAllByRoleId(
        @Param("roleId") UUID roleId,
        @Param("revokedBy") UUID revokedBy,
        @Param("reason") String reason
    );

    /**
     * List assignments with optional filters. All params nullable (null = no filter).
     */
    List<IamRoleAssignmentRow> listAssignments(
        @Param("principalId") UUID principalId,
        @Param("roleId") UUID roleId,
        @Param("roleCode") String roleCode,
        @Param("scopeType") String scopeType,
        @Param("scopeId") String scopeId,
        @Param("status") String status,
        @Param("offset") int offset,
        @Param("limit") int limit
    );

    /** Count active ADMIN-role assignments across all principals (for last-admin protection). */
    int countActiveAdmins(@Param("adminRoleCode") String adminRoleCode, @Param("now") OffsetDateTime now);
}
