package com.hvduong.identity.application.dto.common;

/**
 * Re-export of {@link com.hvduong.identity.common.response.PageResponse}.
 * New code should import from {@code com.hvduong.identity.common.response.PageResponse} directly.
 */
public class PageResponse<T> extends com.hvduong.identity.common.response.PageResponse<T> {
    public PageResponse() { super(); }
    public PageResponse(java.util.List<T> items, int page, int size, long totalItems, int totalPages) {
        super(items, page, size, totalItems, totalPages);
    }
}
