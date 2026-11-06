package com.hvduong.identity.dto.response;

import java.util.List;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UserRoleScopeResponse {
    private String roleCode;
    private String roleName;
    private String scopeType;
    private String scopeId;
    private List<String> permissions;
}

