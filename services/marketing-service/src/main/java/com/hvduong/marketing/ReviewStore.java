package com.hvduong.marketing;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.HexFormat;
import java.util.List;
import java.util.UUID;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

@Repository
class ReviewStore {
    record ProductEvidence(UUID productId, long version, String approvalStatus, String listingStatus,
                           Instant observedAt) {}
    record Snapshot(CampaignPlan submittedPlan, List<ProductEvidence> productEvidence, Long budgetCeilingVnd) {}
    record Request(UUID id, UUID submittedBy, long submittedVersion, Snapshot snapshot) {}
    record Receipt(String hash, String body, int status) {}

    private final JdbcClient jdbc;
    private final ObjectMapper json;

    ReviewStore(JdbcClient jdbc, ObjectMapper json) {
        this.jdbc = jdbc;
        this.json = json;
    }

    Request openRequest(UUID revision) {
        return jdbc.sql("""
                SELECT id, submitted_by, submitted_revision_version, snapshot::text
                FROM approval_requests WHERE revision_id = :revision AND state = 'OPEN'
                """).param("revision", revision).query((rs, n) -> new Request(
                CampaignStore.uuid(rs, "id"), CampaignStore.uuid(rs, "submitted_by"),
                rs.getLong("submitted_revision_version"), parse(rs.getString("snapshot"), Snapshot.class)))
                .optional().orElse(null);
    }

    Object latestView(UUID revision) {
        return jdbc.sql("""
                SELECT r.id, r.state, r.submitted_by, r.submitted_at, r.snapshot::text,
                       d.actor_id, d.decided_at, d.reason, d.approved_budget_vnd,
                       v.actor_id AS revoked_by, v.reason AS revoked_reason, v.occurred_at AS revoked_at
                FROM approval_requests r LEFT JOIN approval_decisions d ON d.request_id = r.id
                LEFT JOIN campaign_approval_revocations v ON v.approval_id = r.id
                WHERE r.revision_id = :revision ORDER BY r.submitted_revision_version DESC LIMIT 1
                """).param("revision", revision).query((rs, n) -> {
                    var snapshot = parse(rs.getString("snapshot"), Snapshot.class);
                    var result = new java.util.LinkedHashMap<String, Object>();
                    result.put("requestId", CampaignStore.uuid(rs, "id"));
                    result.put("state", rs.getString("state"));
                    result.put("submittedBy", CampaignStore.uuid(rs, "submitted_by"));
                    result.put("submittedAt", CampaignStore.instant(rs, "submitted_at"));
                    result.put("submittedPlan", snapshot.submittedPlan());
                    result.put("productEvidence", snapshot.productEvidence());
                    if (rs.getObject("actor_id") != null) {
                        result.put("decidedBy", CampaignStore.uuid(rs, "actor_id"));
                        result.put("decidedAt", CampaignStore.instant(rs, "decided_at"));
                    }
                    if (rs.getString("reason") != null) {
                        result.put("reason", rs.getString("reason"));
                    }
                    if (rs.getObject("approved_budget_vnd") != null) {
                        result.put("approvedBudgetVnd", rs.getLong("approved_budget_vnd"));
                    }
                    if (rs.getObject("revoked_by") != null) {
                        result.put("revocation", java.util.Map.of(
                                "actorId", CampaignStore.uuid(rs, "revoked_by"),
                                "reason", rs.getString("revoked_reason"),
                                "occurredAt", CampaignStore.instant(rs, "revoked_at")));
                    }
                    return result;
                }).optional().orElse(null);
    }


    void open(UUID campaign, UUID revision, UUID request, long version, UUID actor, Snapshot snapshot) {
        String body = encode(snapshot);
        jdbc.sql("""
                INSERT INTO approval_requests (id, campaign_id, revision_id, submitted_revision_version,
                    snapshot, snapshot_hash, policy_version, submitted_by, state)
                VALUES (:id, :campaign, :revision, :version, CAST(:body AS jsonb), :hash, 'pilot-v1', :actor, 'OPEN')
                """).param("id", request).param("campaign", campaign).param("revision", revision)
                .param("version", version).param("body", body).param("hash", hash(body))
                .param("actor", actor).update();
    }

    void close(UUID request, String state, UUID actor, String reason, Long approvedBudget) {
        jdbc.sql("UPDATE approval_requests SET state = :state WHERE id = :id AND state = 'OPEN'")
                .param("id", request).param("state", state).update();
        jdbc.sql("""
                INSERT INTO approval_decisions (request_id, action, actor_id, reason, approved_budget_vnd)
                VALUES (:id, :state, :actor, :reason, :budget)
                """).param("id", request).param("state", state).param("actor", actor)
                .param("reason", reason).param("budget", approvedBudget).update();
    }

    void audit(CampaignStore.Campaign campaign, UUID revision, UUID approval, String action, UUID actor,
               String requestId, String previous, String next, long version, String reason, List<String> changed) {
        jdbc.sql("""
                INSERT INTO audit_entries (id, campaign_id, revision_id, approval_request_id, action, actor_id,
                    department_id, request_id, from_state, to_state, revision_version, reason, changed_fields)
                VALUES (:id, :campaign, :revision, :approval, :action, :actor, :department, :request,
                    :previous, :next, :version, :reason, ARRAY(SELECT jsonb_array_elements_text(CAST(:changed AS jsonb))))
                """).param("id", UUID.randomUUID()).param("campaign", campaign.id()).param("revision", revision)
                .param("approval", approval).param("action", action).param("actor", actor)
                .param("department", campaign.departmentId()).param("request", requestId)
                .param("previous", previous).param("next", next).param("version", version)
                .param("reason", reason).param("changed", encode(changed)).update();
    }

    void lockKey(UUID actor, String department, String action, String key) {
        String lock = encode(List.of(actor, department, action, key));
        jdbc.sql("SELECT 1 FROM pg_advisory_xact_lock(hashtextextended(:lock, 0))")
                .param("lock", lock).query(Integer.class).single();
    }

    Receipt receipt(UUID actor, String department, String action, String key) {
        return jdbc.sql("""
                SELECT request_hash, response_body::text, response_status FROM idempotency_records
                WHERE actor_id = :actor AND department_id = :department AND action = :action AND idempotency_key = :key
                """).param("actor", actor).param("department", department).param("action", action).param("key", key)
                .query((rs, n) -> new Receipt(rs.getString(1), rs.getString(2), rs.getInt(3))).optional().orElse(null);
    }

    void saveReceipt(UUID actor, String department, String action, String key, Receipt receipt) {
        jdbc.sql("""
                INSERT INTO idempotency_records (actor_id, department_id, action, idempotency_key,
                    request_hash, response_body, response_status)
                VALUES (:actor, :department, :action, :key, :hash, CAST(:body AS jsonb), :status)
                """).param("actor", actor).param("department", department).param("action", action)
                .param("key", key).param("hash", receipt.hash()).param("body", receipt.body())
                .param("status", receipt.status()).update();
    }

    String encode(Object value) {
        try {
            return json.writeValueAsString(value);
        } catch (JsonProcessingException error) {
            throw new IllegalStateException("Could not encode Marketing data", error);
        }
    }

    private <T> T parse(String body, Class<T> type) {
        try {
            return json.readValue(body, type);
        } catch (JsonProcessingException error) {
            throw new IllegalStateException("Could not read persisted Marketing data", error);
        }
    }

    static String hash(String value) {
        try {
            return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256")
                    .digest(value.getBytes(StandardCharsets.UTF_8)));
        } catch (NoSuchAlgorithmException error) {
            throw new IllegalStateException(error);
        }
    }
}
