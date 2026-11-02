package com.hvduong.identity.dto.request;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class RegistrationRequest {
    private String email;
    private String firstName;
    private String lastName;
    private String rawPassword;
    private String idempotencyKey;
    private String requestFingerprint;
}

