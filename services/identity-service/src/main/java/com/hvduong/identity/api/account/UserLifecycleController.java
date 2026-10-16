package com.hvduong.identity.api.account;

import com.hvduong.identity.application.account.UserLifecycleService;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/api/v1")
@RequiredArgsConstructor
public class UserLifecycleController {

    private final UserLifecycleService userLifecycleService;

    @Data
    public static class SuspendRequest {
        private String reason;
    }

    @GetMapping("/users/me")
    public ResponseEntity<UserLifecycleService.UserProfileDto> getCurrentUser(@AuthenticationPrincipal Jwt jwt) {
        if (jwt == null) {
            return ResponseEntity.status(401).build();
        }

        String subject = jwt.getSubject();
        String email = jwt.getClaimAsString("email");

        return userLifecycleService.getProfileBySubject(subject, email)
                .map(ResponseEntity::ok)
                .orElseGet(() -> ResponseEntity.notFound().build());
    }

    @PostMapping("/admin/users/{principalId}/suspend")
    public ResponseEntity<Void> suspendUser(
            @PathVariable UUID principalId,
            @RequestBody(required = false) SuspendRequest request,
            @AuthenticationPrincipal Jwt jwt
    ) {
        UUID actorId = null;
        if (jwt != null) {
            try {
                actorId = UUID.fromString(jwt.getSubject());
            } catch (Exception ignored) {}
        }

        String reason = request != null ? request.getReason() : "Suspended by admin";
        userLifecycleService.suspendAccount(principalId, reason, actorId);
        return ResponseEntity.noContent().build();
    }

    @PostMapping("/admin/users/{principalId}/reactivate")
    public ResponseEntity<Void> reactivateUser(
            @PathVariable UUID principalId,
            @AuthenticationPrincipal Jwt jwt
    ) {
        UUID actorId = null;
        if (jwt != null) {
            try {
                actorId = UUID.fromString(jwt.getSubject());
            } catch (Exception ignored) {}
        }

        userLifecycleService.reactivateAccount(principalId, actorId);
        return ResponseEntity.noContent().build();
    }
}

