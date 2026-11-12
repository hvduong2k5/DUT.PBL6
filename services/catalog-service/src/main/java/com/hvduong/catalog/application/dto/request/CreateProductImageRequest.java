package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import lombok.Data;

/** Input DTO để thêm ảnh cho sản phẩm. */
@Data
public class CreateProductImageRequest {

  @NotBlank(message = "objectKey không được bỏ trống")
  @Size(max = 1024)
  private String objectKey;

  @Size(max = 500)
  private String altText;

  /** COVER | GALLERY | THUMBNAIL. Mặc định GALLERY. */
  private String role = "GALLERY";

  @Min(value = 0, message = "position phải lớn hơn hoặc bằng 0")
  private Integer position = 0;

  /** PENDING_SCAN | APPROVED | REJECTED. Mặc định PENDING_SCAN cho ảnh mới tải lên. */
  private String mediaStatus;
}
