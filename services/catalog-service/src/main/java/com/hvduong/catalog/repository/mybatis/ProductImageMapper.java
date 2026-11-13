package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.domain.entity.ProductImage;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

/** MyBatis Mapper interface cho bảng {@code product_images}. */
@Mapper
public interface ProductImageMapper {

  List<ProductImage> findApprovedByProductId(@Param("productId") UUID productId);

  List<ProductImage> findAllByProductId(@Param("productId") UUID productId);

  Optional<ProductImage> findById(@Param("id") UUID id);

  void insert(ProductImage image);

  int update(ProductImage image);

  int deleteById(@Param("id") UUID id);

  /**
   * Hạ các ảnh COVER đang APPROVED khác xuống GALLERY trước khi đặt cover mới. Đảm bảo bất biến
   * "mỗi product chỉ có tối đa một cover APPROVED" mà không vấp unique index.
   */
  int demoteOtherApprovedCovers(
      @Param("productId") UUID productId, @Param("excludeId") UUID excludeId);
}
