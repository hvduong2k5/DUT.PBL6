package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import org.apache.ibatis.annotations.Param;
import java.util.UUID;

public interface IdentityPrincipalMapper {
    IdentityPrincipalRow findById(@Param("principalId") UUID principalId);
}
