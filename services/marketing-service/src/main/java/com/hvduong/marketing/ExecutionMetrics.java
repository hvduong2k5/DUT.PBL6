package com.hvduong.marketing;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.time.LocalDate;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.HashSet;
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
    private final ReviewStore reviews;

    ExecutionMetrics(JdbcClient jdbc, CampaignStore campaigns, ManualExecutions executions, ReviewStore reviews) {
        this.jdbc = jdbc;
        this.campaigns = campaigns;
        this.executions = executions;
        this.reviews = reviews;
    }

    record Comparison(String metricCode, String unit, String status, String target, String actual,
                      String variance, String attainmentPercent) {}
    record Report(String timeZone, LocalDate throughDate, String coverage, long expectedChannelDays,
                  long reportedChannelDays, List<Comparison> items) {}
    private record Totals(long days, BigDecimal impressions, BigDecimal clicks) {}

    @Transactional(isolation = Isolation.REPEATABLE_READ, readOnly = true)
    public Report report(StaffActor actor, UUID campaignId, UUID revisionId) {
        var campaign = campaigns.campaign(campaignId, false);
        if (!actor.can("MARKETING_KPI_VIEW", campaign.departmentId())) {
            throw new MarketingFailure("RESOURCE_NOT_FOUND", 404);
        }
        campaigns.revision(campaignId, revisionId);
        var plan = reviews.approvedSnapshot(revisionId).submittedPlan();
        var zone = ZoneId.of(plan.timeZone());
        LocalDate through = LocalDate.now(zone).minusDays(1);
        long expected = plan.channels().stream().mapToLong(channel ->
                channel.endsAt().minusNanos(1).atZone(zone).toLocalDate().toEpochDay()
                        - channel.startsAt().atZone(zone).toLocalDate().toEpochDay() + 1).sum();
        Totals totals = jdbc.sql("""
                SELECT count(*) AS days, sum(impressions) AS impressions, sum(clicks) AS clicks
                FROM (
                    SELECT DISTINCT ON (m.execution_id, m.metric_date) m.impressions, m.clicks
                    FROM campaign_execution_metrics m JOIN campaign_executions e ON e.id = m.execution_id
                    WHERE e.revision_id = :revision AND m.metric_date <= :through
                    ORDER BY m.execution_id, m.metric_date, m.source_version DESC
                ) latest
                """).param("revision", revisionId).param("through", through)
                .query((rs, row) -> new Totals(rs.getLong("days"), rs.getBigDecimal("impressions"),
                        rs.getBigDecimal("clicks"))).single();
        boolean complete = totals.days() != 0 && totals.days() == expected;
        String coverage = totals.days() == 0 ? "NOT_SYNCED" : complete ? "COMPLETE" : "PARTIAL";
        var comparisons = plan.kpiTargets().stream().map(target -> compare(target, totals, complete)).toList();
        return new Report(plan.timeZone(), through, coverage, expected, totals.days(), comparisons);
    }

    private Comparison compare(CampaignPlan.KpiTarget target, Totals totals, boolean complete) {
        boolean supported = "count".equals(target.unit())
                && ("CLICKS".equals(target.metricCode()) || "IMPRESSIONS".equals(target.metricCode()));
        BigDecimal actual = supported ? ("CLICKS".equals(target.metricCode()) ? totals.clicks() : totals.impressions()) : null;
        BigDecimal goal = new BigDecimal(target.target());
        String variance = actual != null && complete ? actual.subtract(goal).stripTrailingZeros().toPlainString() : null;
        String percent = actual != null && complete && goal.signum() > 0
                ? actual.movePointRight(2).divide(goal, 2, RoundingMode.HALF_UP).toPlainString() : null;
        return new Comparison(target.metricCode(), target.unit(), supported ? "SUPPORTED" : "UNSUPPORTED",
                target.target(), actual == null ? null : actual.toPlainString(), variance, percent);
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
        if (csvKey(executionId).isPresent()) {
            throw new MarketingFailure("STATE_CONFLICT", 409);
        }
        return storeDay(actor, executionId, date, input, requestId);
    }

    private Day storeDay(StaffActor actor, UUID executionId, LocalDate date, Input input, String requestId) {
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

    private record CsvRow(String key, LocalDate date, Input input) {}

    @Transactional
    public List<Day> importCsv(StaffActor actor, UUID campaignId, UUID revisionId, UUID executionId,
                               String csv, String requestId) {
        Window window = window(actor, campaignId, revisionId, executionId, true);
        var rows = parseCsv(csv);
        LocalDate today = LocalDate.now(window.zone());
        if (rows.stream().anyMatch(row -> !window.includes(row.date()) || !row.date().isBefore(today))) {
            throw invalid();
        }
        String key = rows.getFirst().key();
        var bound = csvKey(executionId);
        if (bound.isPresent()) {
            if (!bound.get().equals(key)) {
                throw new MarketingFailure("STATE_CONFLICT", 409);
            }
        } else {
            boolean hasManualMetrics = jdbc.sql("""
                    SELECT EXISTS (SELECT 1 FROM campaign_execution_metrics WHERE execution_id = :id)
                    """).param("id", executionId).query(Boolean.class).single();
            if (hasManualMetrics) {
                throw new MarketingFailure("STATE_CONFLICT", 409);
            }
            int inserted = jdbc.sql("""
                    INSERT INTO campaign_execution_csv_sources (execution_id, external_key)
                    VALUES (:id, :key) ON CONFLICT DO NOTHING
                    """).param("id", executionId).param("key", key).update();
            if (inserted == 0) {
                throw new MarketingFailure("STATE_CONFLICT", 409);
            }
        }
        return rows.stream().map(row -> storeDay(actor, executionId, row.date(), row.input(), requestId)).toList();
    }

    private java.util.Optional<String> csvKey(UUID executionId) {
        return jdbc.sql("""
                SELECT external_key FROM campaign_execution_csv_sources WHERE execution_id = :id
                """).param("id", executionId).query(String.class).optional();
    }

    private List<CsvRow> parseCsv(String csv) {
        if (csv.length() > 8192) {
            throw invalid();
        }
        var lines = csv.replaceFirst("^\uFEFF", "").lines().toList();
        if (lines.size() < 2 || lines.size() > 32
                || !"external_execution_key,date,source_version,impressions,clicks".equals(lines.getFirst())) {
            throw invalid();
        }
        var rows = new ArrayList<CsvRow>();
        var dates = new HashSet<LocalDate>();
        String key = null;
        for (String line : lines.subList(1, lines.size())) {
            var fields = line.split(",", -1);
            if (fields.length != 5 || !fields[0].matches("[A-Za-z0-9._:-]{1,160}")
                    || (key != null && !key.equals(fields[0])) || !fields[1].matches("[0-9]{4}-[0-9]{2}-[0-9]{2}")) {
                throw invalid();
            }
            key = fields[0];
            try {
                var date = LocalDate.parse(fields[1]);
                var input = new Input(csvNumber(fields[2], 1), csvNumber(fields[3], 0), csvNumber(fields[4], 0));
                if (!dates.add(date)) {
                    throw invalid();
                }
                rows.add(new CsvRow(key, date, input));
            } catch (DateTimeParseException error) {
                throw invalid();
            }
        }
        return rows;
    }

    private long csvNumber(String field, long minimum) {
        if (!field.matches("[0-9]{1,16}")) {
            throw invalid();
        }
        long value = Long.parseLong(field);
        if (!valid(value, minimum)) {
            throw invalid();
        }
        return value;
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
