package com.hvduong.marketing;

import java.time.LocalDate;
import java.time.ZoneId;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Isolation;
import org.springframework.transaction.annotation.Transactional;

@Service
class ExecutionMetrics {
    private static final long MAX_VALUE = 9_007_199_254_740_991L;
    record Input(Long sourceVersion, Long impressions, Long clicks) {}
    record Day(LocalDate date, String status, Long sourceVersion, Long impressions, Long clicks) {}
    record Range(String coverage, List<Day> items) {}
    private record Window(LocalDate first, LocalDate last, ZoneId zone) {
        boolean includes(LocalDate date) {
            return !date.isBefore(first) && !date.isAfter(last);
        }
    }

    private final JdbcClient jdbc;
    private final CampaignStore campaigns;
    private final ManualExecutions executions;

    ExecutionMetrics(JdbcClient jdbc, CampaignStore campaigns, ManualExecutions executions) {
        this.jdbc = jdbc;
        this.campaigns = campaigns;
        this.executions = executions;
    }

    @Transactional(isolation = Isolation.REPEATABLE_READ, readOnly = true)
    public Range read(StaffActor actor, UUID campaignId, UUID revisionId, UUID executionId, LocalDate from, LocalDate to) {
        Window window = window(actor, campaignId, revisionId, executionId, false);
        if (!window.includes(from) || !window.includes(to) || to.isBefore(from) || to.toEpochDay() - from.toEpochDay() > 30) {
            throw invalid();
        }
        Map<LocalDate, Day> current = new HashMap<>();
        for (Day day : snapshots(executionId, from, to)) {
            current.put(day.date(), day);
        }
        var items = from.datesUntil(to.plusDays(1))
                .map(date -> current.getOrDefault(date, new Day(date, "MISSING", null, null, null))).toList();
        String coverage = current.isEmpty() ? "NOT_SYNCED" : current.size() == items.size() ? "COMPLETE" : "PARTIAL";
        return new Range(coverage, items);
    }

    @Transactional
    public Day record(StaffActor actor, UUID campaignId, UUID revisionId, UUID executionId,
                      LocalDate date, Input input, String requestId) {
        Window window = window(actor, campaignId, revisionId, executionId, true);
        if (!window.includes(date) || date.isAfter(LocalDate.now(window.zone()))
                || !valid(input.sourceVersion(), 1) || !valid(input.impressions(), 0) || !valid(input.clicks(), 0)) {
            throw invalid();
        }
        Day candidate = new Day(date, "REPORTED", input.sourceVersion(), input.impressions(), input.clicks());
        var existing = snapshots(executionId, date, date);
        if (!existing.isEmpty()) {
            Day latest = existing.getFirst();
            if (candidate.equals(latest)) {
                return latest;
            }
            if (candidate.sourceVersion() <= latest.sourceVersion()) {
                throw new MarketingFailure("STATE_CONFLICT", 409);
            }
        }
        boolean started = jdbc.sql("""
                SELECT EXISTS (SELECT 1 FROM campaign_execution_history
                               WHERE execution_id = :id AND to_state = 'ACTIVE')
                """).param("id", executionId).query(Boolean.class).single();
        if (!started) {
            throw new MarketingFailure("STATE_CONFLICT", 409);
        }
        jdbc.sql("""
                INSERT INTO campaign_execution_metrics
                    (execution_id, metric_date, source_version, impressions, clicks, actor_id, request_id)
                VALUES (:id, :date, :version, :impressions, :clicks, :actor, :request)
                """).param("id", executionId).param("date", date).param("version", input.sourceVersion())
                .param("impressions", input.impressions()).param("clicks", input.clicks())
                .param("actor", actor.id()).param("request", requestId).update();
        return candidate;
    }

    private Window window(StaffActor actor, UUID campaignId, UUID revisionId, UUID id, boolean write) {
        var campaign = campaigns.campaign(campaignId, write);
        String permission = write ? "MARKETING_CHANNEL_MANAGE" : "MARKETING_KPI_VIEW";
        if (!actor.can(permission, campaign.departmentId())) {
            throw write ? new MarketingFailure("FORBIDDEN", 403) : new MarketingFailure("RESOURCE_NOT_FOUND", 404);
        }
        var revision = campaigns.revision(campaignId, revisionId);
        var execution = executions.get(revisionId, id);
        var zone = ZoneId.of(revision.plan().timeZone());
        return new Window(execution.startsAt().atZone(zone).toLocalDate(),
                execution.endsAt().minusNanos(1).atZone(zone).toLocalDate(), zone);
    }

    private List<Day> snapshots(UUID id, LocalDate from, LocalDate to) {
        return jdbc.sql("""
                SELECT DISTINCT ON (metric_date) metric_date, source_version, impressions, clicks
                FROM campaign_execution_metrics WHERE execution_id = :id AND metric_date BETWEEN :from AND :to
                ORDER BY metric_date, source_version DESC
                """).param("id", id).param("from", from).param("to", to)
                .query((rs, row) -> new Day(rs.getObject("metric_date", LocalDate.class), "REPORTED",
                        rs.getLong("source_version"), rs.getLong("impressions"), rs.getLong("clicks"))).list();
    }

    private boolean valid(Long value, long minimum) {
        return value != null && value >= minimum && value <= MAX_VALUE;
    }

    private MarketingFailure invalid() {
        return new MarketingFailure("VALIDATION_FAILED", 422);
    }
}
