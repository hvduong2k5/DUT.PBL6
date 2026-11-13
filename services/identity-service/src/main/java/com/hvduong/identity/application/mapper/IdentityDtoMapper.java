package com.hvduong.identity.application.mapper;

import org.mapstruct.Mapper;
import org.mapstruct.MappingConstants;

/**
 * MapStruct mapper for converting between domain entities and application DTOs.
 * Extend this interface with mapping methods as needed.
 */
@Mapper(componentModel = MappingConstants.ComponentModel.SPRING)
public interface IdentityDtoMapper {
    // Add mapping methods here as the domain model matures.
    // Example:
    // RoleDetailResponse toRoleDetailResponse(IamRoleRow row);
}
