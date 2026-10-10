package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import org.apache.ibatis.annotations.Param;
import java.util.List;
import java.util.UUID;

public interface IamRolePermissionMapper {
    void insert(IamRolePermissionRow row);
    
    List<IamRolePermissionRow> findByRoleId(@Param("roleId") UUID roleId);
    
    void deleteByRoleIdAndPermission(@Param("roleId") UUID roleId, @Param("permissionCode") String permissionCode);
}
