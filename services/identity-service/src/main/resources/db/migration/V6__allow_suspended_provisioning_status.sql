-- Migration V6: Allow SUSPENDED in identity_principals provisioning_status

ALTER TABLE identity_principals 
    DROP CONSTRAINT IF EXISTS identity_principals_provisioning_status_check;

ALTER TABLE identity_principals 
    ADD CONSTRAINT identity_principals_provisioning_status_check
    CHECK (provisioning_status IN ('PENDING', 'ACTIVE', 'FAILED', 'SUSPENDED'));

