package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.Data;

/** Input DTO để cập nhật metadata/trạng thái ảnh sản phẩm. */
@Data
public class UpdateProductImageRequest {

  @Size(max = 500)
  private String altText;

  /** COVER | GALLERY | THUMBNAIL. */
  private String role;

  @Min(value = 0, message = "position phải lớn hơn hoặc bằng 0")
  private Integer position;

  /** PENDING_SCAN | APPROVED | REJECTED. */
  private String mediaStatus;

  @NotNull(message = "expectedVersion không được bỏ trống")
  private Long expectedVersion;
}
