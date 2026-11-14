package com.hvduong.marketing;

import java.time.Instant;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.List;
import java.util.UUID;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
class ManualExecutions {
    enum State { SCHEDULED, ACTIVE, PAUSED, STOP_PENDING, COMPLETED, CANCELLED }
    enum Action { START, PAUSE, COMPLETE, CANCEL, CONFIRM_STOP }
    record Execution(UUID id, String channel, Instant startsAt, Instant endsAt, State state, long version) {}
    record Schedule(String channel, long expectedVersion) {}
    record Change(Action action, long expectedVersion, String reason, String evidenceRef) {}
    record Page(List<Execution> items, int total) {}

    private final JdbcClient jdbc;
    private final CampaignStore campaigns;
    private final CampaignCommands commands;
    private final ReviewStore reviews;

    ManualExecutions(JdbcClient jdbc, CampaignStore campaigns, CampaignCommands commands, ReviewStore reviews) {
        this.jdbc = jdbc;
        this.campaigns = campaigns;
        this.commands = commands;
        this.reviews = reviews;
    }

    record HistoryEntry(long version, UUID actorId, String requestId, String fromState, String toState,
                        String reason, String evidenceRef, Instant occurredAt) {}
    record HistoryPage(List<HistoryEntry> items, long total, Long nextAfterVersion) {}

    @Transactional(readOnly = true, isolation = org.springframework.transaction.annotation.Isolation.REPEATABLE_READ)
    public HistoryPage history(StaffActor actor, UUID campaignId, UUID revisionId, UUID id, long afterVersion, int limit) {
        if (afterVersion < 0 || afterVersion > 9_007_199_254_740_991L || limit < 1 || limit > 100) {
            throw new MarketingFailure("BAD_REQUEST", 400);
        }
        var campaign = campaigns.campaign(campaignId, false);
        if (!actor.can("MARKETING_CAMPAIGN_VIEW", campaign.departmentId())) {
            throw new MarketingFailure("RESOURCE_NOT_FOUND", 404);
        }
        campaigns.revision(campaignId, revisionId);
        var execution = get(revisionId, id);
        var rows = jdbc.sql("""
                SELECT version, actor_id, request_id, from_state, to_state, reason, evidence_ref, occurred_at
                FROM campaign_execution_history WHERE execution_id = :id AND version > :after
                ORDER BY version LIMIT :limit
                """).param("id", id).param("after", afterVersion).param("limit", limit + 1)
                .query((rs, row) -> new HistoryEntry(rs.getLong("version"), CampaignStore.uuid(rs, "actor_id"),
                        rs.getString("request_id"), rs.getString("from_state"), rs.getString("to_state"),
                        rs.getString("reason"), rs.getString("evidence_ref"), CampaignStore.instant(rs, "occurred_at")))
                .list();
        var items = rows.subList(0, Math.min(limit, rows.size()));
        return new HistoryPage(items, execution.version(), rows.size() > limit ? items.getLast().version() : null);
    }


    @Transactional(readOnly = true)
    public Page list(StaffActor actor, UUID campaignId, UUID revisionId) {
        var campaign = campaigns.campaign(campaignId, false);
        if (!actor.can("MARKETING_CAMPAIGN_VIEW", campaign.departmentId())) {
            throw new MarketingFailure("RESOURCE_NOT_FOUND", 404);
        }
        campaigns.revision(campaignId, revisionId);
        var rows = query("SELECT * FROM campaign_executions WHERE revision_id = :revision ORDER BY channel", revisionId).list();
        return new Page(rows, rows.size());
    }

    @Transactional
    public ReviewStore.Receipt schedule(StaffActor actor, UUID campaignId, UUID revisionId, Schedule input,
                                       String key, String requestId) {
        var campaign = authorized(actor, campaignId);
        String channel = required(input.channel());
        return commands.keyed(actor, campaign.departmentId(), "EXECUTION_SCHEDULE", key,
                List.of(campaignId, revisionId, channel, input.expectedVersion()), 201, () -> {
                    var revision = campaigns.revision(campaignId, revisionId);
                    commands.requireVersion(revision, input.expectedVersion());
                    if (!revisionId.equals(campaign.currentRevisionId()) || revision.state() != ReviewWorkflow.State.APPROVED) {
                        throw new MarketingFailure("STATE_CONFLICT", 409);
                    }
                    var approved = jdbc.sql("""
                            SELECT id FROM approval_requests WHERE revision_id = :revision AND state = 'APPROVED' AND NOT EXISTS (SELECT 1 FROM campaign_approval_revocations WHERE approval_id = approval_requests.id)
                            """).param("revision", revisionId).query(UUID.class).optional()
                            .orElseThrow(() -> new MarketingFailure("STATE_CONFLICT", 409));
                    var plan = revision.plan().channels().stream().filter(c -> c.channel().equals(channel)).findFirst()
                            .orElseThrow(() -> new MarketingFailure("VALIDATION_FAILED", 422));
                    if (!plan.endsAt().isAfter(Instant.now())) {
                        throw new MarketingFailure("VALIDATION_FAILED", 422);
                    }
                    UUID id = UUID.randomUUID();
                    int inserted = jdbc.sql("""
                            INSERT INTO campaign_executions (id,campaign_id,revision_id,approval_id,channel,starts_at,ends_at,state,version)
                            VALUES (:id,:campaign,:revision,:approval,:channel,:start,:end,'SCHEDULED',1)
                            ON CONFLICT (revision_id, channel) DO NOTHING
                            """).param("id", id).param("campaign", campaignId).param("revision", revisionId)
                            .param("approval", approved).param("channel", channel).param("start", utc(plan.startsAt()))
                            .param("end", utc(plan.endsAt())).update();
                    if (inserted == 0) {
                        throw new MarketingFailure("STATE_CONFLICT", 409);
                    }
                    var scheduled = get(revisionId, id);
                    history(scheduled, null, actor, requestId, null, null);
                    return scheduled;
                });
    }

    @Transactional
    public ReviewStore.Receipt act(StaffActor actor, UUID campaignId, UUID revisionId, UUID id, Change input,
                                  String key, String requestId) {
        var campaign = authorized(actor, campaignId);
        campaigns.revision(campaignId, revisionId);
        String reason = optional(input.reason());
        String evidence = optional(input.evidenceRef());
        if (input.action() == null || (evidence != null && !evidence.matches("[A-Za-z0-9._:/-]{1,200}"))) {
            throw new MarketingFailure("VALIDATION_FAILED", 422);
        }
        return commands.keyed(actor, campaign.departmentId(), "EXECUTION_ACTION", key,
                List.of(campaignId, revisionId, id, new Change(input.action(), input.expectedVersion(), reason, evidence)),
                200, () -> {
                    var before = get(revisionId, id);
                    if (before.version() != input.expectedVersion()) {
                        throw new MarketingFailure("STALE_REVISION", 409);
                    }
                    var next = transition(before, input.action(), reason, evidence, Instant.now());
                    jdbc.sql("UPDATE campaign_executions SET state = :state, version = version + 1 WHERE id = :id")
                            .param("state", next.name()).param("id", id).update();
                    var after = get(revisionId, id);
                    history(after, before.state(), actor, requestId, reason, evidence);
                    return after;
                });
    }

    record Revoke(long expectedVersion, UUID approvalRequestId, String reason) {}

    @Transactional
    public ReviewStore.Receipt revoke(StaffActor actor, UUID campaignId, UUID revisionId, Revoke input,
                                     String key, String requestId) {
        var campaign = campaigns.campaign(campaignId, true);
        if (!actor.can("MARKETING_PLAN_APPROVE", campaign.departmentId())) {
            throw new MarketingFailure("FORBIDDEN", 403);
        }
        String reason = required(input.reason());
        return commands.keyed(actor, campaign.departmentId(), "APPROVAL_REVOKE", key,
                List.of(campaignId, revisionId, input.approvalRequestId(), input.expectedVersion(), reason), 200, () -> {
                    var revision = campaigns.revision(campaignId, revisionId);
                    commands.requireVersion(revision, input.expectedVersion());
                    if (revision.state() != ReviewWorkflow.State.APPROVED || !revisionId.equals(campaign.currentRevisionId())) {
                        throw new MarketingFailure("STATE_CONFLICT", 409);
                    }
                    int inserted = jdbc.sql("""
                            INSERT INTO campaign_approval_revocations (approval_id,actor_id,request_id,reason)
                            SELECT id,:actor,:request,:reason FROM approval_requests
                            WHERE id = :approval AND revision_id = :revision AND state = 'APPROVED'
                            ON CONFLICT (approval_id) DO NOTHING
                            """).param("actor", actor.id()).param("request", requestId).param("reason", reason)
                            .param("approval", input.approvalRequestId()).param("revision", revisionId).update();
                    if (inserted == 0) {
                        throw new MarketingFailure("STATE_CONFLICT", 409);
                    }
                    for (var execution : query("""
                            SELECT * FROM campaign_executions WHERE revision_id = :revision
                            AND state IN ('SCHEDULED','ACTIVE','PAUSED') ORDER BY channel
                            """, revisionId).list()) {
                        State next = execution.state() == State.SCHEDULED ? State.CANCELLED : State.STOP_PENDING;
                        jdbc.sql("UPDATE campaign_executions SET state = :state, version = version + 1 WHERE id = :id")
                                .param("state", next.name()).param("id", execution.id()).update();
                        history(get(revisionId, execution.id()), execution.state(), actor, requestId, reason, null);
                    }
                    long version = revision.version() + 1;
                    campaigns.changeState(revisionId, new ReviewWorkflow.Transition(revision.state(), version, null));
                    campaigns.touch(campaignId);
                    reviews.audit(campaign, revisionId, input.approvalRequestId(), "REVOKE_APPROVAL", actor.id(),
                            requestId, revision.state().name(), revision.state().name(), version, reason, List.of("approval"));
                    return commands.view(campaign, campaigns.revision(campaignId, revisionId));
                });
    }


    static State transition(Execution execution, Action action, String reason, String evidence, Instant now) {
        State state = execution.state();
        boolean allowed = switch (action) {
            case START -> state == State.SCHEDULED || state == State.PAUSED;
            case PAUSE, COMPLETE -> state == State.ACTIVE;
            case CONFIRM_STOP -> state == State.STOP_PENDING;
            case CANCEL -> state == State.SCHEDULED || state == State.ACTIVE || state == State.PAUSED;
        };
        if (!allowed) {
            throw new MarketingFailure("STATE_CONFLICT", 409);
        }
        if ((action == Action.START && (now.isBefore(execution.startsAt()) || !now.isBefore(execution.endsAt())))
                || ((action == Action.START || action == Action.COMPLETE || action == Action.CONFIRM_STOP) && evidence == null)
                || ((action == Action.PAUSE || action == Action.CANCEL) && reason == null)) {
            throw new MarketingFailure("VALIDATION_FAILED", 422);
        }
        return switch (action) {
            case START -> State.ACTIVE;
            case PAUSE -> State.PAUSED;
            case COMPLETE -> State.COMPLETED;
            case CANCEL, CONFIRM_STOP -> State.CANCELLED;
        };
    }

    private CampaignStore.Campaign authorized(StaffActor actor, UUID campaignId) {
        var campaign = campaigns.campaign(campaignId, true);
        if (!actor.can("MARKETING_EXECUTION_MANAGE", campaign.departmentId())) {
            throw new MarketingFailure("FORBIDDEN", 403);
        }
        return campaign;
    }

    private Execution get(UUID revision, UUID id) {
        return query("SELECT * FROM campaign_executions WHERE revision_id = :revision AND id = :id", revision, id)
                .optional().orElseThrow(() -> new MarketingFailure("RESOURCE_NOT_FOUND", 404));
    }

    private JdbcClient.MappedQuerySpec<Execution> query(String sql, UUID revision, UUID... id) {
        var statement = jdbc.sql(sql).param("revision", revision);
        if (id.length > 0) {
            statement.param("id", id[0]);
        }
        return statement.query((rs, row) -> new Execution(CampaignStore.uuid(rs, "id"), rs.getString("channel"),
                CampaignStore.instant(rs, "starts_at"), CampaignStore.instant(rs, "ends_at"),
                State.valueOf(rs.getString("state")), rs.getLong("version")));
    }

    private void history(Execution execution, State previous, StaffActor actor, String requestId,
                         String reason, String evidence) {
        jdbc.sql("""
                INSERT INTO campaign_execution_history
                (execution_id,version,actor_id,request_id,from_state,to_state,reason,evidence_ref)
                VALUES (:id,:version,:actor,:request,:previous,:next,:reason,:evidence)
                """).param("id", execution.id()).param("version", execution.version()).param("actor", actor.id())
                .param("request", requestId).param("previous", previous == null ? null : previous.name())
                .param("next", execution.state().name()).param("reason", reason).param("evidence", evidence).update();
    }

    private static OffsetDateTime utc(Instant time) {
        return time.atOffset(ZoneOffset.UTC);
    }

    private String required(String value) {
        String result = optional(value);
        if (result == null) {
            throw new MarketingFailure("VALIDATION_FAILED", 422);
        }
        return result;
    }

    private String optional(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        if (value.length() > 200) {
            throw new MarketingFailure("VALIDATION_FAILED", 422);
        }
        return value.strip();
    }
}
