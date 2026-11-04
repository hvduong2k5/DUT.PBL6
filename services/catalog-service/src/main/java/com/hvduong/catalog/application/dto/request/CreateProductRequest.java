package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.Data;

import java.util.UUID;

/** Input DTO để tạo Product mới. */
@Data
public class CreateProductRequest {

    @NotNull(message = "categoryId không được bỏ trống")
    private UUID categoryId;

    @NotBlank(message = "Tên sản phẩm không được bỏ trống")
    @Size(max = 500)
    private String name;

    @NotBlank(message = "Slug không được bỏ trống")
    @Size(max = 500)
    private String slug;

    @Size(max = 5000)
    private String description;

    @Size(max = 2000)
    private String ingredients;

    @Size(max = 1000)
    private String storageGuide;

    private Integer ocopStar;

    @Size(max = 100)
    private String ocopCertificateNo;

    @Size(max = 5000)
    private String story;
}
