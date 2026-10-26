package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Param;

public interface IamPermissionMapper {
    void insert(IamPermissionRow row);
    
    IamPermissionRow findByCode(@Param("permissionCode") String permissionCode);
}
