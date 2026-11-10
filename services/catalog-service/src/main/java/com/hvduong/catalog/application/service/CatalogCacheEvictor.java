package com.hvduong.catalog.application.service;

import org.springframework.cache.annotation.CacheEvict;
import org.springframework.stereotype.Component;

/**
 * Xóa cache danh mục khi có mutation admin (create/update/transition/archive...). Được inject vào
 * {@link CatalogAdminServiceImpl} để tránh circular dependency giữa CatalogService và
 * CatalogAdminService.
 */
@Component
public class CatalogCacheEvictor {

  @CacheEvict(value = "catalog:categories", allEntries = true)
  public void evictCategoriesCache() {
    // No-op method, chỉ để trigger Spring cache eviction
  }
}
