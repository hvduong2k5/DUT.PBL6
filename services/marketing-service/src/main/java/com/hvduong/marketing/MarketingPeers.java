package com.hvduong.marketing;

import com.fasterxml.jackson.databind.JsonNode;
import java.time.Instant;
import java.util.HashSet;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.client.SimpleClientHttpRequestFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientResponseException;
import org.springframework.web.client.RestClientException;

@Component
class MarketingPeers {
    private final RestClient identity;
    private final RestClient catalog;

    MarketingPeers(@Value("${marketing.identity-url}") String identityUrl,
                   @Value("${marketing.catalog-url}") String catalogUrl) {
        var factory = new SimpleClientHttpRequestFactory();
        factory.setConnectTimeout(2000);
        factory.setReadTimeout(3000);
        identity = RestClient.builder().baseUrl(identityUrl).requestFactory(factory).build();
        catalog = RestClient.builder().baseUrl(catalogUrl).requestFactory(factory).build();
    }

    StaffActor staff(String bearer, String session, String origin) {
        var request = identity.get().uri("/api/v1/users/me");
        if (bearer != null) {
            request.header("Authorization", bearer);
        } else if (session != null) {
            request.header("Cookie", "OMAMA_SESSION=" + session);
            if (origin != null) {
                request.header("Origin", origin);
            }
        } else {
            throw new MarketingFailure("AUTHENTICATION_REQUIRED", 401);
        }
        try {
            var employee = data(request.retrieve().body(JsonNode.class));
            if (!"EMPLOYEE".equals(employee.path("principalType").asText())
                    || !"ACTIVE".equals(employee.path("provisioningStatus").asText())) {
                throw new MarketingFailure("FORBIDDEN", 403);
            }
            var grants = new HashSet<StaffActor.Grant>();
            var raw = employee.path("permissionGrants");
            if (!raw.isArray()) {
                throw unavailable();
            }
            for (var grant : raw) {
                grants.add(new StaffActor.Grant(grant.path("permissionCode").asText(),
                        grant.path("scopeType").asText(), grant.path("scopeId").asText(null)));
            }
            return new StaffActor(UUID.fromString(employee.path("principalId").asText()), grants);
        } catch (RestClientResponseException error) {
            throw switch (error.getStatusCode().value()) {
                case 401 -> new MarketingFailure("AUTHENTICATION_REQUIRED", 401);
                case 403 -> new MarketingFailure("FORBIDDEN", 403);
                default -> unavailable();
            };
        } catch (RestClientException | IllegalArgumentException error) {
            throw unavailable();
        }
    }

    ReviewStore.ProductEvidence verifyProduct(CampaignPlan.Product product, String authorization) {
        try {
            var result = data(catalog.get().uri("/api/v1/admin/catalog/products/{id}", product.productId())
                    .header("Authorization", authorization).retrieve().body(JsonNode.class));
            var revision = result.path("revision");
            if (!revision.isIntegralNumber() || !revision.canConvertToLong()
                    || revision.asLong() < 0 || revision.asLong() > 9_007_199_254_740_991L) {
                throw unavailable();
            }
            String approval = result.path("approvalStatus").asText();
            String listing = result.path("listingStatus").asText();
            if (!product.productId().toString().equals(result.path("productId").asText())
                    || !product.slug().equals(result.path("slug").asText())
                    || !"APPROVED".equals(approval) || !"ACTIVE".equals(listing)) {
                throw new MarketingFailure("VALIDATION_FAILED", 422);
            }
            return new ReviewStore.ProductEvidence(product.productId(), revision.asLong(), approval, listing, Instant.now());
        } catch (RestClientResponseException error) {
            if (error.getStatusCode().value() == 403) {
                throw new MarketingFailure("CATALOG_AUTHORIZATION_REQUIRED", 403);
            }
            if (error.getStatusCode().value() == 404) {
                throw new MarketingFailure("VALIDATION_FAILED", 422);
            }
            throw unavailable();
        } catch (RestClientException error) {
            throw unavailable();
        }
    }

    private JsonNode data(JsonNode response) {
        if (response == null || !response.path("success").asBoolean() || !response.path("data").isObject()) {
            throw unavailable();
        }
        return response.path("data");
    }

    private MarketingFailure unavailable() {
        return new MarketingFailure("SERVICE_UNAVAILABLE", 503);
    }
}
