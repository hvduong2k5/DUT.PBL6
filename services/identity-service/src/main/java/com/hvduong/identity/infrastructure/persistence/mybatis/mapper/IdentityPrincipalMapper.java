package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import org.apache.ibatis.annotations.Param;

import java.util.List;
import java.util.UUID;

public interface IdentityPrincipalMapper {
    void insert(IdentityPrincipalRow row);

    IdentityPrincipalRow findById(@Param("principalId") UUID principalId);
    
    IdentityPrincipalRow findByIssuerAndSubject(
        @Param("idpIssuer") String idpIssuer, 
        @Param("idpSubject") String idpSubject
    );

    IdentityPrincipalRow findBySubject(@Param("idpSubject") String idpSubject);

    int updateProvisioningStatus(
        @Param("principalId") UUID principalId,
        @Param("newStatus") String newStatus,
        @Param("idpSubject") String idpSubject,
        @Param("expectedVersion") Long expectedVersion
    );

    List<IdentityPrincipalRow> findPage(
        @Param("principalType") String principalType,
        @Param("provisioningStatus") String provisioningStatus,
        @Param("offset") int offset,
        @Param("limit") int limit
    );

    long countTotal(
        @Param("principalType") String principalType,
        @Param("provisioningStatus") String provisioningStatus
    );

    boolean hasRoleCode(
        @Param("principalId") UUID principalId,
        @Param("roleCode") String roleCode
    );
}
