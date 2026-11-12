package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Ảnh sản phẩm trong admin response. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminProductImageResponse {
  private String imageId;
  private String objectKey;
  private String url;
  private String altText;
  private String role;
  private Integer position;
  private String mediaStatus;
  private Long revision;
  private String createdAt;
  private String updatedAt;
}
