package com.hvduong.identity.application.dto.request;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class RegistrationRequest {
    @jakarta.validation.constraints.NotBlank
    @jakarta.validation.constraints.Email
    @jakarta.validation.constraints.Size(max = 254)
    private String email;
    @jakarta.validation.constraints.NotBlank
    @jakarta.validation.constraints.Size(max = 100)
    private String firstName;
    @jakarta.validation.constraints.NotBlank
    @jakarta.validation.constraints.Size(max = 100)
    private String lastName;
    @jakarta.validation.constraints.NotBlank
    @jakarta.validation.constraints.Size(min = 12, max = 128)
    @lombok.ToString.Exclude
    private String rawPassword;
    @jakarta.validation.constraints.Pattern(regexp = "[A-Za-z0-9_-]{16,128}")
    @jakarta.validation.constraints.NotBlank
    private String idempotencyKey;
    private String requestFingerprint;
}

