package com.hvduong.identity.application.registration.impl;

import com.hvduong.identity.application.registration.RegisterPrincipalUseCase;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.dto.request.RegistrationRequest;
import com.hvduong.identity.dto.response.RegistrationResponse;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import com.hvduong.identity.infrastructure.persistence.entity.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Propagation;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

@Service
@RequiredArgsConstructor
@Slf4j
public class RegistrationOrchestrator implements RegisterPrincipalUseCase {

    private final IdentityPrincipalMapper principalMapper;
    private final RegistrationOperationMapper operationMapper;
    private final OutboxEventMapper outboxEventMapper;
    private final KeycloakAdminClient keycloakAdminClient;
    private final ObjectMapper objectMapper;

    @Override
    public RegistrationResponse registerCustomer(RegistrationRequest request) {
        // Step 1: Check Idempotency (Requires a new transaction to lock/commit immediately)
        RegistrationOperationRow existingOp = operationMapper.findByIdempotency("CUSTOMER_REGISTER", request.getIdempotencyKey());
        if (existingOp != null) {
            if (!existingOp.getRequestFingerprint().equals(request.getRequestFingerprint())) {
                return new RegistrationResponse(null, "CONFLICT", "Idempotency key reused with different payload");
            }
            return new RegistrationResponse(existingOp.getPrincipalId(), existingOp.getOperationStatus(), "Idempotent response");
        }

        // Step 2: Initialize PENDING state in DB (Requires new transaction)
        UUID principalId = UUID.randomUUID();
        UUID operationId = UUID.randomUUID();
        initializePendingRegistration(principalId, operationId, request);

        // Step 3: Call External System (Keycloak) - OUTSIDE OF DB TRANSACTION to avoid holding connection
        String idpSubject;
        try {
            idpSubject = keycloakAdminClient.createUser(
                principalId,
                request.getEmail(), 
                request.getFirstName(), 
                request.getLastName(), 
                request.getRawPassword()
            );
        } catch (Exception ex) {
            log.error("Failed to create user in Keycloak", ex);
            markOperationFailed(operationId, ex.getMessage());
            return new RegistrationResponse(principalId, "FAILED", "Identity provider provisioning failed");
        }

        // Step 4: Finalize mapping and write outbox (In a single transaction)
        finalizeRegistrationAndOutbox(principalId, operationId, request, idpSubject);

        return new RegistrationResponse(principalId, "ACTIVE", "User registered successfully");
    }

    @Transactional(propagation = Propagation.REQUIRES_NEW)
    public void initializePendingRegistration(UUID principalId, UUID operationId, RegistrationRequest request) {
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpIssuer("https://keycloak.omama.vn/realms/omama");
        principal.setPrincipalType("CUSTOMER");
        principal.setProvisioningStatus("PENDING");
        principal.setVersion(0L);
        principalMapper.insert(principal);

        RegistrationOperationRow operation = new RegistrationOperationRow();
        operation.setOperationId(operationId);
        operation.setOperationType("CUSTOMER_REGISTER");
        operation.setIdempotencyKey(request.getIdempotencyKey());
        operation.setRequestFingerprint(request.getRequestFingerprint());
        operation.setPrincipalId(principalId);
        operation.setOperationStatus("PENDING");
        operationMapper.insert(operation);
    }

    @Transactional(propagation = Propagation.REQUIRES_NEW)
    public void markOperationFailed(UUID operationId, String error) {
        operationMapper.updateStatus(operationId, "FAILED", error, OffsetDateTime.now());
        // A reconciliation job will later pick up PENDING principals with FAILED operations
    }

    @Transactional(propagation = Propagation.REQUIRES_NEW)
    public void finalizeRegistrationAndOutbox(UUID principalId, UUID operationId, RegistrationRequest request, String idpSubject) {
        // 1. Update Principal
        principalMapper.updateProvisioningStatus(principalId, "ACTIVE", idpSubject, 0L);

        // 2. Update Operation
        operationMapper.updateStatus(operationId, "COMPLETED", null, OffsetDateTime.now());

        // 3. Write Outbox Event
        try {
            OutboxEventRow outboxEvent = new OutboxEventRow();
            outboxEvent.setEventId(UUID.randomUUID());
            outboxEvent.setAggregateType("IdentityPrincipal");
            outboxEvent.setAggregateId(principalId.toString());
            outboxEvent.setEventType("identity.events.v1.UserRegistered");
            outboxEvent.setSchemaVersion(1);
            outboxEvent.setPartitionKey(principalId.toString());
            
            Map<String, Object> payload = new HashMap<>();
            payload.put("principalId", principalId.toString());
            payload.put("email", request.getEmail());
            payload.put("firstName", request.getFirstName());
            payload.put("lastName", request.getLastName());
            outboxEvent.setPayload(objectMapper.writeValueAsString(payload));
            
            outboxEvent.setDeliveryStatus("PENDING");
            outboxEvent.setAttemptCount(0);
            outboxEvent.setNextAttemptAt(OffsetDateTime.now());
            outboxEvent.setOccurredAt(OffsetDateTime.now());
            
            outboxEventMapper.insert(outboxEvent);
        } catch (JsonProcessingException e) {
            throw new RuntimeException("Failed to serialize outbox payload", e);
        }
    }
}
