package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.util.UUID;
import java.time.OffsetDateTime;

@Data
public class IdentityPrincipalRow {
    private UUID principalId;
    private String idpIssuer;
    private String idpSubject;
    private String principalType;
    private String provisioningStatus;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
    private Long version;
}
