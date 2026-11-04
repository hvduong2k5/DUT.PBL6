package com.hvduong.catalog.infrastructure.outbox;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.catalog.repository.mybatis.OutboxEventMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.*;

class CatalogOutboxWriterTest {

    private OutboxEventMapper outboxEventMapper;
    private ObjectMapper objectMapper;
    private CatalogOutboxWriter writer;

    @BeforeEach
    void setUp() {
        outboxEventMapper = mock(OutboxEventMapper.class);
        objectMapper = new ObjectMapper();
        writer = new CatalogOutboxWriter(outboxEventMapper, objectMapper);
    }

    @Test
    void enqueue_serializesDataAndCallsMapper() {
        Map<String, Object> payloadData = Map.of("title", "Test Product");

        writer.enqueue("Product", "prod-1", "ProductCreated", payloadData);

        verify(outboxEventMapper).insert(
                eq("Product"),
                eq("prod-1"),
                eq("ProductCreated"),
                eq("{\"title\":\"Test Product\"}")
        );
    }

    @Test
    void enqueue_throwsIllegalArgumentException_whenSerializationFails() throws Exception {
        ObjectMapper failingMapper = mock(ObjectMapper.class);
        when(failingMapper.writeValueAsString(any())).thenThrow(new JsonProcessingException("serialization error") {});

        CatalogOutboxWriter failingWriter = new CatalogOutboxWriter(outboxEventMapper, failingMapper);

        assertThatThrownBy(() -> failingWriter.enqueue("Product", "prod-1", "ProductCreated", new Object()))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("Could not serialize Catalog outbox payload");

        verifyNoInteractions(outboxEventMapper);
    }
}

