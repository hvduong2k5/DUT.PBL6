package com.hvduong.identity.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Domain entity representing an identity principal (customer or employee).
 * This is the domain model; the persistence entity is in infrastructure.persistence.entity.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class IdentityPrincipal {
    private UUID principalId;
    private String principalType; // CUSTOMER | EMPLOYEE
    private String idpSubject;    // Keycloak sub
    private String status;        // ACTIVE | SUSPENDED | PENDING_VERIFICATION | DELETED
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
}
