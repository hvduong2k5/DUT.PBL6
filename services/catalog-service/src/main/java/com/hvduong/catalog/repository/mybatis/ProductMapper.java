package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.repository.mybatis.projection.AdminCategoryRow;
import com.hvduong.catalog.repository.mybatis.projection.AdminStatusCountRow;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

/**
 * MyBatis Mapper interface cho bảng {@code products}. SQL định nghĩa trong {@code
 * resources/mapper/sql/ProductMapper.xml}.
 */
@Mapper
public interface ProductMapper {

  List<Product> findPublicProducts(
      @Param("query") String query,
      @Param("categoryId") UUID categoryId,
      @Param("ocopStar") Integer ocopStar,
      @Param("minPrice") Long minPrice,
      @Param("maxPrice") Long maxPrice,
      @Param("sortBy") String sortBy,
      @Param("channel") String channel,
      @Param("offset") int offset,
      @Param("limit") int limit);

  long countPublicProducts(
      @Param("query") String query,
      @Param("categoryId") UUID categoryId,
      @Param("ocopStar") Integer ocopStar,
      @Param("minPrice") Long minPrice,
      @Param("maxPrice") Long maxPrice,
      @Param("channel") String channel);

  Optional<Product> findPublicById(@Param("id") UUID id);

  Optional<Product> findPublicBySlug(@Param("slug") String slug);

  Optional<Product> findById(@Param("id") UUID id);

  List<Product> findAdminProducts(
      @Param("q") String q,
      @Param("status") String status,
      @Param("offset") int offset,
      @Param("limit") int limit);

  long countAdminProducts(@Param("q") String q, @Param("status") String status);

  List<AdminStatusCountRow> countAdminProductsByStatus(@Param("q") String q);

  List<AdminCategoryRow> findAdminCategories(@Param("q") String q);

  boolean existsBySlug(@Param("slug") String slug, @Param("excludeId") UUID excludeId);

  void insert(Product product);

  int update(Product product);

  int updateApprovalStatus(
      @Param("id") UUID id,
      @Param("approvalStatus") String approvalStatus,
      @Param("approvedBy") String approvedBy,
      @Param("expectedVersion") long expectedVersion);

  int updateListingStatus(
      @Param("id") UUID id,
      @Param("listingStatus") String listingStatus,
      @Param("expectedVersion") long expectedVersion);
}
