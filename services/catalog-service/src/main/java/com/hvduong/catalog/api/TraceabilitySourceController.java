package com.hvduong.catalog.api;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Map;
import java.util.UUID;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.server.ResponseStatusException;

import lombok.RequiredArgsConstructor;

@RestController
@RequiredArgsConstructor
public class TraceabilitySourceController {
  private final JdbcTemplate jdbc;

  @Value("${catalog.traceability-token:}")
  private String traceabilityToken;

  @Value("${catalog.internal-token:}")
  private String checkoutToken;

  @GetMapping("/api/v1/internal/traceability/products/{productId}/skus/{sku}")
  public Map<String, String> lookup(
      @PathVariable("productId") UUID productId,
      @PathVariable("sku") String sku,
      @RequestHeader(name = "X-Internal-Token", required = false) String token) {
    if (traceabilityToken.length() < 32
        || traceabilityToken.equals(checkoutToken)
        || token == null
        || !MessageDigest.isEqual(
            traceabilityToken.getBytes(StandardCharsets.UTF_8),
            token.getBytes(StandardCharsets.UTF_8))) {
      throw new ResponseStatusException(HttpStatus.UNAUTHORIZED);
    }
    return jdbc.query(
            """
            SELECT p.version AS product_version, v.version AS variant_version, v.id AS variant_id
            FROM products p JOIN product_variants v ON v.product_id = p.id
            WHERE p.id = ? AND v.sku_code = ?
            """,
            (rs, row) ->
                Map.of(
                    "product_id", productId.toString(),
                    "sku_code", sku,
                    "catalog_version",
                        rs.getString("variant_id")
                            + ":" + rs.getLong("product_version")
                            + ":" + rs.getLong("variant_version")),
            productId,
            sku)
        .stream()
        .findFirst()
        .orElseThrow(() -> new ResponseStatusException(HttpStatus.NOT_FOUND));
  }
}
