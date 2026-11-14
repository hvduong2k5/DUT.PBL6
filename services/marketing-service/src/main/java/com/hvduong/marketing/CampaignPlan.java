package com.hvduong.marketing;

import java.math.BigDecimal;
import java.time.Instant;
import java.time.ZoneId;
import java.util.HashSet;
import java.util.List;
import java.util.UUID;

record CampaignPlan(
        String name, String objective, UUID ownerId, Instant startsAt, Instant endsAt,
        String timeZone, long plannedBudgetVnd, List<Channel> channels,
        List<Product> products, List<KpiTarget> kpiTargets
) {
    record Channel(String channel, Instant startsAt, Instant endsAt, String brief) {}
    record Product(UUID productId, String slug) {}
    record KpiTarget(String metricCode, String unit, String target) {}

    CampaignPlan {
        name = requiredText(name, "name", 200);
        objective = requiredText(objective, "objective", 200);
        require(ownerId != null, "ownerId");
        require(startsAt != null && endsAt != null && endsAt.isAfter(startsAt), "endsAt");
        require(timeZone != null && ZoneId.getAvailableZoneIds().contains(timeZone), "timeZone");
        require(plannedBudgetVnd >= 0 && plannedBudgetVnd <= 9_007_199_254_740_991L, "plannedBudgetVnd");
        require(channels != null && channels.size() <= 20, "channels");
        require(products != null && products.size() <= 50, "products");
        require(kpiTargets != null && kpiTargets.size() <= 50, "kpiTargets");

        var channelCodes = new HashSet<String>();
        for (Channel channel : channels) {
            require(channel != null, "channels");
            require(channelCodes.add(requiredText(channel.channel(), "channels.channel", 200)), "channels.channel");
            require(channel.startsAt() != null && channel.endsAt() != null
                    && channel.endsAt().isAfter(channel.startsAt())
                    && !channel.startsAt().isBefore(startsAt) && !channel.endsAt().isAfter(endsAt), "channels.schedule");
            require(channel.brief() != null && channel.brief().length() <= 4000, "channels.brief");
        }
        var productIds = new HashSet<UUID>();
        for (Product product : products) {
            require(product != null && product.productId() != null && productIds.add(product.productId()), "products.productId");
            requiredText(product.slug(), "products.slug", 200);
        }
        var metricCodes = new HashSet<String>();
        for (KpiTarget target : kpiTargets) {
            require(target != null, "kpiTargets");
            require(metricCodes.add(requiredText(target.metricCode(), "kpiTargets.metricCode", 200)), "kpiTargets.metricCode");
            requiredText(target.unit(), "kpiTargets.unit", 200);
            require(target.target() != null && target.target().matches("[0-9]{1,12}(\\.[0-9]{1,6})?"), "kpiTargets.target");
        }
        channels = channels.stream().map(c -> new Channel(c.channel().strip(), c.startsAt(), c.endsAt(), c.brief().strip())).toList();
        products = products.stream().map(p -> new Product(p.productId(), p.slug().strip())).toList();
        kpiTargets = kpiTargets.stream().map(k -> new KpiTarget(k.metricCode().strip(), k.unit().strip(),
                new BigDecimal(k.target()).stripTrailingZeros().toPlainString())).toList();
    }

    void validateForSubmission() {
        require(!channels.isEmpty(), "channels");
        require(!kpiTargets.isEmpty(), "kpiTargets");
        require(channels.stream().noneMatch(c -> c.brief().isBlank()), "channels.brief");
    }

    private static String requiredText(String value, String field, int maxLength) {
        require(value != null && !value.isBlank() && value.length() <= maxLength, field);
        return value.strip();
    }

    private static void require(boolean valid, String field) {
        if (!valid) {
            throw new IllegalArgumentException(field);
        }
    }
}
