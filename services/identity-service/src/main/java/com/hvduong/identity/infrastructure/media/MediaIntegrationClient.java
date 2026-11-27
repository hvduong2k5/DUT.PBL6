package com.hvduong.identity.infrastructure.media;

import com.omamx.proto.media.v1.BindMediaRequest;
import com.omamx.proto.media.v1.GetAccessUrlRequest;
import com.omamx.proto.media.v1.GetAccessUrlResponse;
import com.omamx.proto.media.v1.MediaInternalServiceGrpc;
import com.omamx.proto.media.v1.ValidateMediaRequest;
import com.omamx.proto.media.v1.ValidateMediaResponse;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import jakarta.annotation.PreDestroy;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.Optional;
import java.util.concurrent.TimeUnit;

/**
 * gRPC client kết nối tới Media Service (port 9007) để xác thực tài liệu KYC B2B / Avatar
 * và cấp Presigned GET URL ngắn hạn cho quản trị viên.
 */
@Slf4j
@Component
public class MediaIntegrationClient {

    private final boolean grpcEnabled;
    private final String grpcTarget;
    private volatile ManagedChannel channel;

    public MediaIntegrationClient(
            @Value("${identity.media.grpc-enabled:false}") boolean grpcEnabled,
            @Value("${identity.media.grpc-target:localhost:9007}") String grpcTarget) {
        this.grpcEnabled = grpcEnabled;
        this.grpcTarget = grpcTarget;
    }

    public boolean validateAndBindDocument(
            String fileId, String objectKey, String expectedPurpose, String entityType, String entityId) {
        if (!grpcEnabled) {
            log.debug("[MEDIA-GRPC] Disabled; skipping validateAndBindDocument for fileId={}", fileId);
            return true;
        }
        try {
            ValidateMediaResponse validation = stub()
                    .withDeadlineAfter(3, TimeUnit.SECONDS)
                    .validateMediaReference(ValidateMediaRequest.newBuilder()
                            .setFileId(fileId == null ? "" : fileId)
                            .setObjectKey(objectKey == null ? "" : objectKey)
                            .setExpectedPurpose(expectedPurpose)
                            .build());
            if (!validation.getIsValid() || !"READY".equalsIgnoreCase(validation.getStatus())) {
                log.warn("[MEDIA-GRPC] Invalid media reference fileId={}, status={}", fileId, validation.getStatus());
                return false;
            }
            stub()
                    .withDeadlineAfter(3, TimeUnit.SECONDS)
                    .bindMediaToEntity(BindMediaRequest.newBuilder()
                            .setFileId(fileId == null ? "" : fileId)
                            .setObjectKey(objectKey == null ? "" : objectKey)
                            .setEntityType(entityType)
                            .setEntityId(entityId)
                            .build());
            return true;
        } catch (Exception ex) {
            log.error("[MEDIA-GRPC] validateAndBindDocument failed for fileId={}: {}", fileId, ex.getMessage());
            return false;
        }
    }

    public Optional<String> getPresignedAccessUrl(
            String fileId, String requestingUserId, String userRole, String disposition, int expirySeconds) {
        if (!grpcEnabled || fileId == null || fileId.isBlank()) {
            return Optional.empty();
        }
        try {
            GetAccessUrlResponse resp = stub()
                    .withDeadlineAfter(3, TimeUnit.SECONDS)
                    .getPresignedAccessUrl(GetAccessUrlRequest.newBuilder()
                            .setFileId(fileId)
                            .setRequestingUserId(requestingUserId == null ? "" : requestingUserId)
                            .setUserRole(userRole == null ? "" : userRole)
                            .setDisposition(disposition == null ? "inline" : disposition)
                            .setExpirySeconds(expirySeconds > 0 ? expirySeconds : 900)
                            .build());
            return Optional.ofNullable(resp.getAccessUrl()).filter(u -> !u.isBlank());
        } catch (Exception ex) {
            log.error("[MEDIA-GRPC] getPresignedAccessUrl failed for fileId={}: {}", fileId, ex.getMessage());
            return Optional.empty();
        }
    }

    private MediaInternalServiceGrpc.MediaInternalServiceBlockingStub stub() {
        if (channel == null) {
            synchronized (this) {
                if (channel == null) {
                    channel = ManagedChannelBuilder.forTarget(grpcTarget).usePlaintext().build();
                }
            }
        }
        return MediaInternalServiceGrpc.newBlockingStub(channel);
    }

    @PreDestroy
    public void shutdown() {
        if (channel != null && !channel.isShutdown()) {
            channel.shutdownNow();
        }
    }
}
