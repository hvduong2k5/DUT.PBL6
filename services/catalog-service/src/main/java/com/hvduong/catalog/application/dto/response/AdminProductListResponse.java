package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

/**
 * Payload nằm trong {@code data} của GET /api/v1/admin/catalog/products.
 * Khớp với {@code CatalogAdminProductListData} trong admin-stable.openapi.yaml.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminProductListResponse {

    private List<AdminProductListItemResponse> items;
    private List<StatusCount> statusCounts;
    private List<CategoryOption> categories;
    private String calculatedAt;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class StatusCount {
        private String status;
        private String label;
        private long count;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CategoryOption {
        private String categoryId;
        private String categoryName;
    }
}
