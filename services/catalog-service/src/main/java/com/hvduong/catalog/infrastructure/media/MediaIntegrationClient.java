package com.hvduong.catalog.infrastructure.media;

import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;

import java.util.concurrent.TimeUnit;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import com.omamx.proto.media.v1.BindMediaRequest;
import com.omamx.proto.media.v1.MediaInternalServiceGrpc;
import com.omamx.proto.media.v1.PublishMediaRequest;
import com.omamx.proto.media.v1.PublishMediaResponse;
import com.omamx.proto.media.v1.ValidateMediaRequest;
import com.omamx.proto.media.v1.ValidateMediaResponse;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import jakarta.annotation.PreDestroy;
import lombok.extern.slf4j.Slf4j;

/**
 * gRPC client kết nối tới Media Service (port 9007) để xác thực objectKey, gắn kết entity và xuất
 * bản ảnh đã duyệt sang vùng public.
 */
@Slf4j
@Component
public class MediaIntegrationClient {

  private final boolean grpcEnabled;
  private final String grpcTarget;
  private volatile ManagedChannel channel;

  public MediaIntegrationClient(
      @Value("${catalog.media.grpc-enabled:false}") boolean grpcEnabled,
      @Value("${catalog.media.grpc-target:localhost:9007}") String grpcTarget) {
    this.grpcEnabled = grpcEnabled;
    this.grpcTarget = grpcTarget;
  }

  public static boolean isExternalUrl(String value) {
    return value != null && (value.startsWith("http://") || value.startsWith("https://"));
  }

  public void validateProductImage(String objectKey) {
    validateImage(objectKey, "CATALOG_PRODUCT");
  }

  public void validateCategoryImage(String objectKey) {
    validateImage(objectKey, "CATALOG_CATEGORY");
  }

  private void validateImage(String objectKey, String expectedPurpose) {
    if (!grpcEnabled) {
      log.debug(
          "[MEDIA-GRPC] Disabled; skipping validateImage({}) for {}", expectedPurpose, objectKey);
      return;
    }
    try {
      ValidateMediaResponse response =
          stub()
              .withDeadlineAfter(3, TimeUnit.SECONDS)
              .validateMediaReference(
                  ValidateMediaRequest.newBuilder()
                      .setObjectKey(objectKey)
                      .setExpectedPurpose(expectedPurpose)
                      .build());
      if (!response.getIsValid() || !"READY".equalsIgnoreCase(response.getStatus())) {
        throw new CatalogException(
            ErrorCode.CATALOG_IMAGE_INVALID,
            "Tham chiếu ảnh không hợp lệ hoặc chưa xử lý xong trên Media Service: " + objectKey);
      }
    } catch (CatalogException ex) {
      throw ex;
    } catch (Exception ex) {
      log.error("[MEDIA-GRPC] Failed to validate objectKey={}: {}", objectKey, ex.getMessage());
      throw new CatalogException(
          ErrorCode.CATALOG_IMAGE_INVALID,
          "Không thể xác thực ảnh với Media Service: " + ex.getMessage());
    }
  }

  public void bindProductImage(String objectKey, String productId) {
    bindMedia(objectKey, "Product", productId);
  }

  public void bindCategoryImage(String objectKey, String categoryId) {
    bindMedia(objectKey, "Category", categoryId);
  }

  private void bindMedia(String objectKey, String entityType, String entityId) {
    if (!grpcEnabled) {
      return;
    }
    try {
      stub()
          .withDeadlineAfter(3, TimeUnit.SECONDS)
          .bindMediaToEntity(
              BindMediaRequest.newBuilder()
                  .setObjectKey(objectKey)
                  .setEntityType(entityType)
                  .setEntityId(entityId)
                  .build());
    } catch (Exception ex) {
      log.warn(
          "[MEDIA-GRPC] BindMediaToEntity failed for objectKey={}, entityType={}, entityId={}: {}",
          objectKey,
          entityType,
          entityId,
          ex.getMessage());
    }
  }

  public void publishProductImage(String objectKey) {
    publishMedia(objectKey);
  }

  public void publishCategoryImage(String objectKey) {
    publishMedia(objectKey);
  }

  private void publishMedia(String objectKey) {
    if (!grpcEnabled) {
      return;
    }
    try {
      PublishMediaResponse response =
          stub()
              .withDeadlineAfter(5, TimeUnit.SECONDS)
              .publishMedia(PublishMediaRequest.newBuilder().setObjectKey(objectKey).build());
      if (!response.getSuccess()) {
        log.warn("[MEDIA-GRPC] PublishMedia returned success=false for {}", objectKey);
      }
    } catch (Exception ex) {
      log.error(
          "[MEDIA-GRPC] PublishMedia failed for objectKey={}: {}", objectKey, ex.getMessage());
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
