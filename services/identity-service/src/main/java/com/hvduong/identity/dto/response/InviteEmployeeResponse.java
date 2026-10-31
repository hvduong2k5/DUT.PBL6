package com.hvduong.identity.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class InviteEmployeeResponse {
    private UUID invitationId;
    private UUID principalId;
    private String status;
    private OffsetDateTime expiresAt;
}

