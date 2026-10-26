package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Mapper
public interface SpecializedPermissionPolicyMapper {

    void insert(SpecializedPermissionPolicyRow row);

    Optional<SpecializedPermissionPolicyRow> findById(@Param("policyId") UUID policyId);

    List<SpecializedPermissionPolicyRow> findActiveByActionCode(@Param("actionCode") String actionCode);

    Optional<SpecializedPermissionPolicyRow> findActiveByActionCodeAndScope(
            @Param("actionCode") String actionCode,
            @Param("scopeType") String scopeType,
            @Param("scopeId") String scopeId
    );
}

