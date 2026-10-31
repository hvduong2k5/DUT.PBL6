package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Param;
import java.util.List;
import java.util.UUID;

public interface IamRoleMapper {
    void insert(IamRoleRow row);
    
    IamRoleRow findById(@Param("roleId") UUID roleId);
    
    IamRoleRow findByCode(@Param("roleCode") String roleCode);

    List<IamRoleRow> findAll();

    int updateDetails(
        @Param("roleId") UUID roleId,
        @Param("roleName") String roleName,
        @Param("description") String description,
        @Param("isEnabled") Boolean isEnabled,
        @Param("updatedBy") UUID updatedBy,
        @Param("expectedVersion") Long expectedVersion
    );

    int deleteById(@Param("roleId") UUID roleId);
}
