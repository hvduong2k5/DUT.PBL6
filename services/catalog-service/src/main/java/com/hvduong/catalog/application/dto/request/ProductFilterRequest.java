package com.hvduong.catalog.application.dto.request;

import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.enums.SortBy;
import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.Positive;
import lombok.Data;

import java.util.UUID;

/** Input DTO cho use case lọc + phân trang sản phẩm. */
@Data
public class ProductFilterRequest {

    private UUID categoryId;

    @Min(3) @Max(5)
    private Integer ocopStar;

    @Min(0)
    private Long minPrice;

    @Min(0)
    private Long maxPrice;

    private SortBy sortBy = SortBy.NEWEST;

    private SalesChannel channel = SalesChannel.D2C_WEB;

    @Positive
    private int page = 1;

    @Min(1) @Max(100)
    private int pageSize = 20;

    public int getOffset() {
        return (page - 1) * pageSize;
    }
}
