package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.domain.entity.Category;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

/** MyBatis Mapper interface cho bảng {@code categories}. */
@Mapper
public interface CategoryMapper {

  List<Category> findRootCategories();

  List<Category> findByParentId(@Param("parentId") UUID parentId);

  List<Category> findAllActive();

  Optional<Category> findById(@Param("id") UUID id);

  Optional<Category> findBySlug(@Param("slug") String slug);

  boolean existsBySlug(@Param("slug") String slug, @Param("excludeId") UUID excludeId);

  int insert(Category category);

  int update(Category category);
}
