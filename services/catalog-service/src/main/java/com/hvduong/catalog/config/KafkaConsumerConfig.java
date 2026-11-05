package com.hvduong.catalog.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.kafka.annotation.EnableKafka;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.listener.DefaultErrorHandler;
import org.springframework.kafka.listener.DeadLetterPublishingRecoverer;
import org.springframework.util.backoff.FixedBackOff;

/**
 * Kafka retry configuration cho stock projection consumer.
 *
 * <p>Thay đổi từ retry vô hạn sang retry hạn chế (3 lần) + Dead Letter Topic.
 * - Lỗi deserialize/validation ({@link IllegalArgumentException}) gửi thẳng DLT, không retry.
 * - Lỗi DB tạm thời retry tối đa 3 lần, sau đó đưa vào DLT để poison message không chặn partition.
 */
@Configuration
@EnableKafka
public class KafkaConsumerConfig {

    @Bean
    DefaultErrorHandler stockProjectionErrorHandler(KafkaTemplate<?, ?> kafkaTemplate) {
        DeadLetterPublishingRecoverer recoverer = new DeadLetterPublishingRecoverer(kafkaTemplate);
        DefaultErrorHandler errorHandler = new DefaultErrorHandler(recoverer,
                new FixedBackOff(1_000L, 3));
        // Lỗi xác thực sự kiện không cần retry — gửi DLT ngay.
        errorHandler.addNotRetryableExceptions(IllegalArgumentException.class);
        return errorHandler;
    }
}
