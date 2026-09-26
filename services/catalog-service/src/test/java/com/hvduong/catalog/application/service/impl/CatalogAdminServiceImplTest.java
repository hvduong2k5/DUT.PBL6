package com.hvduong.catalog.application.service.impl;

import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.mapper.CatalogDtoMapper;
import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.domain.entity.Category;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.repository.mybatis.CategoryMapper;
import com.hvduong.catalog.repository.mybatis.ProductMapper;
import com.hvduong.catalog.repository.mybatis.ProductVariantMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.transaction.support.TransactionTemplate;

import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class CatalogAdminServiceImplTest {

    @Mock private ProductMapper productMapper;
    @Mock private CategoryMapper categoryMapper;
    @Mock private ProductVariantMapper variantMapper;
    @Mock private CatalogDtoMapper dtoMapper;
    @Mock private CatalogServiceImpl catalogService;

    @InjectMocks
    private CatalogAdminServiceImpl adminService;

    private UUID productId;
    
    @BeforeEach
    void setUp() {
        productId = UUID.randomUUID();
    }

    @Test
    void createProduct_Success() {
        CreateProductRequest req = new CreateProductRequest();
        req.setSlug("test-slug");
        req.setCategoryId(UUID.randomUUID());
        req.setName("Test Product");
        
        when(productMapper.existsBySlug(req.getSlug(), null)).thenReturn(false);
        when(categoryMapper.findById(req.getCategoryId())).thenReturn(Optional.of(new Category()));
        
        ProductDetailResponse mockResponse = ProductDetailResponse.builder().build();
        mockResponse.setProductId(productId);
        when(dtoMapper.toDetailResponse(any())).thenReturn(mockResponse);

        ProductDetailResponse result = adminService.createProduct(req, "admin-1");

        assertNotNull(result);
        verify(productMapper, times(1)).insert(any(Product.class));
    }

    @Test
    void createProduct_SlugExists_ThrowsException() {
        CreateProductRequest req = new CreateProductRequest();
        req.setSlug("existing-slug");
        
        when(productMapper.existsBySlug(req.getSlug(), null)).thenReturn(true);

        assertThrows(CatalogException.class, () -> adminService.createProduct(req, "admin-1"));
        verify(productMapper, never()).insert(any());
    }

    @Test
    void updateProduct_Success() {
        UpdateProductRequest req = new UpdateProductRequest();
        req.setSlug("new-slug");
        req.setName("Updated Name");
        
        Product existingProduct = new Product();
        existingProduct.setId(productId);
        existingProduct.setVersion(1L);
        
        when(productMapper.findById(productId)).thenReturn(Optional.of(existingProduct));
        when(productMapper.existsBySlug("new-slug", productId)).thenReturn(false);
        when(productMapper.update(any(Product.class))).thenReturn(1);
        
        ProductDetailResponse mockResponse = ProductDetailResponse.builder().build();
        when(dtoMapper.toDetailResponse(any())).thenReturn(mockResponse);
        
        ProductDetailResponse result = adminService.updateProduct(productId, req, "admin-1");
        
        assertNotNull(result);
        verify(productMapper, times(1)).update(any(Product.class));
    }
}
