package com.hvduong.identity.dto.response;

/** A permission and the scope of the same active role assignment. */
public record PermissionGrantResponse(String permissionCode, String scopeType, String scopeId) {}
