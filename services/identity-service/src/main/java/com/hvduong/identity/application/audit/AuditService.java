package com.hvduong.identity.application.audit;

import java.util.Map;
import java.util.UUID;

public interface AuditService {

    void recordAudit(
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
    );
}
