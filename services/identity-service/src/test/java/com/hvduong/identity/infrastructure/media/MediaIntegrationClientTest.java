package com.hvduong.identity.infrastructure.media;

import org.junit.jupiter.api.Test;

import java.util.Optional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class MediaIntegrationClientTest {

    @Test
    void disabledClient_allowsValidationAndReturnsEmptyPresignedUrl() {
        MediaIntegrationClient client = new MediaIntegrationClient(false, "localhost:9007");

        assertTrue(client.validateAndBindDocument(
                "file-1", "b2b-licenses/2026/11/25/file-1.pdf", "B2B_DOCUMENT", "Organization", "org-1"));
        assertEquals(Optional.empty(), client.getPresignedAccessUrl(
                "file-1", "admin-1", "ADMIN", "inline", 900));
        client.shutdown();
    }
}
