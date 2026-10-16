package com.hvduong.identity.api.admin;

import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleRow;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/api/v1/admin/iam/roles")
@RequiredArgsConstructor
public class IamRoleController {

    private final IamRoleMapper roleMapper;

    @Data
    public static class CreateRoleRequest {
        private String roleCode;
        private String roleName;
        private String description;
    }

    @GetMapping("/{roleCode}")
    public ResponseEntity<IamRoleRow> getRoleByCode(@PathVariable String roleCode) {
        IamRoleRow role = roleMapper.findByCode(roleCode);
        if (role == null) {
            return ResponseEntity.notFound().build();
        }
        return ResponseEntity.ok(role);
    }

    @PostMapping
    public ResponseEntity<UUID> createRole(
            @RequestBody CreateRoleRequest request,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        UUID roleId = UUID.randomUUID();
        UUID creatorId = null;
        try {
            if (actorId != null) {
                creatorId = UUID.fromString(actorId);
            }
        } catch (Exception ignored) {}

        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode(request.getRoleCode());
        row.setRoleName(request.getRoleName());
        row.setDescription(request.getDescription());
        row.setIsProtected(false);
        row.setIsEnabled(true);
        row.setVersion(0L);
        row.setCreatedByPrincipalId(creatorId);

        roleMapper.insert(row);
        return ResponseEntity.ok(roleId);
    }
}

