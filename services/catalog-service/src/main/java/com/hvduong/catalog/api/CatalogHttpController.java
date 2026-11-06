package com.hvduong.catalog.api;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.PropertyNamingStrategies;
import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.application.service.CheckoutSnapshotService;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.enums.SortBy;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.server.ResponseStatusException;
import java.math.BigDecimal;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.*;

/** Public DTO adapter; checkout endpoints are authenticated internal calls. */
@RestController
@RequestMapping("/api/v1")
@RequiredArgsConstructor
public class CatalogHttpController {
    private final CatalogService catalog;
    private final CheckoutSnapshotService snapshots;
    private final ObjectMapper json;
    @Value("${catalog.internal-token:}") private String internalToken;
    private Object dto(Object value) { return json.copy().setPropertyNamingStrategy(PropertyNamingStrategies.SNAKE_CASE).setSerializationInclusion(com.fasterxml.jackson.annotation.JsonInclude.Include.ALWAYS).convertValue(value, Object.class); }
    @GetMapping("/categories")
    public Object categories() { return Map.of("categories", catalog.getAllCategories().stream().map(category -> {
        Map<String,Object> value=new LinkedHashMap<>(); value.put("category_id",category.getCategoryId()); value.put("parent_id",category.getParentId());
        value.put("name",category.getName()); value.put("slug",category.getSlug());value.put("description",Objects.toString(category.getDescription(),""));value.put("image_url",Objects.toString(category.getImageUrl(),""));return value;
    }).toList()); }
    @GetMapping("/products")
    public Object products(@RequestParam(defaultValue="1") int page, @RequestParam(name="page_size",required=false) Integer requestedSize,
            @RequestParam(name="pageSize",required=false) Integer legacySize,
            @RequestParam(required=false) UUID category_id, @RequestParam(required=false) Long min_price,
            @RequestParam(required=false) Long max_price, @RequestParam(defaultValue="NEWEST") SortBy sort_by,
            @RequestParam(defaultValue="D2C_WEB") SalesChannel channel) {
        int size=pageSize(requestedSize,legacySize);
        if (page < 1 || page > 100000) throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
        var filter = new ProductFilterRequest(); filter.setPage(page); filter.setPageSize(size); filter.setCategoryId(category_id);
        filter.setMinPrice(min_price); filter.setMaxPrice(max_price); filter.setSortBy(sort_by); filter.setChannel(channel);
        var result=catalog.getPublicProducts(filter);
        return Map.of("items",result.getItems().stream().map(this::summary).toList(),"total",result.getTotal(),"page",result.getPage(),"page_size",result.getPageSize(),"total_pages",result.getTotalPages());
    }
    @GetMapping("/products/search")
    public Object search(@RequestParam(required=false) String query, @RequestParam(name="q",required=false) String legacyQuery,
            @RequestParam(defaultValue="1") int page, @RequestParam(name="page_size",required=false) Integer requestedSize,
            @RequestParam(name="pageSize",required=false) Integer legacySize) {
        int size=pageSize(requestedSize,legacySize);
        if(query!=null && legacyQuery!=null && !query.equals(legacyQuery))throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
        if(query==null)query=legacyQuery;
        if (page < 1 || page > 100000 || query==null || query.isBlank()) throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
        var result=catalog.searchProducts(query, page, size);
        return Map.of("query",query,"items",result.getItems().stream().map(this::summary).toList(),"total",result.getTotal());
    }
    @GetMapping("/products/{slug}")
    public Object product(@PathVariable String slug, @RequestParam(defaultValue="D2C_WEB") SalesChannel channel) {
        var product = catalog.getPublicProductDetail(slug, channel);
        Map<String,Object> result = new LinkedHashMap<>();
        result.put("product_id", product.getProductId()); result.put("name", product.getName()); result.put("slug", product.getSlug());
        result.put("description", Objects.toString(product.getDescription(), "")); result.put("story", Objects.toString(product.getStory(), ""));
        result.put("ocop_star", product.getOcopStar() == null ? 0 : product.getOcopStar()); result.put("ocop_certificate_no", Objects.toString(product.getOcopCertificateNo(), ""));
        result.put("images", product.getImageUrls() == null ? List.of() : product.getImageUrls());
        result.put("variants", product.getVariants().stream().map(this::variant).toList());
        return result;
    }
    @GetMapping("/products/variants/{sku}")
    public Object sku(@PathVariable String sku) { return Map.of("variant",variant(catalog.getVariantBySkuCode(sku))); }
    private Object variant(ProductVariantResponse variant) {
        if(variant.getPrice()==null)throw new ResponseStatusException(HttpStatus.SERVICE_UNAVAILABLE,"CATALOG_PRICE_UNAVAILABLE");
        Map<String,Object> value = new LinkedHashMap<>(); value.put("variant_id", variant.getVariantId()); value.put("sku_code", variant.getSkuCode()); value.put("name", variant.getName());
        BigDecimal grams = variant.getWeightValue(); if ("KILOGRAM".equals(variant.getWeightUnit())) grams = grams.multiply(BigDecimal.valueOf(1000));
        value.put("weight_gram", grams.longValueExact()); value.put("price", dto(variant.getPrice())); value.put("original_price", dto(variant.getPrice())); value.put("stock_available", variant.getStockAvailable());
        return value;
    }
    private int pageSize(Integer canonical,Integer legacy) {
        if(canonical!=null && legacy!=null && !canonical.equals(legacy))throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
        int size=canonical!=null?canonical:legacy!=null?legacy:20;
        if(size<1||size>100)throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
        return size;
    }
    @SuppressWarnings("unchecked")
    private Object summary(com.hvduong.catalog.application.dto.response.ProductListItemResponse item) {
        var value=(Map<String,Object>)dto(item);
        for(String field:List.of("summary","thumbnail_url","category_name")) if(value.get(field)==null)value.put(field,"");
        if(value.get("ocop_star")==null)value.put("ocop_star",0);
        if(item.getBasePrice()==null)throw new ResponseStatusException(HttpStatus.SERVICE_UNAVAILABLE,"CATALOG_PRICE_UNAVAILABLE");
        return value;
    }
    @PostMapping("/internal/catalog/snapshots")
    public Object snapshot(@RequestHeader(name="X-Internal-Token",required=false) String token, @RequestBody CheckoutSnapshotService.Request request) { authorize(token); return snapshots.create(request); }
    @PostMapping("/internal/catalog/snapshots/{id}/revalidate")
    public Object revalidate(@RequestHeader(name="X-Internal-Token",required=false) String token, @PathVariable String id) { authorize(token); return Map.of("valid", snapshots.revalidate(id)); }
    private void authorize(String token) {
        if (internalToken.length()<32 || token==null || !MessageDigest.isEqual(internalToken.getBytes(StandardCharsets.UTF_8), token.getBytes(StandardCharsets.UTF_8))) throw new ResponseStatusException(HttpStatus.UNAUTHORIZED);
    }
}
