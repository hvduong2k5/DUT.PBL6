package com.hvduong.catalog.common.response;

import lombok.Builder;
import lombok.Getter;

import java.util.List;

/**
 * Wrapper response cho danh sách có phân trang.
 * Khớp với {@code PageInfo} trong proto hvduong.catalog.v1.
 */
@Getter
@Builder
public class PageResponse<T> {

    private final List<T> items;
    private final int page;
    private final int pageSize;
    private final long total;
    private final int totalPages;

    public static <T> PageResponse<T> of(List<T> items, int page, int pageSize, long total) {
        int totalPages = pageSize == 0 ? 0 : (int) Math.ceil((double) total / pageSize);
        return PageResponse.<T>builder()
                .items(items)
                .page(page)
                .pageSize(pageSize)
                .total(total)
                .totalPages(totalPages)
                .build();
    }
}
