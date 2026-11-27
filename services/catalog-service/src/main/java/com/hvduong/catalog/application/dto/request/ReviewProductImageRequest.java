package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Input DTO để kiểm duyệt ảnh sản phẩm (Content Moderation). */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ReviewProductImageRequest {

  @NotBlank(message = "decision không được bỏ trống")
  @Pattern(regexp = "^(APPROVED|REJECTED)$", message = "decision phải là APPROVED hoặc REJECTED")
  private String decision;

  @Size(max = 1000, message = "reason tối đa 1000 ký tự")
  private String reason;

  @NotNull(message = "expectedVersion không được bỏ trống")
  private Long expectedVersion;
}
