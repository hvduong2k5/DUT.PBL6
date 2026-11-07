ALTER TABLE voucher_holds
    DROP CONSTRAINT IF EXISTS voucher_hold_tombstone_shape_check;

ALTER TABLE voucher_holds
    ALTER COLUMN coupon_id SET NOT NULL;
