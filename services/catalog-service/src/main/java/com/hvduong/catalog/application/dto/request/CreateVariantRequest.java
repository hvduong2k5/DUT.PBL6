package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;
import jakarta.validation.constraints.Size;
import lombok.Data;

import java.math.BigDecimal;

/** Input DTO để tạo ProductVariant (SKU) mới. */
@Data
public class CreateVariantRequest {

    @NotBlank(message = "skuCode không được bỏ trống")
    @Size(max = 100)
    private String skuCode;

    @NotBlank(message = "variantName không được bỏ trống")
    @Size(max = 500)
    private String variantName;

    @NotNull(message = "weightValue không được bỏ trống")
    @Positive(message = "weightValue phải lớn hơn 0")
    private BigDecimal weightValue;

    private String weightUnit = "GRAM";

    @Size(max = 200)
    private String flavor;

    @NotBlank(message = "packagingType không được bỏ trống")
    private String packagingType;

    @Positive
    private Integer shelfLifeDays;

    @NotNull(message = "basePrice không được bỏ trống")
    @Positive(message = "basePrice phải lớn hơn 0")
    private Long basePriceUnits;
}
