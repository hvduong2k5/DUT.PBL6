package com.hvduong.identity.api.registration.dto;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;
import java.util.UUID;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class RegistrationResponse {
    private UUID principalId;
    private String status; // e.g. "PENDING", "ACTIVE", "CONFLICT"
    private String message;
}
