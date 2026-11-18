package com.hvduong.marketing;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;

class MarketingPeersTest {
    @Test
    void readsCatalogRevisionAndRejectsAnInactiveProduct() throws Exception {
        UUID id = UUID.randomUUID();
        var product = new CampaignPlan.Product(id, "pilot-product");
        var listing = new AtomicReference<>("ACTIVE");
        var receivedAuthorization = new AtomicReference<String>();
        var catalog = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        catalog.createContext("/api/v1/admin/catalog/products/" + id, request -> {
            try (request) {
                receivedAuthorization.set(request.getRequestHeaders().getFirst("Authorization"));
                byte[] body = """
                        {"success":true,"data":{"productId":"%s","slug":"pilot-product",
                        "revision":7,"approvalStatus":"APPROVED","listingStatus":"%s"}}
                        """.formatted(id, listing.get()).getBytes(StandardCharsets.UTF_8);
                request.getResponseHeaders().set("Content-Type", "application/json");
                request.sendResponseHeaders(200, body.length);
                request.getResponseBody().write(body);
            }
        });
        catalog.start();
        try {
            String url = "http://127.0.0.1:" + catalog.getAddress().getPort();
            var peers = new MarketingPeers(url, url);
            var evidence = peers.verifyProduct(product, "Bearer fixture");
            assertThat(receivedAuthorization.get()).isEqualTo("Bearer fixture");
            assertThat(evidence.productId()).isEqualTo(id);
            assertThat(evidence.version()).isEqualTo(7);
            listing.set("SUSPENDED");
            assertThatThrownBy(() -> peers.verifyProduct(product, "Bearer fixture"))
                    .isInstanceOfSatisfying(MarketingFailure.class, error -> {
                        assertThat(error.status()).isEqualTo(422);
                        assertThat(error.getMessage()).isEqualTo("VALIDATION_FAILED");
                    });
        } finally {
            catalog.stop(0);
        }
    }
}
