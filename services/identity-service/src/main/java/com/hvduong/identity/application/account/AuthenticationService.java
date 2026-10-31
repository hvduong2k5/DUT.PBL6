package com.hvduong.identity.application.account;

import com.hvduong.identity.dto.response.AuthUrlResponse;
import com.hvduong.identity.dto.response.BffCallbackResult;
import com.hvduong.identity.dto.response.SessionResponse;
import com.hvduong.identity.infrastructure.session.BffSession;

import java.util.UUID;

public interface AuthenticationService {

    /**
     * Khởi tạo luồng OIDC Authorization Code + PKCE (RFC 7636).
     * @param redirectUri Đường dẫn frontend cần quay về sau khi hoàn tất đăng nhập.
     * @param provider Nhà cung cấp SSO (ví dụ: google, zalo, apple) nếu đăng nhập mạng xã hội.
     * @return URL xác thực trên Keycloak kèm state và code_challenge.
     */
    AuthUrlResponse initiateLogin(String redirectUri, String provider);

    /**
     * Tiếp nhận Authorization Code và State từ Keycloak callback,
     * đổi code lấy Token qua backchannel nội bộ và tạo phiên làm việc BFF Session.
     */
    BffCallbackResult handleCallback(String code, String state);

    /**
     * Điều hướng khi IdP trả về lỗi trong quá trình xác thực callback.
     */
    String resolveErrorRedirectUri(String state, String error, String errorDescription);

    /**
     * Lấy thông tin phiên làm việc hiện tại của Client từ Cookie.
     */
    SessionResponse getSessionInfo(UUID sessionId);

    /**
     * Làm mới Token trong phiên làm việc của BFF qua Refresh Token.
     */
    BffSession refreshSession(UUID sessionId);

    /**
     * Hủy phiên làm việc phía server và thu hồi Token trên Keycloak.
     */
    void logout(UUID sessionId);
}
