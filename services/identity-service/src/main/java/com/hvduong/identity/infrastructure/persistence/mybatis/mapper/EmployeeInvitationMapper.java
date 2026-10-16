package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.Optional;
import java.util.UUID;

@Mapper
public interface EmployeeInvitationMapper {

    void insert(EmployeeInvitationRow row);

    Optional<EmployeeInvitationRow> findById(@Param("invitationId") UUID invitationId);

    Optional<EmployeeInvitationRow> findPendingByPrincipalId(@Param("principalId") UUID principalId);

    int updateStatus(
            @Param("invitationId") UUID invitationId,
            @Param("newStatus") String newStatus,
            @Param("timestampColumn") String timestampColumn
    );
}

