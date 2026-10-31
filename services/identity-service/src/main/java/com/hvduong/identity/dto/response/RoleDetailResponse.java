package com.hvduong.identity.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.List;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class RoleDetailResponse {
    private UUID roleId;
    private String roleCode;
    private String roleName;
    private String description;
    private Boolean isProtected;
    private Boolean isEnabled;
    private Long version;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
    private List<String> permissions;
}

