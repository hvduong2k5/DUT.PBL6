package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.domain.entity.*;

import org.apache.ibatis.annotations.Param;
import java.time.OffsetDateTime;
import java.util.List;
import java.util.UUID;

public interface IamRolePermissionMapper {
    void insert(IamRolePermissionRow row);
    
    List<IamRolePermissionRow> findByRoleId(@Param("roleId") UUID roleId);
    
    void deleteByRoleIdAndPermission(@Param("roleId") UUID roleId, @Param("permissionCode") String permissionCode);

    void deleteByRoleId(@Param("roleId") UUID roleId);

    /**
     * Compute the effective permission codes for a principal directly from the IAM
     * source of truth: active assignments joined to enabled roles and their permissions.
     * This is the single query used before every Keycloak attribute sync.
     */
    List<String> findEffectivePermissionCodesByPrincipal(
            @Param("principalId") UUID principalId,
            @Param("now") OffsetDateTime now
    );

    /**
     * Compute the effective permission grants for a principal directly from the IAM
     * source of truth. Each element binds a permission to its assignment scope
     * (scope_type + scope_id). No cross-product between elements.
     */
    List<PermissionGrantRow> findEffectivePermissionGrantsByPrincipal(
            @Param("principalId") UUID principalId,
            @Param("now") OffsetDateTime now
    );

    /**
     * Row representing one permission grant with its scope.
     */
    class PermissionGrantRow {
        private String permissionCode;
        private String scopeType;
        private String scopeId;

        public String getPermissionCode() { return permissionCode; }
        public void setPermissionCode(String permissionCode) { this.permissionCode = permissionCode; }
        public String getScopeType() { return scopeType; }
        public void setScopeType(String scopeType) { this.scopeType = scopeType; }
        public String getScopeId() { return scopeId; }
        public void setScopeId(String scopeId) { this.scopeId = scopeId; }
    }
}
