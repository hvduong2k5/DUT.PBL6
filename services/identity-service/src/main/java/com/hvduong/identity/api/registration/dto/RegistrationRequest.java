package com.hvduong.identity.api.registration.dto;

import lombok.Data;

@Data
public class RegistrationRequest {
    private String email;
    private String firstName;
    private String lastName;
    private String rawPassword;
    private String idempotencyKey; // Provided by the frontend or caller
    private String requestFingerprint; // Hash of device/IP/request body for duplicate detection
}
