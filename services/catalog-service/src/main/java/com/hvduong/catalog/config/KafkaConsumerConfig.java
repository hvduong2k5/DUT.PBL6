package com.hvduong.catalog.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.kafka.annotation.EnableKafka;
import org.springframework.context.annotation.Bean;
import org.springframework.kafka.listener.DefaultErrorHandler;
import org.springframework.util.backoff.FixedBackOff;

/** Kafka retry configuration for failures before a stock projection can be committed. */
@Configuration
@EnableKafka
public class KafkaConsumerConfig {

    @Bean
    DefaultErrorHandler stockProjectionErrorHandler() {
        // Keep retrying a database failure: an offset must not be committed until
        // the stock projection transaction has committed successfully.
        return new DefaultErrorHandler(new FixedBackOff(1_000L, FixedBackOff.UNLIMITED_ATTEMPTS));
    }
}
