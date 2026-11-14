package com.hvduong.marketing;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.assertThrows;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.time.Instant;
import java.util.List;
import java.util.Set;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.simple.JdbcClient;

@SpringBootTest
@org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc
@EnabledIfEnvironmentVariable(named = "MARKETING_DB_URL", matches = ".+")
class CampaignCommandsIntegrationTest {
    @Autowired CampaignCommands commands;
    @Autowired ObjectMapper json;
    @Autowired JdbcClient jdbc;
    @Autowired org.springframework.test.web.servlet.MockMvc http;
    @org.springframework.boot.test.mock.mockito.MockBean MarketingPeers peers;

    @Test
    void createsSubmitsAndApprovesExactlyOnce() throws Exception {
        var author = new StaffActor(UUID.randomUUID(), Set.of(grant("PLAN_EDIT"), grant("BUDGET_EDIT"), grant("PLAN_SUBMIT")));
        var reviewer = new StaffActor(UUID.randomUUID(), Set.of(grant("PLAN_APPROVE")));
        var starts = Instant.parse("2026-12-01T00:00:00Z");
        var ends = starts.plusSeconds(3600);
        var plan = new CampaignPlan("Pilot", "Launch", author.id(), starts, ends, "Asia/Ho_Chi_Minh", 100,
                List.of(new CampaignPlan.Channel("EMAIL", starts, ends, "Launch brief")), List.of(),
                List.of(new CampaignPlan.KpiTarget("CLICKS", "count", "100")));
        var created = commands.create(author, "marketing", plan, "integration-create", "create");
        assertThat(commands.create(author, "marketing", plan, "integration-create", "create-retry").status()).isEqualTo(201);
        var draft = json.readTree(created.body());
        UUID campaign = UUID.fromString(draft.path("campaignId").asText());
        UUID revision = UUID.fromString(draft.path("revisionId").asText());
        var submitted = commands.review(author, campaign, revision, 1, ReviewWorkflow.Action.SUBMIT,
                null, null, "integration-submit", "submit", List.of());
        var inReview = json.readTree(submitted.body());
        UUID approval = UUID.fromString(inReview.path("approvalRequest").path("requestId").asText());
        var duplicate = commands.review(author, campaign, revision, 1, ReviewWorkflow.Action.SUBMIT,
                null, null, "integration-submit-two", "submit-retry", List.of());
        assertThat(json.readTree(duplicate.body()).path("approvalRequest").path("requestId").asText()).isEqualTo(approval.toString());
        var approved = commands.review(reviewer, campaign, revision, 2, ReviewWorkflow.Action.APPROVE,
                approval, null, "integration-approve", "approve", List.of());
        assertThat(json.readTree(approved.body()).path("state").asText()).isEqualTo("APPROVED");
        assertThrows(ReviewWorkflow.Rejected.class, () -> commands.review(author, campaign, revision, 2,
                ReviewWorkflow.Action.WITHDRAW, approval, "Late withdrawal", "integration-withdraw", "withdraw", List.of()));
        assertThat(jdbc.sql("SELECT count(*) FROM audit_entries WHERE campaign_id = :id").param("id", campaign)
                .query(Long.class).single()).isEqualTo(3);
    }

    @Test
    void checksEmployeeScopeAtHttpBoundaryAndReturnsEmptyPage() throws Exception {
        var get = org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get(
                "/api/v1/admin/marketing/campaigns");
        org.mockito.Mockito.when(peers.staff(null, null, null))
                .thenThrow(new MarketingFailure("AUTHENTICATION_REQUIRED", 401));
        http.perform(get).andExpect(org.springframework.test.web.servlet.result.MockMvcResultMatchers.status().isUnauthorized())
                .andExpect(org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath("$.requestId").isNotEmpty());

        var actor = new StaffActor(UUID.randomUUID(), Set.of(new StaffActor.Grant(
                "MARKETING_CAMPAIGN_VIEW", "DEPARTMENT", "empty-pilot")));
        org.mockito.Mockito.when(peers.staff("Bearer fixture", null, null)).thenReturn(actor);
        http.perform(org.springframework.test.web.servlet.request.MockMvcRequestBuilders
                        .get("/api/v1/admin/marketing/campaigns").header("Authorization", "Bearer fixture"))
                .andExpect(org.springframework.test.web.servlet.result.MockMvcResultMatchers.status().isOk())
                .andExpect(org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath("$.total").value(0))
                .andExpect(org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath("$.items").isEmpty());
        http.perform(org.springframework.test.web.servlet.request.MockMvcRequestBuilders
                        .get("/api/v1/admin/marketing/campaigns").header("Authorization", "Bearer fixture")
                        .param("departmentId", "other-department"))
                .andExpect(org.springframework.test.web.servlet.result.MockMvcResultMatchers.status().isForbidden());
    }


    private StaffActor.Grant grant(String code) {
        return new StaffActor.Grant("MARKETING_" + code, "DEPARTMENT", "marketing");
    }
}
