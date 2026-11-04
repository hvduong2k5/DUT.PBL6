package com.hvduong.catalog.config;

import org.mybatis.spring.annotation.MapperScan;
import org.springframework.context.annotation.Configuration;

/**
 * Cấu hình MyBatis — scan mapper interfaces.
 * SQL XML được đặt ở {@code resources/mapper/sql/*.xml}.
 */
@Configuration
@MapperScan("com.hvduong.catalog.repository.mybatis")
public class MyBatisConfig {
    // Spring Boot auto-configure DataSource và SqlSessionFactory.
    // Class này chỉ để khai báo @MapperScan một cách tường minh.
}
