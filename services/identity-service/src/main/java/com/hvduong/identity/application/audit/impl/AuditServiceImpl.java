package com.hvduong.identity.application.audit.impl;

import com.hvduong.identity.application.audit.AuditService;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityAuditLogRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityAuditLogMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Propagation;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.Map;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class AuditServiceImpl implements AuditService {

    private final IdentityAuditLogMapper auditLogMapper;
    private final ObjectMapper objectMapper;

    @Override
    @Transactional(propagation = Propagation.REQUIRES_NEW)
    public void recordAudit(
            String actorKind,
            UUID actorPrincipalId,
            String actorClientId,
            String actionCode,
            String targetType,
            String targetId,
            String outcome,
            String reasonCode,
            String correlationId,
            String sourceIp,
            Map<String, Object> details
    ) {
        try {
            IdentityAuditLogRow row = new IdentityAuditLogRow();
            row.setAuditId(UUID.randomUUID());
            row.setOccurredAt(OffsetDateTime.now());
            row.setActorKind(actorKind != null ? actorKind : "SYSTEM");
            row.setActorPrincipalId(actorPrincipalId);
            row.setActorClientId(actorClientId);
            row.setActionCode(actionCode);
            row.setTargetType(targetType);
            row.setTargetId(targetId);
            row.setOutcome(outcome != null ? outcome : "SUCCESS");
            row.setReasonCode(reasonCode);
            row.setCorrelationId(correlationId);
            row.setSourceIp(sourceIp);

            if (details != null && !details.isEmpty()) {
                row.setDetails(objectMapper.writeValueAsString(details));
            } else {
                row.setDetails("{}");
            }

            auditLogMapper.insert(row);
        } catch (Exception ex) {
            log.error("Failed to write identity audit log for action: {}", actionCode, ex);
        }
    }
}

