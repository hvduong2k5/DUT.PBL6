package com.hvduong.identity.application.dto.response;

import java.util.List;
import java.util.UUID;

import com.fasterxml.jackson.annotation.JsonInclude;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * "Configured permissions" for a principal, recomputed from the IAM database.
 *
 * <p>This is not necessarily what a token already issued to the user carries: an
 * access token is immutable once signed and only picks up changes after the next
 * refresh.</p>
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class EffectivePermissionsResponse {
    private UUID principalId;
    private List<String> permissions;
    private String tokenNote;
}
