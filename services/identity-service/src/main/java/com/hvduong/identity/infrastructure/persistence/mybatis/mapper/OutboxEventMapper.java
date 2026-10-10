package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

public interface OutboxEventMapper {
    void insert(OutboxEventRow row);
}
