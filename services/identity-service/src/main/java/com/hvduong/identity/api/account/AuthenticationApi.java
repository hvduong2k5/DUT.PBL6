package com.hvduong.identity.api.account;

import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.response.AuthUrlResponse;
import com.hvduong.identity.dto.response.SessionResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@Tag(name = "Authentication & BFF Sessions", description = "Kiến trúc Backend-for-Frontend (BFF Pattern) theo RFC 9700 & RFC 7636 PKCE")
@RequestMapping("/api/v1/public/auth")
public interface AuthenticationApi {

    @Operation(summary = "Khởi tạo đăng nhập OIDC PKCE (Dual-Mode: 302 Redirect cho Web & JSON cho Mobile)", description = "Mặc định chuyển hướng (HTTP 302) trình duyệt sang Keycloak. Đối với Mobile App hoặc gọi API, truyền query param format=json hoặc Header Accept: application/json để nhận payload JSON chứa authorizationUrl.")
    @GetMapping("/login")
    ResponseEntity<?> initiateLogin(
            @Parameter(description = "URL Frontend/App cần quay về sau khi hoàn tất đăng nhập") @RequestParam(value = "redirect_uri", required = false) String redirectUri,
            @Parameter(description = "Nhà cung cấp SSO trực tiếp (google, facebook, zalo, apple)") @RequestParam(value = "provider", required = false) String provider,
            @Parameter(description = "Định dạng phản hồi: 'json' cho Mobile/API client, hoặc mặc định 302 redirect cho Web") @RequestParam(value = "format", required = false) String format,
            @RequestHeader(value = org.springframework.http.HttpHeaders.ACCEPT, required = false) String acceptHeader,
            @RequestHeader(value = org.springframework.http.HttpHeaders.CONTENT_TYPE, required = false) String contentType
    );

    @Operation(summary = "Tiếp nhận Callback Authorization Code từ Keycloak", description = "Trao đổi Authorization Code lấy Token, lưu phiên server-side, thiết lập HttpOnly Cookie OMAMA_SESSION và redirect 302 về Frontend.")
    @GetMapping("/callback")
    ResponseEntity<?> handleCallback(
            @Parameter(description = "Authorization Code được Keycloak cấp") @RequestParam(value = "code", required = false) String code,
            @Parameter(description = "State ngẫu nhiên chống CSRF") @RequestParam(value = "state", required = false) String state,
            @Parameter(description = "Mã lỗi nếu xác thực thất bại tại IdP") @RequestParam(value = "error", required = false) String error,
            @Parameter(description = "Chi tiết lỗi") @RequestParam(value = "error_description", required = false) String errorDescription
    );

    @Operation(summary = "Kiểm tra phiên đăng nhập hiện tại", description = "Đọc HttpOnly Cookie OMAMA_SESSION, trả về trạng thái phiên và danh tính người dùng mà không để lộ JWT Access Token.")
    @GetMapping("/session")
    ResponseEntity<ApiResponse<SessionResponse>> getSession(
            @Parameter(description = "Cookie phiên làm việc BFF") @CookieValue(value = "OMAMA_SESSION", required = false) String sessionCookie
    );

    @Operation(summary = "Làm mới phiên đăng nhập BFF", description = "Dùng Refresh Token được lưu an toàn trong session server-side để làm mới phiên làm việc.")
    @PostMapping("/tokens/refresh")
    ResponseEntity<ApiResponse<SessionResponse>> refreshSession(
            @Parameter(description = "Cookie phiên làm việc BFF") @CookieValue(value = "OMAMA_SESSION", required = false) String sessionCookie
    );

    @Operation(summary = "Đăng xuất tài khoản", description = "Thu hồi Token trên Keycloak, xóa session trên server và dọn sạch Cookie OMAMA_SESSION.")
    @PostMapping("/logout")
    ResponseEntity<ApiResponse<Void>> logout(
            @Parameter(description = "Cookie phiên làm việc BFF") @CookieValue(value = "OMAMA_SESSION", required = false) String sessionCookie
    );
}
