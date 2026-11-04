package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Đại diện số tiền theo BigInt pattern (units + nanos).
 * VND: nanos luôn = 0.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class MoneyResponse {
    private String currencyCode;
    private long units;
    private int nanos;

    public static MoneyResponse ofVnd(long units) {
        return MoneyResponse.builder()
                .currencyCode("VND")
                .units(units)
                .nanos(0)
                .build();
    }
}
