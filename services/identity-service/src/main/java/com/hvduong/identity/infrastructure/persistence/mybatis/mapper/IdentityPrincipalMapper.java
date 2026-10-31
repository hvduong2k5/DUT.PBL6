package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Param;
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
}
