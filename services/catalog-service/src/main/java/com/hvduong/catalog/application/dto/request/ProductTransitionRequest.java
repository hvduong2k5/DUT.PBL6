package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProductTransitionRequest {

  @NotBlank(message = "Hành động chuyển trạng thái không được để trống")
  @Pattern(
      regexp = "^(SUBMIT_FOR_REVIEW|APPROVE|REJECT|SUSPEND|ARCHIVE)$",
      message =
          "Hành động chuyển đổi không được hỗ trợ. Chỉ chấp nhận: SUBMIT_FOR_REVIEW, APPROVE, REJECT, SUSPEND, ARCHIVE")
  private String transition; // SUBMIT_FOR_REVIEW, APPROVE, REJECT, SUSPEND, ARCHIVE

  @NotNull(message = "expectedVersion không được bỏ trống")
  @Min(value = 0, message = "expectedVersion phải lớn hơn hoặc bằng 0")
  private Long expectedVersion;
}
