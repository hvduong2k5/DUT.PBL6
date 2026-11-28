package com.hvduong.identity.grpc;

import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptors;
import java.io.File;
import java.util.concurrent.TimeUnit;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.SmartLifecycle;
import org.springframework.stereotype.Component;

/** Independent TLS endpoint preserves the legacy Identity listener and its consumers. */
@Component
@ConditionalOnProperty(name = "IDENTITY_TRACEABILITY_GRPC_ENABLED", havingValue = "true")
public class TraceabilityStaffGrpcListener implements SmartLifecycle {
    private final TraceabilityStaffGrpcService service;
    private final int port;
    private final String certificate;
    private final String key;
    private final String token;
    private volatile Server server;

    public TraceabilityStaffGrpcListener(TraceabilityStaffGrpcService service,
            @Value("${IDENTITY_TRACEABILITY_GRPC_PORT:9116}") int port,
            @Value("${IDENTITY_TRACEABILITY_GRPC_TLS_CERT_FILE:}") String certificate,
            @Value("${IDENTITY_TRACEABILITY_GRPC_TLS_KEY_FILE:}") String key,
            @Value("${IDENTITY_TRACEABILITY_GRPC_TOKEN:}") String token) {
        this.service = service;
        this.port = port;
        this.certificate = certificate;
        this.key = key;
        this.token = token;
    }

    @Override
    public void start() {
        if (token.length() < 32 || token.chars().anyMatch(Character::isWhitespace)
                || certificate.isBlank() || key.isBlank() || port < 1 || port > 65535) {
            throw new IllegalStateException("Traceability staff RPC requires TLS and a dedicated credential");
        }
        try {
            server = ServerBuilder.forPort(port).useTransportSecurity(new File(certificate), new File(key))
                    .maxInboundMessageSize(1024).maxInboundMetadataSize(32768)
                    .addService(ServerInterceptors.intercept(service, TraceabilityStaffGrpcService.credentials(token)))
                    .build().start();
        } catch (Exception error) {
            throw new IllegalStateException("Cannot start Traceability staff TLS listener");
        }
    }

    @Override
    public void stop() {
        Server current = server;
        if (current == null) return;
        current.shutdown();
        try {
            if (!current.awaitTermination(5, TimeUnit.SECONDS)) current.shutdownNow();
        } catch (InterruptedException error) {
            current.shutdownNow();
            Thread.currentThread().interrupt();
        } finally {
            server = null;
        }
    }

    @Override public boolean isRunning() { return server != null && !server.isShutdown(); }
    @Override public int getPhase() { return Integer.MAX_VALUE - 100; }
}
