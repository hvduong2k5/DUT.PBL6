package com.hvduong.identity.api.account;

import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.SuspendRequest;
import com.hvduong.identity.dto.request.UpdateUserStatusRequest;
import com.hvduong.identity.dto.response.UserProfileResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Tag(name = "User Lifecycle & Profile", description = "Quản lý hồ sơ và vòng đời tài khoản người dùng theo chuẩn RESTful (Mục 3.3.1)")
@RequestMapping("/api/v1")
public interface UserLifecycleApi {

    @Operation(summary = "Lấy hồ sơ người dùng hiện tại", description = "Trích xuất JWT Access Token để trả về hồ sơ, danh sách vai trò và quyền hạn tổng hợp.")
    @SecurityRequirement(name = "bearerAuth")
    @GetMapping("/users/me")
    ResponseEntity<ApiResponse<UserProfileResponse>> getCurrentUser(@AuthenticationPrincipal Jwt jwt);

    @Operation(summary = "Cập nhật trạng thái người dùng (Admin) [Chuẩn RESTful 3.3.1]", description = "Cập nhật trạng thái tài khoản người dùng (ACTIVE, SUSPENDED) trên Keycloak và cơ sở dữ liệu qua HTTP PATCH.")
    @SecurityRequirement(name = "bearerAuth")
    @PatchMapping("/admin/users/{principalId}/status")
    ResponseEntity<ApiResponse<Void>> updateUserStatus(
            @Parameter(description = "ID của người dùng cần cập nhật trạng thái", required = true) @PathVariable UUID principalId,
            @RequestBody UpdateUserStatusRequest request,
            @AuthenticationPrincipal Jwt jwt
    );

    @Operation(summary = "Khóa tài khoản người dùng (Admin) [Legacy Alias]", description = "Vô hiệu hóa user trên Keycloak, đổi trạng thái DB sang SUSPENDED và ghi Audit Log.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/admin/users/{principalId}/suspend")
    ResponseEntity<ApiResponse<Void>> suspendUser(
            @Parameter(description = "ID của người dùng cần khóa", required = true) @PathVariable UUID principalId,
            @RequestBody(required = false) SuspendRequest request,
            @AuthenticationPrincipal Jwt jwt
    );

    @Operation(summary = "Tái kích hoạt tài khoản người dùng (Admin) [Legacy Alias]", description = "Kích hoạt lại user trên Keycloak và đổi trạng thái DB sang ACTIVE.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/admin/users/{principalId}/reactivate")
    ResponseEntity<ApiResponse<Void>> reactivateUser(
            @Parameter(description = "ID của người dùng cần kích hoạt lại", required = true) @PathVariable UUID principalId,
            @AuthenticationPrincipal Jwt jwt
    );
}
