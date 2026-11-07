package com.hvduong.identity.api.admin;

import com.hvduong.identity.common.response.ApiResponse;
import com.hvduong.identity.common.response.PageResponse;
import com.hvduong.identity.application.dto.request.ResetPasswordRequest;
import com.hvduong.identity.application.dto.response.AdminUserDetailResponse;
import com.hvduong.identity.application.dto.response.AdminUserItemResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Tag(name = "Admin User Management", description = "Quản lý danh sách, hồ sơ, mật khẩu và phiên đăng nhập của người dùng dành riêng cho ADMIN")
@RequestMapping("/api/v1/admin/users")
public interface AdminUserApi {

    @Operation(summary = "Lấy danh sách người dùng phân trang", description = "Truy vấn danh sách người dùng kèm bộ lọc loại tài khoản (CUSTOMER, EMPLOYEE) và trạng thái (ACTIVE, SUSPENDED). Chỉ ADMIN.")
    @SecurityRequirement(name = "bearerAuth")
    @GetMapping
    ResponseEntity<ApiResponse<PageResponse<AdminUserItemResponse>>> listUsers(
            @Parameter(description = "Loại tài khoản: CUSTOMER, EMPLOYEE") @RequestParam(value = "type", required = false) String type,
            @Parameter(description = "Trạng thái: ACTIVE, SUSPENDED") @RequestParam(value = "status", required = false) String status,
            @Parameter(description = "Số trang (bắt đầu từ 0)") @RequestParam(value = "page", defaultValue = "0") int page,
            @Parameter(description = "Kích thước trang") @RequestParam(value = "size", defaultValue = "20") int size,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Xem chi tiết hồ sơ người dùng", description = "Truy vấn chi tiết người dùng kèm thông tin định danh, các vai trò và quyền hạn thực tế. Chỉ ADMIN.")
    @SecurityRequirement(name = "bearerAuth")
    @GetMapping("/{principalId}")
    ResponseEntity<ApiResponse<AdminUserDetailResponse>> getUserDetail(
            @Parameter(description = "ID người dùng", required = true) @PathVariable UUID principalId,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Đặt lại mật khẩu cho người dùng", description = "ADMIN đặt lại mật khẩu mới cho người dùng trên Keycloak IDP.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/{principalId}/password/reset")
    ResponseEntity<ApiResponse<Void>> resetUserPassword(
            @Parameter(description = "ID người dùng", required = true) @PathVariable UUID principalId,
            @Valid @RequestBody ResetPasswordRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );

    @Operation(summary = "Thu hồi toàn bộ phiên đăng nhập của người dùng", description = "ADMIN buộc người dùng đăng xuất ngay lập tức khỏi hệ thống (force logout).")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/{principalId}/sessions/revoke")
    ResponseEntity<ApiResponse<Void>> revokeUserSessions(
            @Parameter(description = "ID người dùng", required = true) @PathVariable UUID principalId,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    );
}

