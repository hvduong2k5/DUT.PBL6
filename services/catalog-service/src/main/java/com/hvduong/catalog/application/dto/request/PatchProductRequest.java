package com.hvduong.catalog.application.dto.request;

import java.util.UUID;

import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.Data;

/**
 * Input DTO cho PATCH Product — cập nhật một phần nội dung sản phẩm.
 *
 * <p>Quy ước PATCH:
 *
 * <ul>
 *   <li>Field {@code null} hoặc không gửi lên = giữ nguyên giá trị hiện tại.
 *   <li>Chuỗi rỗng ({@code ""}) = xóa giá trị của field optional (description, ingredients,
 *       storageGuide, ocopCertificateNo, story).
 *   <li>{@code name}, {@code slug} không được rỗng nếu được gửi lên.
 * </ul>
 */
@Data
public class PatchProductRequest {

  @Size(max = 500)
  private String name;

  @Size(max = 500)
  private String slug;

  private UUID categoryId;

  @Size(max = 5000)
  private String description;

  @Size(max = 2000)
  private String ingredients;

  @Size(max = 1000)
  private String storageGuide;

  @Min(value = 3, message = "ocopStar phải từ 3 đến 5")
  @Max(value = 5, message = "ocopStar phải từ 3 đến 5")
  private Integer ocopStar;

  @Size(max = 100)
  private String ocopCertificateNo;

  @Size(max = 5000)
  private String story;

  @NotNull(message = "expectedVersion không được bỏ trống")
  private Long expectedVersion;
}
