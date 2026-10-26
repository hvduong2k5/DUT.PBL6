package com.hvduong.identity.application.iam;

import com.hvduong.identity.infrastructure.persistence.entity.IamRoleRow;
import lombok.Builder;
import lombok.Data;

import java.time.OffsetDateTime;
import java.util.List;
import java.util.UUID;

public interface RoleManagementService {

    @Data
    class CreateRoleDto {
        private String roleCode;
        private String roleName;
        private String description;
    }

    @Data
    class UpdateRoleDto {
        private String roleName;
        private String description;
        private Boolean isEnabled;
    }

    @Data
    @Builder
    class RoleDetailDto {
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

    List<IamRoleRow> getAllRoles();

    RoleDetailDto getRoleById(UUID roleId);

    UUID createRole(CreateRoleDto dto, UUID creatorId);

    void updateRole(UUID roleId, UpdateRoleDto dto, UUID updaterId);

    void deleteRole(UUID roleId, UUID actorId);

    List<String> getPermissionsForRole(UUID roleId);

    void replaceRolePermissions(UUID roleId, List<String> permissionCodes, UUID actorId);

    void addPermissionToRole(UUID roleId, String permissionCode, UUID actorId);

    void removePermissionFromRole(UUID roleId, String permissionCode, UUID actorId);
}
