package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.domain.entity.ProductVariant;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

/**
 * MyBatis Mapper interface cho bảng {@code product_variants}.
 */
@Mapper
public interface ProductVariantMapper {

    List<ProductVariant> findActiveByProductId(@Param("productId") UUID productId);

    List<ProductVariant> findAllByProductId(@Param("productId") UUID productId);

    Optional<ProductVariant> findBySkuCode(@Param("skuCode") String skuCode);

    Optional<ProductVariant> findById(@Param("id") UUID id);

    boolean existsBySkuCode(@Param("skuCode") String skuCode, @Param("excludeId") UUID excludeId);

    List<ProductVariant> findByFilter(
            @Param("categoryId") UUID categoryId,
            @Param("ocopRating") String ocopRating,
            @Param("activeOnly") Boolean activeOnly,
            @Param("offset") int offset,
            @Param("limit") int limit
    );

    long countByFilter(
            @Param("categoryId") UUID categoryId,
            @Param("ocopRating") String ocopRating,
            @Param("activeOnly") Boolean activeOnly
    );

    void insert(ProductVariant variant);

    int update(ProductVariant variant);

    int updateListingStatus(
            @Param("id") UUID id,
            @Param("listingStatus") String listingStatus,
            @Param("expectedVersion") long expectedVersion
    );
}
