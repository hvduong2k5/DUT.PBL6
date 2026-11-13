-- V11: Exclusion constraint cho khoảng hiệu lực giá kênh
-- Thay unique index uq_channel_prices_active (chỉ cho phép 1 ACTIVE tại mỗi thời điểm)
-- bằng exclusion constraint trên khoảng [effective_from, effective_to) cho cùng (variant_id, channel).
-- Cho phép nhiều giá theo thời gian mà không bị trùng khoảng hiệu lực.

CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Xóa unique index cũ
DROP INDEX IF EXISTS uq_channel_prices_active;

-- Thêm exclusion constraint: không cho 2 dòng cùng (variant_id, channel)
-- có khoảng hiệu lực overlap nhau.
ALTER TABLE channel_prices
    ADD CONSTRAINT excl_channel_prices_no_overlap
    EXCLUDE USING gist (
        variant_id WITH =,
        channel WITH =,
        tstzrange(effective_from, COALESCE(effective_to, 'infinity')) WITH &&
    );

-- Giữ index hỗ trợ query tìm giá hiệu lực
CREATE INDEX IF NOT EXISTS idx_channel_prices_variant_channel_effective
    ON channel_prices(variant_id, channel, effective_from, effective_to);

COMMENT ON CONSTRAINT excl_channel_prices_no_overlap ON channel_prices IS
    'Không cho phép 2 giá cùng variant_id và channel có khoảng [effective_from, effective_to) overlap nhau.';
