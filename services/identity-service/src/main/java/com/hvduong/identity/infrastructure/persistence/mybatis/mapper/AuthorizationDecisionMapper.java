package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Mapper
public interface AuthorizationDecisionMapper {

    void insert(AuthorizationDecisionRow row);

    Optional<AuthorizationDecisionRow> findById(@Param("decisionId") UUID decisionId);

    List<AuthorizationDecisionRow> findByActorPrincipalId(@Param("actorPrincipalId") UUID actorPrincipalId);
}

