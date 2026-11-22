package com.hvduong.catalog.api;

import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.response.CategoryListResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.enums.SortBy;
import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;

import java.math.BigDecimal;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

import org.springframework.http.HttpStatus;
import org.springframework.security.authentication.AnonymousAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.server.ResponseStatusException;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.PropertyNamingStrategies;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.media.Content;
import io.swagger.v3.oas.annotations.media.Schema;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.RequiredArgsConstructor;

/** Public DTO adapter for Catalog read APIs. */
@Tag(name = "Catalog Public", description = "Các API tra cứu danh mục, sản phẩm công khai")
@RestController
@RequestMapping("/api/v1")
@RequiredArgsConstructor
public class CatalogHttpController {
  private final CatalogService catalog;
  private final ObjectMapper json;

  private Object dto(Object value) {
    return json.copy()
        .setPropertyNamingStrategy(PropertyNamingStrategies.SNAKE_CASE)
        .setSerializationInclusion(com.fasterxml.jackson.annotation.JsonInclude.Include.ALWAYS)
        .convertValue(value, Object.class);
  }

  @Operation(
      summary = "Lấy danh sách cây danh mục sản phẩm OCOP",
      description = "Trả về danh sách toàn bộ danh mục sản phẩm đang hoạt động.")
  @ApiResponses({
    @ApiResponse(
        responseCode = "200",
        description = "Thành công",
        content =
            @Content(
                mediaType = "application/json",
                schema = @Schema(implementation = CategoryListResponse.class)))
  })
  @GetMapping("/categories")
  public CategoryListResponse categories() {
    List<CategoryListResponse.CategoryItem> items =
        catalog.getAllCategories().stream()
            .map(
                category ->
                    CategoryListResponse.CategoryItem.builder()
                        .categoryId(category.getCategoryId())
                        .parentId(category.getParentId())
                        .name(category.getName())
                        .slug(category.getSlug())
                        .description(Objects.toString(category.getDescription(), ""))
                        .imageUrl(Objects.toString(category.getImageUrl(), ""))
                        .build())
            .toList();
    return CategoryListResponse.builder().categories(items).build();
  }

  @GetMapping("/products")
  public Object products(
      @RequestParam(defaultValue = "1") int page,
      @RequestParam(name = "page_size", required = false) Integer requestedSize,
      @RequestParam(name = "pageSize", required = false) Integer legacySize,
      @RequestParam(name = "category_id", required = false) UUID category_id,
      @RequestParam(name = "categoryId", required = false) UUID categoryId,
      @RequestParam(name = "min_price", required = false) Long min_price,
      @RequestParam(name = "minPrice", required = false) Long minPrice,
      @RequestParam(name = "max_price", required = false) Long max_price,
      @RequestParam(name = "maxPrice", required = false) Long maxPrice,
      @RequestParam(name = "ocop_star", required = false) Integer ocop_star,
      @RequestParam(name = "ocopStar", required = false) Integer ocopStar,
      @RequestParam(name = "sort_by", required = false) SortBy sort_by,
      @RequestParam(name = "sortBy", required = false) SortBy sortBy,
      @RequestParam(defaultValue = "D2C_WEB") SalesChannel channel) {
    int size = pageSize(requestedSize, legacySize);
    if (page < 1 || page > 100000) throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
    UUID targetCategoryId = category_id != null ? category_id : categoryId;
    Long targetMinPrice = min_price != null ? min_price : minPrice;
    Long targetMaxPrice = max_price != null ? max_price : maxPrice;
    Integer targetOcopStar = ocop_star != null ? ocop_star : ocopStar;
    SortBy targetSortBy =
        sort_by != null ? sort_by : (sortBy != null ? sortBy : SortBy.BEST_SELLING);

    var filter = new ProductFilterRequest();
    filter.setPage(page);
    filter.setPageSize(size);
    filter.setCategoryId(targetCategoryId);
    filter.setMinPrice(targetMinPrice);
    filter.setMaxPrice(targetMaxPrice);
    filter.setOcopStar(targetOcopStar);
    filter.setSortBy(targetSortBy);
    filter.setChannel(channel != null ? channel : SalesChannel.D2C_WEB);
    var result = catalog.getPublicProducts(filter);
    return Map.of(
        "items",
        result.getItems().stream().map(this::summary).toList(),
        "total",
        result.getTotal(),
        "page",
        result.getPage(),
        "page_size",
        result.getPageSize(),
        "total_pages",
        result.getTotalPages());
  }

  @GetMapping("/products/search")
  public Object search(
      @RequestParam(required = false) String query,
      @RequestParam(name = "q", required = false) String legacyQuery,
      @RequestParam(defaultValue = "1") int page,
      @RequestParam(name = "page_size", required = false) Integer requestedSize,
      @RequestParam(name = "pageSize", required = false) Integer legacySize,
      @RequestParam(name = "category_id", required = false) UUID category_id,
      @RequestParam(name = "categoryId", required = false) UUID categoryId,
      @RequestParam(name = "min_price", required = false) Long min_price,
      @RequestParam(name = "minPrice", required = false) Long minPrice,
      @RequestParam(name = "max_price", required = false) Long max_price,
      @RequestParam(name = "maxPrice", required = false) Long maxPrice,
      @RequestParam(name = "ocop_star", required = false) Integer ocop_star,
      @RequestParam(name = "ocopStar", required = false) Integer ocopStar,
      @RequestParam(name = "sort_by", required = false) SortBy sort_by,
      @RequestParam(name = "sortBy", required = false) SortBy sortBy,
      @RequestParam(defaultValue = "D2C_WEB") SalesChannel channel) {
    int size = pageSize(requestedSize, legacySize);
    if (query != null && legacyQuery != null && !query.equals(legacyQuery))
      throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
    if (query == null) query = legacyQuery;
    if (page < 1 || page > 100000 || query == null || query.isBlank())
      throw new ResponseStatusException(HttpStatus.BAD_REQUEST);

    UUID targetCategoryId = category_id != null ? category_id : categoryId;
    Long targetMinPrice = min_price != null ? min_price : minPrice;
    Long targetMaxPrice = max_price != null ? max_price : maxPrice;
    Integer targetOcopStar = ocop_star != null ? ocop_star : ocopStar;
    SortBy targetSortBy = sort_by != null ? sort_by : (sortBy != null ? sortBy : SortBy.NEWEST);
    SalesChannel targetChannel = channel != null ? channel : SalesChannel.D2C_WEB;

    var result =
        catalog.searchProducts(
            query,
            targetCategoryId,
            targetMinPrice,
            targetMaxPrice,
            targetOcopStar,
            targetSortBy,
            targetChannel,
            page,
            size);
    return Map.of(
        "query",
        query,
        "items",
        result.getItems().stream().map(this::summary).toList(),
        "total",
        result.getTotal(),
        "page",
        result.getPage(),
        "page_size",
        result.getPageSize(),
        "total_pages",
        result.getTotalPages());
  }

  @GetMapping("/products/{slug}")
  public Object product(
      @PathVariable String slug, @RequestParam(defaultValue = "D2C_WEB") SalesChannel channel) {
    var product = catalog.getPublicProductDetail(slug, channel);
    Map<String, Object> result = new LinkedHashMap<>();
    result.put("product_id", product.getProductId());
    result.put("name", product.getName());
    result.put("slug", product.getSlug());
    result.put("description", Objects.toString(product.getDescription(), ""));
    result.put("story", Objects.toString(product.getStory(), ""));
    result.put("ocop_star", product.getOcopStar() == null ? 0 : product.getOcopStar());
    result.put("ocop_certificate_no", Objects.toString(product.getOcopCertificateNo(), ""));
    result.put("images", product.getImageUrls() == null ? List.of() : product.getImageUrls());
    result.put("variants", product.getVariants().stream().map(this::variant).toList());
    return result;
  }

  @GetMapping("/products/variants/{sku}")
  public Object sku(
      @PathVariable String sku,
      @RequestParam(name = "channel", required = false, defaultValue = "D2C_WEB")
          String channelParam,
      @AuthenticationPrincipal Jwt jwt,
      Authentication authentication) {
    SalesChannel channel = parseAndAuthorizeChannel(channelParam, jwt, authentication);
    return Map.of("variant", variant(catalog.getVariantBySkuCode(sku, channel)));
  }

  private Object variant(ProductVariantResponse variant) {
    if (variant.getPrice() == null)
      throw new ResponseStatusException(
          HttpStatus.SERVICE_UNAVAILABLE, "CATALOG_PRICE_UNAVAILABLE");
    Map<String, Object> value = new LinkedHashMap<>();
    value.put("variant_id", variant.getVariantId());
    value.put("sku_code", variant.getSkuCode());
    value.put("name", variant.getName());
    if (variant.getChannel() != null) {
      value.put("channel", variant.getChannel());
    }
    BigDecimal grams = variant.getWeightValue();
    if ("KILOGRAM".equals(variant.getWeightUnit()))
      grams = grams.multiply(BigDecimal.valueOf(1000));
    value.put("weight_gram", grams.longValueExact());
    value.put("price", dto(variant.getPrice()));
    if (variant.getPriceVersion() != null) {
      value.put("price_version", variant.getPriceVersion());
    }
    if (variant.getStockStatus() != null) {
      value.put("stock_status", variant.getStockStatus());
    }
    value.put("stock_available", variant.getStockAvailable());
    return value;
  }

  private SalesChannel parseAndAuthorizeChannel(
      String channelParam, Jwt jwt, Authentication authentication) {
    if (authentication == null) {
      authentication = SecurityContextHolder.getContext().getAuthentication();
    }
    if (jwt == null && authentication != null) {
      if (authentication.getPrincipal() instanceof Jwt j) {
        jwt = j;
      } else if (authentication.getCredentials() instanceof Jwt j) {
        jwt = j;
      }
    }

    SalesChannel channel;
    try {
      channel = SalesChannel.from(channelParam);
    } catch (IllegalArgumentException e) {
      throw new CatalogException(
          ErrorCode.INVALID_ARGUMENT, "Kênh bán hàng không hợp lệ: " + channelParam);
    }

    if (channel == SalesChannel.D2C_WEB || channel == SalesChannel.MOBILE_APP) {
      return channel;
    }

    if (authentication == null
        || !authentication.isAuthenticated()
        || authentication instanceof AnonymousAuthenticationToken) {
      throw new CatalogException(ErrorCode.CHANNEL_ACCESS_DENIED);
    }

    Set<String> authorities =
        authentication.getAuthorities().stream()
            .map(GrantedAuthority::getAuthority)
            .collect(Collectors.toSet());

    if (channel == SalesChannel.B2B_WHOLESALE) {
      boolean hasB2b =
          authorities.contains("ROLE_B2B")
              || authorities.contains("ROLE_B2B_CUSTOMER")
              || authorities.contains("ROLE_B2B_BUYER")
              || authorities.contains("ROLE_B2B_ADMIN")
              || authorities.contains("B2B_CUSTOMER")
              || authorities.contains("B2B_BUYER")
              || authorities.contains("B2B_ADMIN")
              || authorities.contains("B2B")
              || authorities.contains("ROLE_ADMIN")
              || authorities.contains("CATALOG_VIEW")
              || authorities.contains("CATALOG_MANAGE");

      if (!hasB2b && jwt != null) {
        List<String> roles = jwt.getClaimAsStringList("roles");
        if (roles != null
            && roles.stream()
                .anyMatch(
                    r ->
                        r.equalsIgnoreCase("b2b")
                            || r.toLowerCase().contains("b2b")
                            || r.equalsIgnoreCase("admin"))) {
          hasB2b = true;
        }
        if (!hasB2b) {
          String customerType = jwt.getClaimAsString("customer_type");
          if ("B2B".equalsIgnoreCase(customerType)) {
            hasB2b = true;
          }
        }
      }

      if (!hasB2b) {
        throw new CatalogException(ErrorCode.CHANNEL_ACCESS_DENIED);
      }
      return channel;
    }

    // POS_QUAY, SHOPEE, TIKTOK: requires staff / admin / integration authority
    boolean hasStaff =
        authorities.contains("ROLE_STAFF")
            || authorities.contains("ROLE_ADMIN")
            || authorities.contains("ROLE_INTEGRATION")
            || authorities.contains("ROLE_POS")
            || authorities.contains("CATALOG_VIEW")
            || authorities.contains("CATALOG_MANAGE")
            || authorities.contains("POS_OPERATOR");

    if (!hasStaff && jwt != null) {
      List<String> roles = jwt.getClaimAsStringList("roles");
      if (roles != null
          && roles.stream()
              .anyMatch(
                  r ->
                      r.equalsIgnoreCase("admin")
                          || r.equalsIgnoreCase("staff")
                          || r.toLowerCase().contains("integration"))) {
        hasStaff = true;
      }
    }

    if (!hasStaff) {
      throw new CatalogException(ErrorCode.CHANNEL_ACCESS_DENIED);
    }

    return channel;
  }

  private int pageSize(Integer canonical, Integer legacy) {
    if (canonical != null && legacy != null && !canonical.equals(legacy))
      throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
    int size = canonical != null ? canonical : legacy != null ? legacy : 20;
    if (size < 1 || size > 100) throw new ResponseStatusException(HttpStatus.BAD_REQUEST);
    return size;
  }

  @SuppressWarnings("unchecked")
  private Object summary(
      com.hvduong.catalog.application.dto.response.ProductListItemResponse item) {
    var value = (Map<String, Object>) dto(item);
    for (String field : List.of("summary", "category_name"))
      if (value.get(field) == null) value.put(field, "");
    if (value.get("thumbnail_url") == null)
      value.put("thumbnail_url", "https://cdn.omama.vn/products/placeholder-thumb.jpg");
    if (value.get("ocop_star") == null) value.put("ocop_star", 0);
    if (item.getBasePrice() == null)
      throw new ResponseStatusException(
          HttpStatus.SERVICE_UNAVAILABLE, "CATALOG_PRICE_UNAVAILABLE");
    return value;
  }
}
