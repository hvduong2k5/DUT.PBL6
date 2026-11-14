package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.domain.entity.*;

import org.apache.ibatis.annotations.Param;
import java.util.List;
import java.util.UUID;

public interface IamRoleMapper {
    void insert(IamRoleRow row);

    /** Returns an enabled role. */
    IamRoleRow findById(@Param("roleId") UUID roleId);

    /** Returns the role while it is unarchived. */
    IamRoleRow findUnarchivedById(@Param("roleId") UUID roleId);

    /**
     * Loads any lifecycle state and takes a row lock (SELECT ... FOR UPDATE) held
     * until the caller's transaction ends. Used to serialize the last-admin
     * check in revokeRole: concurrent revokes of the same role queue on this
     * row, so the admin count they read cannot go stale mid-check.
     */
    IamRoleRow findByIdForUpdate(@Param("roleId") UUID roleId);

    /** Returns an unarchived role by code. */
    IamRoleRow findByCode(@Param("roleCode") String roleCode);

    /** Returns all unarchived roles, ordered by creation time. */
    List<IamRoleRow> findAll();

    int updateDetails(
        @Param("roleId") UUID roleId,
        @Param("roleName") String roleName,
        @Param("description") String description,
        @Param("isEnabled") Boolean isEnabled,
        @Param("updatedBy") UUID updatedBy,
        @Param("expectedVersion") Long expectedVersion
    );

    /** Archive the role with optimistic locking. Returns 1 on success, 0 on conflict. */
    int softDeleteById(
        @Param("roleId") UUID roleId,
        @Param("updatedBy") UUID updatedBy,
        @Param("expectedVersion") Long expectedVersion
    );

    int deleteById(@Param("roleId") UUID roleId);
}
