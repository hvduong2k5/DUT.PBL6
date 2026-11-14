package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.domain.entity.*;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.Optional;
import java.util.UUID;

@Mapper
public interface EmployeeInvitationMapper {

    void insert(EmployeeInvitationRow row);

    Optional<EmployeeInvitationRow> findByIdForUpdate(@Param("invitationId") UUID invitationId);

    Optional<EmployeeInvitationRow> findPendingByPrincipalIdForUpdate(@Param("principalId") UUID principalId);

    int updateStatus(
            @Param("invitationId") UUID invitationId,
            @Param("newStatus") String newStatus
    );
}

