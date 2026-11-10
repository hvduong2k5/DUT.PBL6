package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.NotBlank;
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
  private String transition; // SUBMIT_FOR_REVIEW, APPROVE, REJECT, SUSPEND, ARCHIVE

  private long expectedVersion;
}
