package com.hvduong.catalog.application.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.hvduong.catalog.application.dto.request.CheckoutSnapshotRequest;
import com.hvduong.catalog.infrastructure.snapshot.SkuCheckoutDetail;
import com.hvduong.catalog.repository.mybatis.CheckoutSnapshotMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpStatus;
import org.springframework.web.server.ResponseStatusException;

import java.math.BigDecimal;
import java.time.Instant;
import java.util.List;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class CheckoutSnapshotServiceTest {

    private CheckoutSnapshotMapper mapper;
    private ObjectMapper objectMapper;
    private CheckoutSnapshotService service;

    @BeforeEach
    void setUp() {
        mapper = mock(CheckoutSnapshotMapper.class);
        objectMapper = new ObjectMapper().registerModule(new JavaTimeModule());
        service = new CheckoutSnapshotService(mapper, objectMapper);
    }

    @Test
    void create_successfullyCreatesAndPersistsSnapshot() {
        when(mapper.getCurrentTimestamp()).thenReturn(Instant.now());

        SkuCheckoutDetail detail = new SkuCheckoutDetail(
                "Keo Me Xung",
                1L,
                1L,
                new BigDecimal("500"),
                "GRAM",
                "HOP_GIAY_KRAFT",
                50000L,
                "VND",
                0,
                1L,
                Instant.now().plusSeconds(3600)
        );

        when(mapper.findSkuCheckoutDetails(eq("MX-GION-500G"), eq("D2C_WEB")))
                .thenReturn(List.of(detail));

        var request = new CheckoutSnapshotRequest(
                List.of(new CheckoutSnapshotRequest.Selection("MX-GION-500G", 2)),
                "D2C_WEB"
        );

        var snapshot = service.create(request);

        assertThat(snapshot).isNotNull();
        assertThat(snapshot.subtotal()).isEqualTo(100000L);
        assertThat(snapshot.items()).hasSize(1);
        assertThat(snapshot.items().getFirst().productName()).isEqualTo("Keo Me Xung");

        verify(mapper).insertSnapshot(eq(UUID.fromString(snapshot.snapshotId())), anyString(),
                eq(snapshot.validUntil()));
    }

    @Test
    void create_throwsWhenSkuNotAvailable() {
        when(mapper.getCurrentTimestamp()).thenReturn(Instant.now());
        when(mapper.findSkuCheckoutDetails(eq("UNKNOWN-SKU"), eq("D2C_WEB")))
                .thenReturn(List.of());

        var request = new CheckoutSnapshotRequest(
                List.of(new CheckoutSnapshotRequest.Selection("UNKNOWN-SKU", 1)),
                "D2C_WEB"
        );

        assertThatThrownBy(() -> service.create(request))
                .isInstanceOf(ResponseStatusException.class)
                .extracting(e -> ((ResponseStatusException) e).getStatusCode())
                .isEqualTo(HttpStatus.UNPROCESSABLE_ENTITY);
    }

    @Test
    void revalidate_returnsFalse_whenSnapshotNotFoundOrExpired() {
        String randomId = UUID.randomUUID().toString();
        when(mapper.findValidPayloadById(UUID.fromString(randomId))).thenReturn(null);

        boolean valid = service.revalidate(randomId);

        assertThat(valid).isFalse();
    }
}

