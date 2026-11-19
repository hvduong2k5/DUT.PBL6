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

    @Builder.Default
    private long tokenVersion = 1L;

    @Builder.Default
    private Instant lastRotatedAt = Instant.now();

    public boolean isAccessTokenExpired() {
        return accessTokenExpiresAt != null && Instant.now().isAfter(accessTokenExpiresAt);
    }

    /**
     * Proactive refresh buffer: triggers refresh 30 seconds before access token actually expires
     * to prevent in-flight request failures and clock skew issues.
     */
    public boolean shouldRefreshAccessToken() {
        return accessTokenExpiresAt != null && Instant.now().plusSeconds(30).isAfter(accessTokenExpiresAt);
    }

    public boolean isRefreshTokenExpired() {
        return refreshTokenExpiresAt != null && Instant.now().isAfter(refreshTokenExpiresAt);
    }
}
