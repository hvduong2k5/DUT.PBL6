package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.domain.entity.ProductImage;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.List;
import java.util.UUID;

/**
 * MyBatis Mapper interface cho bảng {@code product_images}.
 */
@Mapper
public interface ProductImageMapper {

    List<ProductImage> findApprovedByProductId(@Param("productId") UUID productId);

    List<ProductImage> findAllByProductId(@Param("productId") UUID productId);

    void insert(ProductImage image);

    int update(ProductImage image);

    int deleteById(@Param("id") UUID id);
}
