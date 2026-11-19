package com.hvduong.identity.common.handler;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import org.junit.jupiter.api.Test;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.security.authorization.AuthorizationDeniedException;

import com.hvduong.identity.common.response.ApiResponse;

/**
 * F-01 regression: method-security denials used to fall through to the
 * generic handler and surface as HTTP 500, disguising a permission problem
 * as a server crash.
 */
class GlobalExceptionHandlerTest {

    private final GlobalExceptionHandler handler = new GlobalExceptionHandler();

    @Test
    void accessDeniedReturns403PermissionDenied() {
        ResponseEntity<ApiResponse<Void>> res =
                handler.handleAccessDenied(new AccessDeniedException("denied"));
        assertEquals(HttpStatus.FORBIDDEN, res.getStatusCode());
        assertNotNull(res.getBody());
        assertEquals("PERMISSION_DENIED", res.getBody().getCode());
    }

    @Test
    void methodSecurityDenialIsAnAccessDenied() {
        // @PreAuthorize throws AuthorizationDeniedException, a subclass of
        // AccessDeniedException, so the 403 handler covers it.
        AccessDeniedException ex = new AuthorizationDeniedException("denied by method security", () -> false);
        ResponseEntity<ApiResponse<Void>> res = handler.handleAccessDenied(ex);
        assertEquals(HttpStatus.FORBIDDEN, res.getStatusCode());
        assertNotNull(res.getBody());
        assertEquals("PERMISSION_DENIED", res.getBody().getCode());
    }
}
