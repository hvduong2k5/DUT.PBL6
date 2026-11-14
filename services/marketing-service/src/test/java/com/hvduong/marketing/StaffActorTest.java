package com.hvduong.marketing;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Set;
import java.util.UUID;
import org.junit.jupiter.api.Test;

class StaffActorTest {
    @Test
    void permissionAndScopeMustComeFromTheSameGrant() {
        var actor = new StaffActor(UUID.randomUUID(), Set.of(
                new StaffActor.Grant("MARKETING_PLAN_EDIT", "DEPARTMENT", "sales"),
                new StaffActor.Grant("MARKETING_CAMPAIGN_VIEW", "DEPARTMENT", "marketing"),
                new StaffActor.Grant("MARKETING_PLAN_SUBMIT", "GLOBAL", null)));
        assertThat(actor.can("MARKETING_PLAN_EDIT", "marketing")).isFalse();
        assertThat(actor.can("MARKETING_CAMPAIGN_VIEW", "marketing")).isTrue();
        assertThat(actor.can("MARKETING_PLAN_SUBMIT", "marketing")).isTrue();
    }
}
