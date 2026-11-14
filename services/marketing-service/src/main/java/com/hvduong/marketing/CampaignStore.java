package com.hvduong.marketing;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.Instant;
import java.util.List;
import java.util.UUID;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

@Repository
class CampaignStore {
    private final JdbcClient jdbc;

    CampaignStore(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    record Campaign(UUID id, String departmentId, UUID currentRevisionId, long nextRevisionNumber) {}
    record Revision(UUID id, UUID campaignId, long number, long version, UUID createdBy,
                    ReviewWorkflow.State state, CampaignPlan plan) {}
    private record PlanHeader(String name, String objective, UUID owner, Instant start, Instant end,
                              String timeZone, long budget) {}

    record Summary(UUID campaignId, String departmentId, UUID currentRevisionId, long revisionNumber,
                   String state, String name, Instant updatedAt) {}

    List<Summary> list(List<String> departments, boolean global, String department, String state,
                       Instant after, UUID afterId, int limit) {
        String filter = filter(departments, global, department, state);
        String cursor = after == null ? "" : " AND (c.updated_at, c.id) < (:after, :afterId)";
        var statement = listParameters(jdbc.sql("""
                SELECT c.id, c.department_id, c.current_revision_id, r.revision_no, r.state, r.name, c.updated_at
                """ + filter + cursor + " ORDER BY c.updated_at DESC, c.id DESC LIMIT :limit"),
                departments, global, department, state).param("limit", limit);
        if (after != null) {
            statement.param("after", timestamp(after)).param("afterId", afterId);
        }
        return statement.query((rs, n) -> new Summary(uuid(rs, "id"), rs.getString("department_id"),
                uuid(rs, "current_revision_id"), rs.getLong("revision_no"), rs.getString("state"),
                rs.getString("name"), instant(rs, "updated_at"))).list();
    }

    long count(List<String> departments, boolean global, String department, String state) {
        return listParameters(jdbc.sql("SELECT count(*) " + filter(departments, global, department, state)),
                departments, global, department, state).query(Long.class).single();
    }

    private String filter(List<String> departments, boolean global, String department, String state) {
        return """
                 FROM campaigns c JOIN campaign_revisions r ON r.id = c.current_revision_id WHERE TRUE
                """ + (global ? "" : departments.isEmpty() ? " AND FALSE" : " AND c.department_id IN (:departments)")
                + (department == null ? "" : " AND c.department_id = :department")
                + (state == null ? "" : " AND r.state = :state");
    }

    private JdbcClient.StatementSpec listParameters(JdbcClient.StatementSpec statement, List<String> departments,
                                                     boolean global, String department, String state) {
        if (!global && !departments.isEmpty()) {
            statement.param("departments", departments);
        }
        if (department != null) {
            statement.param("department", department);
        }
        if (state != null) {
            statement.param("state", state);
        }
        return statement;
    }


    Campaign campaign(UUID id, boolean lock) {
        String sql = """
                SELECT id, department_id, current_revision_id, next_revision_number
                FROM campaigns WHERE id = :id
                """;
        return jdbc.sql(lock ? sql + " FOR UPDATE" : sql).param("id", id)
                .query((rs, n) -> new Campaign(uuid(rs, "id"), rs.getString("department_id"),
                        uuid(rs, "current_revision_id"), rs.getLong("next_revision_number")))
                .optional().orElseThrow(() -> new MarketingFailure("RESOURCE_NOT_FOUND", 404));
    }

    Revision revision(UUID campaignId, UUID revisionId) {
        return jdbc.sql("""
                SELECT * FROM campaign_revisions WHERE campaign_id = :campaign AND id = :id
                """).param("campaign", campaignId).param("id", revisionId)
                .query((rs, n) -> {
                    var header = new PlanHeader(rs.getString("name"), rs.getString("objective"),
                            uuid(rs, "owner_id"), instant(rs, "starts_at"), instant(rs, "ends_at"),
                            rs.getString("timezone"), rs.getLong("planned_budget_vnd"));
                    return new Revision(revisionId, campaignId, rs.getLong("revision_no"),
                            rs.getLong("version"), uuid(rs, "created_by"),
                            ReviewWorkflow.State.valueOf(rs.getString("state")), loadPlan(revisionId, header));
                }).optional().orElseThrow(() -> new MarketingFailure("RESOURCE_NOT_FOUND", 404));
    }

    void createCampaign(UUID id, String department, UUID firstRevision) {
        jdbc.sql("""
                INSERT INTO campaigns (id, department_id, current_revision_id)
                VALUES (:id, :department, :revision)
                """).param("id", id).param("department", department).param("revision", firstRevision).update();
    }

    void insertRevision(UUID campaign, UUID revision, UUID base, long number, UUID actor, CampaignPlan plan) {
        planParameters(jdbc.sql("""
                INSERT INTO campaign_revisions (id, campaign_id, base_revision_id, revision_no, created_by,
                    state, name, objective, owner_id, starts_at, ends_at, timezone, planned_budget_vnd)
                VALUES (:id, :campaign, :base, :number, :actor,
                    'DRAFT', :name, :objective, :owner, :start, :end, :timezone, :budget)
                """), revision, plan).param("campaign", campaign).param("base", base)
                .param("number", number).param("actor", actor).update();
        insertPlanChildren(revision, plan);
    }

    void replacePlan(UUID revision, CampaignPlan plan) {
        planParameters(jdbc.sql("""
                UPDATE campaign_revisions SET name = :name, objective = :objective, owner_id = :owner,
                    starts_at = :start, ends_at = :end, timezone = :timezone, planned_budget_vnd = :budget,
                    version = version + 1 WHERE id = :id
                """), revision, plan).update();
        jdbc.sql("DELETE FROM channel_plans WHERE revision_id = :id").param("id", revision).update();
        jdbc.sql("DELETE FROM product_references WHERE revision_id = :id").param("id", revision).update();
        jdbc.sql("DELETE FROM kpi_targets WHERE revision_id = :id").param("id", revision).update();
        insertPlanChildren(revision, plan);
    }

    void makeCurrent(UUID campaign, UUID revision) {
        jdbc.sql("""
                UPDATE campaigns SET current_revision_id = :revision,
                    next_revision_number = next_revision_number + 1, updated_at = clock_timestamp()
                WHERE id = :campaign
                """).param("campaign", campaign).param("revision", revision).update();
    }

    void touch(UUID campaign) {
        jdbc.sql("UPDATE campaigns SET updated_at = clock_timestamp() WHERE id = :id")
                .param("id", campaign).update();
    }

    void changeState(UUID revision, ReviewWorkflow.Transition transition) {
        jdbc.sql("UPDATE campaign_revisions SET state = :state, version = :version WHERE id = :id")
                .param("id", revision).param("state", transition.state().name())
                .param("version", transition.version()).update();
    }

    private CampaignPlan loadPlan(UUID revision, PlanHeader header) {
        List<CampaignPlan.Channel> channels = jdbc.sql("""
                SELECT * FROM channel_plans WHERE revision_id = :id ORDER BY channel
                """).param("id", revision).query((rs, n) -> new CampaignPlan.Channel(rs.getString("channel"),
                instant(rs, "starts_at"), instant(rs, "ends_at"), rs.getString("brief"))).list();
        List<CampaignPlan.Product> products = jdbc.sql("""
                SELECT * FROM product_references WHERE revision_id = :id ORDER BY product_id
                """).param("id", revision).query((rs, n) ->
                new CampaignPlan.Product(uuid(rs, "product_id"), rs.getString("slug"))).list();
        List<CampaignPlan.KpiTarget> kpis = jdbc.sql("""
                SELECT * FROM kpi_targets WHERE revision_id = :id ORDER BY metric_code
                """).param("id", revision).query((rs, n) -> new CampaignPlan.KpiTarget(rs.getString("metric_code"),
                rs.getString("unit"), rs.getBigDecimal("target").stripTrailingZeros().toPlainString())).list();
        return new CampaignPlan(header.name(), header.objective(), header.owner(), header.start(), header.end(),
                header.timeZone(), header.budget(), channels, products, kpis);
    }

    private JdbcClient.StatementSpec planParameters(JdbcClient.StatementSpec statement, UUID id, CampaignPlan plan) {
        return statement.param("id", id).param("name", plan.name()).param("objective", plan.objective())
                .param("owner", plan.ownerId()).param("start", timestamp(plan.startsAt()))
                .param("end", timestamp(plan.endsAt())).param("timezone", plan.timeZone())
                .param("budget", plan.plannedBudgetVnd());
    }

    private void insertPlanChildren(UUID id, CampaignPlan plan) {
        for (var channel : plan.channels()) {
            jdbc.sql("""
                    INSERT INTO channel_plans (revision_id, channel, starts_at, ends_at, brief)
                    VALUES (:id, :channel, :start, :end, :brief)
                    """).param("id", id).param("channel", channel.channel())
                    .param("start", timestamp(channel.startsAt())).param("end", timestamp(channel.endsAt()))
                    .param("brief", channel.brief()).update();
        }
        for (var product : plan.products()) {
            jdbc.sql("INSERT INTO product_references VALUES (:id, :product, :slug)")
                    .param("id", id).param("product", product.productId()).param("slug", product.slug()).update();
        }
        for (var kpi : plan.kpiTargets()) {
            jdbc.sql("INSERT INTO kpi_targets VALUES (:id, :metric, :unit, CAST(:target AS numeric))")
                    .param("id", id).param("metric", kpi.metricCode()).param("unit", kpi.unit())
                    .param("target", kpi.target()).update();
        }
    }

    static UUID uuid(ResultSet rs, String field) throws SQLException {
        return rs.getObject(field, UUID.class);
    }

    static Instant instant(ResultSet rs, String field) throws SQLException {
        var timestamp = rs.getTimestamp(field);
        return timestamp == null ? null : timestamp.toInstant();
    }

    private static java.time.OffsetDateTime timestamp(Instant instant) {
        return instant.atOffset(java.time.ZoneOffset.UTC);
    }
}
