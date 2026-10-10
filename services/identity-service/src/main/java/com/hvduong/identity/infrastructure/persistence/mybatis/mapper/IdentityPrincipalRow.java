package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.util.UUID;
import java.time.LocalDateTime;

@Data
public class IdentityPrincipalRow {
    private UUID id;
    private String subjectId;
    private String email;
    private String phoneNumber;
    private String status;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
