package com.hvduong.identity.infrastructure.session;

import org.springframework.stereotype.Component;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class BffSessionStore {

    private final Map<UUID, BffSession> sessions = new ConcurrentHashMap<>();
    private static final Duration MAX_SESSION_IDLE = Duration.ofDays(30);

    public BffSession createSession(String idpSubject, UUID principalId, String email, String accessToken, String refreshToken, long expiresIn, long refreshExpiresIn) {
        cleanExpired();
        UUID sessionId = UUID.randomUUID();
        Instant now = Instant.now();
        BffSession session = BffSession.builder()
                .sessionId(sessionId)
                .idpSubject(idpSubject)
                .principalId(principalId)
                .email(email)
                .accessToken(accessToken)
                .refreshToken(refreshToken)
                .accessTokenExpiresAt(now.plusSeconds(expiresIn))
                .refreshTokenExpiresAt(now.plusSeconds(refreshExpiresIn))
                .createdAt(now)
                .lastAccessedAt(now)
                .build();
        sessions.put(sessionId, session);
        return session;
    }

    public Optional<BffSession> getSession(UUID sessionId) {
        if (sessionId == null) return Optional.empty();
        cleanExpired();
        BffSession session = sessions.get(sessionId);
        if (session != null) {
            session.setLastAccessedAt(Instant.now());
            return Optional.of(session);
        }
        return Optional.empty();
    }

    public void updateTokens(UUID sessionId, String newAccessToken, String newRefreshToken, long expiresIn, long refreshExpiresIn) {
        BffSession session = sessions.get(sessionId);
        if (session != null) {
            Instant now = Instant.now();
            session.setAccessToken(newAccessToken);
            if (newRefreshToken != null && !newRefreshToken.isBlank()) {
                session.setRefreshToken(newRefreshToken);
                session.setRefreshTokenExpiresAt(now.plusSeconds(refreshExpiresIn));
            }
            session.setAccessTokenExpiresAt(now.plusSeconds(expiresIn));
            session.setLastAccessedAt(now);
        }
    }

    public void removeSession(UUID sessionId) {
        if (sessionId != null) {
            sessions.remove(sessionId);
        }
    }

    private void cleanExpired() {
        Instant now = Instant.now();
        sessions.entrySet().removeIf(entry -> {
            BffSession s = entry.getValue();
            boolean idleExpired = Duration.between(s.getLastAccessedAt(), now).compareTo(MAX_SESSION_IDLE) > 0;
            return s.isRefreshTokenExpired() || idleExpired;
        });
    }
}

