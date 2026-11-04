package com.hvduong.catalog.application.service;

import com.fasterxml.jackson.annotation.JsonAlias;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.catalog.infrastructure.snapshot.SkuCheckoutDetail;
import com.hvduong.catalog.repository.mybatis.CheckoutSnapshotMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Isolation;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.server.ResponseStatusException;

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

    private final CheckoutSnapshotMapper mapper;
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
            mapper.insertSnapshot(
                    UUID.fromString(snapshot.snapshot_id()),
                    json.writeValueAsString(snapshot),
                    snapshot.valid_until()
            );
        } catch (java.io.IOException error) {
            throw new IllegalStateException(error);
        }
        return snapshot;
    }

    @Transactional(isolation = Isolation.REPEATABLE_READ, readOnly = true)
    public boolean revalidate(String id) {
        UUID snapshotUuid;
        try {
            snapshotUuid = UUID.fromString(id);
        } catch (IllegalArgumentException e) {
            return false;
        }
        String stored = mapper.findValidPayloadById(snapshotUuid);
        if (stored == null) return false;
        try {
            Snapshot pinned = json.readValue(stored, Snapshot.class);
            String channel = pinned.channel() != null ? pinned.channel() : "D2C_WEB";
            var selections = pinned.items().stream()
                    .map(i -> new Selection(i.sku_code(), Math.toIntExact(i.quantity())))
                    .toList();
            Snapshot latest = current(new Request(selections, channel)).snapshot();
            return latest.catalog_version() == pinned.catalog_version() && latest.items().equals(pinned.items());
        } catch (ResponseStatusException unavailable) {
            return false;
        } catch (java.io.IOException error) {
            throw new IllegalStateException(error);
        }
    }

    private Current current(Request request) {
        String channel = request.channel() == null ? "D2C_WEB" : request.channel();
        if (!Set.of("D2C_WEB", "MOBILE_APP").contains(channel) || request.items() == null || request.items().isEmpty() || request.items().size() > 100) {
            throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "INVALID_CATALOG_SELECTION");
        }
        Instant now = mapper.getCurrentTimestamp();
        Instant expiry = now.plusSeconds(300);
        List<Item> items = new ArrayList<>();
        List<String> revisions = new ArrayList<>();
        Set<String> seen = new HashSet<>();
        long total = 0;
        for (Selection selection : request.items()) {
            if (selection.sku() == null || selection.sku().isBlank() || selection.quantity() < 1 || selection.quantity() > 999 || !seen.add(selection.sku())) {
                throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "INVALID_CATALOG_SELECTION");
            }
            List<SkuCheckoutDetail> details = mapper.findSkuCheckoutDetails(selection.sku(), channel);
            if (details.size() != 1) {
                throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "SKU_NOT_AVAILABLE");
            }
            SkuCheckoutDetail row = details.getFirst();
            long price = row.amountUnits() != null ? row.amountUnits() : 0L;
            if (!"VND".equals(row.currencyCode()) || row.amountNanos() == null || row.amountNanos() != 0 || price < 1) {
                throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "UNSUPPORTED_CATALOG_MONEY");
            }
            BigDecimal weight = row.weightValue() != null ? row.weightValue() : BigDecimal.ZERO;
            if ("KILOGRAM".equalsIgnoreCase(row.weightUnit())) {
                weight = weight.multiply(BigDecimal.valueOf(1000));
            }
            long grams = weight.longValueExact();
            if (grams < 1) {
                throw new ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "INVALID_CATALOG_WEIGHT");
            }
            Item item = new Item(selection.sku(), selection.quantity(), row.name(), price, grams, row.packagingType(), true);
            items.add(item);
            total = Math.addExact(total, Math.multiplyExact(price, selection.quantity()));
            revisions.add(item.toString() + ":" + channel + ":" + row.productVersion() + ":" + row.variantVersion() + ":" + row.priceVersion());
            if (row.effectiveTo() != null && row.effectiveTo().isBefore(expiry)) {
                expiry = row.effectiveTo();
            }
        }
        Collections.sort(revisions);
        long version;
        try {
            version = ByteBuffer.wrap(MessageDigest.getInstance("SHA-256").digest(String.join("\n", revisions).getBytes(StandardCharsets.UTF_8))).getLong() & Long.MAX_VALUE;
        } catch (java.security.NoSuchAlgorithmException error) {
            throw new IllegalStateException(error);
        }
        return new Current(new Snapshot(UUID.randomUUID().toString(), version, expiry, items, total, "catalog-checkout-1.0", channel), revisions);
    }
}
