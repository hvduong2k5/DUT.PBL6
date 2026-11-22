package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
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

  /** COVER | GALLERY | THUMBNAIL | NUTRITION | CERTIFICATE. Mặc định GALLERY. */
  @Pattern(
      regexp = "^(COVER|GALLERY|THUMBNAIL|NUTRITION|CERTIFICATE)$",
      message =
          "Role ảnh không hợp lệ. Chỉ chấp nhận: COVER, GALLERY, THUMBNAIL, NUTRITION, CERTIFICATE")
  private String role = "GALLERY";

  @Min(value = 0, message = "position phải lớn hơn hoặc bằng 0")
  private Integer position = 0;
}
