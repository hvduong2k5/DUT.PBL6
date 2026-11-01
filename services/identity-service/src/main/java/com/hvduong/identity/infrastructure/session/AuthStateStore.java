package com.hvduong.identity.infrastructure.session;

import lombok.AllArgsConstructor;
import lombok.Data;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class AuthStateStore {

    @Data
    @AllArgsConstructor
    public static class PendingAuthState {
        private String codeVerifier;
        private String redirectUri;
        private Instant createdAt;
    }

    private final Map<String, PendingAuthState> states = new ConcurrentHashMap<>();
    private static final Duration TTL = Duration.ofMinutes(10);

    public void save(String state, String codeVerifier, String redirectUri) {
        cleanExpired();
        states.put(state, new PendingAuthState(codeVerifier, redirectUri, Instant.now()));
    }

    public Optional<PendingAuthState> consume(String state) {
        cleanExpired();
        PendingAuthState found = states.remove(state);
        if (found != null && Duration.between(found.getCreatedAt(), Instant.now()).compareTo(TTL) <= 0) {
            return Optional.of(found);
        }
        return Optional.empty();
    }

    private void cleanExpired() {
        Instant now = Instant.now();
        states.entrySet().removeIf(entry -> Duration.between(entry.getValue().getCreatedAt(), now).compareTo(TTL) > 0);
    }
}

