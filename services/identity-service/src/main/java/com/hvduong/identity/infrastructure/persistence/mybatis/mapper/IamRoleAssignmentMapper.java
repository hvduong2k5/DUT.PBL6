package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Param;
import java.util.UUID;
import java.time.OffsetDateTime;
import java.util.List;

public interface IamRoleAssignmentMapper {
    void insert(IamRoleAssignmentRow row);
    
    List<IamRoleAssignmentRow> findActiveByPrincipal(@Param("principalId") UUID principalId, @Param("now") OffsetDateTime now);
    
    int revokeAssignment(
        @Param("assignmentId") UUID assignmentId,
        @Param("revokedBy") UUID revokedBy,
        @Param("reason") String reason,
        @Param("expectedVersion") Long expectedVersion
    );

    int countActiveByRoleId(@Param("roleId") UUID roleId, @Param("now") OffsetDateTime now);
}
