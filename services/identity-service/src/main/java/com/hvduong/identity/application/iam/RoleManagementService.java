package com.hvduong.identity.application.iam;

import com.hvduong.identity.dto.request.CreateRoleRequest;
import com.hvduong.identity.dto.request.UpdateRoleRequest;
import com.hvduong.identity.dto.response.RoleDetailResponse;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleRow;

import java.util.List;
import java.util.UUID;

public interface RoleManagementService {

    /**
     * @deprecated Sử dụng {@link CreateRoleRequest}
     */
    @Deprecated
    class CreateRoleDto extends CreateRoleRequest {}

    /**
     * @deprecated Sử dụng {@link UpdateRoleRequest}
     */
    @Deprecated
    class UpdateRoleDto extends UpdateRoleRequest {}

    /**
     * @deprecated Sử dụng {@link RoleDetailResponse}
     */
    @Deprecated
    class RoleDetailDto extends RoleDetailResponse {}

    List<IamRoleRow> getAllRoles();

    RoleDetailResponse getRoleById(UUID roleId);

    UUID createRole(CreateRoleRequest dto, UUID creatorId);

    void updateRole(UUID roleId, UpdateRoleRequest dto, UUID updaterId);

    void deleteRole(UUID roleId, UUID actorId);

    List<String> getPermissionsForRole(UUID roleId);

    void replaceRolePermissions(UUID roleId, List<String> permissionCodes, UUID actorId);

    void addPermissionToRole(UUID roleId, String permissionCode, UUID actorId);

    void removePermissionFromRole(UUID roleId, String permissionCode, UUID actorId);
}
