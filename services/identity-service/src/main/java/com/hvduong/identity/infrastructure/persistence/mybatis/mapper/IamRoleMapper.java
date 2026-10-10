package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import org.apache.ibatis.annotations.Param;
import java.util.UUID;

public interface IamRoleMapper {
    void insert(IamRoleRow row);
    
    IamRoleRow findById(@Param("roleId") UUID roleId);
    
    IamRoleRow findByCode(@Param("roleCode") String roleCode);

    int updateStatus(
        @Param("roleId") UUID roleId,
        @Param("isEnabled") Boolean isEnabled,
        @Param("updatedBy") UUID updatedBy,
        @Param("expectedVersion") Long expectedVersion
    );
}
