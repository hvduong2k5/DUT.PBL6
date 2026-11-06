package com.hvduong.catalog.application.service;

import com.fasterxml.jackson.annotation.JsonAlias;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Isolation;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.server.ResponseStatusException;
import org.springframework.http.HttpStatus;

import java.math.BigDecimal;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.*;

/** One database revision supplies price and all metadata used to ship the order. */
@Service
@RequiredArgsConstructor
public class CheckoutSnapshotService {
    private final JdbcTemplate database;
    private final ObjectMapper json;
    public record Selection(@JsonAlias("sku_code") String sku, int quantity) {}
    public record Request(List<Selection> items, String channel) {}
    public record Item(String sku_code, long quantity, String product_name, long unit_price,
                       long weight_grams, String packaging_type, boolean is_active) {}
    public record Snapshot(String snapshot_id, long catalog_version, Instant valid_until,
                           List<Item> items, long subtotal, String contract_version, String channel) {}
    private record Current(Snapshot snapshot, List<String> revisions) {}

    @Transactional(isolation = Isolation.REPEATABLE_READ)
    public Snapshot create(Request request) {
        Snapshot snapshot = current(request).snapshot();
        try {
            database.update("INSERT INTO checkout_snapshots(id,payload,expires_at) VALUES(?::uuid,?::jsonb,?::timestamptz)",
                    snapshot.snapshot_id(), json.writeValueAsString(snapshot), snapshot.valid_until().toString());
        } catch (java.io.IOException error) { throw new IllegalStateException(error); }
        return snapshot;
    }

    @Transactional(isolation = Isolation.REPEATABLE_READ, readOnly = true)
    public boolean revalidate(String id) {
        UUID.fromString(id);
        var stored = database.queryForList("SELECT payload::text FROM checkout_snapshots WHERE id=?::uuid AND expires_at>NOW()", String.class, id);
        if (stored.isEmpty()) return false;
        try {
            Snapshot pinned = json.readValue(stored.getFirst(), Snapshot.class);
            // Channel is stored with the snapshot envelope, never inferred from a caller's role.
            String channel = database.queryForObject("SELECT payload->>'channel' FROM checkout_snapshots WHERE id=?::uuid", String.class, id);
            if (channel == null) channel = "D2C_WEB";
            var selections = pinned.items().stream().map(i -> new Selection(i.sku_code(), Math.toIntExact(i.quantity()))).toList();
            Snapshot latest = current(new Request(selections, channel)).snapshot();
            return latest.catalog_version() == pinned.catalog_version() && latest.items().equals(pinned.items());
        } catch (ResponseStatusException unavailable) { return false; }
        catch (java.io.IOException error) { throw new IllegalStateException(error); }
    }

    private Current current(Request request) {
        String channel = request.channel() == null ? "D2C_WEB" : request.channel();
        if (!Set.of("D2C_WEB", "MOBILE_APP").contains(channel) || request.items() == null || request.items().isEmpty() || request.items().size() > 100)
            throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "INVALID_CATALOG_SELECTION");
        Instant now = database.queryForObject("SELECT CURRENT_TIMESTAMP", java.sql.Timestamp.class).toInstant();
        Instant expiry = now.plusSeconds(300);
        List<Item> items = new ArrayList<>();
        List<String> revisions = new ArrayList<>();
        Set<String> seen = new HashSet<>();
        long total = 0;
        for (Selection selection : request.items()) {
            if (selection.sku() == null || selection.sku().isBlank() || selection.quantity() < 1 || selection.quantity() > 999 || !seen.add(selection.sku()))
                throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "INVALID_CATALOG_SELECTION");
            var rows = database.queryForList("""
                SELECT p.name,p.version AS product_version,v.version AS variant_version,v.weight_value,v.weight_unit,
                       v.packaging_type,c.amount_units,c.currency_code,c.amount_nanos,c.version AS price_version,c.effective_to
                FROM product_variants v JOIN products p ON p.id=v.product_id JOIN channel_prices c ON c.variant_id=v.id
                WHERE v.sku_code=? AND v.listing_status='ACTIVE' AND p.listing_status='ACTIVE' AND p.approval_status='APPROVED'
                  AND c.channel=?::sales_channel AND c.status='ACTIVE' AND c.effective_from<=CURRENT_TIMESTAMP
                  AND (c.effective_to IS NULL OR c.effective_to>CURRENT_TIMESTAMP)
                """, selection.sku(), channel);
            if (rows.size() != 1) throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "SKU_NOT_AVAILABLE");
            var row = rows.getFirst();
            long price = ((Number) row.get("amount_units")).longValue();
            if (!"VND".equals(row.get("currency_code")) || ((Number) row.get("amount_nanos")).intValue() != 0 || price < 1)
                throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "UNSUPPORTED_CATALOG_MONEY");
            BigDecimal weight = (BigDecimal) row.get("weight_value");
            if ("KILOGRAM".equals(row.get("weight_unit"))) weight = weight.multiply(BigDecimal.valueOf(1000));
            long grams = weight.longValueExact();
            if (grams < 1) throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "INVALID_CATALOG_WEIGHT");
            Item item = new Item(selection.sku(), selection.quantity(), (String) row.get("name"), price, grams, row.get("packaging_type").toString(), true);
            items.add(item);
            total = Math.addExact(total, Math.multiplyExact(price, selection.quantity()));
            revisions.add(item.toString() + ":" + channel + ":" + row.get("product_version") + ":" + row.get("variant_version") + ":" + row.get("price_version"));
            if (row.get("effective_to") instanceof java.sql.Timestamp end && end.toInstant().isBefore(expiry)) expiry = end.toInstant();
        }
        Collections.sort(revisions);
        long version;
        try { version = ByteBuffer.wrap(MessageDigest.getInstance("SHA-256").digest(String.join("\n", revisions).getBytes(StandardCharsets.UTF_8))).getLong() & Long.MAX_VALUE; }
        catch (java.security.NoSuchAlgorithmException error) { throw new IllegalStateException(error); }
        return new Current(new Snapshot(UUID.randomUUID().toString(), version, expiry, items, total, "catalog-checkout-1.0", channel), revisions);
    }
}
