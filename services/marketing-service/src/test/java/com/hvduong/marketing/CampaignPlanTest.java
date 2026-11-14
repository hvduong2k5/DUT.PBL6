package com.hvduong.marketing;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.time.Instant;
import java.util.List;
import java.util.UUID;
import org.junit.jupiter.api.Test;

class CampaignPlanTest {
    private final Instant start = Instant.parse("2026-11-20T00:00:00Z");
    private final Instant end = start.plusSeconds(86_400);

    @Test
    void allowsAnIncompleteDraftButRequiresBriefsToSubmit() {
        var plan = plan(List.of(new CampaignPlan.Channel("EMAIL", start, end, " ")));
        assertThat(assertThrows(IllegalArgumentException.class, plan::validateForSubmission))
                .hasMessage("channels.brief");
        plan(List.of(new CampaignPlan.Channel("EMAIL", start, end, "Holiday launch"))).validateForSubmission();
    }

    @Test
    void rejectsSchedulesOutsideTheCampaignAndDuplicateChannels() {
        assertThat(assertThrows(IllegalArgumentException.class,
                () -> plan(List.of(new CampaignPlan.Channel("EMAIL", start, end.plusSeconds(1), "Launch")))))
                .hasMessage("channels.schedule");
        assertThat(assertThrows(IllegalArgumentException.class, () -> plan(List.of(
                new CampaignPlan.Channel("EMAIL", start, end, "Launch"),
                new CampaignPlan.Channel(" EMAIL ", start, end, "Follow-up")))))
                .hasMessage("channels.channel");
    }

    private CampaignPlan plan(List<CampaignPlan.Channel> channels) {
        return new CampaignPlan("Holiday campaign", "Launch", UUID.randomUUID(), start, end,
                "Asia/Ho_Chi_Minh", 100, channels, List.of(),
                List.of(new CampaignPlan.KpiTarget("CLICKS", "count", "100")));
    }
}
