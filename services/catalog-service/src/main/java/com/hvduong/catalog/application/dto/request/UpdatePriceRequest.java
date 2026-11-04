package com.hvduong.catalog.application.dto.request;

import com.hvduong.catalog.common.enums.SalesChannel;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;
import lombok.Data;

import java.util.List;

/** Input DTO để cập nhật bảng giá Variant theo kênh. */
@Data
public class UpdatePriceRequest {

    @NotEmpty(message = "Danh sách giá không được bỏ trống")
    @Valid
    private List<PriceEntry> prices;

    @Data
    public static class PriceEntry {

        @NotNull
        private SalesChannel channel;

        @NotNull
        @Positive(message = "Giá phải lớn hơn 0")
        private Long amountUnits;

        private String currencyCode = "VND";
    }
}
