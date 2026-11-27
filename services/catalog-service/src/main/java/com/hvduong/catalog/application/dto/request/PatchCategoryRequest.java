package com.hvduong.catalog.application.dto.request;

import java.util.UUID;

import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.PositiveOrZero;
import jakarta.validation.constraints.Size;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Input DTO để cập nhật một phần (PATCH) thông tin danh mục sản phẩm. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class PatchCategoryRequest {

  @Size(max = 255, message = "Tên danh mục tối đa 255 ký tự")
  private String name;

  @Size(max = 255, message = "Slug tối đa 255 ký tự")
  @Pattern(
      regexp = "^$|[a-z0-9]+(?:-[a-z0-9]+)*",
      message = "Slug chỉ chứa chữ thường, số và dấu gạch ngang")
  private String slug;

  private UUID parentId;

  @Size(max = 4000, message = "Mô tả tối đa 4000 ký tự")
  private String description;

  @Size(max = 2048, message = "Đường dẫn ảnh tối đa 2048 ký tự")
  private String imageUrl;

  @PositiveOrZero(message = "Vị trí sắp xếp phải >= 0")
  private Integer position;

  private Boolean isActive;

  @NotNull(message = "expectedVersion không được bỏ trống")
  private Long expectedVersion;
}
