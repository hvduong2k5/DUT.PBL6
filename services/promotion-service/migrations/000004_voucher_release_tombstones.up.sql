ALTER TABLE voucher_holds
    ALTER COLUMN coupon_id DROP NOT NULL;

ALTER TABLE voucher_holds
    ADD CONSTRAINT voucher_hold_tombstone_shape_check CHECK (
        coupon_id IS NOT NULL OR
        (state = 'RELEASED' AND customer_id = '' AND discount_amount = 0 AND finalized_at IS NULL)
    );
