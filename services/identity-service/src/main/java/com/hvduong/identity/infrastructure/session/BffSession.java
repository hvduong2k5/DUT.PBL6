package com.hvduong.identity.infrastructure.session;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class BffSession {
    private UUID sessionId;
    // Public correlation ID is never the bearer credential in OMAMA_SESSION.
    @Builder.Default
    private UUID publicId = UUID.randomUUID();
    private boolean nativeClient;
    private String oauthClientId;
    private String idpSubject;
    private UUID principalId;
    private String email;
    private String accessToken;
    private String refreshToken;
    private Instant accessTokenExpiresAt;
    private Instant refreshTokenExpiresAt;
    private Instant createdAt;
    private Instant lastAccessedAt;

    public boolean isAccessTokenExpired() {
        return accessTokenExpiresAt != null && Instant.now().isAfter(accessTokenExpiresAt);
    }

    public boolean isRefreshTokenExpired() {
        return refreshTokenExpiresAt != null && Instant.now().isAfter(refreshTokenExpiresAt);
    }
}

