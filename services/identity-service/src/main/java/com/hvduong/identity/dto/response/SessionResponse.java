package com.hvduong.identity.dto.response;

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
public class SessionResponse {
    private UUID sessionId;
    private UUID principalId;
    private String email;
    private boolean isAuthenticated;
    private Instant expiresAt;
    private String accessToken;
}
