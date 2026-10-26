package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Mapper
public interface IdentityAuditLogMapper {

    void insert(IdentityAuditLogRow row);

    Optional<IdentityAuditLogRow> findById(@Param("auditId") UUID auditId);

    List<IdentityAuditLogRow> findByActorPrincipalId(@Param("actorPrincipalId") UUID actorPrincipalId);

    List<IdentityAuditLogRow> findByTarget(@Param("targetType") String targetType, @Param("targetId") String targetId);
}

