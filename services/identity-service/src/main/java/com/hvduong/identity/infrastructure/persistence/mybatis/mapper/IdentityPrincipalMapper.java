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
        @Param("roleId") UUID roleId,
        @Param("roleCode") String roleCode,
        @Param("scopeType") String scopeType,
        @Param("scopeId") String scopeId,
        @Param("now") java.time.OffsetDateTime now,
        @Param("search") String search,
        @Param("idpSubjects") List<String> idpSubjects,
        @Param("offset") int offset,
        @Param("limit") int limit
    );

    long countTotal(
        @Param("principalType") String principalType,
        @Param("provisioningStatus") String provisioningStatus,
        @Param("roleId") UUID roleId,
        @Param("roleCode") String roleCode,
        @Param("scopeType") String scopeType,
        @Param("scopeId") String scopeId,
        @Param("now") java.time.OffsetDateTime now,
        @Param("search") String search,
        @Param("idpSubjects") List<String> idpSubjects
    );

    boolean hasRoleCode(
        @Param("principalId") UUID principalId,
        @Param("roleCode") String roleCode
    );
}
