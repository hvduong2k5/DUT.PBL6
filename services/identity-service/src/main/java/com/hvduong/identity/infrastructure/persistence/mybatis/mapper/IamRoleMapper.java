package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Param;
import java.util.List;
import java.util.UUID;

public interface IamRoleMapper {
    void insert(IamRoleRow row);

    /** Returns active role only (is_enabled = TRUE). Returns null if not found or soft-deleted. */
    IamRoleRow findById(@Param("roleId") UUID roleId);

    /** Returns role regardless of is_enabled status. Use for soft-delete guard checks. */
    IamRoleRow findByIdIncludeDisabled(@Param("roleId") UUID roleId);

    /** Returns active role only (is_enabled = TRUE). */
    IamRoleRow findByCode(@Param("roleCode") String roleCode);

    /** Returns all active roles (is_enabled = TRUE), ordered by created_at ASC. */
    List<IamRoleRow> findAll();

    int updateDetails(
        @Param("roleId") UUID roleId,
        @Param("roleName") String roleName,
        @Param("description") String description,
        @Param("isEnabled") Boolean isEnabled,
        @Param("updatedBy") UUID updatedBy,
        @Param("expectedVersion") Long expectedVersion
    );

    /** Soft delete: set is_enabled = FALSE with optimistic lock. Returns 1 on success, 0 on conflict. */
    int softDeleteById(
        @Param("roleId") UUID roleId,
        @Param("updatedBy") UUID updatedBy,
        @Param("expectedVersion") Long expectedVersion
    );

    int deleteById(@Param("roleId") UUID roleId);
}
