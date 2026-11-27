package com.hvduong.catalog.application.dto.request;

import java.util.UUID;

import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.Data;

/** Input DTO để cập nhật Product — patch-style (chỉ field khác null mới được cập nhật). */
@Data
public class UpdateProductRequest {

  private UUID categoryId;

  @Size(max = 500)
  private String name;

  @Size(max = 500)
  private String slug;

  @Size(max = 5000)
  private String description;

  @Size(max = 2000)
  private String ingredients;

  @Size(max = 1000)
  private String storageGuide;

  @Min(value = 3, message = "Sao OCOP phải từ 3 đến 5")
  @Max(value = 5, message = "Sao OCOP phải từ 3 đến 5")
  private Integer ocopStar;

  @Size(max = 100)
  private String ocopCertificateNo;

  @Size(max = 5000)
  private String story;

  /** Bắt buộc cho khóa lạc quan (optimistic locking). */
  @NotNull(message = "expectedVersion không được bỏ trống")
  @Min(value = 0, message = "expectedVersion phải lớn hơn hoặc bằng 0")
  private Long expectedVersion;

  /** Alias tương thích ngược cho version. */
  public Long getVersion() {
    return expectedVersion;
  }

  public void setVersion(Long version) {
    if (this.expectedVersion == null) {
      this.expectedVersion = version;
    }
  }
}
