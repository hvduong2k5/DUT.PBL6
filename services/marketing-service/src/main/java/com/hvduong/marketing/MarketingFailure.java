package com.hvduong.marketing;

final class MarketingFailure extends RuntimeException {
    private final int status;

    MarketingFailure(String code, int status) {
        super(code);
        this.status = status;
    }

    int status() {
        return status;
    }
}
