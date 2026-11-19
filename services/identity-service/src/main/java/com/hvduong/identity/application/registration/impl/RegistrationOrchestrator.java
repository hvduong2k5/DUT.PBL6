package com.hvduong.identity.application.registration.impl;

import com.hvduong.identity.application.registration.RegisterPrincipalUseCase;
import com.hvduong.identity.common.enums.PrincipalType;
import com.hvduong.identity.common.enums.ProvisioningStatus;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.dto.request.RegistrationRequest;
import com.hvduong.identity.application.dto.response.RegistrationResponse;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import com.hvduong.identity.domain.entity.*;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Propagation;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.dao.DuplicateKeyException;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.util.HexFormat;

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
    private final PlatformTransactionManager transactionManager;
    @Value("${spring.security.oauth2.resourceserver.jwt.issuer-uri}")
    private String issuer;
    @Value("${identity.registration.fingerprint-key:}")
    private String fingerprintKey;
    @Value("${app.kafka.topics.identity-events:identity.events.v1}")
    private String identityEventsTopic = "identity.events.v1";

    private TransactionTemplate transaction() {
        TransactionTemplate template = new TransactionTemplate(transactionManager);
        template.setPropagationBehavior(org.springframework.transaction.TransactionDefinition.PROPAGATION_REQUIRES_NEW);
        return template;
    }

    private void normalizeAndFingerprint(RegistrationRequest request) {
        if (fingerprintKey.getBytes(StandardCharsets.UTF_8).length < 32) {
            throw new org.springframework.web.server.ResponseStatusException(org.springframework.http.HttpStatus.SERVICE_UNAVAILABLE, "Registration is not configured");
        }
        request.setEmail(request.getEmail().trim().toLowerCase(java.util.Locale.ROOT));
        request.setFirstName(request.getFirstName().trim());
        request.setLastName(request.getLastName().trim());
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(fingerprintKey.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            // The client cannot choose the fingerprint. HMAC prevents an offline
            // password dictionary attack against a leaked operation row.
            byte[] payload = objectMapper.writeValueAsBytes(java.util.List.of(request.getEmail(), request.getFirstName(), request.getLastName(), request.getRawPassword()));
            request.setRequestFingerprint(HexFormat.of().formatHex(mac.doFinal(payload)));
        } catch (Exception ex) {
            throw new IllegalStateException("Registration fingerprint unavailable");
        }
    }

    private RegistrationResponse replay(RegistrationOperationRow op, RegistrationRequest request) {
        if (!op.getRequestFingerprint().equals(request.getRequestFingerprint()))
            return new RegistrationResponse(null, "CONFLICT", "Idempotency key reused with different payload");
        return new RegistrationResponse(op.getPrincipalId(), op.getOperationStatus(), "Idempotent response");
    }

    @Value("${keycloak.public-url:${keycloak.server-url:https://keycloak.omama.vn}}")
    private String publicUrl = "https://keycloak.omama.vn";

    @Value("${keycloak.realm:omama}")
    private String realm = "omama";

    @Override
    public RegistrationResponse registerCustomer(RegistrationRequest request) {
        normalizeAndFingerprint(request);
        // Step 1: Check Idempotency (Requires a new transaction to lock/commit immediately)
        RegistrationOperationRow existingOp = operationMapper.findByIdempotency("CUSTOMER_REGISTER", request.getIdempotencyKey());
        if (existingOp != null) {
            return replay(existingOp, request);
        }

        // Step 2: Initialize PENDING state in DB (Requires new transaction)
        UUID principalId = UUID.randomUUID();
        UUID operationId = UUID.randomUUID();
        try {
            transaction().executeWithoutResult(status -> initializePendingRegistration(principalId, operationId, request));
        } catch (DuplicateKeyException ex) {
            RegistrationOperationRow winner = operationMapper.findByIdempotency("CUSTOMER_REGISTER", request.getIdempotencyKey());
            if (winner == null) throw ex;
            return replay(winner, request);
        }

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
            log.warn("Keycloak registration failed for operation {}", operationId);
            transaction().executeWithoutResult(status -> markOperationFailed(operationId, "IDP_PROVISIONING_FAILED"));
            return new RegistrationResponse(principalId, "FAILED", "Identity provider provisioning failed");
        }

        // Step 4: Finalize mapping and write outbox (In a single transaction)
        transaction().executeWithoutResult(status -> finalizeRegistrationAndOutbox(principalId, operationId, request, idpSubject));

        return new RegistrationResponse(principalId, "ACTIVE", "User registered successfully");
    }

    @Transactional(propagation = Propagation.REQUIRES_NEW)
    public void initializePendingRegistration(UUID principalId, UUID operationId, RegistrationRequest request) {
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpIssuer(publicUrl.replaceAll("/$", "") + "/realms/" + realm);
        principal.setPrincipalType(PrincipalType.CUSTOMER.name());
        principal.setProvisioningStatus(ProvisioningStatus.PENDING.name());
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
        if (principalMapper.updateProvisioningStatus(principalId, "ACTIVE", idpSubject, 0L) != 1) {
            throw new IllegalStateException("Registration principal version changed");
        }

        // 2. Update Operation
        operationMapper.updateStatus(operationId, "COMPLETED", null, OffsetDateTime.now());

        // 3. Write Outbox Event
        try {
            OutboxEventRow outboxEvent = new OutboxEventRow();
            outboxEvent.setEventId(UUID.randomUUID());
            outboxEvent.setAggregateType("IdentityPrincipal");
            outboxEvent.setAggregateId(principalId.toString());
            outboxEvent.setEventType(identityEventsTopic + ".UserRegistered");
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
