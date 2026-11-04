package com.hvduong.catalog;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cache.annotation.EnableCaching;
import org.springframework.scheduling.annotation.EnableScheduling;

/**
 * Catalog Service — MS-05
 * <p>
 * Nguồn chân lý (SSOT) cho dữ liệu thương mại sản phẩm OCOP:
 * Danh mục, Product mẹ, Variant/SKU, ảnh, giá niêm yết và trạng thái bán.
 * <p>
 * Giao tiếp: gRPC internal only (port 8005) — không có REST API.
 */
@SpringBootApplication
@EnableCaching
@EnableScheduling
public class CatalogApplication {

    public static void main(String[] args) {
        SpringApplication.run(CatalogApplication.class, args);
    }
}
