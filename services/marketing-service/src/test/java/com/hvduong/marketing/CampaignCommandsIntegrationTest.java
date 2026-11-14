package com.hvduong.marketing;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

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


    @Test
    void schedulesApprovedChannelAndRecordsManualActionsOnce() throws Exception {
        var operator = new StaffActor(UUID.randomUUID(), Set.of(grant("PLAN_EDIT"), grant("BUDGET_EDIT"),
                grant("PLAN_SUBMIT"), grant("EXECUTION_MANAGE"), grant("CAMPAIGN_VIEW")));
        var reviewer = new StaffActor(UUID.randomUUID(), Set.of(grant("PLAN_APPROVE")));
        var starts = Instant.now().minusSeconds(60);
        var ends = starts.plusSeconds(3600);
        var plan = new CampaignPlan("Execution pilot", "Launch", operator.id(), starts, ends, "Asia/Ho_Chi_Minh",
                100, List.of(new CampaignPlan.Channel("EMAIL", starts, ends, "Launch brief"),
                        new CampaignPlan.Channel("SOCIAL", starts, ends, "Social brief")), List.of(),
                List.of(new CampaignPlan.KpiTarget("CLICKS", "count", "100")));
        var draft = json.readTree(commands.create(operator, "marketing", plan, "execution-create", "create").body());
        UUID campaign = UUID.fromString(draft.path("campaignId").asText());
        UUID revision = UUID.fromString(draft.path("revisionId").asText());
        var submitted = json.readTree(commands.review(operator, campaign, revision, 1, ReviewWorkflow.Action.SUBMIT,
                null, null, "execution-submit", "submit", List.of()).body());
        UUID approval = UUID.fromString(submitted.path("approvalRequest").path("requestId").asText());
        commands.review(reviewer, campaign, revision, 2, ReviewWorkflow.Action.APPROVE, approval, null,
                "execution-approve", "approve", List.of());

        when(peers.staff("Bearer operator", null, null)).thenReturn(operator);
        var outsider = new StaffActor(UUID.randomUUID(), Set.of(new StaffActor.Grant(
                "MARKETING_CAMPAIGN_VIEW", "DEPARTMENT", "other")));
        when(peers.staff("Bearer outsider", null, null)).thenReturn(outsider);
        String path = "/api/v1/admin/marketing/campaigns/" + campaign + "/revisions/" + revision + "/executions";
        String schedule = """
                {"channel":"EMAIL","expectedVersion":3}
                """;
        var created = http.perform(post(path).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-schedule").contentType("application/json").content(schedule))
                .andExpect(status().isCreated()).andExpect(jsonPath("$.state").value("SCHEDULED"))
                .andReturn().getResponse().getContentAsString();
        UUID id = UUID.fromString(json.readTree(created).path("id").asText());
        http.perform(post(path).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-schedule").contentType("application/json").content(schedule))
                .andExpect(status().isCreated()).andExpect(jsonPath("$.id").value(id.toString()));
        http.perform(post(path).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-duplicate").contentType("application/json").content(schedule))
                .andExpect(status().isConflict());
        assertThrows(MarketingFailure.class, () -> commands.revise(operator, campaign, revision, 3,
                "execution-revise", "revise"));

        String action = path + "/" + id + "/actions";
        http.perform(post(action).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-no-evidence").contentType("application/json")
                        .content("{\"action\":\"START\",\"expectedVersion\":1}"))
                .andExpect(status().isUnprocessableEntity());
        http.perform(post(action).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-start-v1").contentType("application/json")
                        .content("{\"action\":\"START\",\"expectedVersion\":1,\"evidenceRef\":\"manual:ready\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.state").value("ACTIVE"));
        http.perform(post(action).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-stale-v1").contentType("application/json")
                        .content("{\"action\":\"COMPLETE\",\"expectedVersion\":1,\"evidenceRef\":\"manual:report\"}"))
                .andExpect(status().isConflict()).andExpect(jsonPath("$.code").value("STALE_REVISION"));
        http.perform(post(action).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-complete").contentType("application/json")
                        .content("{\"action\":\"COMPLETE\",\"expectedVersion\":2,\"evidenceRef\":\"manual:report\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.state").value("COMPLETED"));
        http.perform(get(path).header("Authorization", "Bearer operator"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.items[0].state").value("COMPLETED"));
        String history = path + "/" + id + "/history";
        http.perform(get(history).header("Authorization", "Bearer operator").param("limit", "2"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.total").value(3))
                .andExpect(jsonPath("$.nextAfterVersion").value(2))
                .andExpect(jsonPath("$.items[0].toState").value("SCHEDULED"))
                .andExpect(jsonPath("$.items[1].toState").value("ACTIVE"));
        http.perform(get(history).header("Authorization", "Bearer operator").param("afterVersion", "2"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.nextAfterVersion").value(org.hamcrest.Matchers.nullValue()))
                .andExpect(jsonPath("$.items[0].evidenceRef").value("manual:report"))
                .andExpect(jsonPath("$.items[0].actorId").value(operator.id().toString()));
        http.perform(get(history).header("Authorization", "Bearer outsider"))
                .andExpect(status().isNotFound());
        assertThat(jdbc.sql("SELECT count(*) FROM campaign_execution_history WHERE execution_id = :id")
                .param("id", id).query(Long.class).single()).isEqualTo(3);

        when(peers.staff("Bearer reviewer", null, null)).thenReturn(reviewer);
        var social = http.perform(post(path).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-social-schedule").contentType("application/json")
                        .content("""
                                {"channel":"SOCIAL","expectedVersion":3}
                                """))
                .andExpect(status().isCreated()).andReturn().getResponse().getContentAsString();
        String socialAction = path + "/" + json.readTree(social).path("id").asText() + "/actions";
        http.perform(post(socialAction).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-social-start").contentType("application/json")
                        .content("""
                                {"action":"START","expectedVersion":1,"evidenceRef":"manual:ready"}
                                """)).andExpect(status().isOk());
        String revoke = path.substring(0, path.lastIndexOf('/')) + "/revoke-approval";
        http.perform(post(revoke).header("Authorization", "Bearer reviewer")
                        .header("Idempotency-Key", "execution-revoke-approval").contentType("application/json")
                        .content(json.writeValueAsString(java.util.Map.of(
                                "expectedVersion", 3, "approvalRequestId", approval, "reason", "Campaign withdrawn"))))
                .andExpect(status().isOk()).andExpect(jsonPath("$.version").value(4))
                .andExpect(jsonPath("$.approvalRequest.revocation.actorId").value(reviewer.id().toString()));
        http.perform(get(path).header("Authorization", "Bearer operator"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.items[0].state").value("COMPLETED"))
                .andExpect(jsonPath("$.items[1].state").value("STOP_PENDING"));
        http.perform(post(socialAction).header("Authorization", "Bearer operator")
                        .header("Idempotency-Key", "execution-social-stop").contentType("application/json")
                        .content("""
                                {"action":"CONFIRM_STOP","expectedVersion":3,"evidenceRef":"manual:stopped"}
                                """)).andExpect(status().isOk()).andExpect(jsonPath("$.state").value("CANCELLED"));
    }


    private StaffActor.Grant grant(String code) {
        return new StaffActor.Grant("MARKETING_" + code, "DEPARTMENT", "marketing");
    }
}
